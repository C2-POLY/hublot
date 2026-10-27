// SPDX-License-Identifier: AGPL-3.0-or-later

package httpapi

import (
	"errors"
	"io"
	"mime"
	"net/http"
	"net/url"
	"strconv"
	"strings"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"

	"github.com/et-technology-consulting/hublot/internal/auth"
	"github.com/et-technology-consulting/hublot/internal/authz"
	"github.com/et-technology-consulting/hublot/internal/db"
	"github.com/et-technology-consulting/hublot/internal/portal"
)

func (s *Server) magicLink(w http.ResponseWriter, r *http.Request) {
	var in struct {
		Email string `json:"email"`
		Lang  string `json:"lang"`
	}
	if !decode(w, r, &in) {
		return
	}
	err := s.Auth.RequestLink(r.Context(), in.Email, in.Lang, clientIP(r))
	switch {
	case errors.Is(err, auth.ErrInvalidEmail):
		writeError(w, http.StatusUnprocessableEntity, "validation", map[string]string{"email": "invalid"})
	case errors.Is(err, auth.ErrRateLimited):
		writeError(w, http.StatusTooManyRequests, "rate_limited", nil)
	case err != nil:
		s.internal(w, r, err)
	default:
		writeJSON(w, http.StatusAccepted, struct{}{})
	}
}

func (s *Server) verify(w http.ResponseWriter, r *http.Request) {
	var in struct {
		Token string `json:"token"`
	}
	if !decode(w, r, &in) {
		return
	}
	user, session, err := s.Auth.Verify(r.Context(), in.Token)
	if errors.Is(err, auth.ErrInvalidToken) {
		writeError(w, http.StatusBadRequest, "invalid_token", nil)
		return
	}
	if err != nil {
		s.internal(w, r, err)
		return
	}
	s.setCookie(w, session)
	sub, err := s.Portal.Subject(r.Context(), user)
	if err != nil {
		s.internal(w, r, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"user": s.meView(user, sub)})
}

func (s *Server) logout(w http.ResponseWriter, r *http.Request) {
	if c, err := r.Cookie(SessionCookie); err == nil {
		if err := s.Auth.Logout(r.Context(), c.Value); err != nil {
			s.internal(w, r, err)
			return
		}
	}
	s.clearCookie(w)
	w.WriteHeader(http.StatusNoContent)
}

type meView struct {
	ID       uuid.UUID            `json:"id"`
	Email    string               `json:"email"`
	Name     string               `json:"name"`
	Lang     string               `json:"lang"`
	OrgRole  string               `json:"orgRole"`
	Projects []portal.ProjectView `json:"projects"`
}

func (s *Server) meView(u db.AppUser, sub authz.Subject) meView {
	return meView{
		ID: u.ID, Email: u.Email, Name: u.Name, Lang: u.Lang, OrgRole: string(u.OrgRole),
		Projects: s.Portal.VisibleProjects(sub),
	}
}

func (s *Server) me(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, http.StatusOK, s.meView(userFrom(r), subjectFrom(r)))
}

func (s *Server) forms(w http.ResponseWriter, r *http.Request) {
	forms, err := s.Portal.Forms(r.Context(), subjectFrom(r), chi.URLParam(r, "slug"))
	if err != nil {
		s.fail(w, r, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"forms": forms})
}

func (s *Server) createRequest(w http.ResponseWriter, r *http.Request) {
	var in portal.CreateInput
	if !decode(w, r, &in) {
		return
	}
	v, err := s.Portal.CreateRequest(r.Context(), subjectFrom(r), userFrom(r), in)
	if err != nil {
		s.fail(w, r, err)
		return
	}
	writeJSON(w, http.StatusCreated, v)
}

func (s *Server) listRequests(w http.ResponseWriter, r *http.Request) {
	var (
		list []portal.RequestView
		err  error
	)
	switch q := r.URL.Query(); q.Get("scope") {
	case "", "mine":
		list, err = s.Portal.ListMine(r.Context(), subjectFrom(r))
	case "all":
		if q.Get("project") == "" {
			writeError(w, http.StatusBadRequest, "bad_request", map[string]string{"project": "required"})
			return
		}
		list, err = s.Portal.ListProject(r.Context(), subjectFrom(r), q.Get("project"))
	default:
		writeError(w, http.StatusBadRequest, "bad_request", map[string]string{"scope": "invalid"})
		return
	}
	if err != nil {
		s.fail(w, r, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"requests": list})
}

