// SPDX-License-Identifier: AGPL-3.0-or-later

package portal

import (
	"context"
	"errors"
	"log/slog"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"

	"github.com/C2-POLY/hublot/internal/authz"
	"github.com/C2-POLY/hublot/internal/db"
	"github.com/C2-POLY/hublot/internal/mail"
)

// PreferencesPath is the SPA page where people choose their notifications.
const PreferencesPath = "/preferences"

// NotifyStatus e-mails a request's author that it reached status: at most once per
// (request, status), only while it still has that status, only to an author who wants
// it and may still see the request. A failed send returns the error, so the job is
// retried; it counts as sent only once the mailer accepted it.
func (s *Service) NotifyStatus(ctx context.Context, id uuid.UUID, status string) error {
	r, err := s.Q.GetRequest(ctx, id)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil
	}
	if err != nil {
		return err
	}
	skip := func(reason string) error {
		slog.InfoContext(ctx, "status e-mail not sent", "request", id, "status", status, "reason", reason)
		return nil
	}
	if r.Status != status {
		return skip("status changed since")
	}
	sent, err := s.Q.NotificationSent(ctx, db.NotificationSentParams{RequestID: id, Status: status})
	if err != nil || sent {
		return err
	}
	author, err := s.Q.GetUser(ctx, r.AuthorID)
	if err != nil {
		return err
	}
	if author.Suspended {
		return skip("author suspended")
	}
	if !author.NotifyStatus {
		return skip("author turned status e-mails off")
	}
	sub, err := s.Subject(ctx, author)
	if err != nil {
		return err
	}
	// The same rule as reading the request: never mail someone who can no longer see it.
	ar, _, ok := s.authzRequest(requestRow(r))
	if !ok || !authz.Can(sub, authz.ViewRequest, ar) {
		return skip("author can no longer see the request")
	}
	m, ok := mail.StatusChanged(author.Email, author.Lang, author.Name, r.Title, status,
		s.Cfg.BaseURL+"/demandes/"+id.String(), s.Cfg.BaseURL+PreferencesPath)
	if !ok {
		return skip("unknown status")
	}
	if err := s.Mailer.Send(ctx, m); err != nil {
		return err
	}
	return s.Q.RecordNotification(ctx, db.RecordNotificationParams{RequestID: id, Status: status})
}

// Preferences are a person's own settings.
type Preferences struct {
	NotifyStatus bool `json:"notifyStatus"`
}

// SetPreferences saves a person's preferences and returns them as stored.
func (s *Service) SetPreferences(ctx context.Context, userID uuid.UUID, p Preferences) (Preferences, error) {
	v, err := s.Q.SetUserNotifyStatus(ctx, db.SetUserNotifyStatusParams{ID: userID, NotifyStatus: p.NotifyStatus})
	if err != nil {
		return Preferences{}, err
	}
	return Preferences{NotifyStatus: v}, nil
}
