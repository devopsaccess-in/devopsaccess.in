package main

import (
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/jackc/pgx/v5"

	"github.com/devopsaccess-in/devopsaccess.in/services/shared/db"

	"github.com/devopsaccess-in/devopsaccess.in/services/api/internal/store"
)

func (s *server) listIncidents(w http.ResponseWriter, r *http.Request) {
	var monitorID *string
	if v := r.URL.Query().Get("monitor_id"); v != "" {
		if !isUUID(v) {
			s.writeError(w, http.StatusBadRequest, "monitor_id must be a UUID")
			return
		}
		monitorID = &v
	}

	var incidents []store.Incident
	err := db.WithTenant(r.Context(), s.pool, tenantID(r.Context()), func(tx pgx.Tx) error {
		var err error
		incidents, err = store.ListIncidents(r.Context(), tx, tenantID(r.Context()), monitorID, 100)
		return err
	})
	if err != nil {
		s.serverError(w, r, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"incidents": incidents})
}

func (s *server) getIncident(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	if !isUUID(id) {
		s.writeError(w, http.StatusNotFound, "not found")
		return
	}
	var incident store.Incident
	err := db.WithTenant(r.Context(), s.pool, tenantID(r.Context()), func(tx pgx.Tx) error {
		var err error
		incident, err = store.GetIncident(r.Context(), tx, tenantID(r.Context()), id)
		return err
	})
	if err != nil {
		s.storeError(w, r, err)
		return
	}
	writeJSON(w, http.StatusOK, incident)
}
