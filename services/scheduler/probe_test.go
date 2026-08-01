package main

import (
	"fmt"
	"net"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

func testJob(url string, expectStatus int) job {
	return job{
		ID: "test", TenantID: "test", Name: "test", URL: url, Method: "GET",
		TimeoutMs: 5000, ExpectedStatus: expectStatus, FailureThreshold: 2,
	}
}

func TestCheckCapturesPhaseTimings(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		time.Sleep(15 * time.Millisecond) // make server time measurable
		w.WriteHeader(http.StatusOK)
	}))
	defer srv.Close()

	r := check(t.Context(), testJob(srv.URL, 200), true)

	if !r.OK || r.Error != "" {
		t.Fatalf("expected success, got ok=%v err=%q phase=%q", r.OK, r.Error, r.FailurePhase)
	}
	if r.Timings.ConnectMs == nil {
		t.Error("connect timing not captured")
	}
	if r.Timings.TTFBMs == nil {
		t.Error("ttfb not captured")
	} else if *r.Timings.TTFBMs < 10 {
		t.Errorf("ttfb %dms should reflect the server's 15ms delay", *r.Timings.TTFBMs)
	}
	// Plain http over loopback: no TLS phase, and no DNS for an IP literal.
	if r.Timings.TLSMs != nil {
		t.Errorf("plain http should have no TLS timing, got %dms", *r.Timings.TLSMs)
	}
	if r.Cert != nil {
		t.Error("plain http should capture no certificate")
	}
}

func TestCheckCapturesTLSCertificate(t *testing.T) {
	srv := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	defer srv.Close()

	// httptest's cert is self-signed, so the handshake fails verification —
	// which is exactly the diagnosis we want to prove works against a real
	// TLS exchange rather than a synthesized error.
	r := check(t.Context(), testJob(srv.URL, 200), true)

	if r.OK {
		t.Fatal("self-signed cert should not verify")
	}
	if r.FailurePhase != "tls" {
		t.Errorf("phase = %q, want tls (err %q)", r.FailurePhase, r.Error)
	}
	if !strings.Contains(r.Error, "not trusted") && !strings.Contains(r.Error, "does not match") {
		t.Errorf("unhelpful TLS diagnosis: %q", r.Error)
	}
	// The TLS phase started, so a connection was made first.
	if r.Timings.ConnectMs == nil {
		t.Error("connect timing not captured before the TLS failure")
	}
}

func TestCheckDiagnosesUnexpectedStatus(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer srv.Close()

	r := check(t.Context(), testJob(srv.URL, 200), true)

	if r.OK {
		t.Fatal("500 should fail a monitor expecting 200")
	}
	if r.FailurePhase != "status" {
		t.Errorf("phase = %q, want status", r.FailurePhase)
	}
	if r.StatusCode == nil || *r.StatusCode != 500 {
		t.Errorf("status code not recorded: %v", r.StatusCode)
	}
	if !strings.Contains(r.Error, "expected status 200, got 500") {
		t.Errorf("error = %q", r.Error)
	}
	// A response arrived, so the breakdown should still be complete.
	if r.Timings.TTFBMs == nil {
		t.Error("ttfb should be captured even on a bad status")
	}
}

func TestCheckDiagnosesConnectionRefused(t *testing.T) {
	// Bind then release a port so nothing is listening on it.
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("listen: %v", err)
	}
	port := ln.Addr().(*net.TCPAddr).Port
	ln.Close()

	r := check(t.Context(), testJob(fmt.Sprintf("http://127.0.0.1:%d/", port), 200), true)

	if r.OK {
		t.Fatal("expected failure against a closed port")
	}
	if r.FailurePhase != "tcp" {
		t.Errorf("phase = %q, want tcp (err %q)", r.FailurePhase, r.Error)
	}
	if !strings.Contains(r.Error, "refused") {
		t.Errorf("error = %q, want connection refused", r.Error)
	}
}

func TestCheckSSRFGuardBlocksPrivateTarget(t *testing.T) {
	// With the guard ON (insecureTargets=false), a loopback target must be
	// refused at dial time and diagnosed as blocked — never contacted.
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		t.Error("guarded probe must not reach a loopback server")
		w.WriteHeader(http.StatusOK)
	}))
	defer srv.Close()

	r := check(t.Context(), testJob(srv.URL, 200), false)

	if r.OK {
		t.Fatal("loopback target should be blocked")
	}
	if r.FailurePhase != "blocked" {
		t.Errorf("phase = %q, want blocked (err %q)", r.FailurePhase, r.Error)
	}
}
