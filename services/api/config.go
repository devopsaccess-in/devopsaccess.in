package main

import (
	"fmt"
	"os"
)

// config holds runtime configuration, all sourced from the environment so no
// secrets live in the repo. SMTP defaults target the VM's local Postfix relay
// (no auth) — used only for the channel-test endpoint; incident alerting
// itself lives in services/scheduler.
type config struct {
	listenAddr    string
	databaseURL   string // connects as the uptime_api role (RLS enforced)
	auth0Domain   string // e.g. devopsaccess.eu.auth0.com
	auth0Audience string

	smtpHost string
	smtpPort string
	mailFrom string

	// allowPrivateTargets disables the SSRF guards on monitor URLs and
	// channel webhooks (private IPs, non-80/443 ports, http webhooks).
	// E2E-test hook ONLY — never set in production.
	allowPrivateTargets bool
}

func loadConfig() (config, error) {
	c := config{
		listenAddr:    envOr("API_LISTEN_ADDR", "127.0.0.1:8081"),
		databaseURL:   os.Getenv("DATABASE_URL"),
		auth0Domain:   os.Getenv("AUTH0_DOMAIN"),
		auth0Audience: envOr("AUTH0_AUDIENCE", "https://api.devopsaccess.in"),
		smtpHost:      envOr("SMTP_HOST", "127.0.0.1"),
		smtpPort:      envOr("SMTP_PORT", "25"),
		mailFrom:      envOr("ALERT_FROM", "support@devopsaccess.in"),

		allowPrivateTargets: os.Getenv("UPTIME_ALLOW_PRIVATE_TARGETS") == "true",
	}
	if c.databaseURL == "" {
		return config{}, fmt.Errorf("DATABASE_URL is required")
	}
	if c.auth0Domain == "" {
		return config{}, fmt.Errorf("AUTH0_DOMAIN is required")
	}
	return c, nil
}

func envOr(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}
