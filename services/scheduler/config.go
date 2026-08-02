package main

import (
	"fmt"
	"os"
	"strconv"
)

// config holds runtime configuration, all sourced from the environment. The
// scheduler connects as the uptime_scheduler role (BYPASSRLS) — it works
// across all tenants by design.
type config struct {
	databaseURL string
	workers     int
	tickSeconds int

	smtpHost string
	smtpPort string
	mailFrom string

	// allowPrivateTargets disables the SSRF guards on probe and webhook
	// URLs. E2E-test hook ONLY — never set in production.
	allowPrivateTargets bool
}

func loadConfig() (config, error) {
	c := config{
		databaseURL: os.Getenv("DATABASE_URL"),
		workers:     20,
		tickSeconds: 10,
		smtpHost:    envOr("SMTP_HOST", "127.0.0.1"),
		smtpPort:    envOr("SMTP_PORT", "25"),
		mailFrom:    envOr("ALERT_FROM", "support@devopsaccess.in"),

		allowPrivateTargets: os.Getenv("UPTIME_ALLOW_PRIVATE_TARGETS") == "true",
	}
	if v := os.Getenv("SCHEDULER_WORKERS"); v != "" {
		n, err := strconv.Atoi(v)
		if err != nil || n < 1 || n > 100 {
			return config{}, fmt.Errorf("SCHEDULER_WORKERS must be 1-100")
		}
		c.workers = n
	}
	// E2E-test hook: faster claim/notify loop so tests are not bound to the
	// production 10s cadence.
	if v := os.Getenv("SCHEDULER_TICK_SECONDS"); v != "" {
		n, err := strconv.Atoi(v)
		if err != nil || n < 1 || n > 60 {
			return config{}, fmt.Errorf("SCHEDULER_TICK_SECONDS must be 1-60")
		}
		c.tickSeconds = n
	}
	if c.databaseURL == "" {
		return config{}, fmt.Errorf("DATABASE_URL is required")
	}
	return c, nil
}

func envOr(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}
