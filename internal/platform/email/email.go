// Package email defines the transactional email provider abstraction and
// concrete adapters for local development and SMTP delivery.
package email

import (
	"context"
	"crypto/tls"
	"errors"
	"fmt"
	"net"
	"net/smtp"
	"os"
	"path/filepath"
	"strings"
	"time"
)

// Message is a single transactional email.
type Message struct {
	To      string
	Subject string
	Text    string
}

// Sender delivers transactional email. Implementations must respect context
// cancellation and return classified, non-sensitive errors.
type Sender interface {
	Send(ctx context.Context, msg Message) error
}

// ErrPermanent marks an error that must not be retried.
var ErrPermanent = errors.New("permanent email delivery failure")

// LogSender writes messages to a local directory for inspection. It never
// sends real email and never logs message bodies.
type LogSender struct {
	dir string
}

func NewLogSender(dir string) (*LogSender, error) {
	if dir == "" {
		dir = ".tmp/emails"
	}
	if err := os.MkdirAll(dir, 0o750); err != nil {
		return nil, fmt.Errorf("create email directory: %w", err)
	}
	return &LogSender{dir: dir}, nil
}

func (s *LogSender) Send(ctx context.Context, msg Message) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	filename := fmt.Sprintf("%d.eml", time.Now().UnixNano())
	content := fmt.Sprintf("To: %s\nSubject: %s\nContent-Type: text/plain; charset=utf-8\n\n%s\n",
		msg.To, msg.Subject, msg.Text)
	if err := os.WriteFile(filepath.Join(s.dir, filename), []byte(content), 0o600); err != nil {
		return fmt.Errorf("write local email: %w", err)
	}
	return nil
}

// sanitizeHeader strips CR/LF so user-controlled values cannot inject SMTP
// headers or split messages.
func sanitizeHeader(value string) string {
	return strings.TrimSpace(strings.NewReplacer("\r", " ", "\n", " ").Replace(value))
}

// SMTPSender delivers mail through SMTP. When UseTLS is set, STARTTLS is
// required; the sender never silently downgrades to plaintext.
type SMTPSender struct {
	Host     string
	Port     int
	Username string
	Password string
	UseTLS   bool
	From     string
	FromName string
	Timeout  time.Duration
}

func (s *SMTPSender) Send(ctx context.Context, msg Message) error {
	addr := net.JoinHostPort(s.Host, fmt.Sprintf("%d", s.Port))
	dialer := &net.Dialer{Timeout: 10 * time.Second}
	conn, err := dialer.DialContext(ctx, "tcp", addr)
	if err != nil {
		return fmt.Errorf("smtp dial: %w", err)
	}
	defer conn.Close()

	client, err := smtp.NewClient(conn, s.Host)
	if err != nil {
		return fmt.Errorf("smtp client: %w", err)
	}
	defer client.Close()

	if s.UseTLS {
		ok, _ := client.Extension("STARTTLS")
		if !ok {
			return fmt.Errorf("%w: smtp server does not offer STARTTLS", ErrPermanent)
		}
		if err := client.StartTLS(&tls.Config{ServerName: s.Host, MinVersion: tls.VersionTLS12}); err != nil {
			return fmt.Errorf("smtp starttls: %w", err)
		}
	}
	if s.Username != "" {
		if err := client.Auth(smtp.PlainAuth("", s.Username, s.Password, s.Host)); err != nil {
			return fmt.Errorf("%w: smtp auth failed", ErrPermanent)
		}
	}
	from := sanitizeHeader(s.From)
	if from == "" {
		from = "no-reply@veyro.local"
	}
	to := sanitizeHeader(msg.To)
	if err := client.Mail(from); err != nil {
		return fmt.Errorf("%w: smtp envelope rejected", ErrPermanent)
	}
	if err := client.Rcpt(to); err != nil {
		return fmt.Errorf("%w: smtp recipient rejected", ErrPermanent)
	}
	w, err := client.Data()
	if err != nil {
		return fmt.Errorf("smtp data: %w", err)
	}
	body := strings.ReplaceAll(msg.Text, "\n", "\r\n")
	header := fmt.Sprintf("From: %s <%s>\r\nTo: %s\r\nSubject: %s\r\nMIME-Version: 1.0\r\nContent-Type: text/plain; charset=utf-8\r\n\r\n",
		sanitizeHeader(s.FromName), from, to, sanitizeHeader(msg.Subject))
	if _, err := w.Write([]byte(header + body)); err != nil {
		return fmt.Errorf("smtp write: %w", err)
	}
	if err := w.Close(); err != nil {
		return fmt.Errorf("smtp close data: %w", err)
	}
	return client.Quit()
}
