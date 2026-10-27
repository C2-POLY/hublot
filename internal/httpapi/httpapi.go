// SPDX-License-Identifier: AGPL-3.0-or-later

// Package httpapi is the portal's HTTP surface: the JSON API under /api (docs/api.md),
// signed attachment links under /files, the GitHub webhook, and the SPA for everything else.
package httpapi

import (
	"context"
	"encoding/json"
	"errors"
	"io/fs"
	"log/slog"
	"net"
	"net/http"
	"strings"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"

	"github.com/C2-POLY/hublot/internal/auth"
	"github.com/C2-POLY/hublot/internal/authz"
	"github.com/C2-POLY/hublot/internal/config"
	"github.com/C2-POLY/hublot/internal/db"
	"github.com/C2-POLY/hublot/internal/forge"
	"github.com/C2-POLY/hublot/internal/portal"
)

const SessionCookie = "hublot_session"

type Server struct {
	Cfg    *config.Config
	Auth   *auth.Service
	Portal *portal.Service
	// Web is the built SPA (web/dist); nil serves a short notice instead.
	Web fs.FS
}

type ctxKey int

const (
	userKey ctxKey = iota
	subjectKey
)

func (s *Server) Handler() http.Handler {
	r := chi.NewRouter()
	r.Use(middleware.RequestID)
	if s.Cfg.TrustProxyHeaders {
		// Only behind a reverse proxy that overwrites these headers: clients could forge them.
		r.Use(middleware.RealIP)
	}
	r.Use(requestLogger, middleware.Recoverer, securityHeaders)

	r.Get("/healthz", func(w http.ResponseWriter, _ *http.Request) { w.WriteHeader(http.StatusNoContent) })
	r.Get("/files/{id}", s.file)
	// Outside /api: GitHub signs its deliveries, it has no session nor CSRF header.
	r.Post("/webhooks/github", s.githubWebhook)
	if fake, ok := s.Portal.Forge.(*forge.Fake); ok && s.Cfg.Forge.Driver == "fake" {
		r.Post("/dev/fake/issues/{owner}/{repo}/{number}", s.fakeIssueChange(fake))
	}

	r.Route("/api", func(r chi.Router) {
		r.Use(csrfGuard, noStore)
		r.Post("/auth/magic-link", s.magicLink)
		r.Post("/auth/verify", s.verify)
		r.Post("/auth/logout", s.logout)

		r.Group(func(r chi.Router) {
			r.Use(s.requireSession)
			r.Get("/me", s.me)
			r.Get("/me/preferences", s.preferences)
			r.Patch("/me/preferences", s.setPreferences)
			r.Get("/projects/{slug}/forms", s.forms)
			r.Post("/attachments", s.uploadAttachment)
			r.Get("/attachments/{id}", s.getAttachment)
			r.Delete("/attachments/{id}", s.deleteAttachment)
			r.Post("/requests", s.createRequest)
			r.Get("/requests", s.listRequests)
			r.Get("/requests/{id}", s.getRequest)
			r.Get("/triage", s.triageInbox)
			r.Post("/triage/{id}/{action}", s.triage)
		})
		r.NotFound(func(w http.ResponseWriter, _ *http.Request) { writeError(w, http.StatusNotFound, "not_found", nil) })
	})

	r.Handle("/*", s.spa())
	return r
}

// csrfGuard requires a custom header on every state-changing API call. Browsers do not let
// another site set it without a CORS preflight, which this server never grants.
func csrfGuard(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet && r.Method != http.MethodHead && r.Header.Get("X-Hublot") != "1" {
			writeError(w, http.StatusForbidden, "forbidden", nil)
			return
		}
		next.ServeHTTP(w, r)
	})
}

func noStore(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Cache-Control", "no-store")
		next.ServeHTTP(w, r)
	})
}

func securityHeaders(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		h := w.Header()
		h.Set("X-Content-Type-Options", "nosniff")
		h.Set("Referrer-Policy", "same-origin")
		h.Set("X-Frame-Options", "DENY")
		next.ServeHTTP(w, r)
	})
}

