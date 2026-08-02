package main

import (
	"strings"
	"testing"
	"time"
)

func TestComposeAlert(t *testing.T) {
	start := time.Now().Add(-10 * time.Minute)
	resolved := time.Now()

	t.Run("http monitor down includes the URL", func(t *testing.T) {
		subject, body := composeAlert(pendingIncident{
			MonitorName: "prod api",
			MonitorURL:  "https://api.example.com/health",
			StartedAt:   start,
			Cause:       "connection refused",
		}, false)

		if !strings.HasPrefix(subject, "DOWN: prod api") {
			t.Errorf("subject = %q", subject)
		}
		for _, want := range []string{"URL: https://api.example.com/health", "connection refused", "Since:"} {
			if !strings.Contains(body, want) {
				t.Errorf("body missing %q:\n%s", want, body)
			}
		}
	})

	t.Run("heartbeat omits the empty URL line", func(t *testing.T) {
		_, body := composeAlert(pendingIncident{
			MonitorName: "nightly backup",
			MonitorURL:  "", // heartbeats have no target URL
			StartedAt:   start,
			Cause:       "heartbeat is 8s late",
		}, false)

		if strings.Contains(body, "URL:") {
			t.Fatalf("heartbeat alert should not render a URL line:\n%s", body)
		}
		if !strings.Contains(body, "Monitor: nightly backup") || !strings.Contains(body, "8s late") {
			t.Errorf("body lost its content:\n%s", body)
		}
	})

	t.Run("recovery reports downtime", func(t *testing.T) {
		subject, body := composeAlert(pendingIncident{
			MonitorName: "prod api",
			MonitorURL:  "https://api.example.com",
			StartedAt:   start,
			ResolvedAt:  &resolved,
		}, true)

		if !strings.HasPrefix(subject, "RESOLVED: prod api") {
			t.Errorf("subject = %q", subject)
		}
		if !strings.Contains(body, "Downtime:") {
			t.Errorf("recovery body should state downtime:\n%s", body)
		}
	})

	t.Run("subjects carry no CRLF (header-injection guard)", func(t *testing.T) {
		subject, _ := composeAlert(pendingIncident{
			MonitorName: "prod api", StartedAt: start,
		}, false)
		if strings.ContainsAny(subject, "\r\n") {
			t.Fatalf("subject contains CR/LF: %q", subject)
		}
	})
}
