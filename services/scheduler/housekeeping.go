package main

import (
	"context"
	"time"
)

// purgeOldResults enforces the 30-day monitor_results retention. Runs at
// startup and then nightly; the volume at MVP scale (per-minute checks) makes
// a single DELETE fine.
func (p *prober) purgeOldResults(ctx context.Context) {
	ctx, cancel := context.WithTimeout(ctx, 5*time.Minute)
	defer cancel()

	tag, err := p.pool.Exec(ctx,
		`DELETE FROM monitor_results WHERE checked_at < now() - interval '30 days'`)
	if err != nil {
		p.log.Error().Err(err).Msg("purge old results failed")
		return
	}
	if n := tag.RowsAffected(); n > 0 {
		p.log.Info().Int64("rows", n).Msg("purged old monitor results")
	}

	// Audit entries are kept longer than raw check results: they are the
	// record customers and support actually go back through.
	auditTag, err := p.pool.Exec(ctx,
		`DELETE FROM audit_log WHERE created_at < now() - interval '90 days'`)
	if err != nil {
		p.log.Error().Err(err).Msg("purge old audit entries failed")
		return
	}
	if n := auditTag.RowsAffected(); n > 0 {
		p.log.Info().Int64("rows", n).Msg("purged old audit entries")
	}
}
