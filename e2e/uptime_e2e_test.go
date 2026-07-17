//go:build e2e

package e2e

import (
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync/atomic"
	"testing"
	"time"
)

type meResp struct {
	User struct {
		ID    string `json:"id"`
		Email string `json:"email"`
	} `json:"user"`
	Tenant struct {
		ID   string `json:"id"`
		Name string `json:"name"`
		Slug string `json:"slug"`
	} `json:"tenant"`
}

type monitorResp struct {
	ID    string `json:"id"`
	Name  string `json:"name"`
	State string `json:"state"`
}

// FEATURES.md: Signup & tenant provisioning.
func TestSignupProvisioning(t *testing.T) {
	sub := uniqueSub("signup")
	c := newClient(t, sub, "ramesh@startup.in", "Ramesh")

	var first meResp
	c.mustDo(t, "GET", "/api/me", nil, &first, http.StatusOK)
	if first.Tenant.ID == "" || first.Tenant.Slug == "" {
		t.Fatalf("first login did not provision a tenant: %+v", first)
	}
	if first.User.Email != "ramesh@startup.in" {
		t.Fatalf("email claim not stored: %+v", first.User)
	}
	if !strings.HasPrefix(first.Tenant.Slug, "ramesh") {
		t.Fatalf("slug %q not derived from email localpart", first.Tenant.Slug)
	}

	// Idempotent: same subject, same tenant.
	var second meResp
	c.mustDo(t, "GET", "/api/me", nil, &second, http.StatusOK)
	if second.Tenant.ID != first.Tenant.ID {
		t.Fatalf("second /api/me created a new tenant (%s != %s)", second.Tenant.ID, first.Tenant.ID)
	}

	// Same email localpart, different subject → distinct tenant, unique slug.
	other := newClient(t, uniqueSub("signup2"), "ramesh@othercorp.in", "Other Ramesh")
	var third meResp
	other.mustDo(t, "GET", "/api/me", nil, &third, http.StatusOK)
	if third.Tenant.ID == first.Tenant.ID || third.Tenant.Slug == first.Tenant.Slug {
		t.Fatalf("slug collision not resolved: %+v vs %+v", third.Tenant, first.Tenant)
	}

	// No token → 401. Token without /api/me first → 403 on tenant routes.
	anon := &apiClient{http: c.http}
	if status := anon.do(t, "GET", "/api/monitors", nil, nil); status != http.StatusUnauthorized {
		t.Fatalf("unauthenticated /api/monitors = %d, want 401", status)
	}
	fresh := newClient(t, uniqueSub("fresh"), "", "")
	if status := fresh.do(t, "GET", "/api/monitors", nil, nil); status != http.StatusForbidden {
		t.Fatalf("no-tenant /api/monitors = %d, want 403", status)
	}
}

// FEATURES.md: Tenant isolation (RLS + app scoping).
func TestTenantIsolation(t *testing.T) {
	owner := newClient(t, uniqueSub("owner"), "owner@a.in", "")
	intruder := newClient(t, uniqueSub("intruder"), "intruder@b.in", "")
	owner.mustDo(t, "GET", "/api/me", nil, nil, http.StatusOK)
	intruder.mustDo(t, "GET", "/api/me", nil, nil, http.StatusOK)

	target := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	defer target.Close()

	var m monitorResp
	owner.mustDo(t, "POST", "/api/monitors", monitorPayload("isolated", target.URL, 2), &m, http.StatusCreated)

	// Cross-tenant access must be indistinguishable from a missing row.
	if s := intruder.do(t, "GET", "/api/monitors/"+m.ID, nil, nil); s != http.StatusNotFound {
		t.Fatalf("cross-tenant GET = %d, want 404", s)
	}
	if s := intruder.do(t, "PATCH", "/api/monitors/"+m.ID, map[string]any{"name": "stolen"}, nil); s != http.StatusNotFound {
		t.Fatalf("cross-tenant PATCH = %d, want 404", s)
	}
	if s := intruder.do(t, "DELETE", "/api/monitors/"+m.ID, nil, nil); s != http.StatusNotFound {
		t.Fatalf("cross-tenant DELETE = %d, want 404", s)
	}
	if s := intruder.do(t, "GET", "/api/monitors/"+m.ID+"/results", nil, nil); s != http.StatusNotFound {
		t.Fatalf("cross-tenant results = %d, want 404", s)
	}

	var list struct {
		Monitors []monitorResp `json:"monitors"`
	}
	intruder.mustDo(t, "GET", "/api/monitors", nil, &list, http.StatusOK)
	for _, im := range list.Monitors {
		if im.ID == m.ID {
			t.Fatal("intruder's monitor list contains the owner's monitor")
		}
	}
	owner.mustDo(t, "DELETE", "/api/monitors/"+m.ID, nil, nil, http.StatusNoContent)
}

