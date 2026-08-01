package main

import (
	"strings"
	"testing"
	"time"
)

func TestCertRung(t *testing.T) {
	day := 24 * time.Hour
	tests := []struct {
		name     string
		left     time.Duration
		warned   int
		wantRung int
		wantOwed bool
	}{
		{"plenty of time", 60 * day, 0, 0, false},
		{"just outside the window", 15 * day, 0, 0, false},
		{"enters 14-day window", 13 * day, 0, certWarnDays, true},
		{"14-day already sent", 13 * day, certWarnDays, 0, false},
		{"enters 3-day window from unwarned", 2 * day, 0, certUrgentDays, true},
		{"enters 3-day window after 14-day warning", 2 * day, certWarnDays, certUrgentDays, true},
		{"3-day already sent", 2 * day, certUrgentDays, 0, false},
		{"already expired, never warned", -1 * day, 0, certUrgentDays, true},
		{"already expired, urgent sent", -1 * day, certUrgentDays, 0, false},
		{"renewed cert resets (warned cleared by the prober)", 90 * day, 0, 0, false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			rung, owed := certRung(tt.left, tt.warned)
			if owed != tt.wantOwed || rung != tt.wantRung {
				t.Fatalf("certRung(%v, %d) = (%d, %v), want (%d, %v)",
					tt.left, tt.warned, rung, owed, tt.wantRung, tt.wantOwed)
			}
		})
	}
}

func TestComposeCertAlert(t *testing.T) {
	t.Run("upcoming expiry", func(t *testing.T) {
		subject, body := composeCertAlert(expiringCert{
			MonitorName: "prod api",
			ExpiresAt:   time.Now().Add(3 * 24 * time.Hour),
			Issuer:      "Let's Encrypt",
		})
		if !strings.Contains(subject, "prod api") || !strings.Contains(subject, "expires in") {
			t.Errorf("subject = %q", subject)
		}
		for _, want := range []string{"prod api", "Let's Encrypt", "avoid an outage"} {
			if !strings.Contains(body, want) {
				t.Errorf("body missing %q:\n%s", want, body)
			}
		}
	})

	t.Run("already expired", func(t *testing.T) {
		subject, body := composeCertAlert(expiringCert{
			MonitorName: "prod api",
			ExpiresAt:   time.Now().Add(-48 * time.Hour),
			Issuer:      "",
		})
		if !strings.HasPrefix(subject, "EXPIRED:") {
			t.Errorf("subject = %q, want EXPIRED prefix", subject)
		}
		if !strings.Contains(body, "2 days ago") {
			t.Errorf("body should say how long ago it expired:\n%s", body)
		}
		if !strings.Contains(body, "unknown") {
			t.Errorf("empty issuer should render as 'unknown':\n%s", body)
		}
	})

	t.Run("subject has no CRLF (header-injection guard)", func(t *testing.T) {
		subject, _ := composeCertAlert(expiringCert{
			MonitorName: "prod api",
			ExpiresAt:   time.Now().Add(24 * time.Hour),
		})
		if strings.ContainsAny(subject, "\r\n") {
			t.Fatalf("subject contains CR/LF: %q", subject)
		}
	})
}
