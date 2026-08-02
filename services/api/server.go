package main

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"regexp"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/rs/zerolog"

	"github.com/devopsaccess-in/devopsaccess.in/services/api/internal/auth"
	"github.com/devopsaccess-in/devopsaccess.in/services/api/internal/store"
	"github.com/devopsaccess-in/devopsaccess.in/services/shared/notify"
)

// server bundles the dependencies handlers need.
type server struct {
	cfg    config
	pool   *pgxpool.Pool
	log    zerolog.Logger
	mailer *notify.Mailer
	// probe is the SSRF-guarded client used for customer-supplied URLs
	// (Slack webhook test sends).
	probe *http.Client
}

type ctxKey int

const tenantIDKey ctxKey = 0

// tenantID returns the tenant established by requireTenant.
func tenantID(ctx context.Context) string {
	id, _ := ctx.Value(tenantIDKey).(string)
	return id
}

// actor identifies the authenticated caller for the audit trail. Email comes
// from the same optional token claims /api/me uses; it is stored alongside the
// subject so the log stays readable without joining users.
func actor(ctx context.Context) store.Actor {
	sub, _ := auth.Sub(ctx)
	var email string
	if claims, ok := auth.Claims(ctx); ok {
		email = auth.StringClaim(claims, "https://devopsaccess.in/email", "email")
	}
	return store.Actor{Sub: sub, Email: email}
}

// requireTenant resolves the caller's tenant from the verified token subject.
// Users who have never hit /api/me have no tenant yet — the dashboard always
// calls /api/me first, so that is a client bug surfaced as 403.
func (s *server) requireTenant(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		sub, ok := auth.Sub(r.Context())
		if !ok {
			s.writeError(w, http.StatusUnauthorized, "unauthenticated")
			return
		}
		id, err := store.TenantForSub(r.Context(), s.pool, sub)
		if errors.Is(err, store.ErrNotFound) {
			s.writeError(w, http.StatusForbidden, "no tenant for user; call /api/me first")
			return
		}
		if err != nil {
			s.serverError(w, r, err)
			return
		}
		next.ServeHTTP(w, r.WithContext(context.WithValue(r.Context(), tenantIDKey, id)))
	})
}

func writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(v)
}

func (s *server) writeError(w http.ResponseWriter, status int, msg string) {
	writeJSON(w, status, map[string]string{"error": msg})
}

// serverError logs the real error and returns an opaque 500.
func (s *server) serverError(w http.ResponseWriter, r *http.Request, err error) {
	s.log.Error().Err(err).Str("method", r.Method).Str("path", r.URL.Path).Msg("request failed")
	s.writeError(w, http.StatusInternalServerError, "internal error")
}

// storeError maps store errors: ErrNotFound → 404 (cross-tenant reads must be
// indistinguishable from missing rows), anything else → 500.
func (s *server) storeError(w http.ResponseWriter, r *http.Request, err error) {
	if errors.Is(err, store.ErrNotFound) {
		s.writeError(w, http.StatusNotFound, "not found")
		return
	}
	s.serverError(w, r, err)
}

// decodeJSON reads a bounded JSON body into dst.
func decodeJSON(w http.ResponseWriter, r *http.Request, dst any) error {
	r.Body = http.MaxBytesReader(w, r.Body, 64<<10)
	dec := json.NewDecoder(r.Body)
	if err := dec.Decode(dst); err != nil {
		return fmt.Errorf("invalid JSON body")
	}
	return nil
}

var uuidRe = regexp.MustCompile(`^[0-9a-fA-F]{8}-[0-9a-fA-F]{4}-[0-9a-fA-F]{4}-[0-9a-fA-F]{4}-[0-9a-fA-F]{12}$`)

// isUUID pre-validates id path params so malformed ids become 404s instead of
// Postgres cast errors.
func isUUID(s string) bool {
	return uuidRe.MatchString(s)
}
