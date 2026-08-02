package main

import (
	"fmt"
	"net/http"
	"net/mail"
	"net/url"
	"strings"

	"github.com/go-chi/chi/v5"
	"github.com/jackc/pgx/v5"

	"github.com/devopsaccess-in/devopsaccess.in/services/shared/db"
	"github.com/devopsaccess-in/devopsaccess.in/services/shared/notify"

	"github.com/devopsaccess-in/devopsaccess.in/services/api/internal/store"
)

func (s *server) listChannels(w http.ResponseWriter, r *http.Request) {
	var channels []store.Channel
	err := db.WithTenant(r.Context(), s.pool, tenantID(r.Context()), func(tx pgx.Tx) error {
		var err error
		channels, err = store.ListChannels(r.Context(), tx, tenantID(r.Context()))
		return err
	})
	if err != nil {
		s.serverError(w, r, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"channels": channels})
}

func (s *server) createChannel(w http.ResponseWriter, r *http.Request) {
	var in struct {
		Type   string         `json:"type"`
		Config map[string]any `json:"config"`
	}
	if err := decodeJSON(w, r, &in); err != nil {
		s.writeError(w, http.StatusBadRequest, err.Error())
		return
	}
	config, err := validateChannel(in.Type, in.Config, s.cfg.allowPrivateTargets)
	if err != nil {
		s.writeError(w, http.StatusBadRequest, err.Error())
		return
	}

	var c store.Channel
	err = db.WithTenant(r.Context(), s.pool, tenantID(r.Context()), func(tx pgx.Tx) error {
		var err error
		c, err = store.CreateChannel(r.Context(), tx, tenantID(r.Context()), in.Type, config)
		if err != nil {
			return err
		}
		return store.Audit(r.Context(), tx, tenantID(r.Context()), actor(r.Context()),
			store.ActionChannelCreate,
			fmt.Sprintf("added %s alert channel (%s)", c.Type, channelTargetLabel(c)),
			&c.ID, map[string]any{"type": c.Type})
	})
	if err != nil {
		s.serverError(w, r, err)
		return
	}
	writeJSON(w, http.StatusCreated, c)
}

// channelTargetLabel names a channel for the audit trail without leaking the
// secret: an email address is the user's own and safe to show, but a Slack
// webhook URL is a credential — only its host goes in the log.
func channelTargetLabel(c store.Channel) string {
	switch c.Type {
	case "email":
		to, _ := c.Config["to"].(string)
		return to
	case "slack_webhook":
		raw, _ := c.Config["url"].(string)
		if u, err := url.Parse(raw); err == nil && u.Host != "" {
			return u.Host
		}
		return "webhook"
	default:
		return c.Type
	}
}

// validateChannel checks the type-specific config and returns it with only
// the known keys kept, so arbitrary JSON never lands in the database.
// allowPrivate permits http webhooks (E2E-test hook).
func validateChannel(typ string, config map[string]any, allowPrivate bool) (map[string]any, error) {
	switch typ {
	case "email":
		to, _ := config["to"].(string)
		addr, err := mail.ParseAddress(strings.TrimSpace(to))
		if err != nil {
			return nil, fmt.Errorf(`email channel needs config.to, a valid address`)
		}
		return map[string]any{"to": addr.Address}, nil
	case "slack_webhook":
		raw, _ := config["url"].(string)
		u, err := url.Parse(strings.TrimSpace(raw))
		httpsOK := u != nil && (u.Scheme == "https" || (allowPrivate && u.Scheme == "http"))
		if err != nil || !httpsOK || u.Host == "" {
			return nil, fmt.Errorf(`slack_webhook channel needs config.url, an https webhook URL`)
		}
		return map[string]any{"url": u.String()}, nil
	default:
		return nil, fmt.Errorf(`type must be "email" or "slack_webhook"`)
	}
}

func (s *server) deleteChannel(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	if !isUUID(id) {
		s.writeError(w, http.StatusNotFound, "not found")
		return
	}
	err := db.WithTenant(r.Context(), s.pool, tenantID(r.Context()), func(tx pgx.Tx) error {
		c, err := store.GetChannel(r.Context(), tx, tenantID(r.Context()), id)
		if err != nil {
			return err
		}
		if err := store.DeleteChannel(r.Context(), tx, tenantID(r.Context()), id); err != nil {
			return err
		}
		return store.Audit(r.Context(), tx, tenantID(r.Context()), actor(r.Context()),
			store.ActionChannelDelete,
			fmt.Sprintf("removed %s alert channel (%s)", c.Type, channelTargetLabel(c)),
			nil, map[string]any{"type": c.Type, "channel_id": c.ID})
	})
	if err != nil {
		s.storeError(w, r, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

// testChannel sends a test notification so users can verify wiring before an
// incident depends on it.
func (s *server) testChannel(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	if !isUUID(id) {
		s.writeError(w, http.StatusNotFound, "not found")
		return
	}
	var c store.Channel
	err := db.WithTenant(r.Context(), s.pool, tenantID(r.Context()), func(tx pgx.Tx) error {
		var err error
		c, err = store.GetChannel(r.Context(), tx, tenantID(r.Context()), id)
		return err
	})
	if err != nil {
		s.storeError(w, r, err)
		return
	}

	const msg = "Test alert from DevOps Access — this channel is wired up correctly."
	switch c.Type {
	case "email":
		to, _ := c.Config["to"].(string)
		err = s.mailer.Send(r.Context(), to, "Test alert from DevOps Access", msg)
	case "slack_webhook":
		u, _ := c.Config["url"].(string)
		// s.probe is SSRF-guarded: the webhook URL is customer input.
		err = notify.Slack(r.Context(), s.probe, u, msg)
	default:
		s.writeError(w, http.StatusBadRequest, "unknown channel type")
		return
	}
	// Audit the attempt either way: a test send leaves our relay, so the trail
	// should show who triggered it and whether it landed.
	sendErr := err
	auditErr := db.WithTenant(r.Context(), s.pool, tenantID(r.Context()), func(tx pgx.Tx) error {
		outcome := "delivered"
		if sendErr != nil {
			outcome = "failed"
		}
		return store.Audit(r.Context(), tx, tenantID(r.Context()), actor(r.Context()),
			store.ActionChannelTest,
			fmt.Sprintf("sent a test alert to %s channel (%s) — %s",
				c.Type, channelTargetLabel(c), outcome),
			&c.ID, map[string]any{"type": c.Type, "delivered": sendErr == nil})
	})
	if auditErr != nil {
		s.log.Error().Err(auditErr).Str("channel_id", id).Msg("audit channel test failed")
	}

	if sendErr != nil {
		s.log.Warn().Err(sendErr).Str("channel_id", id).Msg("channel test failed")
		s.writeError(w, http.StatusBadGateway, "test notification failed to send")
		return
	}
	writeJSON(w, http.StatusOK, map[string]bool{"ok": true})
}
