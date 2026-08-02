package main

import (
	"net/http"

	"github.com/jackc/pgx/v5"

	"github.com/devopsaccess-in/devopsaccess.in/services/shared/db"

	"github.com/devopsaccess-in/devopsaccess.in/services/api/internal/auth"
	"github.com/devopsaccess-in/devopsaccess.in/services/api/internal/store"
)

// handleMe returns the caller's user + tenant, provisioning both on first
// login. Email/name come from optional token claims (an Auth0 Action adds the
// namespaced ones to the access token); absent claims just mean a generic
// tenant name until the user sets one.
func (s *server) handleMe(w http.ResponseWriter, r *http.Request) {
	claims, ok := auth.Claims(r.Context())
	if !ok {
		s.writeError(w, http.StatusUnauthorized, "unauthenticated")
		return
	}
	sub, _ := auth.Sub(r.Context())
	email := auth.StringClaim(claims, "https://devopsaccess.in/email", "email")
	name := auth.StringClaim(claims, "https://devopsaccess.in/name", "name")

	user, tenant, err := store.EnsureUser(r.Context(), s.pool, sub, email, name)
	if err != nil {
		s.serverError(w, r, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"user": user, "tenant": tenant})
}

// updateSettings updates tenant-level settings. Currently just the public
// status-page toggle. Tenant-scoped (requireTenant) so a user can only change
// their own workspace.
func (s *server) updateSettings(w http.ResponseWriter, r *http.Request) {
	var in struct {
		PublicStatusEnabled *bool `json:"public_status_enabled"`
	}
	if err := decodeJSON(w, r, &in); err != nil {
		s.writeError(w, http.StatusBadRequest, err.Error())
		return
	}
	if in.PublicStatusEnabled == nil {
		s.writeError(w, http.StatusBadRequest, "no settings to update")
		return
	}
	err := db.WithTenant(r.Context(), s.pool, tenantID(r.Context()), func(tx pgx.Tx) error {
		if err := store.SetPublicStatus(r.Context(), tx, tenantID(r.Context()), *in.PublicStatusEnabled); err != nil {
			return err
		}
		state := "enabled"
		if !*in.PublicStatusEnabled {
			state = "disabled"
		}
		return store.Audit(r.Context(), tx, tenantID(r.Context()), actor(r.Context()),
			store.ActionSettingsUpdate,
			"public status page "+state,
			nil, map[string]any{"public_status_enabled": *in.PublicStatusEnabled})
	})
	if err != nil {
		s.storeError(w, r, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]bool{"public_status_enabled": *in.PublicStatusEnabled})
}

// listAudit returns the tenant's activity trail — who changed what, when.
func (s *server) listAudit(w http.ResponseWriter, r *http.Request) {
	var entries []store.AuditEntry
	err := db.WithTenant(r.Context(), s.pool, tenantID(r.Context()), func(tx pgx.Tx) error {
		var err error
		entries, err = store.ListAudit(r.Context(), tx, tenantID(r.Context()), 200)
		return err
	})
	if err != nil {
		s.serverError(w, r, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"entries": entries})
}
