// Package store is the persistence layer for the uptime API. Tenant-scoped
// functions take a Querier that MUST be a transaction opened via
// db.WithTenant so RLS applies; each query still carries an explicit
// tenant-scoped predicate where natural (defense in depth — project rule).
// Identity lookups (users, tenants, memberships) run on the pool: those
// tables carry no RLS because they are what establishes the tenant context.
package store

import (
	"context"
	"crypto/rand"
	"encoding/base64"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"
)

// ErrNotFound is returned when a row does not exist — or is invisible to the
// caller's tenant, which must look identical to the client (404, not 403).
var ErrNotFound = errors.New("not found")

// Querier is satisfied by both pgx.Tx and *pgxpool.Pool.
type Querier interface {
	Exec(ctx context.Context, sql string, args ...any) (pgconn.CommandTag, error)
	Query(ctx context.Context, sql string, args ...any) (pgx.Rows, error)
	QueryRow(ctx context.Context, sql string, args ...any) pgx.Row
}

type User struct {
	ID        string    `json:"id"`
	Auth0Sub  string    `json:"-"`
	Email     string    `json:"email"`
	Name      string    `json:"name"`
	CreatedAt time.Time `json:"created_at"`
}

type Tenant struct {
	ID                  string    `json:"id"`
	Name                string    `json:"name"`
	Slug                string    `json:"slug"`
	PublicStatusEnabled bool      `json:"public_status_enabled"`
	CreatedAt           time.Time `json:"created_at"`
}

type Monitor struct {
	ID       string `json:"id"`
	TenantID string `json:"-"`
	Name     string `json:"name"`
	// Kind is "http" (we call the target) or "heartbeat" (the target calls us).
	Kind string `json:"kind"`
	// Heartbeat fields; zero/empty for http monitors.
	PeriodSeconds int        `json:"period_seconds"`
	GraceSeconds  int        `json:"grace_seconds"`
	PingToken     *string    `json:"ping_token"`
	LastPingAt    *time.Time `json:"last_ping_at"`

	URL              string     `json:"url"`
	Method           string     `json:"method"`
	IntervalSeconds  int        `json:"interval_seconds"`
	TimeoutMs        int        `json:"timeout_ms"`
	ExpectedStatus   int        `json:"expected_status"`
	FailureThreshold int        `json:"failure_threshold"`
	Enabled          bool       `json:"enabled"`
	State            string     `json:"state"`
	ConsecutiveFails int        `json:"consecutive_fails"`
	LastCheckedAt    *time.Time `json:"last_checked_at"`
	TLSExpiresAt     *time.Time `json:"tls_expires_at"`
	TLSIssuer        string     `json:"tls_issuer"`
	CreatedAt        time.Time  `json:"created_at"`
	UpdatedAt        time.Time  `json:"updated_at"`
}

type Result struct {
	ID         int64     `json:"id"`
	MonitorID  string    `json:"monitor_id"`
	CheckedAt  time.Time `json:"checked_at"`
	OK         bool      `json:"ok"`
	StatusCode *int      `json:"status_code"`
	LatencyMs  *int      `json:"latency_ms"`
	Error      string    `json:"error"`
	// Per-phase breakdown; nil when the phase did not occur (no DNS for an
	// IP literal, no TLS for plain http, no TTFB without a response).
	DNSMs        *int   `json:"dns_ms"`
	ConnectMs    *int   `json:"connect_ms"`
	TLSMs        *int   `json:"tls_ms"`
	TTFBMs       *int   `json:"ttfb_ms"`
	FailurePhase string `json:"failure_phase"`
}

type Incident struct {
	ID          string     `json:"id"`
	MonitorID   string     `json:"monitor_id"`
	MonitorName string     `json:"monitor_name"`
	StartedAt   time.Time  `json:"started_at"`
	ResolvedAt  *time.Time `json:"resolved_at"`
	Cause       string     `json:"cause"`
}

type Channel struct {
	ID        string          `json:"id"`
	Type      string          `json:"type"`
	Config    map[string]any  `json:"config"`
	Enabled   bool            `json:"enabled"`
	CreatedAt time.Time       `json:"created_at"`
}

