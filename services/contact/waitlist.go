package main

import (
	"context"
	"encoding/json"
	"net/http"
	"strings"
	"time"

	"github.com/rs/zerolog"
	"golang.org/x/time/rate"
)

type waitlistHandler struct {
	store     *store
	mailer    *mailer
	limiter   *ipLimiter
	turnstile *turnstileVerifier
	global    *rate.Limiter
	log       zerolog.Logger
}

type waitlistRequest struct {
	Email     string `json:"email"`
	Name      string `json:"name"`
	Website   string `json:"website"`   // honeypot: must be empty
	Turnstile string `json:"turnstile"` // Cloudflare Turnstile token
}

func (h *waitlistHandler) handle(w http.ResponseWriter, r *http.Request) {
	ip := clientIP(r)
	if !h.limiter.allow(ip) {
		writeJSON(w, http.StatusTooManyRequests, map[string]string{"error": "Too many requests. Please try again later."})
		return
	}

	var req waitlistRequest
	if err := json.NewDecoder(http.MaxBytesReader(w, r.Body, 16*1024)).Decode(&req); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "Invalid request."})
		return
	}

	// Honeypot: pretend success, drop silently.
	if strings.TrimSpace(req.Website) != "" {
		h.log.Info().Str("ip", ip).Msg("waitlist honeypot triggered")
		writeJSON(w, http.StatusOK, map[string]bool{"ok": true})
		return
	}

	email := strings.TrimSpace(req.Email)
	if email == "" || !emailRe.MatchString(email) || len(email) > maxEmailLen {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "A valid email is required."})
		return
	}
	name := strings.TrimSpace(req.Name)
	if len(name) > maxNameLen {
		name = name[:maxNameLen]
	}

	ctx, cancel := context.WithTimeout(r.Context(), 15*time.Second)
	defer cancel()

	if !passAntiAbuse(ctx, w, h.turnstile, h.global, req.Turnstile, ip, h.log) {
		return
	}

	inserted, err := h.store.addWaitlist(ctx, email, name, "site")
	if err != nil {
		h.log.Error().Err(err).Str("ip", ip).Msg("waitlist insert failed")
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "Could not save your signup. Please try again later."})
		return
	}

	if inserted {
		// Best-effort notification; never fail the request on email error.
		if err := h.mailer.notify(ctx, email, "New waitlist signup", "Email: "+email+"\nName:  "+name); err != nil {
			h.log.Error().Err(err).Msg("waitlist notify email failed")
		}
		h.log.Info().Str("email", email).Msg("waitlist signup")
	}

	writeJSON(w, http.StatusOK, map[string]bool{"ok": true})
}
