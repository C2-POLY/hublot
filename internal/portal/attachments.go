// SPDX-License-Identifier: AGPL-3.0-or-later

package portal

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"net/http"
	"path/filepath"
	"strings"
	"time"
	"unicode"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"

	"github.com/et-technology-consulting/hublot/internal/authz"
	"github.com/et-technology-consulting/hublot/internal/db"
	"github.com/et-technology-consulting/hublot/internal/storage"
)

// Allowed types, recognised from the content itself and never from the name.
// SVG and HTML are refused: they can carry scripts.
var allowedTypes = map[string]string{
	"image/png":                 "image/png",
	"image/jpeg":                "image/jpeg",
	"image/gif":                 "image/gif",
	"image/webp":                "image/webp",
	"application/pdf":           "application/pdf",
	"text/plain; charset=utf-8": "text/plain; charset=utf-8",
}

type AttachmentView struct {
	ID          uuid.UUID `json:"id"`
	Name        string    `json:"name"`
	ContentType string    `json:"contentType"`
	Size        int64     `json:"size"`
	URL         string    `json:"url"`
}

func attachmentView(a db.Attachment) AttachmentView {
	return AttachmentView{
		ID: a.ID, Name: a.Name, ContentType: a.ContentType, Size: a.Size,
		URL: "/api/attachments/" + a.ID.String(),
	}
}

// Upload stores a file before the request is filed. Only someone who may file a
// request somewhere may upload, so storage cannot be filled by anyone signed in.
func (s *Service) Upload(ctx context.Context, sub authz.Subject, filename string, r io.Reader) (AttachmentView, error) {
	if !s.canFileSomewhere(sub) {
		return AttachmentView{}, ErrForbidden
	}
	head := make([]byte, 512)
	n, err := io.ReadFull(r, head)
	if err != nil && !errors.Is(err, io.ErrUnexpectedEOF) && !errors.Is(err, io.EOF) {
		return AttachmentView{}, err
	}
	head = head[:n]
	if n == 0 {
		return AttachmentView{}, ErrUnsupported
	}
	contentType, ok := allowedTypes[http.DetectContentType(head)]
	if !ok {
		return AttachmentView{}, ErrUnsupported
	}

	key := fmt.Sprintf("attachments/%s/%s", time.Now().UTC().Format("2006/01"), uuid.NewString())
	counter := &countingReader{r: io.LimitReader(io.MultiReader(bytes.NewReader(head), r), s.Cfg.Attachments.MaxBytes+1)}
	if err := s.Store.Put(ctx, key, counter, -1, contentType); err != nil {
		return AttachmentView{}, err
	}
	if counter.n > s.Cfg.Attachments.MaxBytes {
		_ = s.Store.Delete(ctx, key)
		return AttachmentView{}, ErrTooLarge
	}
	a, err := s.Q.CreateAttachment(ctx, db.CreateAttachmentParams{
		AuthorID: sub.UserID, ObjectKey: key, Name: cleanName(filename, contentType),
		ContentType: contentType, Size: counter.n,
	})
	if err != nil {
		_ = s.Store.Delete(ctx, key)
		return AttachmentView{}, err
	}
	return attachmentView(a), nil
}

func (s *Service) canFileSomewhere(sub authz.Subject) bool {
	for _, p := range s.activeProjects() {
		if authz.Can(sub, authz.CreateRequest, authzProject(p)) {
			return true
		}
	}
	return false
}

