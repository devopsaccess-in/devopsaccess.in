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
	URL              string `json:"url"`
	Method           string `json:"method"`
	IntervalSeconds  int    `json:"interval_seconds"`
	TimeoutMs        int    `json:"timeout_ms"`
	ExpectedStatus   int    `json:"expected_status"`
	FailureThreshold int    `json:"failure_threshold"`
}

func (in *monitorInput) applyDefaults() {
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
	if err := validateMonitorFields(in.Name, in.Method, in.IntervalSeconds, in.TimeoutMs, in.ExpectedStatus, in.FailureThreshold); err != nil {
		s.writeError(w, http.StatusBadRequest, err.Error())
		return
	}
	if err := validateMonitorURL(r.Context(), in.URL, defaultLookup, s.cfg.allowPrivateTargets); err != nil {
		s.writeError(w, http.StatusBadRequest, err.Error())
		return
	}

	var m store.Monitor
	err := db.WithTenant(r.Context(), s.pool, tenantID(r.Context()), func(tx pgx.Tx) error {
		var err error
		m, err = store.CreateMonitor(r.Context(), tx, tenantID(r.Context()), store.NewMonitor{
			Name:             strings.TrimSpace(in.Name),
			URL:              strings.TrimSpace(in.URL),
			Method:           in.Method,
			IntervalSeconds:  in.IntervalSeconds,
			TimeoutMs:        in.TimeoutMs,
			ExpectedStatus:   in.ExpectedStatus,
			FailureThreshold: in.FailureThreshold,
		})
		return err
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
	}
	if err := decodeJSON(w, r, &in); err != nil {
		s.writeError(w, http.StatusBadRequest, err.Error())
		return
	}
	p := store.MonitorPatch{
		Name: in.Name, URL: in.URL, Method: in.Method,
		IntervalSeconds: in.IntervalSeconds, TimeoutMs: in.TimeoutMs,
		ExpectedStatus: in.ExpectedStatus, FailureThreshold: in.FailureThreshold,
		Enabled: in.Enabled,
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
		return err
	})
	if err != nil {
		s.storeError(w, r, err)
		return
	}
	writeJSON(w, http.StatusOK, m)
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
		return store.DeleteMonitor(r.Context(), tx, tenantID(r.Context()), id)
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
	window, err := parseWindow(r.URL.Query().Get("window"), 24*time.Hour, 7*24*time.Hour)
	if err != nil {
		s.writeError(w, http.StatusBadRequest, err.Error())
		return
	}

	var results []store.Result
	err = db.WithTenant(r.Context(), s.pool, tenantID(r.Context()), func(tx pgx.Tx) error {
		// Confirm the monitor exists in this tenant so an unknown id is a 404,
		// not an empty list.
		if _, err := store.GetMonitor(r.Context(), tx, tenantID(r.Context()), id); err != nil {
			return err
		}
		var err error
		results, err = store.ListResults(r.Context(), tx, tenantID(r.Context()), id, time.Now().Add(-window), 5000)
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

// parseWindow parses "24h" / "7d" style windows, clamping to max.
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
	default:
		return 0, fmt.Errorf("window must look like 24h or 7d")
	}
	n, err := strconv.Atoi(s[:len(s)-1])
	if err != nil || n <= 0 {
		return 0, fmt.Errorf("window must look like 24h or 7d")
	}
	w := time.Duration(n) * unit
	if w > max {
		return 0, fmt.Errorf("window too large (max %s)", max)
	}
	return w, nil
}
