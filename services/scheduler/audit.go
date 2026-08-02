package main

import (
	"context"
	"fmt"

	"github.com/jackc/pgx/v5"
)

// Audit actions the scheduler records. These MUST match the constants in
// services/api/internal/store/audit.go (that package is internal to the API
// module, so the two writers share the vocabulary by convention — the
// dashboard renders whatever string it finds).
const (
	auditIncidentOpen    = "incident.open"
	auditIncidentResolve = "incident.resolve"
)

// audit appends a system-attributed entry to the tenant's trail, in the same
// transaction as the change it describes. actor_sub is left empty: nobody
// clicked this, the scheduler did it.
//
// The scheduler connects with BYPASSRLS, so no tenant context is needed —
// but tenant_id is always written explicitly so the row is scoped correctly
// for everyone who reads it under RLS.
func (p *prober) audit(ctx context.Context, tx pgx.Tx, j job, action, summary string) error {
	if _, err := tx.Exec(ctx, `INSERT INTO audit_log
		(tenant_id, actor_sub, action, entity_id, summary, details)
		VALUES ($1, '', $2, $3, $4, $5)`,
		j.TenantID, action, j.ID, truncate(summary, 500),
		map[string]any{"kind": j.Kind}); err != nil {
		return fmt.Errorf("write audit entry: %w", err)
	}
	return nil
}
