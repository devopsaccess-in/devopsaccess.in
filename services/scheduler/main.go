// Command scheduler is the uptime prober + alerter in one binary: every 10s
// it claims monitors that are due, probes them through an SSRF-guarded
// client, advances each monitor's up/down state machine (opening and
// resolving incidents), and delivers email/Slack notifications. It connects
// as the uptime_scheduler Postgres role (BYPASSRLS) — cross-tenant by design.
package main

import (
	"context"
	"os"
	"os/signal"
	"sync"
	"syscall"
	"time"

	"github.com/rs/zerolog"

	"github.com/devopsaccess-in/devopsaccess.in/services/shared/db"
	"github.com/devopsaccess-in/devopsaccess.in/services/shared/notify"
	"github.com/devopsaccess-in/devopsaccess.in/services/shared/safehttp"
)

func main() {
	log := zerolog.New(os.Stderr).With().Timestamp().Str("svc", "scheduler").Logger()

	cfg, err := loadConfig()
	if err != nil {
		log.Fatal().Err(err).Msg("invalid configuration")
	}

	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	connectCtx, cancel := context.WithTimeout(ctx, 60*time.Second)
	pool, err := db.Connect(connectCtx, cfg.databaseURL)
	cancel()
	if err != nil {
		log.Fatal().Err(err).Msg("database connect failed")
	}
	defer pool.Close()

	p := &prober{
		pool: pool,
		log:  log,
		mailer: &notify.Mailer{
			Host: cfg.smtpHost, Port: cfg.smtpPort, From: cfg.mailFrom,
		},
		// Slack webhook URLs are customer input: SSRF-guarded client only.
		slackClient: safehttp.Client(10 * time.Second),
		jobs:        make(chan job, 200),
	}

	var wg sync.WaitGroup
	for range cfg.workers {
		wg.Add(1)
		go func() {
			defer wg.Done()
			p.worker(ctx)
		}()
	}
	log.Info().Int("workers", cfg.workers).Msg("scheduler started")

	p.purgeOldResults(ctx)
	tick := time.NewTicker(10 * time.Second)
	defer tick.Stop()
	purge := time.NewTicker(24 * time.Hour)
	defer purge.Stop()

	for running := true; running; {
		select {
		case <-ctx.Done():
			running = false
		case <-tick.C:
			p.tick(ctx)
			p.notifyIncidents(ctx)
		case <-purge.C:
			p.purgeOldResults(ctx)
		}
	}

	close(p.jobs)
	wg.Wait()
	log.Info().Msg("scheduler stopped")
}
