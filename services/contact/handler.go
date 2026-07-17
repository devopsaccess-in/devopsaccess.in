package main

import (
	"context"
	"encoding/json"
	"net/http"
	"regexp"
	"strings"
	"time"

	"github.com/rs/zerolog"
	"golang.org/x/time/rate"
)

var emailRe = regexp.MustCompile(`^[^@\s]+@[^@\s]+\.[^@\s]+$`)

const (
	maxNameLen    = 120
	maxEmailLen   = 254
	maxMessageLen = 5000
)

type contactRequest struct {
	Name      string `json:"name"`
	Email     string `json:"email"`
	Message   string `json:"message"`
	Website   string `json:"website"`   // honeypot: must be empty
	Turnstile string `json:"turnstile"` // Cloudflare Turnstile token
}

type contactHandler struct {
	mailer    *mailer
	limiter   *ipLimiter
	turnstile *turnstileVerifier
	global    *rate.Limiter
	log       zerolog.Logger
}

// passAntiAbuse runs Turnstile verification and the global email cap, writing an
// error and returning false if the request should be blocked. Shared by the
// contact and waitlist handlers.
func passAntiAbuse(ctx context.Context, w http.ResponseWriter, ts *turnstileVerifier, global *rate.Limiter, token, ip string, log zerolog.Logger) bool {
	ok, codes, err := ts.verify(ctx, token, ip)
	if err != nil {
		log.Error().Err(err).Msg("turnstile verify error")
		writeJSON(w, http.StatusBadGateway, map[string]string{"error": "Verification temporarily unavailable. Please try again."})
		return false
	}
	if !ok {
		log.Warn().Strs("turnstile_codes", codes).Str("ip", ip).Msg("turnstile rejected")
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "Verification failed. Please retry."})
		return false
	}
	if global != nil && !global.Allow() {
		log.Warn().Str("ip", ip).Msg("global email cap hit")
		writeJSON(w, http.StatusTooManyRequests, map[string]string{"error": "We're receiving a lot of messages right now. Please try again shortly."})
		return false
	}
	return true
}

func (h *contactHandler) handle(w http.ResponseWriter, r *http.Request) {
	ip := clientIP(r)
	if !h.limiter.allow(ip) {
		writeJSON(w, http.StatusTooManyRequests, map[string]string{"error": "Too many requests. Please try again later."})
		return
	}

	var req contactRequest
	if err := json.NewDecoder(http.MaxBytesReader(w, r.Body, 64*1024)).Decode(&req); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "Invalid request."})
		return
	}

	// Honeypot: a real user never fills this hidden field. Pretend success so
	// bots get no signal, but drop the message.
	if strings.TrimSpace(req.Website) != "" {
		h.log.Info().Str("ip", ip).Msg("honeypot triggered, dropping submission")
		writeJSON(w, http.StatusOK, map[string]bool{"ok": true})
		return
	}

	s, ok := validate(req)
	if !ok {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "A valid email and a message are required."})
		return
	}

	ctx, cancel := context.WithTimeout(r.Context(), 15*time.Second)
	defer cancel()

	if !passAntiAbuse(ctx, w, h.turnstile, h.global, req.Turnstile, ip, h.log) {
		return
	}

	if err := h.mailer.send(ctx, s); err != nil {
		h.log.Error().Err(err).Str("ip", ip).Msg("failed to send contact email")
		writeJSON(w, http.StatusBadGateway, map[string]string{"error": "Could not send your message. Please email support@devopsaccess.in."})
		return
	}

	h.log.Info().Str("ip", ip).Str("email", s.Email).Msg("contact message relayed")
	writeJSON(w, http.StatusOK, map[string]bool{"ok": true})
}

// validate trims, length-checks, and verifies required fields.
func validate(req contactRequest) (submission, bool) {
	name := strings.TrimSpace(req.Name)
	email := strings.TrimSpace(req.Email)
	message := strings.TrimSpace(req.Message)

	if email == "" || !emailRe.MatchString(email) || len(email) > maxEmailLen {
		return submission{}, false
	}
	if message == "" || len(message) > maxMessageLen {
		return submission{}, false
	}
	if len(name) > maxNameLen {
		name = name[:maxNameLen]
	}
	return submission{Name: name, Email: email, Message: message}, true
}

// clientIP prefers proxy headers set by nginx, falling back to RemoteAddr.
func clientIP(r *http.Request) string {
	if v := r.Header.Get("X-Real-IP"); v != "" {
		return v
	}
	if v := r.Header.Get("X-Forwarded-For"); v != "" {
		if i := strings.IndexByte(v, ','); i >= 0 {
			return strings.TrimSpace(v[:i])
		}
		return strings.TrimSpace(v)
	}
	if i := strings.LastIndexByte(r.RemoteAddr, ':'); i >= 0 {
		return r.RemoteAddr[:i]
	}
	return r.RemoteAddr
}

func writeJSON(w http.ResponseWriter, status int, body any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(body)
}
