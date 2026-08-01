package main

import (
	"context"
	"fmt"
	"net/http"
	"time"
	"unicode/utf8"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/rs/zerolog"
)

// job is a claimed monitor due for a check.
type job struct {
	ID               string
	TenantID         string
	Name             string
	Kind             string // "http" (we fetch) or "heartbeat" (they ping us)
	URL              string
	Method           string
	TimeoutMs        int
	ExpectedStatus   int
	FailureThreshold int
	State            string
	ConsecutiveFails int
	// Heartbeat fields.
	PeriodSeconds int
	GraceSeconds  int
	LastPingAt    *time.Time
	CreatedAt     time.Time
}

// prober claims due monitors, probes them, and applies the state machine.
// It connects as uptime_scheduler (BYPASSRLS): cross-tenant by design.
type prober struct {
	pool   *pgxpool.Pool
	log    zerolog.Logger
	mailer interface {
		Send(ctx context.Context, to, subject, body string) error
	}
	slackClient *http.Client
	jobs        chan job
	// insecureTargets probes with a plain HTTP client instead of the
	// SSRF-guarded one. E2E-test hook ONLY.
	insecureTargets bool
}

// claimDue atomically claims monitors whose interval has elapsed by stamping
// last_checked_at at claim time — a claimed monitor cannot be dispatched
// twice, even while its check is still in flight. SKIP LOCKED keeps a second
// scheduler instance (deploy overlap) from double-claiming.
func (p *prober) claimDue(ctx context.Context, limit int) ([]job, error) {
	rows, err := p.pool.Query(ctx, `UPDATE monitors SET last_checked_at = now()
		WHERE id IN (
			SELECT id FROM monitors
			WHERE enabled
			  AND (last_checked_at IS NULL
			       OR last_checked_at + interval_seconds * interval '1 second' <= now())
			ORDER BY last_checked_at ASC NULLS FIRST
			LIMIT $1
			FOR UPDATE SKIP LOCKED
		)
		RETURNING id, tenant_id, name, kind, url, method, timeout_ms, expected_status,
			failure_threshold, state, consecutive_fails,
			period_seconds, grace_seconds, last_ping_at, created_at`, limit)
	if err != nil {
		return nil, fmt.Errorf("claim due monitors: %w", err)
	}
	defer rows.Close()

	var jobs []job
	for rows.Next() {
		var j job
		if err := rows.Scan(&j.ID, &j.TenantID, &j.Name, &j.Kind, &j.URL, &j.Method, &j.TimeoutMs,
			&j.ExpectedStatus, &j.FailureThreshold, &j.State, &j.ConsecutiveFails,
			&j.PeriodSeconds, &j.GraceSeconds, &j.LastPingAt, &j.CreatedAt); err != nil {
			return nil, fmt.Errorf("scan claimed monitor: %w", err)
		}
		jobs = append(jobs, j)
	}
	return jobs, rows.Err()
}

