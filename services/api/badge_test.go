package main

import (
	"strings"
	"testing"
)

func TestUptimeColor(t *testing.T) {
	tests := []struct {
		pct  float64
		want string
	}{
		{100, colorBright},
		{99.95, colorBright},
		{99.5, colorBright},
		{99.2, colorGreen},
		{99.0, colorGreen},
		{97.0, colorYellow},
		{95.0, colorYellow},
		{94.9, colorRed},
		{80.0, colorRed},
	}
	for _, tt := range tests {
		if got := uptimeColor(tt.pct); got != tt.want {
			t.Errorf("uptimeColor(%.2f) = %s, want %s", tt.pct, got, tt.want)
		}
	}
}

func TestFormatPct(t *testing.T) {
	tests := []struct {
		in   float64
		want string
	}{
		{100, "100%"},
		{100.0, "100%"},
		{99.9399, "99.94%"},
		{99.005, "99.00%"},
		{0, "0.00%"},
	}
	for _, tt := range tests {
		if got := formatPct(tt.in); got != tt.want {
			t.Errorf("formatPct(%v) = %q, want %q", tt.in, got, tt.want)
		}
	}
}

func TestBadgeLabel(t *testing.T) {
	if got := badgeLabel("", "uptime"); got != "uptime" {
		t.Errorf("default uptime label = %q", got)
	}
	if got := badgeLabel("", "status"); got != "status" {
		t.Errorf("default status label = %q", got)
	}
	if got := badgeLabel("API", "uptime"); got != "API" {
		t.Errorf("custom label = %q", got)
	}
	long := strings.Repeat("x", 60)
	if got := badgeLabel(long, "uptime"); len(got) != 40 {
		t.Errorf("label not capped to 40: len=%d", len(got))
	}
}

func TestRenderBadge(t *testing.T) {
	svg := renderBadge("uptime", "99.94%", colorBright)

	if !strings.HasPrefix(svg, "<svg") || !strings.HasSuffix(svg, "</svg>") {
		t.Fatal("not a well-formed svg envelope")
	}
	for _, must := range []string{
		`role="img"`,        // sanity
		colorBright,         // the value-segment color
		colorLabel,          // the label-segment color
		`>uptime</text>`,    // label text present
		`>99.94%</text>`,    // value text present
		`aria-label="uptime: 99.94%"`,
	} {
		if !strings.Contains(svg, must) {
			t.Errorf("svg missing %q\n%s", must, svg)
		}
	}
}

func TestRenderBadgeEscapesInjection(t *testing.T) {
	// A caller-supplied label must not be able to break out of the markup.
	svg := renderBadge(`"><script>alert(1)</script>`, "up", colorBright)
	if strings.Contains(svg, "<script>") {
		t.Fatalf("unescaped script tag leaked into svg:\n%s", svg)
	}
	if !strings.Contains(svg, "&lt;script&gt;") {
		t.Fatalf("expected escaped label in svg:\n%s", svg)
	}
}
