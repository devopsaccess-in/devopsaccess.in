package main

import (
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/jackc/pgx/v5"

	"github.com/devopsaccess-in/devopsaccess.in/services/shared/db"

	"github.com/devopsaccess-in/devopsaccess.in/services/api/internal/store"
)

// monitorInput is the create payload; zero-valued optional fields take
// defaults.
type monitorInput struct {
	Name             string `json:"name"`
	Kind             string `json:"kind"`
	URL              string `json:"url"`
	Method           string `json:"method"`
	IntervalSeconds  int    `json:"interval_seconds"`
	TimeoutMs        int    `json:"timeout_ms"`
	ExpectedStatus   int    `json:"expected_status"`
	FailureThreshold int    `json:"failure_threshold"`
	PeriodSeconds    int    `json:"period_seconds"`
	GraceSeconds     int    `json:"grace_seconds"`
}

func (in *monitorInput) applyDefaults() {
	if in.Kind == "" {
		in.Kind = "http"
	}
	if in.Kind == "heartbeat" {
		// A heartbeat is never fetched: no URL, and one missed window is
		// already a real failure, so it alerts on the first late evaluation.
		in.URL = ""
		if in.PeriodSeconds == 0 {
			in.PeriodSeconds = 3600
		}
		if in.GraceSeconds == 0 {
			in.GraceSeconds = 300
		}
		if in.FailureThreshold == 0 {
			in.FailureThreshold = 1
		}
	}
	if in.Method == "" {
		in.Method = "GET"
	}
	if in.IntervalSeconds == 0 {
		in.IntervalSeconds = 60
	}
	if in.TimeoutMs == 0 {
		in.TimeoutMs = 10000
	}
	if in.ExpectedStatus == 0 {
		in.ExpectedStatus = 200
	}
	if in.FailureThreshold == 0 {
		in.FailureThreshold = 2
	}
}

// validateMonitorFields checks everything except the URL (which needs a DNS
// lookup and is validated separately). Mirrors the DB CHECK constraints so
// users get 400s with messages instead of opaque 500s.
func validateMonitorFields(name, method string, interval, timeout, expected, threshold int) error {
	if strings.TrimSpace(name) == "" || len(name) > 100 {
		return fmt.Errorf("name is required (max 100 chars)")
	}
	if strings.ContainsFunc(name, func(r rune) bool { return r == '\n' || r == '\r' || (r < 0x20 && r != '\t') }) {
		return fmt.Errorf("name must not contain control characters")
	}
	if method != "GET" && method != "HEAD" {
		return fmt.Errorf("method must be GET or HEAD")
	}
	if interval < 60 || interval > 300 {
		return fmt.Errorf("interval_seconds must be between 60 and 300")
	}
	if timeout < 1000 || timeout > 30000 {
		return fmt.Errorf("timeout_ms must be between 1000 and 30000")
	}
	if expected < 100 || expected > 599 {
		return fmt.Errorf("expected_status must be a valid HTTP status")
	}
	if threshold < 1 || threshold > 10 {
		return fmt.Errorf("failure_threshold must be between 1 and 10")
	}
	return nil
}

// validateHeartbeatFields checks the heartbeat-only inputs. Mirrors the DB
// CHECK constraints in migration 0004.
func validateHeartbeatFields(period, grace int) error {
	if period < 60 || period > 604800 {
		return fmt.Errorf("period_seconds must be between 60 (1 minute) and 604800 (7 days)")
	}
	if grace < 30 || grace > 86400 {
		return fmt.Errorf("grace_seconds must be between 30 and 86400 (1 day)")
	}
	return nil
}

func (s *server) listMonitors(w http.ResponseWriter, r *http.Request) {
	var monitors []store.Monitor
	err := db.WithTenant(r.Context(), s.pool, tenantID(r.Context()), func(tx pgx.Tx) error {
		var err error
		monitors, err = store.ListMonitors(r.Context(), tx, tenantID(r.Context()))
		return err
	})
	if err != nil {
		s.serverError(w, r, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"monitors": monitors})
}

