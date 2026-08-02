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
		// CRLF in the name would enable SMTP header injection via the alert subject.
		{"name": "evil\r\nBcc: victim@example.com", "url": "https://example.com"},
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

	// Public status is opt-in: 404 until the tenant enables it (guessable
	// slugs must reveal nothing).
	anon := &apiClient{http: c.http}
	if s := anon.do(t, "GET", "/api/status/"+me.Tenant.Slug, nil, nil); s != http.StatusNotFound {
		t.Fatalf("status page before opt-in = %d, want 404", s)
	}
	c.mustDo(t, "PATCH", "/api/settings", map[string]any{"public_status_enabled": true}, nil, http.StatusOK)

	// Public status API reflects the outage — no auth.
	var status struct {
		Monitors []struct {
			Name  string `json:"name"`
			State string `json:"state"`
		} `json:"monitors"`
		Incidents []struct {
			ID    string `json:"id"`
			Cause string `json:"cause"`
		} `json:"incidents"`
	}
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
	// The public payload must not leak the technical cause (which can carry
	// the monitor URL / internal error detail).
	for _, inc := range status.Incidents {
		if inc.Cause != "" {
			t.Fatalf("public status incident leaked cause: %q", inc.Cause)
		}
	}
	if s := anon.do(t, "GET", "/api/status/no-such-tenant-slug", nil, nil); s != http.StatusNotFound {
		t.Fatalf("unknown slug = %d, want 404", s)
	}

	// Embeddable badge (public, gated on the same opt-in). The monitor is
	// down right now, so the status badge should read "down".
	badgeResp, err := c.http.Get(apiBase + "/api/badge/" + me.Tenant.Slug + "/" + m.ID + ".svg?metric=status")
	if err != nil {
		t.Fatalf("fetch badge: %v", err)
	}
	badgeBody, _ := io.ReadAll(badgeResp.Body)
	badgeResp.Body.Close()
	if ct := badgeResp.Header.Get("Content-Type"); !strings.HasPrefix(ct, "image/svg+xml") {
		t.Fatalf("badge content-type = %q, want svg", ct)
	}
	if b := string(badgeBody); !strings.HasPrefix(b, "<svg") || !strings.Contains(b, ">down</text>") {
		t.Fatalf("status badge not rendering 'down':\n%.400s", b)
	}
	// A monitor id under a DIFFERENT (nonexistent) slug must render the neutral
	// badge, never that monitor's real state — no cross-tenant leak.
	leakResp, err := c.http.Get(apiBase + "/api/badge/no-such-slug/" + m.ID + ".svg?metric=status")
	if err != nil {
		t.Fatalf("fetch leak badge: %v", err)
	}
	leakBody, _ := io.ReadAll(leakResp.Body)
	leakResp.Body.Close()
	if strings.Contains(string(leakBody), ">down</text>") {
		t.Fatalf("badge leaked monitor state under a wrong slug:\n%.400s", leakBody)
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
			OK           bool   `json:"ok"`
			ConnectMs    *int   `json:"connect_ms"`
			TTFBMs       *int   `json:"ttfb_ms"`
			FailurePhase string `json:"failure_phase"`
		} `json:"results"`
	}
	c.mustDo(t, "GET", "/api/monitors/"+m.ID+"/results?window=24h", nil, &results, http.StatusOK)
	var oks, fails, withTimings, statusPhase int
	for _, r := range results.Results {
		if r.OK {
			oks++
		} else {
			fails++
			if r.FailurePhase == "status" {
				statusPhase++
			}
		}
		// Deep probe: every check records its phase breakdown (the target is
		// plain http on loopback, so TCP connect + TTFB are always present).
		if r.ConnectMs != nil && r.TTFBMs != nil {
			withTimings++
		}
	}
	if oks == 0 || fails < 2 {
		t.Fatalf("results window missing checks: %d ok / %d failed", oks, fails)
	}
	if withTimings != len(results.Results) {
		t.Fatalf("phase timings missing: %d of %d results have connect+ttfb", withTimings, len(results.Results))
	}
	if statusPhase == 0 {
		t.Fatalf("failures should be diagnosed with failure_phase=status, got none in %d failures", fails)
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

// FEATURES.md: audit trail — who changed what, when. Also the answer to
// "why did my monitor disappear".
func TestAuditTrail(t *testing.T) {
	sub := uniqueSub("audit")
	c := newClient(t, sub, "auditor@a.in", "")
	var me meResp
	c.mustDo(t, "GET", "/api/me", nil, &me, http.StatusOK)

	type auditResp struct {
		Entries []struct {
			Action     string `json:"action"`
			Summary    string `json:"summary"`
			ActorEmail string `json:"actor_email"`
		} `json:"entries"`
	}
	actions := func() map[string]string {
		var a auditResp
		c.mustDo(t, "GET", "/api/audit", nil, &a, http.StatusOK)
		byAction := map[string]string{}
		for _, e := range a.Entries {
			byAction[e.Action] = e.Summary
		}
		return byAction
	}

	// Provisioning opens the trail.
	if got := actions(); got["user.first_login"] == "" {
		t.Fatalf("first login not audited: %+v", got)
	}

	target := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	defer target.Close()

	var m monitorResp
	c.mustDo(t, "POST", "/api/monitors", monitorPayload("audited monitor", target.URL, 2), &m, http.StatusCreated)
	c.mustDo(t, "PATCH", "/api/monitors/"+m.ID, map[string]any{"enabled": false}, &m, http.StatusOK)
	c.mustDo(t, "DELETE", "/api/monitors/"+m.ID, nil, nil, http.StatusNoContent)

	got := actions()
	for _, want := range []string{"monitor.create", "monitor.update", "monitor.delete"} {
		if got[want] == "" {
			t.Errorf("missing audit action %q; have %v", want, got)
		}
	}
	// The name must survive the delete — that is the whole point.
	if !strings.Contains(got["monitor.delete"], "audited monitor") {
		t.Errorf("delete entry should name the monitor: %q", got["monitor.delete"])
	}
	if !strings.Contains(got["monitor.update"], "paused") {
		t.Errorf("update entry should describe the change: %q", got["monitor.update"])
	}
	// Actions are attributed to the caller.
	var a auditResp
	c.mustDo(t, "GET", "/api/audit", nil, &a, http.StatusOK)
	for _, e := range a.Entries {
		if e.Action == "monitor.create" && e.ActorEmail != "auditor@a.in" {
			t.Errorf("create not attributed: %q", e.ActorEmail)
		}
	}

	// A Slack webhook URL is a credential: only its host may reach the log.
	var slackCh struct {
		ID string `json:"id"`
	}
	c.mustDo(t, "POST", "/api/channels",
		map[string]any{"type": "slack_webhook", "config": map[string]string{"url": slackSink.url()}},
		&slackCh, http.StatusCreated)
	c.mustDo(t, "GET", "/api/audit", nil, &a, http.StatusOK)
	for _, e := range a.Entries {
		if strings.Contains(e.Summary, "/hook") {
			t.Fatalf("audit summary leaked the webhook path: %q", e.Summary)
		}
	}

	// Another tenant sees none of it.
	other := newClient(t, uniqueSub("audit-other"), "other@b.in", "")
	other.mustDo(t, "GET", "/api/me", nil, nil, http.StatusOK)
	var otherAudit auditResp
	other.mustDo(t, "GET", "/api/audit", nil, &otherAudit, http.StatusOK)
	for _, e := range otherAudit.Entries {
		if strings.Contains(e.Summary, "audited monitor") {
			t.Fatalf("audit trail leaked across tenants: %q", e.Summary)
		}
	}

	c.mustDo(t, "DELETE", "/api/channels/"+slackCh.ID, nil, nil, http.StatusNoContent)
}

// FEATURES.md: heartbeat ("dead man's switch") monitors — the job pings us,
// and silence is the failure.
func TestHeartbeatMonitor(t *testing.T) {
	sub := uniqueSub("heartbeat")
	c := newClient(t, sub, "cron@a.in", "")
	c.mustDo(t, "GET", "/api/me", nil, nil, http.StatusOK)

	alertTo := fmt.Sprintf("cron-%s@e2e.in", strings.TrimPrefix(sub, "auth0|"))
	var ch struct {
		ID string `json:"id"`
	}
	c.mustDo(t, "POST", "/api/channels",
		map[string]any{"type": "email", "config": map[string]string{"to": alertTo}},
		&ch, http.StatusCreated)

	// Shortest allowed cadence so the test isn't slow: ping every 60s, 30s grace.
	var m struct {
		ID        string  `json:"id"`
		Kind      string  `json:"kind"`
		PingToken *string `json:"ping_token"`
		State     string  `json:"state"`
	}
	c.mustDo(t, "POST", "/api/monitors", map[string]any{
		"name": "nightly backup", "kind": "heartbeat",
		"period_seconds": 60, "grace_seconds": 30, "interval_seconds": 60,
	}, &m, http.StatusCreated)

	if m.Kind != "heartbeat" || m.PingToken == nil || len(*m.PingToken) < 20 {
		t.Fatalf("heartbeat not provisioned with a token: %+v", m)
	}

	pingURL := apiBase + "/api/ping/" + *m.PingToken

	// A fresh heartbeat is healthy for its first window (creating one must not
	// page you immediately).
	expedite(t, m.ID)
	waitForState(t, c, m.ID, "up", 30*time.Second)

	// The ping endpoint is public and needs no auth — this is what a cron job
	// runs. Unknown tokens must 404 and reveal nothing.
	anon := &apiClient{http: c.http}
	resp, err := anon.http.Get(pingURL)
	if err != nil {
		t.Fatalf("ping: %v", err)
	}
	body, _ := io.ReadAll(resp.Body)
	resp.Body.Close()
	if resp.StatusCode != http.StatusOK || !strings.Contains(string(body), "ok") {
		t.Fatalf("ping = %d %q, want 200 ok", resp.StatusCode, body)
	}
	bad, err := anon.http.Get(apiBase + "/api/ping/" + strings.Repeat("z", 27))
	if err != nil {
		t.Fatalf("bad ping: %v", err)
	}
	bad.Body.Close()
	if bad.StatusCode != http.StatusNotFound {
		t.Fatalf("unknown ping token = %d, want 404", bad.StatusCode)
	}

	// Stop pinging and push the clock past period+grace: the heartbeat goes
	// down, opens an incident, and alerts — with no HTTP probe involved.
	stalePing(t, m.ID, 5*time.Minute)
	waitForState(t, c, m.ID, "down", 60*time.Second)

	var incidents struct {
		Incidents []struct {
			ID    string `json:"id"`
			Cause string `json:"cause"`
		} `json:"incidents"`
	}
	c.mustDo(t, "GET", "/api/incidents?monitor_id="+m.ID, nil, &incidents, http.StatusOK)
	if len(incidents.Incidents) != 1 {
		t.Fatalf("want one heartbeat incident, got %+v", incidents.Incidents)
	}
	if !strings.Contains(incidents.Incidents[0].Cause, "late") {
		t.Errorf("incident cause should explain the lateness: %q", incidents.Incidents[0].Cause)
	}
	mailSink.waitFor(t, alertTo, "DOWN: nightly backup", 20*time.Second)

	// The job runs again: one ping is the full recovery signal.
	resp2, err := anon.http.Get(pingURL)
	if err != nil {
		t.Fatalf("recovery ping: %v", err)
	}
	resp2.Body.Close()

	var after struct {
		State string `json:"state"`
	}
	c.mustDo(t, "GET", "/api/monitors/"+m.ID, nil, &after, http.StatusOK)
	if after.State != "up" {
		t.Fatalf("state after recovery ping = %q, want up", after.State)
	}
	c.mustDo(t, "GET", "/api/incidents?monitor_id="+m.ID, nil, &incidents, http.StatusOK)
	if len(incidents.Incidents) != 1 {
		t.Fatalf("recovery should not open a second incident: %+v", incidents.Incidents)
	}
	mailSink.waitFor(t, alertTo, "RESOLVED: nightly backup", 25*time.Second)

	c.mustDo(t, "DELETE", "/api/monitors/"+m.ID, nil, nil, http.StatusNoContent)
	c.mustDo(t, "DELETE", "/api/channels/"+ch.ID, nil, nil, http.StatusNoContent)
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
	c.mustDo(t, "PATCH", "/api/settings", map[string]any{"public_status_enabled": true}, nil, http.StatusOK)

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
