package main

import (
	"context"
	"crypto/tls"
	"crypto/x509"
	"fmt"
	"net/http"
	"net/http/httptrace"
	"sync"
	"time"

	"github.com/devopsaccess-in/devopsaccess.in/services/shared/safehttp"
)

// phaseTimings is the per-phase breakdown of one HTTP check, in milliseconds.
// A nil field means the phase did not happen: no DNS for an IP literal, no TLS
// for plain http, no TTFB when the request never got a response.
type phaseTimings struct {
	DNSMs     *int
	ConnectMs *int
	TLSMs     *int
	TTFBMs    *int
}

// certInfo describes the leaf certificate presented by an https target.
type certInfo struct {
	ExpiresAt time.Time
	Issuer    string
}

// checkResult is the observed outcome of one probe.
type checkResult struct {
	OK         bool
	StatusCode *int
	LatencyMs  int
	Error      string
	// FailurePhase is the machine-readable phase that failed ("dns", "tcp",
	// "tls", "timeout", "status", "request", ""). Empty on success.
	FailurePhase string
	Timings      phaseTimings
	Cert         *certInfo
}

// tracer collects httptrace callbacks. The hooks can fire on different
// goroutines, so every field is guarded.
type tracer struct {
	mu sync.Mutex

	dnsStart, dnsDone         time.Time
	connectStart, connectDone time.Time
	tlsStart, tlsDone         time.Time
	firstByte                 time.Time
	start                     time.Time

	cert *certInfo
}

func (t *tracer) trace() *httptrace.ClientTrace {
	set := func(dst *time.Time) {
		t.mu.Lock()
		defer t.mu.Unlock()
		if dst.IsZero() { // keep the first occurrence (redirects re-run phases)
			*dst = time.Now()
		}
	}
	return &httptrace.ClientTrace{
		DNSStart:     func(httptrace.DNSStartInfo) { set(&t.dnsStart) },
		DNSDone:      func(httptrace.DNSDoneInfo) { set(&t.dnsDone) },
		ConnectStart: func(string, string) { set(&t.connectStart) },
		ConnectDone:  func(string, string, error) { set(&t.connectDone) },
		TLSHandshakeStart: func() { set(&t.tlsStart) },
		TLSHandshakeDone: func(cs tls.ConnectionState, err error) {
			set(&t.tlsDone)
			if err != nil || len(cs.PeerCertificates) == 0 {
				return
			}
			leaf := cs.PeerCertificates[0]
			t.mu.Lock()
			defer t.mu.Unlock()
			if t.cert == nil {
				t.cert = &certInfo{ExpiresAt: leaf.NotAfter, Issuer: issuerName(leaf)}
			}
		},
		GotFirstResponseByte: func() { set(&t.firstByte) },
	}
}

// timings converts the collected timestamps into durations.
func (t *tracer) timings() phaseTimings {
	t.mu.Lock()
	defer t.mu.Unlock()

	ms := func(from, to time.Time) *int {
		if from.IsZero() || to.IsZero() || to.Before(from) {
			return nil
		}
		v := int(to.Sub(from).Milliseconds())
		return &v
	}
	var ttfb *int
	if !t.firstByte.IsZero() && !t.start.IsZero() {
		v := int(t.firstByte.Sub(t.start).Milliseconds())
		ttfb = &v
	}
	return phaseTimings{
		DNSMs:     ms(t.dnsStart, t.dnsDone),
		ConnectMs: ms(t.connectStart, t.connectDone),
		TLSMs:     ms(t.tlsStart, t.tlsDone),
		TTFBMs:    ttfb,
	}
}

// reached reports which phases completed — the input to failure diagnosis.
func (t *tracer) reached() (dns, tcp, tlsDone bool) {
	t.mu.Lock()
	defer t.mu.Unlock()
	return !t.dnsDone.IsZero(), !t.connectDone.IsZero(), !t.tlsDone.IsZero()
}

func issuerName(c *x509.Certificate) string {
	if c.Issuer.CommonName != "" {
		return truncate(c.Issuer.CommonName, 100)
	}
	if len(c.Issuer.Organization) > 0 {
		return truncate(c.Issuer.Organization[0], 100)
	}
	return ""
}

// check probes the monitor's URL through the SSRF-guarded client, capturing a
// per-phase timing breakdown and the TLS leaf certificate. Any transport error
// or unexpected status is a failure, diagnosed down to the phase that broke.
func check(ctx context.Context, j job, insecureTargets bool) checkResult {
	timeout := time.Duration(j.TimeoutMs) * time.Millisecond
	client := safehttp.Client(timeout)
	if insecureTargets {
		// E2E-test hook. Mirrors safehttp.Client's transport minus the SSRF
		// guard — DisableKeepAlives especially: a pooled connection skips the
		// DNS/TCP/TLS phases, so without it the harness would exercise a
		// different timing path than production ever does.
		client = &http.Client{
			Timeout:   timeout,
			Transport: &http.Transport{DisableKeepAlives: true},
		}
	}

	ctx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	t := &tracer{start: time.Now()}
	req, err := http.NewRequestWithContext(httptrace.WithClientTrace(ctx, t.trace()), j.Method, j.URL, nil)
	if err != nil {
		return checkResult{Error: "invalid request", FailurePhase: "request"}
	}
	req.Header.Set("User-Agent", "DevOpsAccess-Uptime/1.0 (+https://devopsaccess.in)")

	resp, err := client.Do(req)
	latency := int(time.Since(t.start).Milliseconds())
	timings := t.timings()

	if err != nil {
		dns, tcp, tlsOK := t.reached()
		phase, msg := diagnose(err, j.URL, dns, tcp, tlsOK, timeout)
		return checkResult{
			LatencyMs:    latency,
			Error:        msg,
			FailurePhase: phase,
			Timings:      timings,
			Cert:         t.cert,
		}
	}
	defer resp.Body.Close()

	res := checkResult{
		StatusCode: &resp.StatusCode,
		LatencyMs:  latency,
		Timings:    timings,
		Cert:       t.cert,
	}
	if resp.StatusCode == j.ExpectedStatus {
		res.OK = true
	} else {
		res.Error = fmt.Sprintf("expected status %d, got %d", j.ExpectedStatus, resp.StatusCode)
		res.FailurePhase = "status"
	}
	return res
}