func (s *server) createMonitor(w http.ResponseWriter, r *http.Request) {
	var in monitorInput
	if err := decodeJSON(w, r, &in); err != nil {
		s.writeError(w, http.StatusBadRequest, err.Error())
		return
	}
	in.applyDefaults()
	if in.Kind != "http" && in.Kind != "heartbeat" {
		s.writeError(w, http.StatusBadRequest, `kind must be "http" or "heartbeat"`)
		return
	}
	if err := validateMonitorFields(in.Name, in.Method, in.IntervalSeconds, in.TimeoutMs, in.ExpectedStatus, in.FailureThreshold); err != nil {
		s.writeError(w, http.StatusBadRequest, err.Error())
		return
	}
	if in.Kind == "heartbeat" {
		if err := validateHeartbeatFields(in.PeriodSeconds, in.GraceSeconds); err != nil {
			s.writeError(w, http.StatusBadRequest, err.Error())
			return
		}
	} else if err := validateMonitorURL(r.Context(), in.URL, defaultLookup, s.cfg.allowPrivateTargets); err != nil {
		s.writeError(w, http.StatusBadRequest, err.Error())
		return
	}

	var m store.Monitor
	err := db.WithTenant(r.Context(), s.pool, tenantID(r.Context()), func(tx pgx.Tx) error {
		var err error
		m, err = store.CreateMonitor(r.Context(), tx, tenantID(r.Context()), store.NewMonitor{
			Name:             strings.TrimSpace(in.Name),
			Kind:             in.Kind,
			URL:              strings.TrimSpace(in.URL),
			Method:           in.Method,
			IntervalSeconds:  in.IntervalSeconds,
			TimeoutMs:        in.TimeoutMs,
			ExpectedStatus:   in.ExpectedStatus,
			FailureThreshold: in.FailureThreshold,
			PeriodSeconds:    in.PeriodSeconds,
			GraceSeconds:     in.GraceSeconds,
		})
		if err != nil {
			return err
		}
		target := m.URL
		if m.Kind == "heartbeat" {
			target = fmt.Sprintf("every %ds", m.PeriodSeconds)
		}
		return store.Audit(r.Context(), tx, tenantID(r.Context()), actor(r.Context()),
			store.ActionMonitorCreate,
			fmt.Sprintf("created %s monitor %q (%s)", m.Kind, m.Name, target),
			&m.ID, map[string]any{"kind": m.Kind, "url": m.URL})
	})
	if err != nil {
		s.serverError(w, r, err)
		return
	}
	writeJSON(w, http.StatusCreated, m)
}

func (s *server) getMonitor(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	if !isUUID(id) {
		s.writeError(w, http.StatusNotFound, "not found")
		return
	}
	var m store.Monitor
	err := db.WithTenant(r.Context(), s.pool, tenantID(r.Context()), func(tx pgx.Tx) error {
		var err error
		m, err = store.GetMonitor(r.Context(), tx, tenantID(r.Context()), id)
		return err
	})
	if err != nil {
		s.storeError(w, r, err)
		return
	}
	writeJSON(w, http.StatusOK, m)
}