// FEATURES.md: Monitor management (validation surface).
func TestMonitorValidation(t *testing.T) {
	c := newClient(t, uniqueSub("valid"), "valid@a.in", "")
	c.mustDo(t, "GET", "/api/me", nil, nil, http.StatusOK)

	bad := []map[string]any{
		{"name": "", "url": "https://example.com"},
		{"name": "m", "url": "ftp://example.com"},
		{"name": "m", "url": "https://example.com", "method": "POST"},
		{"name": "m", "url": "https://example.com", "interval_seconds": 30},
		{"name": "m", "url": "https://example.com", "failure_threshold": 99},
		{"name": "m", "url": "https://example.com", "timeout_ms": 100},
	}
	for i, payload := range bad {
		if s := c.do(t, "POST", "/api/monitors", payload, nil); s != http.StatusBadRequest {
			t.Fatalf("bad payload %d accepted (status %d): %v", i, s, payload)
		}
	}

	// Unknown ids: malformed → 404, well-formed-but-absent → 404.
	if s := c.do(t, "GET", "/api/monitors/not-a-uuid", nil, nil); s != http.StatusNotFound {
		t.Fatalf("malformed id = %d, want 404", s)
	}
	if s := c.do(t, "GET", "/api/monitors/00000000-0000-4000-8000-000000000000", nil, nil); s != http.StatusNotFound {
		t.Fatalf("absent id = %d, want 404", s)
	}
}

// FEATURES.md: the full incident pipeline — checks, threshold crossing,
// incident, email + Slack alerts, recovery notice, uptime + results windows,
// public status API. This is the E2E-gate flow with local stand-ins.
func TestIncidentPipeline(t *testing.T) {
	sub := uniqueSub("pipeline")
	c := newClient(t, sub, "pipeline@a.in", "")
	var me meResp
	c.mustDo(t, "GET", "/api/me", nil, &me, http.StatusOK)

	// Target that can be broken and fixed.
	var healthy atomic.Bool
	healthy.Store(true)
	target := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		if healthy.Load() {
			w.WriteHeader(http.StatusOK)
			return
		}
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer target.Close()

	// Alert channels: email + Slack (pointing at the sinks).
	alertTo := fmt.Sprintf("oncall-%s@e2e.in", strings.TrimPrefix(sub, "auth0|"))
	var emailCh, slackCh struct {
		ID string `json:"id"`
	}
	c.mustDo(t, "POST", "/api/channels",
		map[string]any{"type": "email", "config": map[string]string{"to": alertTo}},
		&emailCh, http.StatusCreated)
	c.mustDo(t, "POST", "/api/channels",
		map[string]any{"type": "slack_webhook", "config": map[string]string{"url": slackSink.url()}},
		&slackCh, http.StatusCreated)

	// Channel test sends arrive at the sinks.
	c.mustDo(t, "POST", "/api/channels/"+emailCh.ID+"/test", nil, nil, http.StatusOK)
	mailSink.waitFor(t, alertTo, "Test alert", 15*time.Second)
	c.mustDo(t, "POST", "/api/channels/"+slackCh.ID+"/test", nil, nil, http.StatusOK)
	slackSink.waitFor(t, "Test alert", 15*time.Second)

	// Monitor comes up (threshold 2 exercises fail accumulation).
	var m monitorResp
	c.mustDo(t, "POST", "/api/monitors", monitorPayload("prod api", target.URL, 2), &m, http.StatusCreated)
	waitForState(t, c, m.ID, "up", 30*time.Second)

	// Break it → two failed checks → down + incident + alerts.
	healthy.Store(false)
	waitForState(t, c, m.ID, "down", 60*time.Second)

	var incidents struct {
		Incidents []struct {
			ID         string  `json:"id"`
			ResolvedAt *string `json:"resolved_at"`
			Cause      string  `json:"cause"`
		} `json:"incidents"`
	}
	c.mustDo(t, "GET", "/api/incidents?monitor_id="+m.ID, nil, &incidents, http.StatusOK)
	if len(incidents.Incidents) != 1 || incidents.Incidents[0].ResolvedAt != nil {
		t.Fatalf("want exactly one open incident, got %+v", incidents.Incidents)
	}
	if !strings.Contains(incidents.Incidents[0].Cause, "consecutive failures") {
		t.Fatalf("incident cause missing failure context: %q", incidents.Incidents[0].Cause)
	}
	mailSink.waitFor(t, alertTo, "DOWN: prod api", 20*time.Second)
	slackSink.waitFor(t, "DOWN: prod api", 20*time.Second)

	// Public status API reflects the outage — no auth.
	var status struct {
		Monitors []struct {
			Name  string `json:"name"`
			State string `json:"state"`
		} `json:"monitors"`
		Incidents []struct {
			ID string `json:"id"`
		} `json:"incidents"`
	}
	anon := &apiClient{http: c.http}
	anon.mustDo(t, "GET", "/api/status/"+me.Tenant.Slug, nil, &status, http.StatusOK)
	foundDown := false
	for _, sm := range status.Monitors {
		if sm.Name == "prod api" && sm.State == "down" {
			foundDown = true
		}
	}
	if !foundDown || len(status.Incidents) == 0 {
		t.Fatalf("status page missing outage: %+v", status)
	}
	if s := anon.do(t, "GET", "/api/status/no-such-tenant-slug", nil, nil); s != http.StatusNotFound {
		t.Fatalf("unknown slug = %d, want 404", s)
	}

	// Fix it → recovery on first success + recovery notices.
	healthy.Store(true)
	waitForState(t, c, m.ID, "up", 60*time.Second)
	c.mustDo(t, "GET", "/api/incidents?monitor_id="+m.ID, nil, &incidents, http.StatusOK)
	if len(incidents.Incidents) != 1 || incidents.Incidents[0].ResolvedAt == nil {
		t.Fatalf("incident not resolved: %+v", incidents.Incidents)
	}
	mailSink.waitFor(t, alertTo, "RESOLVED: prod api", 20*time.Second)
	slackSink.waitFor(t, "RESOLVED: prod api", 20*time.Second)

	// History endpoints carry the data.
	var results struct {
		Results []struct {
			OK bool `json:"ok"`
		} `json:"results"`
	}
	c.mustDo(t, "GET", "/api/monitors/"+m.ID+"/results?window=24h", nil, &results, http.StatusOK)
	var oks, fails int
	for _, r := range results.Results {
		if r.OK {
			oks++
		} else {
			fails++
		}
	}
	if oks == 0 || fails < 2 {
		t.Fatalf("results window missing checks: %d ok / %d failed", oks, fails)
	}
	var uptime struct {
		Total     int64    `json:"total"`
		UptimePct *float64 `json:"uptime_pct"`
	}
	c.mustDo(t, "GET", "/api/monitors/"+m.ID+"/uptime?window=7d", nil, &uptime, http.StatusOK)
	if uptime.Total == 0 || uptime.UptimePct == nil || *uptime.UptimePct >= 100 || *uptime.UptimePct <= 0 {
		t.Fatalf("uptime looks wrong: %+v", uptime)
	}

	// Pause stops future claims; delete cleans up (results cascade).
	c.mustDo(t, "PATCH", "/api/monitors/"+m.ID, map[string]any{"enabled": false}, &m, http.StatusOK)
	c.mustDo(t, "DELETE", "/api/monitors/"+m.ID, nil, nil, http.StatusNoContent)
	if s := c.do(t, "GET", "/api/monitors/"+m.ID, nil, nil); s != http.StatusNotFound {
		t.Fatalf("deleted monitor still readable (%d)", s)
	}
	c.mustDo(t, "DELETE", "/api/channels/"+emailCh.ID, nil, nil, http.StatusNoContent)
	c.mustDo(t, "DELETE", "/api/channels/"+slackCh.ID, nil, nil, http.StatusNoContent)
}

