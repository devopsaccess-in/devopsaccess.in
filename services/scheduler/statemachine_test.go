package main

import "testing"

func TestEvaluate(t *testing.T) {
	tests := []struct {
		name      string
		state     string
		fails     int
		threshold int
		ok        bool
		want      transition
	}{
		{"unknown first success", "unknown", 0, 2, true,
			transition{NewState: "up", NewFails: 0}},
		{"up stays up", "up", 0, 2, true,
			transition{NewState: "up", NewFails: 0}},
		{"up success resets fails", "up", 1, 2, true,
			transition{NewState: "up", NewFails: 0}},
		{"up first failure below threshold", "up", 0, 2, false,
			transition{NewState: "up", NewFails: 1}},
		{"up crosses threshold", "up", 1, 2, false,
			transition{NewState: "down", NewFails: 2, OpenIncident: true}},
		{"threshold one goes straight down", "up", 0, 1, false,
			transition{NewState: "down", NewFails: 1, OpenIncident: true}},
		{"unknown failure below threshold", "unknown", 0, 2, false,
			transition{NewState: "unknown", NewFails: 1}},
		{"unknown crosses threshold", "unknown", 1, 2, false,
			transition{NewState: "down", NewFails: 2, OpenIncident: true}},
		{"down stays down, no second incident", "down", 2, 2, false,
			transition{NewState: "down", NewFails: 3}},
		{"down recovers on first success", "down", 5, 2, true,
			transition{NewState: "up", NewFails: 0, ResolveIncident: true}},
		{"high threshold accumulates", "up", 3, 10, false,
			transition{NewState: "up", NewFails: 4}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := evaluate(tt.state, tt.fails, tt.threshold, tt.ok)
			if got != tt.want {
				t.Fatalf("evaluate(%q, %d, %d, %v) = %+v, want %+v",
					tt.state, tt.fails, tt.threshold, tt.ok, got, tt.want)
			}
		})
	}
}
