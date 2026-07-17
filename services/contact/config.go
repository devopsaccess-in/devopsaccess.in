package main

import (
	"fmt"
	"os"
)

// config holds runtime configuration, all sourced from the environment so no
// secrets live in the repo. SMTP credentials are a Google Workspace mailbox +
// app password.
type config struct {
	listenAddr string
	smtpHost   string
	smtpPort   string
	smtpUser   string
	smtpPass   string
	mailFrom   string
	mailTo     string
	allowedOri string // allowed CORS origin for the browser form

	// Optional Phase-2 features. Empty => the feature's endpoint is disabled.
	databaseURL     string // Postgres (waitlist + payments)
	razorpaySecret  string // Razorpay webhook signing secret
	turnstileSecret string // Cloudflare Turnstile secret (empty => verification off)
}

func loadConfig() (config, error) {
	c := config{
		listenAddr: envOr("CONTACT_LISTEN_ADDR", "127.0.0.1:8080"),
		smtpHost:   envOr("SMTP_HOST", "smtp.gmail.com"),
		smtpPort:   envOr("SMTP_PORT", "587"),
		smtpUser:   os.Getenv("SMTP_USER"),
		smtpPass:   os.Getenv("SMTP_PASS"),
		mailTo:     envOr("CONTACT_TO", "support@devopsaccess.in"),
		allowedOri: envOr("CONTACT_ALLOWED_ORIGIN", "https://devopsaccess.in"),

		databaseURL:     os.Getenv("DATABASE_URL"),
		razorpaySecret:  os.Getenv("RAZORPAY_WEBHOOK_SECRET"),
		turnstileSecret: os.Getenv("TURNSTILE_SECRET"),
	}
	c.mailFrom = envOr("CONTACT_FROM", c.smtpUser)

	// SMTP_USER/SMTP_PASS are optional: when sending via the local Postfix relay
	// (127.0.0.1:25) there is no auth — the relay authenticates to Gmail. A
	// recipient and from address are still required.
	if c.mailTo == "" || c.mailFrom == "" {
		return config{}, fmt.Errorf("CONTACT_TO and a from address (CONTACT_FROM/SMTP_USER) are required")
	}
	return c, nil
}

func envOr(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}
