package main

import (
	"bytes"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/rs/zerolog"
)

func TestValidate(t *testing.T) {
	cases := []struct {
		name string
		req  contactRequest
		ok   bool
	}{
		{"good", contactRequest{Name: "Vikram", Email: "v@example.com", Message: "hi"}, true},
		{"missing email", contactRequest{Message: "hi"}, false},
		{"bad email", contactRequest{Email: "nope", Message: "hi"}, false},
		{"missing message", contactRequest{Email: "v@example.com"}, false},
		{"whitespace message", contactRequest{Email: "v@example.com", Message: "   "}, false},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			if _, ok := validate(c.req); ok != c.ok {
				t.Fatalf("validate(%+v) ok=%v, want %v", c.req, ok, c.ok)
			}
		})
	}
}

// honeypot submissions return 200 but must not attempt to send mail.
func TestHoneypotDropsSilently(t *testing.T) {
	h := &contactHandler{
		mailer:  newMailer(config{smtpHost: "x", smtpPort: "1", smtpUser: "u", smtpPass: "p", mailFrom: "f", mailTo: "t"}),
		limiter: newIPLimiter(100, 100),
		log:     zerolog.Nop(),
	}
	body := `{"email":"v@example.com","message":"hi","website":"http://spam"}`
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/api/contact", bytes.NewBufferString(body))
	h.handle(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("honeypot status = %d, want 200", rec.Code)
	}
}

func TestRateLimit(t *testing.T) {
	l := newIPLimiter(1, 1) // 1/hour, burst 1
	if !l.allow("1.2.3.4") {
		t.Fatal("first request should be allowed")
	}
	if l.allow("1.2.3.4") {
		t.Fatal("second request from same IP should be blocked")
	}
	if !l.allow("5.6.7.8") {
		t.Fatal("different IP should be allowed")
	}
}

func TestClientIPPrefersRealIP(t *testing.T) {
	req := httptest.NewRequest(http.MethodPost, "/", nil)
	req.Header.Set("X-Real-IP", "9.9.9.9")
	req.RemoteAddr = "10.0.0.1:5555"
	if got := clientIP(req); got != "9.9.9.9" {
		t.Fatalf("clientIP = %q, want 9.9.9.9", got)
	}
}
