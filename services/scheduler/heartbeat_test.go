package main

import (
	"strings"
	"testing"
	"time"
)

func TestEvaluateHeartbeat(t *testing.T) {
	now := time.Date(2026, 8, 2, 12, 0, 0, 0, time.UTC)
	ago := func(d time.Duration) *time.Time { t := now.Add(-d); return &t }

	// Hourly heartbeat with 5 minutes of grace => healthy up to 65 minutes.
	const period, grace = 3600, 300

	tests := []struct {
		name       string
		lastPing   *time.Time
		createdAt  time.Time
		wantOK     bool
		wantErrHas string
	}{
		{
			name:      "pinged just now",
			lastPing:  ago(1 * time.Minute),
			createdAt: now.Add(-24 * time.Hour),
			wantOK:    true,
		},
		{
			name:      "inside the period",
			lastPing:  ago(59 * time.Minute),
			createdAt: now.Add(-24 * time.Hour),
			wantOK:    true,
		},
		{
			name:      "inside the grace window",
			lastPing:  ago(64 * time.Minute),
			createdAt: now.Add(-24 * time.Hour),
			wantOK:    true,
		},
		{
			name:       "just past grace is late",
			lastPing:   ago(66 * time.Minute),
			createdAt:  now.Add(-24 * time.Hour),
			wantOK:     false,
			wantErrHas: "late",
		},
		{
			name:       "very late reports how late",
			lastPing:   ago(25 * time.Hour),
			createdAt:  now.Add(-48 * time.Hour),
			wantOK:     false,
			wantErrHas: "late",
		},
		{
			name:      "never pinged but still inside the first window",
			lastPing:  nil,
			createdAt: now.Add(-30 * time.Minute),
			wantOK:    true,
		},
		{
			name:       "never pinged and the first window elapsed",
			lastPing:   nil,
			createdAt:  now.Add(-3 * time.Hour),
			wantOK:     false,
			wantErrHas: "no heartbeat received since the monitor was created",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r := evaluateHeartbeat(now, tt.lastPing, tt.createdAt, period, grace)
			if r.OK != tt.wantOK {
				t.Fatalf("OK = %v, want %v (err %q)", r.OK, tt.wantOK, r.Error)
			}
			if tt.wantOK {
				if r.Error != "" || r.FailurePhase != "" {
					t.Fatalf("healthy result should carry no error: %+v", r)
				}
				return
			}
			if r.FailurePhase != "heartbeat" {
				t.Errorf("phase = %q, want heartbeat", r.FailurePhase)
			}
			if !strings.Contains(r.Error, tt.wantErrHas) {
				t.Errorf("error %q does not contain %q", r.Error, tt.wantErrHas)
			}
		})
	}
}

// A daily backup job: the long-period case the feature exists for.
func TestEvaluateHeartbeatDailyJob(t *testing.T) {
	now := time.Date(2026, 8, 2, 3, 0, 0, 0, time.UTC)
	const period, grace = 86400, 3600 // daily, 1h grace

	yesterday := now.Add(-23 * time.Hour)
	if r := evaluateHeartbeat(now, &yesterday, now.Add(-90*24*time.Hour), period, grace); !r.OK {
		t.Fatalf("a backup that ran 23h ago should be healthy: %q", r.Error)
	}

	stale := now.Add(-3 * 24 * time.Hour)
	r := evaluateHeartbeat(now, &stale, now.Add(-90*24*time.Hour), period, grace)
	if r.OK {
		t.Fatal("a backup silent for 3 days must be down")
	}
	// 3 days silent minus the 25h window = 47h late.
	if !strings.Contains(r.Error, "47 hours late") {
		t.Errorf("error should quantify the lateness, got %q", r.Error)
	}
	if !strings.Contains(r.Error, "expected every 24 hours") {
		t.Errorf("error should state the expected cadence, got %q", r.Error)
	}
}
