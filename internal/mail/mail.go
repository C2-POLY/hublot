// SPDX-License-Identifier: AGPL-3.0-or-later

// Package mail sends the portal's e-mails: short, like a message, one button (PRD §5 ter).
package mail

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"log/slog"
	"mime"
	"net"
	"net/smtp"
	"strconv"
	"strings"
	"sync"
	"time"
)

type Message struct {
	To      string
	Subject string
	Text    string
}

type Mailer interface {
	Send(ctx context.Context, m Message) error
}

// SMTP sends through a standard SMTP server (STARTTLS when the server offers it).
type SMTP struct {
	From     string
	Host     string
	Port     int
	Username string
	Password string
}

func (s SMTP) Send(_ context.Context, m Message) error {
	addr := net.JoinHostPort(s.Host, strconv.Itoa(s.Port))
	var auth smtp.Auth
	if s.Username != "" {
		auth = smtp.PlainAuth("", s.Username, s.Password, s.Host)
	}
	if err := smtp.SendMail(addr, auth, envelope(s.From), []string{m.To}, s.render(m)); err != nil {
		return fmt.Errorf("smtp: %w", err)
	}
	return nil
}

func (s SMTP) render(m Message) []byte {
	var b strings.Builder
	header := func(k, v string) { fmt.Fprintf(&b, "%s: %s\r\n", k, v) }
	header("From", s.From)
	header("To", m.To)
	header("Subject", mime.QEncoding.Encode("utf-8", m.Subject))
	header("Date", time.Now().Format(time.RFC1123Z))
	header("Message-ID", fmt.Sprintf("<%s@%s>", randomID(), domainOf(s.From)))
	header("MIME-Version", "1.0")
	header("Content-Type", "text/plain; charset=utf-8")
	header("Content-Transfer-Encoding", "8bit")
	b.WriteString("\r\n")
	b.WriteString(strings.ReplaceAll(m.Text, "\n", "\r\n"))
	return []byte(b.String())
}

// envelope extracts the bare address from "Name <addr>".
func envelope(from string) string {
	if i := strings.LastIndex(from, "<"); i >= 0 {
		return strings.TrimSuffix(from[i+1:], ">")
	}
	return from
}

func domainOf(from string) string {
	_, d, _ := strings.Cut(envelope(from), "@")
	if d == "" {
		return "hublot.local"
	}
	return d
}

func randomID() string {
	b := make([]byte, 12)
	_, _ = rand.Read(b)
	return hex.EncodeToString(b)
}

// Log writes e-mails to the log instead of sending them: for local development only.
// It also keeps them in memory so tests can read the magic link.
type Log struct {
	mu   sync.Mutex
	sent []Message
}

func (l *Log) Send(ctx context.Context, m Message) error {
	l.mu.Lock()
	l.sent = append(l.sent, m)
	l.mu.Unlock()
	slog.InfoContext(ctx, "mail (log driver, not sent)", "to", m.To, "subject", m.Subject, "text", m.Text)
	return nil
}

func (l *Log) Sent() []Message {
	l.mu.Lock()
	defer l.mu.Unlock()
	return append([]Message(nil), l.sent...)
}
