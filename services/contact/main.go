// Command contact is a small stateless HTTP service that relays website contact
// form submissions to support@devopsaccess.in via an SMTP relay. It is fronted
// by nginx (reverse-proxied at /api/) and bound to localhost.
package main

import (
	"context"
	"errors"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/rs/zerolog"
)

func main() {
	log := zerolog.New(os.Stderr).With().Timestamp().Str("svc", "contact").Logger()

	cfg, err := loadConfig()
	if err != nil {
		log.Fatal().Err(err).Msg("invalid configuration")
	}

	mailer := newMailer(cfg)
	limiter := newIPLimiter(5, 3)      // ~5/hour per IP, burst 3
	global := newGlobalLimiter(60, 10) // backstop: 60 emails/hour overall
	turnstile := newTurnstileVerifier(cfg.turnstileSecret)
	if turnstile.enabled() {
		log.Info().Msg("Turnstile verification enabled")
	}

	// Optional Postgres for waitlist + payments.
	var st *store
	if cfg.databaseURL != "" {
		ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		st, err = newStore(ctx, cfg.databaseURL)
		cancel()
		if err != nil {
			log.Fatal().Err(err).Msg("database init failed")
		}
		defer st.close()
		log.Info().Msg("database connected; waitlist + payments enabled")
	} else {
		log.Info().Msg("DATABASE_URL unset; waitlist + payments disabled")
	}

	r := chi.NewRouter()
	r.Use(middleware.RealIP)
	r.Use(middleware.Recoverer)
	r.Use(middleware.Timeout(20 * time.Second))

	r.Get("/healthz", func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("ok"))
	})

	r.Post("/api/contact", (&contactHandler{
		mailer: mailer, limiter: limiter, turnstile: turnstile, global: global, log: log,
	}).handle)

	// Free Site Health & Security check (works without the DB; recording is best-effort).
	r.Post("/api/sitecheck", (&sitecheckHandler{
		store: st, turnstile: turnstile, limiter: newIPLimiter(30, 5), log: log,
	}).handle)

	if st != nil {
		r.Post("/api/waitlist", (&waitlistHandler{
			store: st, mailer: mailer, limiter: limiter, turnstile: turnstile, global: global, log: log,
		}).handle)
		r.Post("/api/webhooks/razorpay", (&razorpayHandler{store: st, mailer: mailer, secret: cfg.razorpaySecret, log: log}).handle)
	}

	srv := &http.Server{
		Addr:              cfg.listenAddr,
		Handler:           r,
		ReadHeaderTimeout: 5 * time.Second,
		ReadTimeout:       15 * time.Second,
		WriteTimeout:      20 * time.Second,
		IdleTimeout:       60 * time.Second,
	}

	// Graceful shutdown on SIGINT/SIGTERM.
	go func() {
		log.Info().Str("addr", cfg.listenAddr).Msg("contact service listening")
		if err := srv.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			log.Fatal().Err(err).Msg("server error")
		}
	}()

	stop := make(chan os.Signal, 1)
	signal.Notify(stop, syscall.SIGINT, syscall.SIGTERM)
	<-stop

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	if err := srv.Shutdown(ctx); err != nil {
		log.Error().Err(err).Msg("graceful shutdown failed")
	}
	log.Info().Msg("contact service stopped")
}
