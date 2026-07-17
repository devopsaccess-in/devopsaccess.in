package main

import (
	"testing"
	"time"
)

func TestParseWindow(t *testing.T) {
	tests := []struct {
		name    string
		in      string
		def     time.Duration
		max     time.Duration
		want    time.Duration
		wantErr bool
	}{
		{"empty uses default", "", 24 * time.Hour, 7 * 24 * time.Hour, 24 * time.Hour, false},
		{"hours", "12h", 24 * time.Hour, 7 * 24 * time.Hour, 12 * time.Hour, false},
		{"days", "7d", 24 * time.Hour, 30 * 24 * time.Hour, 7 * 24 * time.Hour, false},
		{"at max", "30d", 7 * 24 * time.Hour, 30 * 24 * time.Hour, 30 * 24 * time.Hour, false},
		{"over max", "31d", 7 * 24 * time.Hour, 30 * 24 * time.Hour, 0, true},
		{"zero", "0d", 24 * time.Hour, 7 * 24 * time.Hour, 0, true},
		{"negative", "-1h", 24 * time.Hour, 7 * 24 * time.Hour, 0, true},
		{"no unit", "24", 24 * time.Hour, 7 * 24 * time.Hour, 0, true},
		{"garbage", "abc", 24 * time.Hour, 7 * 24 * time.Hour, 0, true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := parseWindow(tt.in, tt.def, tt.max)
			if (err != nil) != tt.wantErr {
				t.Fatalf("parseWindow(%q) error = %v, wantErr %v", tt.in, err, tt.wantErr)
			}
			if got != tt.want {
				t.Fatalf("parseWindow(%q) = %v, want %v", tt.in, got, tt.want)
			}
		})
	}
}

func TestValidateMonitorFields(t *testing.T) {
	tests := []struct {
		name                                  string
		mName, method                         string
		interval, timeout, expected, threshold int
		wantErr                               bool
	}{
		{"valid defaults", "prod api", "GET", 60, 10000, 200, 2, false},
		{"HEAD ok", "m", "HEAD", 300, 30000, 204, 10, false},
		{"empty name", "", "GET", 60, 10000, 200, 2, true},
		{"whitespace name", "   ", "GET", 60, 10000, 200, 2, true},
		{"POST rejected", "m", "POST", 60, 10000, 200, 2, true},
		{"interval too small", "m", "GET", 59, 10000, 200, 2, true},
		{"interval too large", "m", "GET", 301, 10000, 200, 2, true},
		{"timeout too small", "m", "GET", 60, 999, 200, 2, true},
		{"timeout too large", "m", "GET", 60, 30001, 200, 2, true},
		{"status too small", "m", "GET", 60, 10000, 99, 2, true},
		{"status too large", "m", "GET", 60, 10000, 600, 2, true},
		{"threshold zero", "m", "GET", 60, 10000, 200, 0, true},
		{"threshold too large", "m", "GET", 60, 10000, 200, 11, true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validateMonitorFields(tt.mName, tt.method, tt.interval, tt.timeout, tt.expected, tt.threshold)
			if (err != nil) != tt.wantErr {
				t.Fatalf("error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestValidateChannel(t *testing.T) {
	tests := []struct {
		name    string
		typ     string
		config  map[string]any
		wantErr bool
	}{
		{"valid email", "email", map[string]any{"to": "ops@example.com"}, false},
		{"email with display name", "email", map[string]any{"to": "Ops <ops@example.com>"}, false},
		{"email missing to", "email", map[string]any{}, true},
		{"email invalid", "email", map[string]any{"to": "not-an-email"}, true},
		{"valid slack", "slack_webhook", map[string]any{"url": "https://hooks.slack.com/services/T0/B0/xyz"}, false},
		{"slack http rejected", "slack_webhook", map[string]any{"url": "http://hooks.slack.com/services/x"}, true},
		{"slack missing url", "slack_webhook", map[string]any{}, true},
		{"unknown type", "pager", map[string]any{}, true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg, err := validateChannel(tt.typ, tt.config)
			if (err != nil) != tt.wantErr {
				t.Fatalf("error = %v, wantErr %v", err, tt.wantErr)
			}
			if err == nil && len(cfg) != 1 {
				t.Fatalf("config should keep exactly the known key, got %v", cfg)
			}
		})
	}
}
