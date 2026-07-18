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
	// Status pages are opt-in: an un-enabled tenant is indistinguishable from
	// a missing one, so enumerating guessable slugs reveals nothing.
	if !tenant.PublicStatusEnabled {
		s.writeError(w, http.StatusNotFound, "not found")
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

	// The public page shows only that an incident occurred and its timing —
	// never the technical cause, which can reference internal error detail.
	// (Probe causes are already URL-stripped upstream; this is defense in
	// depth so the public surface stays minimal.)
	publicIncidents := make([]map[string]any, 0, len(incidents))
	for _, i := range incidents {
		publicIncidents = append(publicIncidents, map[string]any{
			"id":           i.ID,
			"monitor_name": i.MonitorName,
			"started_at":   i.StartedAt,
			"resolved_at":  i.ResolvedAt,
		})
	}

	writeJSON(w, http.StatusOK, map[string]any{
		"name":      tenant.Name,
		"slug":      tenant.Slug,
		"monitors":  monitors,
		"incidents": publicIncidents,
	})
}