const monitorCols = `id, tenant_id, name, kind, period_seconds, grace_seconds,
	ping_token, last_ping_at, url, method, interval_seconds, timeout_ms,
	expected_status, failure_threshold, enabled, state, consecutive_fails,
	last_checked_at, tls_expires_at, tls_issuer, created_at, updated_at`

func scanMonitor(row pgx.Row) (Monitor, error) {
	var m Monitor
	err := row.Scan(&m.ID, &m.TenantID, &m.Name, &m.Kind, &m.PeriodSeconds, &m.GraceSeconds,
		&m.PingToken, &m.LastPingAt,
		&m.URL, &m.Method, &m.IntervalSeconds,
		&m.TimeoutMs, &m.ExpectedStatus, &m.FailureThreshold, &m.Enabled, &m.State,
		&m.ConsecutiveFails, &m.LastCheckedAt, &m.TLSExpiresAt, &m.TLSIssuer,
		&m.CreatedAt, &m.UpdatedAt)
	if errors.Is(err, pgx.ErrNoRows) {
		return Monitor{}, ErrNotFound
	}
	if err != nil {
		return Monitor{}, fmt.Errorf("scan monitor: %w", err)
	}
	return m, nil
}

// --- monitors (tenant-scoped: call inside db.WithTenant) ---

func ListMonitors(ctx context.Context, q Querier, tenantID string) ([]Monitor, error) {
	rows, err := q.Query(ctx, `SELECT `+monitorCols+` FROM monitors
		WHERE tenant_id = $1 ORDER BY created_at`, tenantID)
	if err != nil {
		return nil, fmt.Errorf("list monitors: %w", err)
	}
	defer rows.Close()

	monitors := []Monitor{}
	for rows.Next() {
		m, err := scanMonitor(rows)
		if err != nil {
			return nil, err
		}
		monitors = append(monitors, m)
	}
	return monitors, rows.Err()
}

// NewMonitor is the validated input for CreateMonitor. Kind "http" uses
// URL/Method/ExpectedStatus/TimeoutMs; kind "heartbeat" uses
// PeriodSeconds/GraceSeconds and gets a generated PingToken.
type NewMonitor struct {
	Name             string
	Kind             string
	URL              string
	Method           string
	IntervalSeconds  int
	TimeoutMs        int
	ExpectedStatus   int
	FailureThreshold int
	PeriodSeconds    int
	GraceSeconds     int
}

func CreateMonitor(ctx context.Context, q Querier, tenantID string, n NewMonitor) (Monitor, error) {
	if n.Kind == "" {
		n.Kind = "http"
	}
	// period/grace are meaningless for http monitors but still carry NOT NULL
	// CHECK constraints, so fall back to the column defaults rather than
	// inserting a zero the constraint rejects.
	if n.PeriodSeconds == 0 {
		n.PeriodSeconds = 3600
	}
	if n.GraceSeconds == 0 {
		n.GraceSeconds = 300
	}
	var token *string
	if n.Kind == "heartbeat" {
		t, err := NewPingToken()
		if err != nil {
			return Monitor{}, err
		}
		token = &t
	}
	m, err := scanMonitor(q.QueryRow(ctx, `INSERT INTO monitors
		(tenant_id, name, kind, url, method, interval_seconds, timeout_ms,
		 expected_status, failure_threshold, period_seconds, grace_seconds, ping_token)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12)
		RETURNING `+monitorCols,
		tenantID, n.Name, n.Kind, n.URL, n.Method, n.IntervalSeconds, n.TimeoutMs,
		n.ExpectedStatus, n.FailureThreshold, n.PeriodSeconds, n.GraceSeconds, token))
	if err != nil {
		return Monitor{}, err
	}

	// Mirror the token into the RLS-free lookup table so an unauthenticated
	// ping can resolve it (see migration 0004). Same transaction, and the FK
	// cascade removes it with the monitor.
	if token != nil {
		if _, err := q.Exec(ctx, `INSERT INTO heartbeat_tokens (token, monitor_id, tenant_id)
			VALUES ($1, $2, $3)`, *token, m.ID, tenantID); err != nil {
			return Monitor{}, fmt.Errorf("register ping token: %w", err)
		}
	}
	return m, nil
}