func requestLogger(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()
		ww := middleware.NewWrapResponseWriter(w, r.ProtoMajor)
		next.ServeHTTP(ww, r)
		if strings.HasPrefix(r.URL.Path, "/assets/") {
			return
		}
		slog.InfoContext(r.Context(), "http", "method", r.Method, "path", r.URL.Path,
			"status", ww.Status(), "ms", time.Since(start).Milliseconds(), "id", middleware.GetReqID(r.Context()))
	})
}

// sessionUser returns the person behind the session cookie, or auth.ErrNoSession.
func (s *Server) sessionUser(w http.ResponseWriter, r *http.Request) (db.AppUser, error) {
	c, err := r.Cookie(SessionCookie)
	if err != nil {
		return db.AppUser{}, auth.ErrNoSession
	}
	user, err := s.Auth.SessionUser(r.Context(), c.Value)
	if errors.Is(err, auth.ErrNoSession) {
		s.clearCookie(w)
	}
	return user, err
}

func (s *Server) requireSession(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		user, err := s.sessionUser(w, r)
		if errors.Is(err, auth.ErrNoSession) {
			writeError(w, http.StatusUnauthorized, "unauthenticated", nil)
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
		ctx := context.WithValue(r.Context(), userKey, user)
		ctx = context.WithValue(ctx, subjectKey, sub)
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

func userFrom(r *http.Request) db.AppUser       { return r.Context().Value(userKey).(db.AppUser) }
func subjectFrom(r *http.Request) authz.Subject { return r.Context().Value(subjectKey).(authz.Subject) }

func (s *Server) setCookie(w http.ResponseWriter, value string) {
	http.SetCookie(w, &http.Cookie{
		Name: SessionCookie, Value: value, Path: "/", HttpOnly: true, Secure: s.Cfg.Secure(),
		SameSite: http.SameSiteLaxMode, MaxAge: int(s.Cfg.SessionLifetime.Seconds()),
	})
}

func (s *Server) clearCookie(w http.ResponseWriter) {
	http.SetCookie(w, &http.Cookie{
		Name: SessionCookie, Value: "", Path: "/", HttpOnly: true, Secure: s.Cfg.Secure(),
		SameSite: http.SameSiteLaxMode, MaxAge: -1,
	})
}

func clientIP(r *http.Request) string {
	host, _, err := net.SplitHostPort(r.RemoteAddr)
	if err != nil {
		return r.RemoteAddr
	}
	return host
}

func decode(w http.ResponseWriter, r *http.Request, v any) bool {
	r.Body = http.MaxBytesReader(w, r.Body, 1<<20)
	dec := json.NewDecoder(r.Body)
	if err := dec.Decode(v); err != nil {
		writeError(w, http.StatusBadRequest, "bad_request", nil)
		return false
	}
	return true
}

func writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(v)
}

type apiError struct {
	Code    string            `json:"code"`
	Message string            `json:"message,omitempty"`
	Fields  map[string]string `json:"fields,omitempty"`
}

func writeError(w http.ResponseWriter, status int, code string, fields map[string]string) {
	writeJSON(w, status, map[string]apiError{"error": {Code: code, Message: http.StatusText(status), Fields: fields}})
}

func (s *Server) internal(w http.ResponseWriter, r *http.Request, err error) {
	slog.ErrorContext(r.Context(), "internal error", "path", r.URL.Path, "err", err, "id", middleware.GetReqID(r.Context()))
	writeError(w, http.StatusInternalServerError, "internal", nil)
}

// fail maps domain errors to answers.
func (s *Server) fail(w http.ResponseWriter, r *http.Request, err error) {
	var verr *portal.ValidationError
	switch {
	case errors.As(err, &verr):
		writeError(w, http.StatusUnprocessableEntity, "validation", verr.Fields)
	case errors.Is(err, portal.ErrNotFound):
		writeError(w, http.StatusNotFound, "not_found", nil)
	case errors.Is(err, portal.ErrForbidden):
		writeError(w, http.StatusForbidden, "forbidden", nil)
	case errors.Is(err, portal.ErrTooLarge):
		writeError(w, http.StatusRequestEntityTooLarge, "too_large", nil)
	case errors.Is(err, portal.ErrUnsupported):
		writeError(w, http.StatusUnsupportedMediaType, "unsupported_type", nil)
	default:
		s.internal(w, r, err)
	}
}
