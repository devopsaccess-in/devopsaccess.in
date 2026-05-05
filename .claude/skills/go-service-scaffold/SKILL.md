---
name: go-service-scaffold
description: Use this skill whenever the user asks to create a new Go service, add a new HTTP endpoint, wire a new gRPC service, or scaffold backend code in DevOps Access. Covers service layout, chi/zerolog/pgx patterns, OTel instrumentation, graceful shutdown, and testing conventions. Triggers on phrases like "new Go service", "add endpoint", "scaffold service", "Go handler", "gRPC service".
---

# DevOps Access Go Service Conventions

## Service directory structure
services/<svc>/
├── cmd/
│   └── <svc>/
│       └── main.go           # entry: flags, signal handling, startup
├── internal/
│   ├── app/                  # wire-up, DI, top-level start/stop
│   ├── config/               # config loading, validation
│   ├── http/                 # chi routers, handlers, middleware
│   ├── grpc/                 # gRPC server (if applicable)
│   ├── domain/               # business logic, pure (no HTTP/DB deps)
│   ├── storage/              # DB access (pgx), object store
│   ├── observability/        # zerolog, otel setup
│   └── testutil/             # test helpers (fake clients, fixtures)
├── migrations/               # SQL migrations (if owns a DB)
├── go.mod
├── go.sum
└── Dockerfile
## Standard main.go

```go
package main

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"syscall"

	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
)

func main() {
	zerolog.TimeFieldFormat = zerolog.TimeFormatUnix
	log.Logger = log.Output(zerolog.ConsoleWriter{Out: os.Stderr}).
		With().Caller().Logger()

	ctx, cancel := signal.NotifyContext(context.Background(),
		syscall.SIGINT, syscall.SIGTERM)
	defer cancel()

	if err := run(ctx); err != nil {
		log.Fatal().Err(err).Msg("service failed")
	}
}

func run(ctx context.Context) error {
	cfg, err := config.Load()
	if err != nil {
		return fmt.Errorf("loading config: %w", err)
	}

	a, err := app.New(ctx, cfg)
	if err != nil {
		return fmt.Errorf("initializing app: %w", err)
	}

	return a.Run(ctx)
}
```

## HTTP handler pattern

```go
// internal/http/tenants.go
package http

import (
	"encoding/json"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/rs/zerolog/hlog"
)

func (h *Handler) GetTenant(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	tenantID := chi.URLParam(r, "tenantID")

	t, err := h.svc.GetTenant(ctx, tenantID)
	if err != nil {
		hlog.FromRequest(r).Error().
			Err(err).
			Str("tenant_id", tenantID).
			Msg("get tenant failed")
		httpError(w, err)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(t)
}
```

## Router setup (middleware order matters)

```go
router := chi.NewRouter()
router.Use(middleware.RequestID)
router.Use(hlog.NewHandler(log.Logger))
router.Use(hlog.AccessHandler(logAccess))
router.Use(middleware.Recoverer)
router.Use(otelhttp.NewMiddleware("api-server"))
router.Use(auth.Middleware(cfg.JWTKey))
router.Use(tenant.ScopeMiddleware)
```

## Hard rules

1. Context first parameter, error last: `func Foo(ctx context.Context, ...) (T, error)`
2. Wrap errors with context: `fmt.Errorf("getting tenant %s: %w", id, err)`
3. No panics outside `main`. No `log.Fatal` outside `main`.
4. Every goroutine must handle its own panic (`defer` a `recover`).
5. Every external call (DB, HTTP, gRPC) must have a context deadline. No unbounded waits.
6. Every HTTP endpoint must have an OTel span (via `otelhttp` middleware).
7. Every DB query goes through `pgx` — no `database/sql` + `lib/pq`.
8. Structured logs only (zerolog). No `fmt.Println` or `log.Println`.

## Testing

- Unit tests: `foo_test.go` alongside `foo.go`, same package
- Table-driven preferred: `tests := []struct { name, input string; want T }{...}`
- Integration tests: separate file with `//go:build integration` tag, `testcontainers-go` for Postgres
- Mock external services with interfaces; never HTTP-stub inside tests

## Common pitfalls

- `pgx.Conn` is NOT safe for concurrent use. Use `pgxpool.Pool`.
- Chi middleware order matters: `RequestID` → `Logger` → `Recoverer` → `otelhttp` → auth → tenant scoping.
- Go contexts cancel cascades; to outlive the request use `context.WithoutCancel` (Go 1.21+).
- JSON unmarshal into `map[string]interface{}` loses int64 precision. Use `json.Number` or typed structs.
- Returning `errors.Is(err, pgx.ErrNoRows)` from a storage layer leaks DB details to callers. Wrap to a domain error.