// FEATURES.md: public status page (dashboard UI). Runs only when the built
// dashboard is provided (E2E_DASHBOARD_DIR) — auth'd dashboard pages need a
// real Auth0 tenant and stay a manual gate.
func TestDashboardStatusPage(t *testing.T) {
	if dashBase == "" {
		t.Skip("E2E_DASHBOARD_DIR not set — dashboard not running")
	}
	c := newClient(t, uniqueSub("dash"), "dash@a.in", "")
	var me meResp
	c.mustDo(t, "GET", "/api/me", nil, &me, http.StatusOK)

	target := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	defer target.Close()
	var m monitorResp
	c.mustDo(t, "POST", "/api/monitors", monitorPayload("dash monitor", target.URL, 2), &m, http.StatusCreated)
	waitForState(t, c, m.ID, "up", 30*time.Second)

	resp, err := c.http.Get(dashBase + "/status/" + me.Tenant.Slug)
	if err != nil {
		t.Fatalf("fetch status page: %v", err)
	}
	defer resp.Body.Close()
	body, _ := io.ReadAll(resp.Body)
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("status page = %d, body: %.500s", resp.StatusCode, body)
	}
	html := string(body)
	if !strings.Contains(html, "dash monitor") || !strings.Contains(html, me.Tenant.Name) {
		t.Fatalf("status page HTML missing monitor/tenant: %.500s", html)
	}

	resp2, err := c.http.Get(dashBase + "/status/no-such-tenant-slug")
	if err != nil {
		t.Fatalf("fetch missing status page: %v", err)
	}
	resp2.Body.Close()
	if resp2.StatusCode != http.StatusNotFound {
		t.Fatalf("unknown slug page = %d, want 404", resp2.StatusCode)
	}
}
