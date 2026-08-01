package main

import (
	"context"
	"crypto/x509"
	"errors"
	"fmt"
	"net"
	"net/url"
	"strings"
	"syscall"
	"testing"
	"time"
)

// wrapURL mirrors how net/http surfaces transport errors: wrapped in a
// *url.Error whose Error() embeds the full target URL.
func wrapURL(target string, err error) error {
	return &url.Error{Op: "Get", URL: target, Err: err}
}

func TestDiagnose(t *testing.T) {
	const target = "https://example.com/health?token=SUPERSECRET"
	timeout := 10 * time.Second

	expiredCert := &x509.Certificate{NotAfter: time.Now().Add(-48 * time.Hour)}

	tests := []struct {
		name                     string
		err                      error
		dns, tcp, tlsOK          bool
		wantPhase                string
		wantMsgContains          string
	}{
		{
			name: "expired certificate names the expiry",
			err: wrapURL(target, x509.CertificateInvalidError{
				Cert: expiredCert, Reason: x509.Expired,
			}),
			dns: true, tcp: true,
			wantPhase:       "tls",
			wantMsgContains: "certificate expired 2 days ago",
		},
		{
			name:            "hostname mismatch",
			err:             wrapURL(target, x509.HostnameError{Host: "example.com"}),
			dns:             true,
			tcp:             true,
			wantPhase:       "tls",
			wantMsgContains: "does not match",
		},
		{
			name:            "untrusted issuer",
			err:             wrapURL(target, x509.UnknownAuthorityError{}),
			dns:             true,
			tcp:             true,
			wantPhase:       "tls",
			wantMsgContains: "not trusted",
		},
		{
			name:            "dns not found",
			err:             wrapURL(target, &net.DNSError{Err: "no such host", IsNotFound: true}),
			wantPhase:       "dns",
			wantMsgContains: "host not found",
		},
		{
			name:            "connection refused",
			err:             wrapURL(target, syscall.ECONNREFUSED),
			dns:             true,
			wantPhase:       "tcp",
			wantMsgContains: "refused",
		},
		{
			name:            "connection reset",
			err:             wrapURL(target, syscall.ECONNRESET),
			dns:             true,
			tcp:             true,
			wantPhase:       "tcp",
			wantMsgContains: "reset",
		},
		{
			name:            "ssrf guard blocked a private address",
			err:             wrapURL(target, errors.New("blocked non-public address 10.0.0.1")),
			wantPhase:       "blocked",
			wantMsgContains: "non-public",
		},
		{
			name:            "too many redirects",
			err:             wrapURL(target, errors.New("too many redirects")),
			dns:             true,
			tcp:             true,
			tlsOK:           true,
			wantPhase:       "redirect",
			wantMsgContains: "redirects",
		},
		{
			name:            "timeout before dns resolved",
			err:             wrapURL(target, context.DeadlineExceeded),
			wantPhase:       "dns",
			wantMsgContains: "DNS lookup",
		},
		{
			name:            "timeout before tcp connected",
			err:             wrapURL(target, context.DeadlineExceeded),
			dns:             true,
			wantPhase:       "tcp",
			wantMsgContains: "connecting",
		},
		{
			name:            "timeout during tls handshake",
			err:             wrapURL(target, context.DeadlineExceeded),
			dns:             true,
			tcp:             true,
			wantPhase:       "tls",
			wantMsgContains: "TLS handshake",
		},
		{
			name:            "timeout waiting for response",
			err:             wrapURL(target, context.DeadlineExceeded),
			dns:             true,
			tcp:             true,
			tlsOK:           true,
			wantPhase:       "timeout",
			wantMsgContains: "waiting for a response",
		},
		{
			name:            "unknown error falls back to furthest phase",
			err:             wrapURL(target, errors.New("something odd")),
			dns:             true,
			tcp:             true,
			tlsOK:           true,
			wantPhase:       "response",
			wantMsgContains: "no valid HTTP response",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			phase, msg := diagnose(tt.err, target, tt.dns, tt.tcp, tt.tlsOK, timeout)
			if phase != tt.wantPhase {
				t.Errorf("phase = %q, want %q (msg %q)", phase, tt.wantPhase, msg)
			}
			if !strings.Contains(msg, tt.wantMsgContains) {
				t.Errorf("message %q does not contain %q", msg, tt.wantMsgContains)
			}
			// Security: a cause is stored on incidents and sent in alerts, so
			// it must never echo the target URL (which can carry secrets).
			if strings.Contains(msg, "SUPERSECRET") || strings.Contains(msg, "example.com/health") {
				t.Errorf("diagnosis leaked the target URL: %q", msg)
			}
		})
	}
}