func (s *Server) getRequest(w http.ResponseWriter, r *http.Request) {
	id, ok := pathID(w, r)
	if !ok {
		return
	}
	v, err := s.Portal.GetRequest(r.Context(), subjectFrom(r), id)
	if err != nil {
		s.fail(w, r, err)
		return
	}
	writeJSON(w, http.StatusOK, v)
}

func (s *Server) uploadAttachment(w http.ResponseWriter, r *http.Request) {
	// Leave room for the multipart envelope; the exact limit is checked while storing.
	r.Body = http.MaxBytesReader(w, r.Body, s.Cfg.Attachments.MaxBytes+1<<20)
	mr, err := r.MultipartReader()
	if err != nil {
		writeError(w, http.StatusBadRequest, "bad_request", nil)
		return
	}
	for {
		part, err := mr.NextPart()
		if err == io.EOF {
			writeError(w, http.StatusBadRequest, "bad_request", map[string]string{"file": "required"})
			return
		}
		var maxErr *http.MaxBytesError
		if errors.As(err, &maxErr) {
			writeError(w, http.StatusRequestEntityTooLarge, "too_large", nil)
			return
		}
		if err != nil {
			writeError(w, http.StatusBadRequest, "bad_request", nil)
			return
		}
		if part.FormName() != "file" {
			part.Close()
			continue
		}
		v, err := s.Portal.Upload(r.Context(), subjectFrom(r), part.FileName(), part)
		part.Close()
		if errors.As(err, &maxErr) {
			writeError(w, http.StatusRequestEntityTooLarge, "too_large", nil)
			return
		}
		if err != nil {
			s.fail(w, r, err)
			return
		}
		writeJSON(w, http.StatusCreated, v)
		return
	}
}

func (s *Server) getAttachment(w http.ResponseWriter, r *http.Request) {
	id, ok := pathID(w, r)
	if !ok {
		return
	}
	a, rc, err := s.Portal.OpenAttachment(r.Context(), subjectFrom(r), id)
	if err != nil {
		s.fail(w, r, err)
		return
	}
	defer rc.Close()
	serveFile(w, a, rc)
	// Access can be withdrawn: never let the browser keep a copy.
	w.Header().Set("Cache-Control", "no-store")
}

func (s *Server) deleteAttachment(w http.ResponseWriter, r *http.Request) {
	id, ok := pathID(w, r)
	if !ok {
		return
	}
	if err := s.Portal.DeleteAttachment(r.Context(), subjectFrom(r), id); err != nil {
		s.fail(w, r, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

// file serves the attachment links written in issue bodies. Signed out, it sends the
// person to sign in and back here; signed in, the usual rules apply.
func (s *Server) file(w http.ResponseWriter, r *http.Request) {
	id, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		http.NotFound(w, r)
		return
	}
	user, err := s.sessionUser(w, r)
	if errors.Is(err, auth.ErrNoSession) {
		http.Redirect(w, r, "/connexion?next="+url.QueryEscape("/files/"+id.String()), http.StatusFound)
		return
	}
	if err != nil {
		s.internal(w, r, err)
		return
	}
	sub, err := s.Portal.Subject(r.Context(), user)
	if err != nil {
		s.internal(w, r, err)
		return
	}
	a, rc, err := s.Portal.OpenAttachment(r.Context(), sub, id)
	if errors.Is(err, portal.ErrNotFound) {
		http.NotFound(w, r)
		return
	}
	if err != nil {
		s.internal(w, r, err)
		return
	}
	defer rc.Close()
	serveFile(w, a, rc)
	w.Header().Set("Cache-Control", "no-store")
}

// serveFile sends a stored file so that it can never run as a page of the portal.
func serveFile(w http.ResponseWriter, a db.Attachment, rc io.Reader) {
	h := w.Header()
	h.Set("Content-Type", a.ContentType)
	h.Set("Content-Length", strconv.FormatInt(a.Size, 10))
	h.Set("Content-Security-Policy", "default-src 'none'; img-src 'self'; style-src 'unsafe-inline'; sandbox")
	h.Set("Cache-Control", "private, max-age=300")
	disposition := "attachment"
	if strings.HasPrefix(a.ContentType, "image/") || a.ContentType == "application/pdf" {
		disposition = "inline"
	}
	h.Set("Content-Disposition", mime.FormatMediaType(disposition, map[string]string{"filename": a.Name}))
	_, _ = io.Copy(w, rc)
}

func pathID(w http.ResponseWriter, r *http.Request) (uuid.UUID, bool) {
	id, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		writeError(w, http.StatusNotFound, "not_found", nil)
		return uuid.Nil, false
	}
	return id, true
}