// NewPingToken mints the unguessable capability embedded in a heartbeat's
// ping URL: 160 bits of crypto/rand, URL-safe.
func NewPingToken() (string, error) {
	b := make([]byte, 20)
	if _, err := rand.Read(b); err != nil {
		return "", fmt.Errorf("generate ping token: %w", err)
	}
	return base64.RawURLEncoding.EncodeToString(b), nil
}

// TokenLookup resolves a ping token to its monitor and tenant. Reads the
// RLS-free heartbeat_tokens table: an unauthenticated ping has no tenant
// context, so this is what establishes one (see migration 0004).
func TokenLookup(ctx context.Context, q Querier, token string) (monitorID, tenantID string, err error) {
	err = q.QueryRow(ctx, `SELECT monitor_id, tenant_id FROM heartbeat_tokens
		WHERE token = $1`, token).Scan(&monitorID, &tenantID)
	if errors.Is(err, pgx.ErrNoRows) {
		return "", "", ErrNotFound
	}
	if err != nil {
		return "", "", fmt.Errorf("lookup ping token: %w", err)
	}
	return monitorID, tenantID, nil
}

// RecordPing marks a heartbeat as just-seen, clears its failure state, and
// resolves an open incident if it was down. tx must already be scoped to the
// monitor's tenant (via db.WithTenant) so RLS applies to the writes.
// Returns ErrNotFound if the monitor is disabled or no longer a heartbeat.
func RecordPing(ctx context.Context, q Querier, tenantID, monitorID string) error {
	tag, err := q.Exec(ctx, `UPDATE monitors
		SET last_ping_at = now(), state = 'up', consecutive_fails = 0, updated_at = now()
		WHERE id = $1 AND tenant_id = $2 AND kind = 'heartbeat' AND enabled`,
		monitorID, tenantID)
	if err != nil {
		return fmt.Errorf("record ping: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return ErrNotFound
	}

	// A ping is the recovery signal for a late heartbeat.
	if _, err := q.Exec(ctx, `UPDATE incidents SET resolved_at = now()
		WHERE monitor_id = $1 AND tenant_id = $2 AND resolved_at IS NULL`,
		monitorID, tenantID); err != nil {
		return fmt.Errorf("resolve heartbeat incident: %w", err)
	}
	return nil
}

func GetMonitor(ctx context.Context, q Querier, tenantID, id string) (Monitor, error) {
	return scanMonitor(q.QueryRow(ctx, `SELECT `+monitorCols+` FROM monitors
		WHERE id = $1 AND tenant_id = $2`, id, tenantID))
}

// MonitorPatch carries optional fields for a partial update; nil = unchanged.
type MonitorPatch struct {
	Name             *string
	URL              *string
	Method           *string
	IntervalSeconds  *int
	TimeoutMs        *int
	ExpectedStatus   *int
	FailureThreshold *int
	Enabled          *bool
	// Heartbeat-only.
	PeriodSeconds *int
	GraceSeconds  *int
}

func UpdateMonitor(ctx context.Context, q Querier, tenantID, id string, p MonitorPatch) (Monitor, error) {
	sets := []string{"updated_at = now()"}
	args := []any{id, tenantID}
	add := func(col string, v any) {
		args = append(args, v)
		sets = append(sets, fmt.Sprintf("%s = $%d", col, len(args)))
	}
	if p.Name != nil {
		add("name", *p.Name)
	}
	if p.URL != nil {
		// A new target means past state is meaningless: restart from unknown.
		add("url", *p.URL)
		sets = append(sets, "state = 'unknown'", "consecutive_fails = 0")
	}
	if p.Method != nil {
		add("method", *p.Method)
	}
	if p.IntervalSeconds != nil {
		add("interval_seconds", *p.IntervalSeconds)
	}
	if p.TimeoutMs != nil {
		add("timeout_ms", *p.TimeoutMs)
	}
	if p.ExpectedStatus != nil {
		add("expected_status", *p.ExpectedStatus)
	}
	if p.FailureThreshold != nil {
		add("failure_threshold", *p.FailureThreshold)
	}
	if p.Enabled != nil {
		add("enabled", *p.Enabled)
	}
	if p.PeriodSeconds != nil {
		add("period_seconds", *p.PeriodSeconds)
	}
	if p.GraceSeconds != nil {
		add("grace_seconds", *p.GraceSeconds)
	}
	m, err := scanMonitor(q.QueryRow(ctx, `UPDATE monitors SET `+strings.Join(sets, ", ")+`
		WHERE id = $1 AND tenant_id = $2 RETURNING `+monitorCols, args...))
	if err != nil {
		return Monitor{}, err
	}

	// A URL change resets the state machine to unknown, and pausing stops
	// probing — either way an open incident would otherwise never resolve
	// (the scheduler only resolves down->up transitions) and would show as
	// "ongoing" forever, blocking future incidents via the open-incident
	// unique index. Close it here, in the same transaction. notify_state is
	// forced terminal so this administrative close never emits a spurious
	// down or recovery alert.
	urlChanged := p.URL != nil
	paused := p.Enabled != nil && !*p.Enabled
	if urlChanged || paused {
		if _, err := q.Exec(ctx, `UPDATE incidents
			SET resolved_at = now(), notify_state = 'recovered_notified'
			WHERE monitor_id = $1 AND tenant_id = $2 AND resolved_at IS NULL`,
			id, tenantID); err != nil {
			return Monitor{}, fmt.Errorf("resolve open incident on config change: %w", err)
		}
	}
	return m, nil
}

func DeleteMonitor(ctx context.Context, q Querier, tenantID, id string) error {
	tag, err := q.Exec(ctx, `DELETE FROM monitors WHERE id = $1 AND tenant_id = $2`, id, tenantID)
	if err != nil {
		return fmt.Errorf("delete monitor: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return ErrNotFound
	}
	return nil
}

// --- results + uptime (tenant-scoped) ---

func ListResults(ctx context.Context, q Querier, tenantID, monitorID string, since time.Time, limit int) ([]Result, error) {
	rows, err := q.Query(ctx, `SELECT id, monitor_id, checked_at, ok, status_code, latency_ms, error,
			dns_ms, connect_ms, tls_ms, ttfb_ms, failure_phase
		FROM monitor_results
		WHERE monitor_id = $1 AND tenant_id = $2 AND checked_at >= $3
		ORDER BY checked_at DESC LIMIT $4`, monitorID, tenantID, since, limit)
	if err != nil {
		return nil, fmt.Errorf("list results: %w", err)
	}
	defer rows.Close()

	results := []Result{}
	for rows.Next() {
		var r Result
		if err := rows.Scan(&r.ID, &r.MonitorID, &r.CheckedAt, &r.OK, &r.StatusCode, &r.LatencyMs, &r.Error,
			&r.DNSMs, &r.ConnectMs, &r.TLSMs, &r.TTFBMs, &r.FailurePhase); err != nil {
			return nil, fmt.Errorf("scan result: %w", err)
		}
		results = append(results, r)
	}
	return results, rows.Err()
}

// Uptime returns ok and total check counts since the given time.
func Uptime(ctx context.Context, q Querier, tenantID, monitorID string, since time.Time) (ok, total int64, err error) {
	err = q.QueryRow(ctx, `SELECT count(*) FILTER (WHERE ok), count(*)
		FROM monitor_results
		WHERE monitor_id = $1 AND tenant_id = $2 AND checked_at >= $3`,
		monitorID, tenantID, since).Scan(&ok, &total)
	if err != nil {
		return 0, 0, fmt.Errorf("uptime: %w", err)
	}
	return ok, total, nil
}

// --- incidents (tenant-scoped) ---

func ListIncidents(ctx context.Context, q Querier, tenantID string, monitorID *string, limit int) ([]Incident, error) {
	rows, err := q.Query(ctx, `SELECT i.id, i.monitor_id, m.name, i.started_at, i.resolved_at, i.cause
		FROM incidents i JOIN monitors m ON m.id = i.monitor_id
		WHERE i.tenant_id = $1 AND ($2::uuid IS NULL OR i.monitor_id = $2::uuid)
		ORDER BY i.started_at DESC LIMIT $3`, tenantID, monitorID, limit)
	if err != nil {
		return nil, fmt.Errorf("list incidents: %w", err)
	}
	defer rows.Close()

	incidents := []Incident{}
	for rows.Next() {
		var i Incident
		if err := rows.Scan(&i.ID, &i.MonitorID, &i.MonitorName, &i.StartedAt, &i.ResolvedAt, &i.Cause); err != nil {
			return nil, fmt.Errorf("scan incident: %w", err)
		}
		incidents = append(incidents, i)
	}
	return incidents, rows.Err()
}

func GetIncident(ctx context.Context, q Querier, tenantID, id string) (Incident, error) {
	var i Incident
	err := q.QueryRow(ctx, `SELECT i.id, i.monitor_id, m.name, i.started_at, i.resolved_at, i.cause
		FROM incidents i JOIN monitors m ON m.id = i.monitor_id
		WHERE i.id = $1 AND i.tenant_id = $2`, id, tenantID).
		Scan(&i.ID, &i.MonitorID, &i.MonitorName, &i.StartedAt, &i.ResolvedAt, &i.Cause)
	if errors.Is(err, pgx.ErrNoRows) {
		return Incident{}, ErrNotFound
	}
	if err != nil {
		return Incident{}, fmt.Errorf("get incident: %w", err)
	}
	return i, nil
}

// --- alert channels (tenant-scoped) ---

func ListChannels(ctx context.Context, q Querier, tenantID string) ([]Channel, error) {
	rows, err := q.Query(ctx, `SELECT id, type, config, enabled, created_at
		FROM alert_channels WHERE tenant_id = $1 ORDER BY created_at`, tenantID)
	if err != nil {
		return nil, fmt.Errorf("list channels: %w", err)
	}
	defer rows.Close()

	channels := []Channel{}
	for rows.Next() {
		var c Channel
		if err := rows.Scan(&c.ID, &c.Type, &c.Config, &c.Enabled, &c.CreatedAt); err != nil {
			return nil, fmt.Errorf("scan channel: %w", err)
		}
		channels = append(channels, c)
	}
	return channels, rows.Err()
}

func CreateChannel(ctx context.Context, q Querier, tenantID, typ string, config map[string]any) (Channel, error) {
	var c Channel
	err := q.QueryRow(ctx, `INSERT INTO alert_channels (tenant_id, type, config)
		VALUES ($1, $2, $3) RETURNING id, type, config, enabled, created_at`,
		tenantID, typ, config).
		Scan(&c.ID, &c.Type, &c.Config, &c.Enabled, &c.CreatedAt)
	if err != nil {
		return Channel{}, fmt.Errorf("create channel: %w", err)
	}
	return c, nil
}

func GetChannel(ctx context.Context, q Querier, tenantID, id string) (Channel, error) {
	var c Channel
	err := q.QueryRow(ctx, `SELECT id, type, config, enabled, created_at
		FROM alert_channels WHERE id = $1 AND tenant_id = $2`, id, tenantID).
		Scan(&c.ID, &c.Type, &c.Config, &c.Enabled, &c.CreatedAt)
	if errors.Is(err, pgx.ErrNoRows) {
		return Channel{}, ErrNotFound
	}
	if err != nil {
		return Channel{}, fmt.Errorf("get channel: %w", err)
	}
	return c, nil
}

func DeleteChannel(ctx context.Context, q Querier, tenantID, id string) error {
	tag, err := q.Exec(ctx, `DELETE FROM alert_channels WHERE id = $1 AND tenant_id = $2`, id, tenantID)
	if err != nil {
		return fmt.Errorf("delete channel: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return ErrNotFound
	}
	return nil
}

// --- identity + provisioning (pool-level; no tenant context yet) ---

// TenantForSub resolves the tenant for a verified Auth0 subject. The ORDER BY
// MUST match userTenant (which backs /api/me) so a user with more than one
// membership is scoped to the same tenant the dashboard displays.
func TenantForSub(ctx context.Context, q Querier, sub string) (string, error) {
	var tenantID string
	err := q.QueryRow(ctx, `SELECT t.id
		FROM users u
		JOIN tenant_members tm ON tm.user_id = u.id
		JOIN tenants t ON t.id = tm.tenant_id
		WHERE u.auth0_sub = $1
		ORDER BY t.created_at LIMIT 1`, sub).Scan(&tenantID)
	if errors.Is(err, pgx.ErrNoRows) {
		return "", ErrNotFound
	}
	if err != nil {
		return "", fmt.Errorf("tenant for sub: %w", err)
	}
	return tenantID, nil
}

func TenantBySlug(ctx context.Context, q Querier, slug string) (Tenant, error) {
	var t Tenant
	err := q.QueryRow(ctx, `SELECT id, name, slug, public_status_enabled, created_at
		FROM tenants WHERE slug = $1`, slug).
		Scan(&t.ID, &t.Name, &t.Slug, &t.PublicStatusEnabled, &t.CreatedAt)
	if errors.Is(err, pgx.ErrNoRows) {
		return Tenant{}, ErrNotFound
	}
	if err != nil {
		return Tenant{}, fmt.Errorf("tenant by slug: %w", err)
	}
	return t, nil
}

// SetPublicStatus toggles a tenant's public status page on or off.
func SetPublicStatus(ctx context.Context, q Querier, tenantID string, enabled bool) error {
	tag, err := q.Exec(ctx, `UPDATE tenants SET public_status_enabled = $2 WHERE id = $1`,
		tenantID, enabled)
	if err != nil {
		return fmt.Errorf("set public status: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return ErrNotFound
	}
	return nil
}

// EnsureUser returns the user + tenant for an Auth0 subject, provisioning a
// user, personal tenant, and owner membership on first login. Concurrent
// first-login requests are serialized with an advisory lock so a double-fire
// of /api/me cannot create two tenants.
func EnsureUser(ctx context.Context, pool *pgxpool.Pool, sub, email, name string) (User, Tenant, error) {
	if u, t, err := userTenant(ctx, pool, sub); err == nil {
		return u, t, nil
	} else if !errors.Is(err, ErrNotFound) {
		return User{}, Tenant{}, err
	}

	tx, err := pool.Begin(ctx)
	if err != nil {
		return User{}, Tenant{}, fmt.Errorf("begin provisioning: %w", err)
	}
	defer func() { _ = tx.Rollback(ctx) }()

	if _, err := tx.Exec(ctx, `SELECT pg_advisory_xact_lock(hashtext($1))`, sub); err != nil {
		return User{}, Tenant{}, fmt.Errorf("provisioning lock: %w", err)
	}
	// Re-check inside the lock: a concurrent request may have provisioned.
	if u, t, err := userTenant(ctx, tx, sub); err == nil {
		return u, t, tx.Commit(ctx)
	} else if !errors.Is(err, ErrNotFound) {
		return User{}, Tenant{}, err
	}

	var u User
	err = tx.QueryRow(ctx, `INSERT INTO users (auth0_sub, email, name) VALUES ($1, $2, $3)
		ON CONFLICT (auth0_sub) DO UPDATE
			SET email = COALESCE(NULLIF(EXCLUDED.email, ''), users.email),
			    name  = COALESCE(NULLIF(EXCLUDED.name, ''), users.name)
		RETURNING id, auth0_sub, email, name, created_at`, sub, email, name).
		Scan(&u.ID, &u.Auth0Sub, &u.Email, &u.Name, &u.CreatedAt)
	if err != nil {
		return User{}, Tenant{}, fmt.Errorf("insert user: %w", err)
	}

	t, err := createTenantWithUniqueSlug(ctx, tx, tenantBaseName(email, name))
	if err != nil {
		return User{}, Tenant{}, err
	}
	if _, err := tx.Exec(ctx, `INSERT INTO tenant_members (tenant_id, user_id, role)
		VALUES ($1, $2, 'owner')`, t.ID, u.ID); err != nil {
		return User{}, Tenant{}, fmt.Errorf("insert membership: %w", err)
	}

	// Open the workspace's audit trail with its own creation. audit_log has
	// RLS, and this transaction has no tenant context yet (the tenant is being
	// created in it), so set one for the remainder of the transaction.
	if _, err := tx.Exec(ctx, `SELECT set_config('app.tenant_id', $1, true)`, t.ID); err != nil {
		return User{}, Tenant{}, fmt.Errorf("set tenant for audit: %w", err)
	}
	if err := Audit(ctx, tx, t.ID, Actor{Sub: sub, Email: email}, ActionUserFirstLogin,
		fmt.Sprintf("workspace %q created on first login", t.Slug), nil,
		map[string]any{"slug": t.Slug}); err != nil {
		return User{}, Tenant{}, err
	}
	if err := tx.Commit(ctx); err != nil {
		return User{}, Tenant{}, fmt.Errorf("commit provisioning: %w", err)
	}
	return u, t, nil
}

func userTenant(ctx context.Context, q Querier, sub string) (User, Tenant, error) {
	var u User
	var t Tenant
	err := q.QueryRow(ctx, `SELECT u.id, u.auth0_sub, u.email, u.name, u.created_at,
			t.id, t.name, t.slug, t.public_status_enabled, t.created_at
		FROM users u
		JOIN tenant_members tm ON tm.user_id = u.id
		JOIN tenants t ON t.id = tm.tenant_id
		WHERE u.auth0_sub = $1
		ORDER BY t.created_at LIMIT 1`, sub).
		Scan(&u.ID, &u.Auth0Sub, &u.Email, &u.Name, &u.CreatedAt,
			&t.ID, &t.Name, &t.Slug, &t.PublicStatusEnabled, &t.CreatedAt)
	if errors.Is(err, pgx.ErrNoRows) {
		return User{}, Tenant{}, ErrNotFound
	}
	if err != nil {
		return User{}, Tenant{}, fmt.Errorf("user tenant: %w", err)
	}
	return u, t, nil
}

func createTenantWithUniqueSlug(ctx context.Context, q Querier, base string) (Tenant, error) {
	slugBase := Slugify(base)
	for i := range 40 {
		candidate := slugBase
		if i > 0 {
			candidate = fmt.Sprintf("%s-%d", slugBase, i+1)
		}
		var t Tenant
		err := q.QueryRow(ctx, `INSERT INTO tenants (name, slug) VALUES ($1, $2)
			ON CONFLICT (slug) DO NOTHING
			RETURNING id, name, slug, public_status_enabled, created_at`, base, candidate).
			Scan(&t.ID, &t.Name, &t.Slug, &t.PublicStatusEnabled, &t.CreatedAt)
		if errors.Is(err, pgx.ErrNoRows) {
			continue // slug taken, try the next suffix
		}
		if err != nil {
			return Tenant{}, fmt.Errorf("insert tenant: %w", err)
		}
		return t, nil
	}
	return Tenant{}, fmt.Errorf("could not find a free slug for %q", slugBase)
}

// tenantBaseName picks a human name for the auto-provisioned tenant: the
// email localpart, else the profile name, else a generic fallback.
func tenantBaseName(email, name string) string {
	if at := strings.IndexByte(email, '@'); at > 0 {
		return email[:at]
	}
	if name != "" {
		return name
	}
	return "team"
}

// Slugify lowercases and reduces s to [a-z0-9-], collapsing runs and
// trimming to 30 chars. An empty result becomes "team".
func Slugify(s string) string {
	var b strings.Builder
	lastDash := true // suppress a leading dash
	for _, r := range strings.ToLower(s) {
		switch {
		case r >= 'a' && r <= 'z' || r >= '0' && r <= '9':
			b.WriteRune(r)
			lastDash = false
		default:
			if !lastDash {
				b.WriteByte('-')
				lastDash = true
			}
		}
		if b.Len() >= 30 {
			break
		}
	}
	out := strings.Trim(b.String(), "-")
	if out == "" {
		return "team"
	}
	return out
}
