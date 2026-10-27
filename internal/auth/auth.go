// SPDX-License-Identifier: AGPL-3.0-or-later

// Package auth signs people in with a magic link: no password is ever stored (PRD §9, §10).
// Links work once, for 15 minutes, are stored hashed and rate limited.
package auth

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"errors"
	"fmt"
	"log/slog"
	"net/mail"
	"net/url"
	"strings"
	"time"
	"unicode"
	"unicode/utf8"

	"github.com/jackc/pgx/v5"

	"github.com/et-technology-consulting/hublot/internal/config"
	"github.com/et-technology-consulting/hublot/internal/db"
	hmail "github.com/et-technology-consulting/hublot/internal/mail"
)

const (
	LinkLifetime = 15 * time.Minute
	// At most this many links per address, and per IP, within LinkLifetime.
	MaxLinksPerEmail = 3
	MaxLinksPerIP    = 20
)

var (
	ErrInvalidEmail = errors.New("auth: invalid e-mail address")
	ErrRateLimited  = errors.New("auth: too many links requested")
	ErrInvalidToken = errors.New("auth: invalid or expired link")
	ErrNoSession    = errors.New("auth: no session")
)

type Service struct {
	Q      *db.Queries
	Mailer hmail.Mailer
	Cfg    *config.Config
	Now    func() time.Time
}

func (s *Service) now() time.Time {
	if s.Now != nil {
		return s.Now()
	}
	return time.Now()
}

// NormalizeEmail returns the bare lowercase address, or ErrInvalidEmail.
func NormalizeEmail(raw string) (string, error) {
	raw = strings.TrimSpace(raw)
	addr, err := mail.ParseAddress(raw)
	if err != nil || addr.Address != raw || addr.Name != "" || len(raw) > 254 {
		return "", ErrInvalidEmail
	}
	return strings.ToLower(addr.Address), nil
}

// RequestLink e-mails a sign-in link. Outside the allowed domains it does nothing and
// still returns nil, so the answer never tells whether an address is known.
func (s *Service) RequestLink(ctx context.Context, rawEmail, lang, ip string) error {
	email, err := NormalizeEmail(rawEmail)
	if err != nil {
		return err
	}
	since := s.now().Add(-LinkLifetime)
	if n, err := s.Q.CountLoginTokensByIPSince(ctx, db.CountLoginTokensByIPSinceParams{Ip: ip, CreatedAt: since}); err != nil {
		return err
	} else if n >= MaxLinksPerIP {
		return ErrRateLimited
	}
	if n, err := s.Q.CountLoginTokensByEmailSince(ctx, db.CountLoginTokensByEmailSinceParams{Email: email, CreatedAt: since}); err != nil {
		return err
	} else if n >= MaxLinksPerEmail {
		return ErrRateLimited
	}
	refused := !s.Cfg.DomainAllowed(email)
	if u, err := s.Q.GetUserByEmail(ctx, email); err == nil && u.Suspended {
		refused = true
	} else if err == nil && lang == "" {
		lang = u.Lang
	}
	lang = normalizeLang(lang)

	// A refused attempt is recorded too, already expired and never sent: otherwise the
	// rate limit would answer 429 only for allowed addresses and reveal them.
	expires := s.now().Add(LinkLifetime)
	if refused {
		expires = s.now()
	}
	token, hash := newToken()
	err = s.Q.CreateLoginToken(ctx, db.CreateLoginTokenParams{
		TokenHash: hash, Email: email, Lang: lang, Ip: ip, ExpiresAt: expires,
	})
	if err != nil {
		return err
	}
	if refused {
		slog.InfoContext(ctx, "magic link refused: domain not allowed or person suspended")
		return nil
	}
	link := s.Cfg.BaseURL + "/connexion/verifier?token=" + url.QueryEscape(token)
	return s.Mailer.Send(ctx, hmail.MagicLink(email, lang, link, int(LinkLifetime/time.Minute)))
}

// Verify consumes a link token and opens a session. It returns the raw session token for the cookie.
func (s *Service) Verify(ctx context.Context, token string) (db.AppUser, string, error) {
	if token == "" {
		return db.AppUser{}, "", ErrInvalidToken
	}
	row, err := s.Q.ConsumeLoginToken(ctx, hashToken(token))
	if errors.Is(err, pgx.ErrNoRows) {
		return db.AppUser{}, "", ErrInvalidToken
	}
	if err != nil {
		return db.AppUser{}, "", err
	}
	role := db.OrgRoleMember
	if s.Cfg.IsBootstrapOwner(row.Email) {
		role = db.OrgRoleOwner
	}
	user, err := s.Q.UpsertUserOnLogin(ctx, db.UpsertUserOnLoginParams{
		Email: row.Email, Name: defaultName(row.Email), OrgRole: role, Lang: row.Lang,
	})
	if err != nil {
		return db.AppUser{}, "", err
	}
	if user.Suspended {
		return db.AppUser{}, "", ErrInvalidToken
	}
	session, hash := newToken()
	if err := s.Q.CreateSession(ctx, db.CreateSessionParams{
		TokenHash: hash, UserID: user.ID, ExpiresAt: s.now().Add(s.Cfg.SessionLifetime),
	}); err != nil {
		return db.AppUser{}, "", err
	}
	return user, session, nil
}

// SessionUser returns the person behind a session cookie.
func (s *Service) SessionUser(ctx context.Context, session string) (db.AppUser, error) {
	if session == "" {
		return db.AppUser{}, ErrNoSession
	}
	hash := hashToken(session)
	user, err := s.Q.GetSessionUser(ctx, hash)
	if errors.Is(err, pgx.ErrNoRows) {
		return db.AppUser{}, ErrNoSession
	}
	if err != nil {
		return db.AppUser{}, err
	}
	if err := s.Q.TouchSession(ctx, hash); err != nil {
		slog.WarnContext(ctx, "touch session", "err", err)
	}
	return user, nil
}

func (s *Service) Logout(ctx context.Context, session string) error {
	if session == "" {
		return nil
	}
	return s.Q.DeleteSession(ctx, hashToken(session))
}

func newToken() (raw string, hash []byte) {
	b := make([]byte, 32)
	if _, err := rand.Read(b); err != nil {
		panic(fmt.Sprintf("crypto/rand: %v", err))
	}
	raw = base64.RawURLEncoding.EncodeToString(b)
	return raw, hashToken(raw)
}

func hashToken(raw string) []byte {
	sum := sha256.Sum256([]byte(raw))
	return sum[:]
}

func normalizeLang(lang string) string {
	if lang == "en" {
		return "en"
	}
	return "fr"
}

// defaultName turns marie.dupont@acme.fr into "Marie Dupont" until the person sets their name.
func defaultName(email string) string {
	local, _, _ := strings.Cut(email, "@")
	parts := strings.FieldsFunc(local, func(r rune) bool { return r == '.' || r == '_' || r == '-' || r == '+' })
	for i, p := range parts {
		r, size := utf8.DecodeRuneInString(p)
		parts[i] = string(unicode.ToUpper(r)) + p[size:]
	}
	return strings.Join(parts, " ")
}