func (s *server) updateMonitor(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	if !isUUID(id) {
		s.writeError(w, http.StatusNotFound, "not found")
		return
	}
	var in struct {
		Name             *string `json:"name"`
		URL              *string `json:"url"`
		Method           *string `json:"method"`
		IntervalSeconds  *int    `json:"interval_seconds"`
		TimeoutMs        *int    `json:"timeout_ms"`
		ExpectedStatus   *int    `json:"expected_status"`
		FailureThreshold *int    `json:"failure_threshold"`
		Enabled          *bool   `json:"enabled"`
		PeriodSeconds    *int    `json:"period_seconds"`
		GraceSeconds     *int    `json:"grace_seconds"`
	}
	if err := decodeJSON(w, r, &in); err != nil {
		s.writeError(w, http.StatusBadRequest, err.Error())
		return
	}
	p := store.MonitorPatch{
		Name: in.Name, URL: in.URL, Method: in.Method,
		IntervalSeconds: in.IntervalSeconds, TimeoutMs: in.TimeoutMs,
		ExpectedStatus: in.ExpectedStatus, FailureThreshold: in.FailureThreshold,
		Enabled: in.Enabled, PeriodSeconds: in.PeriodSeconds, GraceSeconds: in.GraceSeconds,
	}

	if p.Name != nil {
		*p.Name = strings.TrimSpace(*p.Name)
	}
	if err := validatePatch(r, p, s.cfg.allowPrivateTargets); err != nil {
		s.writeError(w, http.StatusBadRequest, err.Error())
		return
	}

	var m store.Monitor
	err := db.WithTenant(r.Context(), s.pool, tenantID(r.Context()), func(tx pgx.Tx) error {
		var err error
		m, err = store.UpdateMonitor(r.Context(), tx, tenantID(r.Context()), id, p)
		if err != nil {
			return err
		}
		return store.Audit(r.Context(), tx, tenantID(r.Context()), actor(r.Context()),
			store.ActionMonitorUpdate,
			fmt.Sprintf("updated monitor %q: %s", m.Name, describePatch(p)),
			&m.ID, patchDetails(p))
	})
	if err != nil {
		s.storeError(w, r, err)
		return
	}
	writeJSON(w, http.StatusOK, m)
}

// describePatch renders the changed fields for the audit summary, e.g.
// "paused; interval_seconds -> 120".
func describePatch(p store.MonitorPatch) string {
	var parts []string
	if p.Enabled != nil {
		if *p.Enabled {
			parts = append(parts, "resumed")
		} else {
			parts = append(parts, "paused")
		}
	}
	if p.Name != nil {
		parts = append(parts, fmt.Sprintf("name -> %q", *p.Name))
	}
	if p.URL != nil {
		parts = append(parts, fmt.Sprintf("url -> %s", *p.URL))
	}
	if p.Method != nil {
		parts = append(parts, fmt.Sprintf("method -> %s", *p.Method))
	}
	if p.IntervalSeconds != nil {
		parts = append(parts, fmt.Sprintf("interval_seconds -> %d", *p.IntervalSeconds))
	}
	if p.TimeoutMs != nil {
		parts = append(parts, fmt.Sprintf("timeout_ms -> %d", *p.TimeoutMs))
	}
	if p.ExpectedStatus != nil {
		parts = append(parts, fmt.Sprintf("expected_status -> %d", *p.ExpectedStatus))
	}
	if p.FailureThreshold != nil {
		parts = append(parts, fmt.Sprintf("failure_threshold -> %d", *p.FailureThreshold))
	}
	if p.PeriodSeconds != nil {
		parts = append(parts, fmt.Sprintf("period_seconds -> %d", *p.PeriodSeconds))
	}
	if p.GraceSeconds != nil {
		parts = append(parts, fmt.Sprintf("grace_seconds -> %d", *p.GraceSeconds))
	}
	if len(parts) == 0 {
		return "no changes"
	}
	return strings.Join(parts, "; ")
}

// patchDetails is the machine-readable form of the same change set.
func patchDetails(p store.MonitorPatch) map[string]any {
	d := map[string]any{}
	if p.Enabled != nil {
		d["enabled"] = *p.Enabled
	}
	if p.Name != nil {
		d["name"] = *p.Name
	}
	if p.URL != nil {
		d["url"] = *p.URL
	}
	if p.Method != nil {
		d["method"] = *p.Method
	}
	if p.IntervalSeconds != nil {
		d["interval_seconds"] = *p.IntervalSeconds
	}
	if p.TimeoutMs != nil {
		d["timeout_ms"] = *p.TimeoutMs
	}
	if p.ExpectedStatus != nil {
		d["expected_status"] = *p.ExpectedStatus
	}
	if p.FailureThreshold != nil {
		d["failure_threshold"] = *p.FailureThreshold
	}
	if p.PeriodSeconds != nil {
		d["period_seconds"] = *p.PeriodSeconds
	}
	if p.GraceSeconds != nil {
		d["grace_seconds"] = *p.GraceSeconds
	}
	return d
}