func TestDiagnoseHTTPTargetSkipsTLSPhase(t *testing.T) {
	// Plain http has no TLS phase: a timeout after connecting is a response
	// timeout, not a TLS timeout.
	phase, msg := diagnose(wrapURL("http://example.com", context.DeadlineExceeded),
		"http://example.com", true, true, false, 5*time.Second)
	if phase != "timeout" {
		t.Fatalf("phase = %q, want timeout (msg %q)", phase, msg)
	}
}

func TestHumanDur(t *testing.T) {
	tests := []struct {
		d    time.Duration
		want string
	}{
		{72 * time.Hour, "3 days"},
		{5 * time.Hour, "5 hours"},
		{30 * time.Minute, "30 minutes"},
		{10 * time.Second, "10s"},
		{250 * time.Millisecond, "250ms"},
	}
	for _, tt := range tests {
		if got := humanDur(tt.d); got != tt.want {
			t.Errorf("humanDur(%v) = %q, want %q", tt.d, got, tt.want)
		}
	}
}

func TestHumanAgo(t *testing.T) {
	if got := humanAgo(time.Now().Add(-72 * time.Hour)); got != "3 days ago" {
		t.Errorf("past = %q, want '3 days ago'", got)
	}
	if got := humanAgo(time.Now().Add(72 * time.Hour)); !strings.HasPrefix(got, "in ") {
		t.Errorf("future = %q, want 'in ...'", got)
	}
}

// Guard the phase vocabulary against the DB CHECK constraint in
// migrations/0003_deep_probe.sql — a phase the column rejects would fail every
// insert for that monitor.
func TestDiagnosePhasesMatchSchemaConstraint(t *testing.T) {
	allowed := map[string]bool{
		"": true, "dns": true, "tcp": true, "tls": true, "timeout": true,
		"status": true, "request": true, "blocked": true, "redirect": true,
		"response": true,
	}
	errs := []error{
		wrapURL("https://x.test", x509.CertificateInvalidError{Reason: x509.Expired}),
		wrapURL("https://x.test", x509.HostnameError{}),
		wrapURL("https://x.test", x509.UnknownAuthorityError{}),
		wrapURL("https://x.test", &net.DNSError{IsTimeout: true}),
		wrapURL("https://x.test", syscall.ECONNREFUSED),
		wrapURL("https://x.test", syscall.EHOSTUNREACH),
		wrapURL("https://x.test", errors.New("blocked non-public address 127.0.0.1")),
		wrapURL("https://x.test", fmt.Errorf("port 8443 not allowed")),
		wrapURL("https://x.test", context.DeadlineExceeded),
		wrapURL("https://x.test", errors.New("unknown")),
	}
	for _, combo := range [][3]bool{{false, false, false}, {true, false, false}, {true, true, false}, {true, true, true}} {
		for _, err := range errs {
			phase, _ := diagnose(err, "https://x.test", combo[0], combo[1], combo[2], time.Second)
			if !allowed[phase] {
				t.Errorf("phase %q not permitted by the schema CHECK constraint", phase)
			}
		}
	}
}