// apply records the result and advances the monitor's state machine in one
// transaction. Incident notifications happen in the notifier pass, keyed off
// notify_state, so a crash between commit and send never loses an alert.
func (p *prober) apply(ctx context.Context, j job, r checkResult) error {
	tx, err := p.pool.Begin(ctx)
	if err != nil {
		return fmt.Errorf("begin apply: %w", err)
	}
	defer func() { _ = tx.Rollback(ctx) }()

	if _, err := tx.Exec(ctx, `INSERT INTO monitor_results
		(monitor_id, tenant_id, ok, status_code, latency_ms, error,
		 dns_ms, connect_ms, tls_ms, ttfb_ms, failure_phase)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11)`,
		j.ID, j.TenantID, r.OK, r.StatusCode, r.LatencyMs, r.Error,
		r.Timings.DNSMs, r.Timings.ConnectMs, r.Timings.TLSMs, r.Timings.TTFBMs,
		r.FailurePhase); err != nil {
		return fmt.Errorf("insert result: %w", err)
	}

	tr := evaluate(j.State, j.ConsecutiveFails, j.FailureThreshold, r.OK)
	if _, err := tx.Exec(ctx, `UPDATE monitors
		SET state = $2, consecutive_fails = $3, updated_at = now()
		WHERE id = $1`, j.ID, tr.NewState, tr.NewFails); err != nil {
		return fmt.Errorf("update monitor state: %w", err)
	}

	// Record the observed leaf certificate. A later expiry than we last saw
	// means the cert was renewed, so the expiry-warning ladder resets.
	if r.Cert != nil {
		if _, err := tx.Exec(ctx, `UPDATE monitors
			SET tls_expires_at = $2,
			    tls_issuer = $3,
			    tls_warned_threshold = CASE
			        WHEN tls_expires_at IS NULL OR $2 > tls_expires_at THEN 0
			        ELSE tls_warned_threshold
			    END
			WHERE id = $1`, j.ID, r.Cert.ExpiresAt, r.Cert.Issuer); err != nil {
			return fmt.Errorf("update tls info: %w", err)
		}
	}

	if tr.OpenIncident {
		cause := truncate(fmt.Sprintf("%d consecutive failures: %s", tr.NewFails, r.Error), 500)
		if _, err := tx.Exec(ctx, `INSERT INTO incidents (tenant_id, monitor_id, cause)
			VALUES ($1, $2, $3)
			ON CONFLICT (monitor_id) WHERE resolved_at IS NULL DO NOTHING`,
			j.TenantID, j.ID, cause); err != nil {
			return fmt.Errorf("open incident: %w", err)
		}
	}
	if tr.ResolveIncident {
		if _, err := tx.Exec(ctx, `UPDATE incidents SET resolved_at = now()
			WHERE monitor_id = $1 AND resolved_at IS NULL`, j.ID); err != nil {
			return fmt.Errorf("resolve incident: %w", err)
		}
	}
	if err := tx.Commit(ctx); err != nil {
		return fmt.Errorf("commit apply: %w", err)
	}
	return nil
}

// worker consumes jobs until the channel closes. Each iteration recovers its
// own panics so one bad check cannot kill the pool.
func (p *prober) worker(ctx context.Context) {
	for j := range p.jobs {
		p.runOne(ctx, j)
	}
}

func (p *prober) runOne(ctx context.Context, j job) {
	defer func() {
		if v := recover(); v != nil {
			p.log.Error().Interface("panic", v).Str("monitor_id", j.ID).Msg("check panicked")
		}
	}()
	// A heartbeat is evaluated against the clock, not fetched — everything
	// downstream (results, state machine, incidents, alerts) is identical.
	var r checkResult
	if j.Kind == "heartbeat" {
		r = evaluateHeartbeat(time.Now(), j.LastPingAt, j.CreatedAt, j.PeriodSeconds, j.GraceSeconds)
	} else {
		r = check(ctx, j, p.insecureTargets)
	}

	if err := p.apply(ctx, j, r); err != nil {
		p.log.Error().Err(err).Str("monitor_id", j.ID).Msg("apply check result failed")
		return
	}
	p.log.Debug().Str("monitor_id", j.ID).Str("kind", j.Kind).Bool("ok", r.OK).
		Int("latency_ms", r.LatencyMs).Msg("checked")
}

// tick claims everything due and enqueues it for the worker pool.
func (p *prober) tick(ctx context.Context) {
	jobs, err := p.claimDue(ctx, 100)
	if err != nil {
		p.log.Error().Err(err).Msg("claim failed")
		return
	}
	for _, j := range jobs {
		select {
		case p.jobs <- j:
		case <-ctx.Done():
			return
		}
	}
}

// truncate shortens s to at most n bytes on a rune boundary, so the result is
// always valid UTF-8 (Postgres rejects invalid byte sequences, which would
// otherwise fail the monitor_results INSERT and stall the whole pipeline).
func truncate(s string, n int) string {
	if len(s) <= n {
		return s
	}
	// Back up to the start of the rune that straddles the cut.
	for n > 0 && !utf8.RuneStart(s[n]) {
		n--
	}
	return s[:n]
}