// OpenAttachment returns a file the subject may see: their own upload, or one linked
// to a request they may see. Anything else answers ErrNotFound.
func (s *Service) OpenAttachment(ctx context.Context, sub authz.Subject, id uuid.UUID) (db.Attachment, io.ReadCloser, error) {
	a, err := s.Q.GetAttachment(ctx, id)
	if errors.Is(err, pgx.ErrNoRows) {
		return db.Attachment{}, nil, ErrNotFound
	}
	if err != nil {
		return db.Attachment{}, nil, err
	}
	resource := authz.Attachment{AuthorID: a.AuthorID}
	if a.RequestID.Valid {
		r, err := s.Q.GetRequest(ctx, a.RequestID.UUID)
		if err != nil {
			return db.Attachment{}, nil, ErrNotFound
		}
		ar, _, ok := s.authzRequest(requestRow(r))
		if !ok {
			return db.Attachment{}, nil, ErrNotFound
		}
		resource.Request = &ar
	}
	if !authz.Can(sub, authz.ViewAttachment, resource) {
		return db.Attachment{}, nil, ErrNotFound
	}
	if resource.Request != nil && a.AuthorID != sub.UserID && authz.ViaOrgRoleOnly(sub, resource.Request.Project.ID) {
		if err := s.audit(ctx, sub.UserID, "attachment.view", "attachment", a.ID.String(), nil); err != nil {
			return db.Attachment{}, nil, err
		}
	}
	return s.open(ctx, a)
}

// DeleteAttachment removes an upload its author changed their mind about,
// as long as it is not part of a filed request.
func (s *Service) DeleteAttachment(ctx context.Context, sub authz.Subject, id uuid.UUID) error {
	key, err := s.Q.DeleteUnlinkedAttachment(ctx, db.DeleteUnlinkedAttachmentParams{ID: id, AuthorID: sub.UserID})
	if errors.Is(err, pgx.ErrNoRows) {
		return ErrNotFound
	}
	if err != nil {
		return err
	}
	return s.Store.Delete(ctx, key)
}

func (s *Service) ownUnlinked(ctx context.Context, author uuid.UUID, ids []uuid.UUID, invalid map[string]string) ([]db.Attachment, error) {
	if len(ids) > maxAttachments {
		invalid["attachments"] = "too_many"
		return nil, nil
	}
	out := make([]db.Attachment, 0, len(ids))
	seen := map[uuid.UUID]bool{}
	for _, id := range ids {
		if seen[id] {
			continue
		}
		seen[id] = true
		a, err := s.Q.GetAttachment(ctx, id)
		if errors.Is(err, pgx.ErrNoRows) || (err == nil && (a.AuthorID != author || a.RequestID.Valid)) {
			invalid["attachments"] = "invalid"
			continue
		}
		if err != nil {
			return nil, err
		}
		out = append(out, a)
	}
	return out, nil
}

// FileURL is the link written in the issue body. It opens only with a Hublot session
// allowed to see the attachment, like any other page of the portal: team members
// reading the issue on GitHub sign in to Hublot too.
func (s *Service) FileURL(id uuid.UUID) string {
	return s.Cfg.BaseURL + "/files/" + id.String()
}

func (s *Service) open(ctx context.Context, a db.Attachment) (db.Attachment, io.ReadCloser, error) {
	rc, err := s.Store.Open(ctx, a.ObjectKey)
	if errors.Is(err, storage.ErrNotFound) {
		return db.Attachment{}, nil, ErrNotFound
	}
	return a, rc, err
}

// cleanName keeps a readable base name without control characters or path parts.
func cleanName(name, contentType string) string {
	name = filepath.Base(strings.ReplaceAll(name, "\\", "/"))
	name = strings.Map(func(r rune) rune {
		if unicode.IsControl(r) || r == '"' || r == '/' {
			return -1
		}
		return r
	}, name)
	name = strings.TrimSpace(name)
	if r := []rune(name); len(r) > 120 {
		name = string(r[:120])
	}
	if name == "" || name == "." {
		name = "fichier"
		if strings.HasPrefix(contentType, "image/") {
			name = "capture"
		}
	}
	return name
}

type countingReader struct {
	r io.Reader
	n int64
}

func (c *countingReader) Read(p []byte) (int, error) {
	n, err := c.r.Read(p)
	c.n += int64(n)
	return n, err
}
