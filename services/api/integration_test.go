//go:build integration

// Tenant-isolation integration test (the RLS half of the E2E gate). Run with:
//
//	DATABASE_URL=postgres://uptime_api:...@localhost/uptime_test \
//	  go test -tags integration ./services/api
//
// The connecting role must NOT be a superuser and must not have BYPASSRLS —
// superusers skip RLS entirely and would make this test fail for the wrong
// reason. The test applies migrations itself and cleans up the rows it made.
package main

import (
	"context"
	"errors"
	"fmt"
	"testing"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/rs/zerolog"

	"github.com/devopsaccess-in/devopsaccess.in/services/shared/db"

	"github.com/devopsaccess-in/devopsaccess.in/services/api/internal/store"
)

func TestTenantIsolationRLS(t *testing.T) {
	url := envOr("DATABASE_URL", "")
	if url == "" {
		t.Skip("DATABASE_URL not set")
	}
	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()

	pool, err := db.Connect(ctx, url)
	if err != nil {
		t.Fatalf("connect: %v", err)
	}
	defer pool.Close()
	if err := migrate(ctx, pool, zerolog.Nop()); err != nil {
		t.Fatalf("migrate: %v", err)
	}

	var isSuper, bypassRLS bool
	if err := pool.QueryRow(ctx,
		`SELECT rolsuper, rolbypassrls FROM pg_roles WHERE rolname = current_user`).
		Scan(&isSuper, &bypassRLS); err != nil {
		t.Fatalf("check role: %v", err)
	}
	if isSuper || bypassRLS {
		t.Fatalf("connecting role bypasses RLS (super=%v bypassrls=%v); use uptime_api", isSuper, bypassRLS)
	}

	suffix := time.Now().UnixNano()
	makeTenant := func(n int) store.Tenant {
		var tn store.Tenant
		slug := fmt.Sprintf("rls-test-%d-%d", n, suffix)
		if err := pool.QueryRow(ctx, `INSERT INTO tenants (name, slug) VALUES ($1, $1)
			RETURNING id, name, slug, created_at`, slug).
			Scan(&tn.ID, &tn.Name, &tn.Slug, &tn.CreatedAt); err != nil {
			t.Fatalf("create tenant: %v", err)
		}
		return tn
	}
	t1, t2 := makeTenant(1), makeTenant(2)
	t.Cleanup(func() {
		// tenants has no RLS; monitors etc. cascade.
		_, _ = pool.Exec(context.Background(),
			`DELETE FROM tenants WHERE id IN ($1, $2)`, t1.ID, t2.ID)
	})

	// Tenant 1 creates a monitor.
	var m store.Monitor
	if err := db.WithTenant(ctx, pool, t1.ID, func(tx pgx.Tx) error {
		var err error
		m, err = store.CreateMonitor(ctx, tx, t1.ID, store.NewMonitor{
			Name: "rls probe", URL: "https://example.com", Method: "GET",
			IntervalSeconds: 60, TimeoutMs: 10000, ExpectedStatus: 200, FailureThreshold: 2,
		})
		return err
	}); err != nil {
		t.Fatalf("tenant1 create monitor: %v", err)
	}

	// Tenant 2 must not see it by id — indistinguishable from missing (404).
	err = db.WithTenant(ctx, pool, t2.ID, func(tx pgx.Tx) error {
		_, err := store.GetMonitor(ctx, tx, t2.ID, m.ID)
		return err
	})
	if !errors.Is(err, store.ErrNotFound) {
		t.Fatalf("tenant2 GetMonitor(tenant1 id) error = %v, want ErrNotFound", err)
	}

	// RLS alone must hide it even without the app-layer predicate: query by
	// bare id inside tenant2's transaction.
	err = db.WithTenant(ctx, pool, t2.ID, func(tx pgx.Tx) error {
		var count int
		if err := tx.QueryRow(ctx, `SELECT count(*) FROM monitors WHERE id = $1`, m.ID).Scan(&count); err != nil {
			return err
		}
		if count != 0 {
			return fmt.Errorf("RLS leak: tenant2 sees %d of tenant1's monitors", count)
		}
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}

	// Tenant 2 must not be able to update or delete it either.
	err = db.WithTenant(ctx, pool, t2.ID, func(tx pgx.Tx) error {
		tag, err := tx.Exec(ctx, `UPDATE monitors SET name = 'stolen' WHERE id = $1`, m.ID)
		if err != nil {
			return err
		}
		if tag.RowsAffected() != 0 {
			return fmt.Errorf("RLS leak: tenant2 updated tenant1's monitor")
		}
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}

	// Tenant 2 must not be able to insert rows into tenant 1 (WITH CHECK).
	err = db.WithTenant(ctx, pool, t2.ID, func(tx pgx.Tx) error {
		_, err := tx.Exec(ctx, `INSERT INTO monitors (tenant_id, name, url) VALUES ($1, 'sneak', 'https://example.com')`, t1.ID)
		return err
	})
	if err == nil {
		t.Fatal("RLS leak: tenant2 inserted a monitor into tenant1")
	}

	// The audit trail is tenant data too: tenant 2 must not read tenant 1's
	// entries, and must not be able to forge one against tenant 1.
	if err := db.WithTenant(ctx, pool, t1.ID, func(tx pgx.Tx) error {
		return store.Audit(ctx, tx, t1.ID, store.Actor{Sub: "auth0|t1"},
			store.ActionMonitorCreate, "rls audit probe", &m.ID, nil)
	}); err != nil {
		t.Fatalf("tenant1 audit write: %v", err)
	}
	err = db.WithTenant(ctx, pool, t2.ID, func(tx pgx.Tx) error {
		entries, err := store.ListAudit(ctx, tx, t2.ID, 100)
		if err != nil {
			return err
		}
		for _, e := range entries {
			if e.Summary == "rls audit probe" {
				return fmt.Errorf("RLS leak: tenant2 read tenant1's audit entry")
			}
		}
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}

	// Forging an entry into another tenant must fail the WITH CHECK. Its own
	// transaction: the rejected statement aborts the surrounding one.
	err = db.WithTenant(ctx, pool, t2.ID, func(tx pgx.Tx) error {
		return store.Audit(ctx, tx, t1.ID, store.Actor{Sub: "auth0|t2"},
			store.ActionMonitorCreate, "forged", nil, nil)
	})
	if err == nil {
		t.Fatal("RLS leak: tenant2 wrote an audit entry for tenant1")
	}
	var forged int
	if err := pool.QueryRow(ctx,
		`SELECT count(*) FROM audit_log WHERE summary = 'forged'`).Scan(&forged); err != nil {
		t.Fatalf("count forged: %v", err)
	}
	if forged != 0 {
		t.Fatalf("RLS leak: %d forged audit rows persisted", forged)
	}

	// A transaction with NO tenant context sees nothing.
	tx, err := pool.Begin(ctx)
	if err != nil {
		t.Fatalf("begin: %v", err)
	}
	defer tx.Rollback(ctx)
	var count int
	if err := tx.QueryRow(ctx, `SELECT count(*) FROM monitors WHERE id = $1`, m.ID).Scan(&count); err != nil {
		t.Fatalf("no-context query: %v", err)
	}
	if count != 0 {
		t.Fatalf("RLS leak: no-tenant transaction sees %d monitors", count)
	}
}
