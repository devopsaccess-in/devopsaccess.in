package main

import (
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/jackc/pgx/v5"

	"github.com/devopsaccess-in/devopsaccess.in/services/shared/db"

	"github.com/devopsaccess-in/devopsaccess.in/services/api/internal/store"
)

// Badge colors (shields.io palette).
const (
	colorBright = "#4c1"     // healthy
	colorGreen  = "#97ca00"  // good
	colorYellow = "#dfb317"  // degraded
	colorRed    = "#e05d44"  // down / poor
	colorGray   = "#9f9f9f"  // unknown / unavailable
	colorLabel  = "#555"     // left segment
)

// handleBadge serves an embeddable SVG status badge for a monitor:
//
//	GET /api/badge/{slug}/{monitorID}.svg?metric=uptime&days=30&label=...
//
// Public, no auth — but gated on the tenant's public-status opt-in, so a badge
// can never expose a monitor the owner hasn't chosen to publish. Any
// not-found / not-enabled / bad-input case renders an identical neutral
// "unavailable" badge (HTTP 200) rather than a broken image, and reveals
// nothing about which tenants or monitors exist.
func (s *server) handleBadge(w http.ResponseWriter, r *http.Request) {
	slug := chi.URLParam(r, "slug")
	monitorID := strings.TrimSuffix(chi.URLParam(r, "monitor"), ".svg")

	metric := r.URL.Query().Get("metric")
	if metric != "status" {
		metric = "uptime"
	}
	label := badgeLabel(r.URL.Query().Get("label"), metric)

	value, color := s.badgeValue(r, slug, monitorID, metric)
	writeBadge(w, label, value, color)
}

// badgeValue resolves the right-hand text and color. It never returns an
// error to the caller — every failure path collapses to the neutral
// "unavailable" badge so embeds degrade gracefully and leak nothing.
func (s *server) badgeValue(r *http.Request, slug, monitorID, metric string) (value, color string) {
	if !isUUID(monitorID) {
		return "unavailable", colorGray
	}
	tenant, err := store.TenantBySlug(r.Context(), s.pool, slug)
	if err != nil || !tenant.PublicStatusEnabled {
		return "unavailable", colorGray
	}

	days := badgeDays(r.URL.Query().Get("days"))

	var m store.Monitor
	var ok, total int64
	err = db.WithTenant(r.Context(), s.pool, tenant.ID, func(tx pgx.Tx) error {
		var err error
		m, err = store.GetMonitor(r.Context(), tx, tenant.ID, monitorID)
		if err != nil {
			return err
		}
		if metric == "uptime" {
			ok, total, err = store.Uptime(r.Context(), tx, tenant.ID, monitorID, time.Now().Add(-days))
		}
		return err
	})
	if err != nil || !m.Enabled {
		return "unavailable", colorGray
	}

	if metric == "status" {
		switch m.State {
		case "up":
			return "up", colorBright
		case "down":
			return "down", colorRed
		default:
			return "pending", colorGray
		}
	}

	// metric == uptime
	if total == 0 {
		return "no data", colorGray
	}
	pct := float64(ok) / float64(total) * 100
	return formatPct(pct), uptimeColor(pct)
}

func badgeLabel(custom, metric string) string {
	if custom != "" {
		if len(custom) > 40 {
			custom = custom[:40]
		}
		return custom
	}
	if metric == "status" {
		return "status"
	}
	return "uptime"
}

func badgeDays(v string) time.Duration {
	switch v {
	case "1":
		return 24 * time.Hour
	case "7":
		return 7 * 24 * time.Hour
	case "90":
		return 90 * 24 * time.Hour
	default:
		return 30 * 24 * time.Hour
	}
}

func formatPct(pct float64) string {
	if pct >= 100 {
		return "100%"
	}
	return strconv.FormatFloat(pct, 'f', 2, 64) + "%"
}

func uptimeColor(pct float64) string {
	switch {
	case pct >= 99.5:
		return colorBright
	case pct >= 99:
		return colorGreen
	case pct >= 95:
		return colorYellow
	default:
		return colorRed
	}
}
