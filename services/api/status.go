package main

import (
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/jackc/pgx/v5"

	"github.com/devopsaccess-in/devopsaccess.in/services/shared/db"

	"github.com/devopsaccess-in/devopsaccess.in/services/api/internal/store"
)

// statusMonitor is the public view of a monitor: no URL, no config — just
// name, state, and 30-day uptime.
type statusMonitor struct {
	ID        string   `json:"id"`
	Name      string   `json:"name"`
	State     string   `json:"state"`
	UptimePct *float64 `json:"uptime_pct"`
}

// handleStatus serves the public status page data for a tenant slug. No auth:
// the slug is the capability. RLS still scopes every data query.
func (s *server) handleStatus(w http.ResponseWriter, r *http.Request) {
	slug := chi.URLParam(r, "slug")
	tenant, err := store.TenantBySlug(r.Context(), s.pool, slug)
	if err != nil {
		s.storeError(w, r, err)
		return
	}

	since := time.Now().Add(-30 * 24 * time.Hour)
	var monitors []statusMonitor
	var incidents []store.Incident
	err = db.WithTenant(r.Context(), s.pool, tenant.ID, func(tx pgx.Tx) error {
		all, err := store.ListMonitors(r.Context(), tx, tenant.ID)
		if err != nil {
			return err
		}
		for _, m := range all {
			if !m.Enabled {
				continue
			}
			sm := statusMonitor{ID: m.ID, Name: m.Name, State: m.State}
			ok, total, err := store.Uptime(r.Context(), tx, tenant.ID, m.ID, since)
			if err != nil {
				return err
			}
			if total > 0 {
				pct := float64(ok) / float64(total) * 100
				sm.UptimePct = &pct
			}
			monitors = append(monitors, sm)
		}
		incidents, err = store.ListIncidents(r.Context(), tx, tenant.ID, nil, 20)
		return err
	})
	if err != nil {
		s.serverError(w, r, err)
		return
	}
	if monitors == nil {
		monitors = []statusMonitor{}
	}

	writeJSON(w, http.StatusOK, map[string]any{
		"name":      tenant.Name,
		"slug":      tenant.Slug,
		"monitors":  monitors,
		"incidents": incidents,
	})
}
