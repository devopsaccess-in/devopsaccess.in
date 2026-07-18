package main

import (
	"context"
	"fmt"
	"net/smtp"
	"strings"
	"time"
)

// submission is a validated contact message.
type submission struct {
	Name    string
	Email   string
	Message string
}

// mailer sends contact submissions via an SMTP relay (Google Workspace).
type mailer struct {
	host, port, user, pass string
	from, to               string
}

func newMailer(c config) *mailer {
	return &mailer{
		host: c.smtpHost, port: c.smtpPort,
		user: c.smtpUser, pass: c.smtpPass,
		from: c.mailFrom, to: c.mailTo,
	}
}

// send relays a contact submission.
func (m *mailer) send(ctx context.Context, s submission) error {
	return m.sendRaw(ctx, m.build(s))
}

// notify sends a plain internal notification (waitlist signups, payments).
func (m *mailer) notify(ctx context.Context, replyTo, subject, body string) error {
	var b strings.Builder
	fmt.Fprintf(&b, "From: devopsaccess <%s>\r\n", m.from)
	fmt.Fprintf(&b, "To: %s\r\n", m.to)
	if replyTo != "" {
		fmt.Fprintf(&b, "Reply-To: %s\r\n", replyTo)
	}
	fmt.Fprintf(&b, "Subject: %s\r\n", subject)
	fmt.Fprintf(&b, "Date: %s\r\n", time.Now().Format(time.RFC1123Z))
	b.WriteString("MIME-Version: 1.0\r\n")
	b.WriteString("Content-Type: text/plain; charset=UTF-8\r\n\r\n")
	b.WriteString(body)
	b.WriteString("\r\n")
	return m.sendRaw(ctx, []byte(b.String()))
}

// sendRaw bounds the blocking net/smtp call with ctx (it has no context support,
// so we run it in a goroutine and wait).
func (m *mailer) sendRaw(ctx context.Context, msg []byte) error {
	// No auth when relaying through the local Postfix smarthost (empty user).
	var auth smtp.Auth
	if m.user != "" {
		auth = smtp.PlainAuth("", m.user, m.pass, m.host)
	}
	addr := m.host + ":" + m.port

	done := make(chan error, 1)
	go func() {
		done <- smtp.SendMail(addr, auth, m.from, []string{m.to}, msg)
	}()

	select {
	case <-ctx.Done():
		return fmt.Errorf("smtp send timed out: %w", ctx.Err())
	case err := <-done:
		if err != nil {
			return fmt.Errorf("smtp send: %w", err)
		}
		return nil
	}
}

func (m *mailer) build(s submission) []byte {
	var b strings.Builder
	fmt.Fprintf(&b, "From: devopsaccess contact <%s>\r\n", m.from)
	fmt.Fprintf(&b, "To: %s\r\n", m.to)
	if s.Email != "" {
		fmt.Fprintf(&b, "Reply-To: %s\r\n", s.Email)
	}
	fmt.Fprintf(&b, "Subject: New contact from %s\r\n", fallback(s.Name, "website visitor"))
	fmt.Fprintf(&b, "Date: %s\r\n", time.Now().Format(time.RFC1123Z))
	b.WriteString("MIME-Version: 1.0\r\n")
	b.WriteString("Content-Type: text/plain; charset=UTF-8\r\n")
	b.WriteString("\r\n")
	fmt.Fprintf(&b, "Name:  %s\r\n", s.Name)
	fmt.Fprintf(&b, "Email: %s\r\n\r\n", s.Email)
	b.WriteString(s.Message)
	b.WriteString("\r\n")
	return []byte(b.String())
}

func fallback(v, def string) string {
	if strings.TrimSpace(v) == "" {
		return def
	}
	return v
}
