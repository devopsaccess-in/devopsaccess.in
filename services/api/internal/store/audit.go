package store

import (
	"context"
	"fmt"
	"time"
)

// Audit action names. Kept as constants so the vocabulary is greppable and
// the dashboard can render each one without guessing.
const (
	ActionUserFirstLogin  = "user.first_login"
	ActionMonitorCreate   = "monitor.create"
	ActionMonitorUpdate   = "monitor.update"
	ActionMonitorDelete   = "monitor.delete"
	ActionChannelCreate   = "channel.create"
	ActionChannelDelete   = "channel.delete"
	ActionChannelTest     = "channel.test"
	ActionSettingsUpdate  = "settings.update"
	ActionIncidentOpen    = "incident.open"
	ActionIncidentResolve = "incident.resolve"
)

// AuditEntry is one recorded action.
type AuditEntry struct {
	ID         int64          `json:"id"`
	ActorSub   string         `json:"-"` // internal identifier, not exposed
	ActorEmail string         `json:"actor_email"`
	Action     string         `json:"action"`
	EntityID   *string        `json:"entity_id"`
	Summary    string         `json:"summary"`
	Details    map[string]any `json:"details"`
	CreatedAt  time.Time      `json:"created_at"`
}

// Actor identifies who performed an action. A zero Actor means the system
// itself (the scheduler).
type Actor struct {
	Sub   string
	Email string
}

// SystemActor is the attribution for actions nobody triggered by hand.
var SystemActor = Actor{}

// Audit appends an entry. q MUST be the same transaction as the mutation being
// recorded, so the trail commits or rolls back with it.
func Audit(ctx context.Context, q Querier, tenantID string, actor Actor, action, summary string, entityID *string, details map[string]any) error {
	if details == nil {
		details = map[string]any{}
	}
	if _, err := q.Exec(ctx, `INSERT INTO audit_log
		(tenant_id, actor_sub, actor_email, action, entity_id, summary, details)
		VALUES ($1, $2, $3, $4, $5, $6, $7)`,
		tenantID, actor.Sub, actor.Email, action, entityID, summary, details); err != nil {
		return fmt.Errorf("write audit entry: %w", err)
	}
	return nil
}

// ListAudit returns the tenant's most recent entries, newest first.
func ListAudit(ctx context.Context, q Querier, tenantID string, limit int) ([]AuditEntry, error) {
	rows, err := q.Query(ctx, `SELECT id, actor_sub, actor_email, action, entity_id,
			summary, details, created_at
		FROM audit_log WHERE tenant_id = $1
		ORDER BY created_at DESC, id DESC LIMIT $2`, tenantID, limit)
	if err != nil {
		return nil, fmt.Errorf("list audit: %w", err)
	}
	defer rows.Close()

	entries := []AuditEntry{}
	for rows.Next() {
		var e AuditEntry
		if err := rows.Scan(&e.ID, &e.ActorSub, &e.ActorEmail, &e.Action, &e.EntityID,
			&e.Summary, &e.Details, &e.CreatedAt); err != nil {
			return nil, fmt.Errorf("scan audit entry: %w", err)
		}
		entries = append(entries, e)
	}
	return entries, rows.Err()
}

// PurgeAudit drops entries older than the retention window. Returns how many
// rows went.
func PurgeAudit(ctx context.Context, q Querier, olderThan time.Duration) (int64, error) {
	tag, err := q.Exec(ctx,
		`DELETE FROM audit_log WHERE created_at < now() - $1::interval`,
		fmt.Sprintf("%d seconds", int(olderThan.Seconds())))
	if err != nil {
		return 0, fmt.Errorf("purge audit: %w", err)
	}
	return tag.RowsAffected(), nil
}
