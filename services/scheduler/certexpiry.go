package main

import (
	"context"
	"fmt"
	"time"
)

// Certificate-expiry warning rungs, in days remaining. A monitor climbs down
// the ladder at most once per rung per certificate (tls_warned_threshold),
// so a renewal resets it and a restart never re-sends.
const (
	certWarnDays   = 14
	certUrgentDays = 3
)

// expiringCert is an https monitor whose leaf certificate is near expiry.
type expiringCert struct {
	MonitorID   string
	TenantID    string
	MonitorName string
	ExpiresAt   time.Time
	Issuer      string
	Warned      int
}

// notifyCertExpiry warns tenants before a TLS certificate takes their site
// down — the outage this product exists to prevent, caught in advance rather
// than reported after. Runs on the same tick as incident notification.
func (p *prober) notifyCertExpiry(ctx context.Context) {
	rows, err := p.pool.Query(ctx, `SELECT id, tenant_id, name, tls_expires_at,
			tls_issuer, tls_warned_threshold
		FROM monitors
		WHERE enabled
		  AND tls_expires_at IS NOT NULL
		  AND tls_expires_at <= now() + ($1 || ' days')::interval
		ORDER BY tls_expires_at
		LIMIT 100`, certWarnDays)
	if err != nil {
		p.log.Error().Err(err).Msg("list expiring certs failed")
		return
	}
	defer rows.Close()

	var due []expiringCert
	for rows.Next() {
		var c expiringCert
		if err := rows.Scan(&c.MonitorID, &c.TenantID, &c.MonitorName, &c.ExpiresAt,
			&c.Issuer, &c.Warned); err != nil {
			p.log.Error().Err(err).Msg("scan expiring cert failed")
			return
		}
		due = append(due, c)
	}
	if err := rows.Err(); err != nil {
		p.log.Error().Err(err).Msg("iterate expiring certs failed")
		return
	}

	for _, c := range due {
		rung, ok := certRung(time.Until(c.ExpiresAt), c.Warned)
		if !ok {
			continue
		}
		p.warnCertExpiry(ctx, c, rung)
	}
}

// certRung decides which warning rung (if any) is owed for a certificate with
// `left` remaining, given the rung already sent. Pure — unit tested.
func certRung(left time.Duration, warned int) (rung int, owed bool) {
	days := left.Hours() / 24
	switch {
	case days <= certUrgentDays && warned != certUrgentDays:
		return certUrgentDays, true
	case days <= certWarnDays && warned == 0:
		return certWarnDays, true
	default:
		return 0, false
	}
}

func (p *prober) warnCertExpiry(ctx context.Context, c expiringCert, rung int) {
	channels, err := p.tenantChannels(ctx, c.TenantID)
	if err != nil {
		p.log.Error().Err(err).Str("monitor_id", c.MonitorID).Msg("load channels failed")
		return
	}
	subject, body := composeCertAlert(c)

	attempted, failed := p.sendAll(ctx, channels, subject, body,
		p.log.With().Str("monitor_id", c.MonitorID).Logger())
	// Nothing delivered => leave the rung unset so the next tick retries,
	// same principle as incident alerts.
	if attempted > 0 && failed == attempted {
		return
	}

	if _, err := p.pool.Exec(ctx, `UPDATE monitors SET tls_warned_threshold = $2
		WHERE id = $1`, c.MonitorID, rung); err != nil {
		p.log.Error().Err(err).Str("monitor_id", c.MonitorID).Msg("record cert warning failed")
		return
	}
	p.log.Info().Str("monitor_id", c.MonitorID).Int("rung_days", rung).
		Time("expires_at", c.ExpiresAt).Msg("sent TLS expiry warning")
}

// composeCertAlert renders the warning. Timestamps in IST (India-first).
func composeCertAlert(c expiringCert) (subject, body string) {
	ist := time.FixedZone("IST", 5*3600+1800)
	when := c.ExpiresAt.In(ist).Format("02 Jan 2006 15:04 MST")
	left := time.Until(c.ExpiresAt)

	issuer := c.Issuer
	if issuer == "" {
		issuer = "unknown"
	}

	if left <= 0 {
		subject = fmt.Sprintf("EXPIRED: TLS certificate for %s", c.MonitorName)
		body = fmt.Sprintf("The TLS certificate for %s expired %s (%s).\nIssuer: %s\n\n"+
			"Visitors are seeing a security warning. Renew it now.\n\n— DevOps Access uptime monitoring",
			c.MonitorName, humanAgo(c.ExpiresAt), when, issuer)
		return subject, body
	}

	subject = fmt.Sprintf("TLS certificate for %s expires in %s", c.MonitorName, humanDur(left))
	body = fmt.Sprintf("The TLS certificate for %s expires in %s (%s).\nIssuer: %s\n\n"+
		"Renew it before then to avoid an outage.\n\n— DevOps Access uptime monitoring",
		c.MonitorName, humanDur(left), when, issuer)
	return subject, body
}
