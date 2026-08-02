package main

import (
	"net/http"
	"strings"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"

	"github.com/devopsaccess-in/devopsaccess.in/services/api/internal/auth"
)

// accessLog emits one structured line per request: what was called, by whom,
// how it ended, and how long it took. This is the operational half of
// observability (the audit trail is the product half) — it answers "was the
// API even called, and what did it return" when a customer reports a problem.
//
// Deliberately recorded here rather than read from nginx: only the app knows
// the tenant and the authenticated subject, and those are what make a log line
// answerable. Paths are route patterns ("/api/monitors/{id}"), never raw URLs,
// so ids and tokens stay out of the log.
func (s *server) accessLog(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()
		ww := middleware.NewWrapResponseWriter(w, r.ProtoMajor)

		next.ServeHTTP(ww, r)

		ev := s.log.Info()
		if ww.Status() >= 500 {
			ev = s.log.Error()
		} else if ww.Status() >= 400 {
			ev = s.log.Warn()
		}

		ev.Str("method", r.Method).
			Str("route", routePattern(r)).
			Int("status", ww.Status()).
			Int("bytes", ww.BytesWritten()).
			Dur("duration", time.Since(start)).
			Str("request_id", middleware.GetReqID(r.Context()))

		// Identity, when the request carried any.
		if tid := tenantID(r.Context()); tid != "" {
			ev.Str("tenant_id", tid)
		}
		if sub, ok := auth.Sub(r.Context()); ok {
			ev.Str("actor_sub", sub)
		}
		if ip := r.Header.Get("X-Real-IP"); ip != "" {
			ev.Str("ip", ip)
		}
		ev.Msg("request")
	})
}

// routePattern returns the matched chi route template, falling back to a
// coarse prefix when nothing matched (404s). Never the raw path: heartbeat
// ping URLs contain a secret token, and monitor ids are noise in aggregates.
func routePattern(r *http.Request) string {
	if rctx := chi.RouteContext(r.Context()); rctx != nil {
		if p := rctx.RoutePattern(); p != "" {
			return p
		}
	}
	path := r.URL.Path
	if i := strings.IndexByte(strings.TrimPrefix(path, "/"), '/'); i >= 0 {
		return path[:i+2] + "…"
	}
	return path
}
