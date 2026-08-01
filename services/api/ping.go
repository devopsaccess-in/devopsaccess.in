package main

import (
	"errors"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/jackc/pgx/v5"

	"github.com/devopsaccess-in/devopsaccess.in/services/shared/db"

	"github.com/devopsaccess-in/devopsaccess.in/services/api/internal/store"
)

// handlePing receives a heartbeat: the "I ran successfully" signal from a
// cron job, backup script, or pipeline.
//
//	curl -fsS https://app.devopsaccess.in/api/ping/<token>
//
// Public and unauthenticated — the token IS the capability (160 bits from
// crypto/rand). Accepts GET (curl/wget friendly) and POST. Deliberately
// forgiving and fast: it records the ping, clears the failure state, and
// resolves an open incident. An unknown/disabled token gets a plain 404 with
// no detail, so tokens can't be probed for existence.
//
// The response is text/plain "ok" rather than JSON: this is pasted into
// shell one-liners, where `curl -fsS` output ends up in cron mail.
func (s *server) handlePing(w http.ResponseWriter, r *http.Request) {
	token := chi.URLParam(r, "token")
	// Bound the work an attacker can cause with garbage tokens: real ones are
	// a fixed 27-char base64url string.
	if len(token) < 16 || len(token) > 64 {
		http.Error(w, "not found", http.StatusNotFound)
		return
	}

	// The token establishes the tenant (heartbeat_tokens carries no RLS);
	// the write then runs inside that tenant's scope like every other query.
	monitorID, tenantID, err := store.TokenLookup(r.Context(), s.pool, token)
	if err == nil {
		err = db.WithTenant(r.Context(), s.pool, tenantID, func(tx pgx.Tx) error {
			return store.RecordPing(r.Context(), tx, tenantID, monitorID)
		})
	}
	if errors.Is(err, store.ErrNotFound) {
		http.Error(w, "not found", http.StatusNotFound)
		return
	}
	if err != nil {
		s.log.Error().Err(err).Msg("record ping failed")
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}

	s.log.Debug().Str("monitor_id", monitorID).Str("tenant_id", tenantID).Msg("heartbeat ping")
	w.Header().Set("Content-Type", "text/plain; charset=utf-8")
	w.Header().Set("Cache-Control", "no-store")
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write([]byte("ok\n"))
}