// validatePatch checks only the fields present, reusing the create-time rules.
func validatePatch(r *http.Request, p store.MonitorPatch, allowPrivate bool) error {
	name, method := "monitor", "GET"
	interval, timeout, expected, threshold := 60, 10000, 200, 2
	if p.Name != nil {
		name = *p.Name
	}
	if p.Method != nil {
		method = *p.Method
	}
	if p.IntervalSeconds != nil {
		interval = *p.IntervalSeconds
	}
	if p.TimeoutMs != nil {
		timeout = *p.TimeoutMs
	}
	if p.ExpectedStatus != nil {
		expected = *p.ExpectedStatus
	}
	if p.FailureThreshold != nil {
		threshold = *p.FailureThreshold
	}
	if err := validateMonitorFields(name, method, interval, timeout, expected, threshold); err != nil {
		return err
	}
	// Heartbeat cadence, when either half is being changed. Defaults stand in
	// for the half that isn't, matching the DB's own constraints.
	if p.PeriodSeconds != nil || p.GraceSeconds != nil {
		period, grace := 3600, 300
		if p.PeriodSeconds != nil {
			period = *p.PeriodSeconds
		}
		if p.GraceSeconds != nil {
			grace = *p.GraceSeconds
		}
		if err := validateHeartbeatFields(period, grace); err != nil {
			return err
		}
	}
	if p.URL != nil {
		return validateMonitorURL(r.Context(), *p.URL, defaultLookup, allowPrivate)
	}
	return nil
}

