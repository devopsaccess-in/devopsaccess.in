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

	smtpHost string
	smtpPort string
	mailFrom string
}

func loadConfig() (config, error) {
	c := config{
		databaseURL: os.Getenv("DATABASE_URL"),
		workers:     20,
		smtpHost:    envOr("SMTP_HOST", "127.0.0.1"),
		smtpPort:    envOr("SMTP_PORT", "25"),
		mailFrom:    envOr("ALERT_FROM", "alerts@devopsaccess.in"),
	}
	if v := os.Getenv("SCHEDULER_WORKERS"); v != "" {
		n, err := strconv.Atoi(v)
		if err != nil || n < 1 || n > 100 {
			return config{}, fmt.Errorf("SCHEDULER_WORKERS must be 1-100")
		}
		c.workers = n
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
