// Package db provides the Postgres pool used by the uptime services, plus the
// tenant-scoping helper that makes Row Level Security effective: every
// tenant-scoped query runs inside a transaction that has SET LOCAL
// app.tenant_id, which the RLS policies read via current_setting().
package db

import (
	"context"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

// Connect opens a pool and pings with a short retry so services tolerate
// Postgres still starting after a reboot.
func Connect(ctx context.Context, url string) (*pgxpool.Pool, error) {
	pool, err := pgxpool.New(ctx, url)
	if err != nil {
		return nil, fmt.Errorf("connect postgres: %w", err)
	}

	var pingErr error
	for i := 0; i < 10; i++ {
		if pingErr = pool.Ping(ctx); pingErr == nil {
			return pool, nil
		}
		select {
		case <-ctx.Done():
			pool.Close()
			return nil, fmt.Errorf("ping postgres: %w", ctx.Err())
		case <-time.After(time.Second):
		}
	}
	pool.Close()
	return nil, fmt.Errorf("ping postgres: %w", pingErr)
}

// WithTenant runs fn inside a transaction scoped to tenantID. RLS policies on
// every tenant table enforce the scope even if a query forgets its WHERE
// tenant_id — defense in depth, both layers required by project convention.
func WithTenant(ctx context.Context, pool *pgxpool.Pool, tenantID string, fn func(pgx.Tx) error) error {
	tx, err := pool.Begin(ctx)
	if err != nil {
		return fmt.Errorf("begin tx: %w", err)
	}
	defer func() { _ = tx.Rollback(ctx) }()

	// set_config with is_local=true scopes the setting to this transaction.
	if _, err := tx.Exec(ctx, `SELECT set_config('app.tenant_id', $1, true)`, tenantID); err != nil {
		return fmt.Errorf("set tenant: %w", err)
	}
	if err := fn(tx); err != nil {
		return err
	}
	if err := tx.Commit(ctx); err != nil {
		return fmt.Errorf("commit tx: %w", err)
	}
	return nil
}
