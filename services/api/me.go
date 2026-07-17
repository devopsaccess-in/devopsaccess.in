package main

import (
	"net/http"

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
	if err := store.SetPublicStatus(r.Context(), s.pool, tenantID(r.Context()), *in.PublicStatusEnabled); err != nil {
		s.storeError(w, r, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]bool{"public_status_enabled": *in.PublicStatusEnabled})
}