func (s *server) deleteMonitor(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	if !isUUID(id) {
		s.writeError(w, http.StatusNotFound, "not found")
		return
	}
	err := db.WithTenant(r.Context(), s.pool, tenantID(r.Context()), func(tx pgx.Tx) error {
		// Read the name first: after the delete there is nothing left to
		// name in the audit trail, which is exactly when someone asks.
		m, err := store.GetMonitor(r.Context(), tx, tenantID(r.Context()), id)
		if err != nil {
			return err
		}
		if err := store.DeleteMonitor(r.Context(), tx, tenantID(r.Context()), id); err != nil {
			return err
		}
		return store.Audit(r.Context(), tx, tenantID(r.Context()), actor(r.Context()),
			store.ActionMonitorDelete,
			fmt.Sprintf("deleted %s monitor %q", m.Kind, m.Name),
			nil, map[string]any{"kind": m.Kind, "url": m.URL, "monitor_id": m.ID})
	})
	if err != nil {
		s.storeError(w, r, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (s *server) monitorResults(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	if !isUUID(id) {
		s.writeError(w, http.StatusNotFound, "not found")
		return
	}
	window, err := parseWindow(r.URL.Query().Get("window"), 24*time.Hour, maxSeriesWindow)
	if err != nil {
		s.writeError(w, http.StatusBadRequest, err.Error())
		return
	}
	// Raw rows are for reading individual checks (the latest result's phase
	// breakdown), not for drawing charts — /series does that. Default small
	// and cap hard so this endpoint can't be used to pull a whole month.
	limit := 200
	if v := r.URL.Query().Get("limit"); v != "" {
		n, err := strconv.Atoi(v)
		if err != nil || n < 1 || n > 1000 {
			s.writeError(w, http.StatusBadRequest, "limit must be between 1 and 1000")
			return
		}
		limit = n
	}

	var results []store.Result
	err = db.WithTenant(r.Context(), s.pool, tenantID(r.Context()), func(tx pgx.Tx) error {
		// Confirm the monitor exists in this tenant so an unknown id is a 404,
		// not an empty list.
		if _, err := store.GetMonitor(r.Context(), tx, tenantID(r.Context()), id); err != nil {
			return err
		}
		var err error
		results, err = store.ListResults(r.Context(), tx, tenantID(r.Context()), id, time.Now().Add(-window), limit)
		return err
	})
	if err != nil {
		s.storeError(w, r, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"window": window.String(), "results": results})
}

func (s *server) monitorUptime(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	if !isUUID(id) {
		s.writeError(w, http.StatusNotFound, "not found")
		return
	}
	window, err := parseWindow(r.URL.Query().Get("window"), 7*24*time.Hour, 30*24*time.Hour)
	if err != nil {
		s.writeError(w, http.StatusBadRequest, err.Error())
		return
	}

	var ok, total int64
	err = db.WithTenant(r.Context(), s.pool, tenantID(r.Context()), func(tx pgx.Tx) error {
		if _, err := store.GetMonitor(r.Context(), tx, tenantID(r.Context()), id); err != nil {
			return err
		}
		var err error
		ok, total, err = store.Uptime(r.Context(), tx, tenantID(r.Context()), id, time.Now().Add(-window))
		return err
	})
	if err != nil {
		s.storeError(w, r, err)
		return
	}

	resp := map[string]any{"window": window.String(), "ok": ok, "total": total, "uptime_pct": nil}
	if total > 0 {
		resp["uptime_pct"] = float64(ok) / float64(total) * 100
	}
	writeJSON(w, http.StatusOK, resp)
}

// parseWindow parses "30m" / "24h" / "7d" style windows, clamping to max.
func parseWindow(s string, def, max time.Duration) (time.Duration, error) {
	if s == "" {
		return def, nil
	}
	var unit time.Duration
	switch {
	case strings.HasSuffix(s, "d"):
		unit = 24 * time.Hour
	case strings.HasSuffix(s, "h"):
		unit = time.Hour
	case strings.HasSuffix(s, "m"):
		unit = time.Minute
	default:
		return 0, fmt.Errorf("window must look like 30m, 24h or 7d")
	}
	n, err := strconv.Atoi(s[:len(s)-1])
	if err != nil || n <= 0 {
		return 0, fmt.Errorf("window must look like 30m, 24h or 7d")
	}
	w := time.Duration(n) * unit
	if w > max {
		return 0, fmt.Errorf("window too large (max %s)", max)
	}
	return w, nil
}

// maxSeriesWindow bounds how far back a chart can ask. Matches the 30-day
// monitor_results retention — asking for more would return a misleadingly
// empty stretch.
const maxSeriesWindow = 30 * 24 * time.Hour

// monitorSeries returns the monitor's results aggregated into time buckets.
// Charts use this instead of /results: a 30-day window is tens of thousands of
// rows raw, but a fixed ~120 points bucketed, which is all a chart can draw.
func (s *server) monitorSeries(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	if !isUUID(id) {
		s.writeError(w, http.StatusNotFound, "not found")
		return
	}
	window, err := parseWindow(r.URL.Query().Get("window"), 24*time.Hour, maxSeriesWindow)
	if err != nil {
		s.writeError(w, http.StatusBadRequest, err.Error())
		return
	}
	buckets := 120
	if v := r.URL.Query().Get("buckets"); v != "" {
		n, err := strconv.Atoi(v)
		if err != nil || n < 1 || n > 500 {
			s.writeError(w, http.StatusBadRequest, "buckets must be between 1 and 500")
			return
		}
		buckets = n
	}

	var points []store.SeriesPoint
	err = db.WithTenant(r.Context(), s.pool, tenantID(r.Context()), func(tx pgx.Tx) error {
		if _, err := store.GetMonitor(r.Context(), tx, tenantID(r.Context()), id); err != nil {
			return err
		}
		var err error
		points, err = store.Series(r.Context(), tx, tenantID(r.Context()), id,
			time.Now().Add(-window), buckets)
		return err
	})
	if err != nil {
		s.storeError(w, r, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{
		"window":  window.String(),
		"buckets": points,
	})
}
