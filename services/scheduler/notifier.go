package main

import (
	"context"
	"fmt"
	"time"

	"github.com/devopsaccess-in/devopsaccess.in/services/shared/notify"
)

// pendingIncident is an incident that owes its tenant a notification.
type pendingIncident struct {
	ID          string
	TenantID    string
	MonitorName string
	MonitorURL  string
	StartedAt   time.Time
	ResolvedAt  *time.Time
	Cause       string
	NotifyState string
}

// notifyIncidents sends down-alerts for incidents still 'pending' and
// recovery notices for resolved incidents still 'notified'. notify_state
// advances only after the send attempt, so a scheduler restart never loses an
// alert; sends themselves are attempted once per channel and failures are
// logged, not retried forever (a permanently broken webhook must not spam the
// loop — users can verify channels via the API's test endpoint).
func (p *prober) notifyIncidents(ctx context.Context) {
	rows, err := p.pool.Query(ctx, `SELECT i.id, i.tenant_id, m.name, m.url,
			i.started_at, i.resolved_at, i.cause, i.notify_state
		FROM incidents i JOIN monitors m ON m.id = i.monitor_id
		WHERE i.notify_state = 'pending'
		   OR (i.notify_state = 'notified' AND i.resolved_at IS NOT NULL)
		ORDER BY i.started_at
		LIMIT 50`)
	if err != nil {
		p.log.Error().Err(err).Msg("list pending incidents failed")
		return
	}
	defer rows.Close()

	var pending []pendingIncident
	for rows.Next() {
		var i pendingIncident
		if err := rows.Scan(&i.ID, &i.TenantID, &i.MonitorName, &i.MonitorURL,
			&i.StartedAt, &i.ResolvedAt, &i.Cause, &i.NotifyState); err != nil {
			p.log.Error().Err(err).Msg("scan pending incident failed")
			return
		}
		pending = append(pending, i)
	}
	if err := rows.Err(); err != nil {
		p.log.Error().Err(err).Msg("iterate pending incidents failed")
		return
	}

	for _, i := range pending {
		p.notifyOne(ctx, i)
	}
}

func (p *prober) notifyOne(ctx context.Context, i pendingIncident) {
	recovery := i.NotifyState == "notified" // resolved, owes a recovery notice
	subject, body := composeAlert(i, recovery)

	channels, err := p.tenantChannels(ctx, i.TenantID)
	if err != nil {
		p.log.Error().Err(err).Str("incident_id", i.ID).Msg("load channels failed")
		return
	}
	for _, c := range channels {
		sendCtx, cancel := context.WithTimeout(ctx, 15*time.Second)
		var sendErr error
		switch c.typ {
		case "email":
			sendErr = p.mailer.Send(sendCtx, c.target, subject, body)
		case "slack_webhook":
			sendErr = notify.Slack(sendCtx, p.slackClient, c.target, subject+"\n"+body)
		}
		cancel()
		if sendErr != nil {
			p.log.Warn().Err(sendErr).Str("incident_id", i.ID).Str("channel_type", c.typ).
				Msg("notification send failed")
		}
	}

	next := "notified"
	if recovery {
		next = "recovered_notified"
	}
	if _, err := p.pool.Exec(ctx, `UPDATE incidents SET notify_state = $2
		WHERE id = $1 AND notify_state = $3`, i.ID, next, i.NotifyState); err != nil {
		p.log.Error().Err(err).Str("incident_id", i.ID).Msg("advance notify_state failed")
	}
}

type channelTarget struct {
	typ    string
	target string // email address or webhook URL
}

func (p *prober) tenantChannels(ctx context.Context, tenantID string) ([]channelTarget, error) {
	rows, err := p.pool.Query(ctx, `SELECT type,
			COALESCE(config->>'to', config->>'url', '')
		FROM alert_channels WHERE tenant_id = $1 AND enabled`, tenantID)
	if err != nil {
		return nil, fmt.Errorf("list channels: %w", err)
	}
	defer rows.Close()

	var out []channelTarget
	for rows.Next() {
		var c channelTarget
		if err := rows.Scan(&c.typ, &c.target); err != nil {
			return nil, fmt.Errorf("scan channel: %w", err)
		}
		if c.target != "" {
			out = append(out, c)
		}
	}
	return out, rows.Err()
}

// composeAlert renders the notification text for a down alert or a recovery
// notice. Timestamps are shown in IST — India-first product rule.
func composeAlert(i pendingIncident, recovery bool) (subject, body string) {
	ist := time.FixedZone("IST", 5*3600+1800)
	started := i.StartedAt.In(ist).Format("02 Jan 2006 15:04:05 MST")

	if recovery && i.ResolvedAt != nil {
		dur := i.ResolvedAt.Sub(i.StartedAt).Round(time.Second)
		subject = fmt.Sprintf("RESOLVED: %s is back up", i.MonitorName)
		body = fmt.Sprintf("Monitor: %s\nURL: %s\nDown since: %s\nDowntime: %s\n\n— DevOps Access uptime monitoring",
			i.MonitorName, i.MonitorURL, started, dur)
		return subject, body
	}
	subject = fmt.Sprintf("DOWN: %s is failing", i.MonitorName)
	body = fmt.Sprintf("Monitor: %s\nURL: %s\nSince: %s\nCause: %s\n\n— DevOps Access uptime monitoring",
		i.MonitorName, i.MonitorURL, started, i.Cause)
	return subject, body
}
