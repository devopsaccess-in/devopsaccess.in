// Package notify sends alert notifications: email through the VM's local
// Postfix relay (no auth) and Slack through customer-supplied incoming
// webhooks. Used by the scheduler for incident alerts and by the API for the
// channel-test endpoint. Slack sends must use an SSRF-guarded client
// (safehttp.Client) because the webhook URL is customer input.
package notify

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/smtp"
	"strings"
	"time"
)

// Mailer sends plain-text mail via an SMTP relay. Empty User means no auth
// (the local Postfix smarthost case).
type Mailer struct {
	Host string
	Port string
	From string
	User string
	Pass string
}

// Send delivers a plain-text message, bounded by ctx (net/smtp has no context
// support, so the blocking call runs in a goroutine — contact service
// precedent). Header fields (to, subject) are stripped of CR/LF so
// user-controlled content (e.g. a monitor name in the subject) can never
// inject additional SMTP headers or recipients.
func (m *Mailer) Send(ctx context.Context, to, subject, body string) error {
	to = stripHeaderChars(to)
	subject = stripHeaderChars(subject)

	var b strings.Builder
	fmt.Fprintf(&b, "From: DevOps Access Alerts <%s>\r\n", m.From)
	fmt.Fprintf(&b, "To: %s\r\n", to)
	fmt.Fprintf(&b, "Subject: %s\r\n", subject)
	fmt.Fprintf(&b, "Date: %s\r\n", time.Now().Format(time.RFC1123Z))
	b.WriteString("MIME-Version: 1.0\r\n")
	b.WriteString("Content-Type: text/plain; charset=UTF-8\r\n\r\n")
	b.WriteString(body)
	b.WriteString("\r\n")

	var auth smtp.Auth
	if m.User != "" {
		auth = smtp.PlainAuth("", m.User, m.Pass, m.Host)
	}
	addr := m.Host + ":" + m.Port

	done := make(chan error, 1)
	go func() {
		done <- smtp.SendMail(addr, auth, m.From, []string{to}, []byte(b.String()))
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

// stripHeaderChars removes CR and LF so a value placed in an email header
// cannot terminate the header and inject new ones (header injection).
func stripHeaderChars(s string) string {
	return strings.NewReplacer("\r", "", "\n", "").Replace(s)
}

// Slack posts a simple text message to an incoming-webhook URL.
func Slack(ctx context.Context, client *http.Client, webhookURL, text string) error {
	payload, err := json.Marshal(map[string]string{"text": text})
	if err != nil {
		return fmt.Errorf("marshal slack payload: %w", err)
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, webhookURL, bytes.NewReader(payload))
	if err != nil {
		return fmt.Errorf("build slack request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := client.Do(req)
	if err != nil {
		return fmt.Errorf("post slack webhook: %w", err)
	}
	defer resp.Body.Close()
	_, _ = io.Copy(io.Discard, io.LimitReader(resp.Body, 4096))
	if resp.StatusCode >= 300 {
		return fmt.Errorf("slack webhook returned status %d", resp.StatusCode)
	}
	return nil
}
