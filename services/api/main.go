// Command api is the uptime-monitoring control plane: monitors, incidents,
// alert channels, and the public status page. It is fronted by nginx on the
// app.devopsaccess.in vhost (reverse-proxied at /api/) and bound to
// localhost. Auth is Auth0 RS256 bearer tokens; every tenant query runs
// inside db.WithTenant so Postgres RLS backs the app-layer scoping.
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

	"github.com/devopsaccess-in/devopsaccess.in/services/shared/db"
	"github.com/devopsaccess-in/devopsaccess.in/services/shared/notify"
	"github.com/devopsaccess-in/devopsaccess.in/services/shared/safehttp"

	"github.com/devopsaccess-in/devopsaccess.in/services/api/internal/auth"
)

func main() {
	log := zerolog.New(os.Stderr).With().Timestamp().Str("svc", "api").Logger()

	cfg, err := loadConfig()
	if err != nil {
		log.Fatal().Err(err).Msg("invalid configuration")
	}

	initCtx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	pool, err := db.Connect(initCtx, cfg.databaseURL)
	if err != nil {
		cancel()
		log.Fatal().Err(err).Msg("database connect failed")
	}
	if err := migrate(initCtx, pool, log); err != nil {
		cancel()
		log.Fatal().Err(err).Msg("migrations failed")
	}
	cancel()
	defer pool.Close()

	probe := safehttp.Client(10 * time.Second)
	if cfg.allowPrivateTargets {
		// E2E-test hook: webhooks may point at local sink servers.
		log.Warn().Msg("UPTIME_ALLOW_PRIVATE_TARGETS=true — SSRF guards disabled, never use in production")
		probe = &http.Client{Timeout: 10 * time.Second}
	}
	s := &server{
		cfg:  cfg,
		pool: pool,
		log:  log,
		mailer: &notify.Mailer{
			Host: cfg.smtpHost, Port: cfg.smtpPort, From: cfg.mailFrom,
		},
		probe: probe,
	}
	verifier := auth.NewVerifier(cfg.auth0Domain, cfg.auth0Audience)

	r := chi.NewRouter()
	r.Use(middleware.RealIP)
	r.Use(middleware.Recoverer)
	r.Use(middleware.Timeout(30 * time.Second))

	r.Get("/healthz", func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("ok"))
	})
	r.Get("/api/status/{slug}", s.handleStatus)
	r.Get("/api/badge/{slug}/{monitor}", s.handleBadge)

	r.Group(func(pr chi.Router) {
		pr.Use(verifier.Middleware)
		pr.Get("/api/me", s.handleMe)

		pr.Group(func(tr chi.Router) {
			tr.Use(s.requireTenant)
			tr.Route("/api/monitors", func(mr chi.Router) {
				mr.Get("/", s.listMonitors)
				mr.Post("/", s.createMonitor)
				mr.Get("/{id}", s.getMonitor)
				mr.Patch("/{id}", s.updateMonitor)
				mr.Delete("/{id}", s.deleteMonitor)
				mr.Get("/{id}/results", s.monitorResults)
				mr.Get("/{id}/uptime", s.monitorUptime)
			})
			tr.Patch("/api/settings", s.updateSettings)
			tr.Get("/api/incidents", s.listIncidents)
			tr.Get("/api/incidents/{id}", s.getIncident)
			tr.Route("/api/channels", func(cr chi.Router) {
				cr.Get("/", s.listChannels)
				cr.Post("/", s.createChannel)
				cr.Delete("/{id}", s.deleteChannel)
				cr.Post("/{id}/test", s.testChannel)
			})
		})
	})

	srv := &http.Server{
		Addr:              cfg.listenAddr,
		Handler:           r,
		ReadHeaderTimeout: 5 * time.Second,
		ReadTimeout:       15 * time.Second,
		WriteTimeout:      35 * time.Second,
		IdleTimeout:       60 * time.Second,
	}

	go func() {
		log.Info().Str("addr", cfg.listenAddr).Msg("api listening")
		if err := srv.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			log.Fatal().Err(err).Msg("server error")
		}
	}()

	stop := make(chan os.Signal, 1)
	signal.Notify(stop, syscall.SIGINT, syscall.SIGTERM)
	<-stop

	shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer shutdownCancel()
	if err := srv.Shutdown(shutdownCtx); err != nil {
		log.Error().Err(err).Msg("graceful shutdown failed")
	}
	log.Info().Msg("api stopped")
}
