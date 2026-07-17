//go:build e2e

// Package e2e black-box-tests the uptime product: it builds the real API and
// scheduler binaries, runs them against a disposable Postgres with the real
// RLS roles, fakes Auth0 with a local JWKS server, and captures alerts with
// local SMTP and Slack sinks. See README.md.
package e2e

import (
	"context"
	"crypto/rand"
	"crypto/rsa"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"math/big"
	"net"
	"net/http"
	"net/url"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
)

const (
	e2eDBName     = "uptime_e2e"
	e2eDBPassword = "e2e-password"
	audience      = "https://api.devopsaccess.in"
	testKid       = "e2e-key-1"
)

// Package-level handles the tests use; set up once in TestMain.
var (
	adminPool *pgxpool.Pool
	apiBase   string
	dashBase  string // "" when E2E_DASHBOARD_DIR is not set
	issuer    string // trailing slash included
	signKey   *rsa.PrivateKey
	mailSink  *smtpSink
	slackSink *webhookSink
)

func envOr(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}

func TestMain(m *testing.M) {
	code, err := run(m)
	if err != nil {
		fmt.Fprintln(os.Stderr, "e2e setup:", err)
		code = 1
	}
	os.Exit(code)
}

func run(m *testing.M) (int, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
	defer cancel()

	repoRoot, err := filepath.Abs("..")
	if err != nil {
		return 0, err
	}

	// --- disposable database + real roles --------------------------------
	adminURL := envOr("E2E_ADMIN_DATABASE_URL",
		"postgres://postgres:postgres@127.0.0.1:5432/postgres?sslmode=disable")
	adminPool, err = pgxpool.New(ctx, adminURL)
	if err != nil {
		return 0, fmt.Errorf("connect admin postgres: %w", err)
	}
	defer adminPool.Close()
	if err := adminPool.Ping(ctx); err != nil {
		return 0, fmt.Errorf("ping admin postgres (start one: docker run --rm -d -p 5432:5432 -e POSTGRES_PASSWORD=postgres postgres:16): %w", err)
	}
	if err := setupDatabase(ctx, adminURL); err != nil {
		return 0, err
	}
	hostPort, err := dbHostPort(adminURL)
	if err != nil {
		return 0, err
	}
	apiDSN := fmt.Sprintf("postgres://uptime_api:%s@%s/%s?sslmode=disable", e2eDBPassword, hostPort, e2eDBName)
	schedDSN := fmt.Sprintf("postgres://uptime_scheduler:%s@%s/%s?sslmode=disable", e2eDBPassword, hostPort, e2eDBName)

	// adminPool must talk to the e2e database (expedite updates run there).
	adminPool.Close()
	adminPool, err = pgxpool.New(ctx, replaceDBName(adminURL, e2eDBName))
	if err != nil {
		return 0, fmt.Errorf("connect admin to %s: %w", e2eDBName, err)
	}
	defer adminPool.Close()

	// --- fake Auth0 (JWKS) + notification sinks --------------------------
	jwksURL, key, stopJWKS, err := startJWKS()
	if err != nil {
		return 0, err
	}
	defer stopJWKS()
	signKey = key
	issuer = jwksURL + "/"

	mailSink, err = newSMTPSink()
	if err != nil {
		return 0, err
	}
	defer mailSink.close()
	slackSink, err = newWebhookSink()
	if err != nil {
		return 0, err
	}
	defer slackSink.close()

	// --- build + start the real services ---------------------------------
	binDir, err := os.MkdirTemp("", "uptime-e2e-bin")
	if err != nil {
		return 0, err
	}
	defer os.RemoveAll(binDir)
	apiBin := filepath.Join(binDir, "uptime-api")
	schedBin := filepath.Join(binDir, "uptime-scheduler")
	if err := goBuild(filepath.Join(repoRoot, "services", "api"), apiBin); err != nil {
		return 0, err
	}
	if err := goBuild(filepath.Join(repoRoot, "services", "scheduler"), schedBin); err != nil {
		return 0, err
	}

	apiPort, err := freePort()
	if err != nil {
		return 0, err
	}
	apiBase = fmt.Sprintf("http://127.0.0.1:%d", apiPort)
	apiCmd, err := startProcess(apiBin, []string{
		fmt.Sprintf("API_LISTEN_ADDR=127.0.0.1:%d", apiPort),
		"DATABASE_URL=" + apiDSN,
		"AUTH0_DOMAIN=" + jwksURL, // scheme-prefixed => used as issuer verbatim
		"AUTH0_AUDIENCE=" + audience,
		"SMTP_HOST=" + mailSink.host(),
		"SMTP_PORT=" + mailSink.port(),
		"ALERT_FROM=alerts@e2e.devopsaccess.in",
		"UPTIME_ALLOW_PRIVATE_TARGETS=true",
	})
	if err != nil {
		return 0, err
	}
	defer stopProcess(apiCmd)
	if err := waitHTTP(apiBase+"/healthz", 30*time.Second); err != nil {
		return 0, fmt.Errorf("api never became healthy: %w", err)
	}

	schedCmd, err := startProcess(schedBin, []string{
		"DATABASE_URL=" + schedDSN,
		"SCHEDULER_TICK_SECONDS=1",
		"SCHEDULER_WORKERS=5",
		"SMTP_HOST=" + mailSink.host(),
		"SMTP_PORT=" + mailSink.port(),
		"ALERT_FROM=alerts@e2e.devopsaccess.in",
		"UPTIME_ALLOW_PRIVATE_TARGETS=true",
	})
	if err != nil {
		return 0, err
	}
	defer stopProcess(schedCmd)

	// --- dashboard (optional) --------------------------------------------
	if dir := os.Getenv("E2E_DASHBOARD_DIR"); dir != "" {
		nodeBin, lookErr := exec.LookPath("node")
		if lookErr != nil {
			fmt.Fprintln(os.Stderr, "e2e: node not found, dashboard tests will be skipped")
		} else {
			dashPort, err := freePort()
			if err != nil {
				return 0, err
			}
			dashBase = fmt.Sprintf("http://127.0.0.1:%d", dashPort)
			dashCmd := exec.Command(nodeBin, "server.js")
			dashCmd.Dir = dir
			dashCmd.Env = append(os.Environ(),
				"NODE_ENV=production",
				fmt.Sprintf("PORT=%d", dashPort),
				"HOSTNAME=127.0.0.1",
				"API_INTERNAL_URL="+apiBase,
				"APP_BASE_URL="+dashBase,
				"AUTH0_DOMAIN=placeholder.auth0.com",
				"AUTH0_CLIENT_ID=placeholder",
				"AUTH0_CLIENT_SECRET=placeholder",
				"AUTH0_SECRET=e2e-placeholder-secret-32-chars!!",
			)
			dashCmd.Stdout = os.Stderr
			dashCmd.Stderr = os.Stderr
			if err := dashCmd.Start(); err != nil {
				return 0, fmt.Errorf("start dashboard: %w", err)
			}
			defer stopProcess(dashCmd)
			if err := waitHTTP(dashBase+"/status/warmup-probe", 30*time.Second); err != nil {
				return 0, fmt.Errorf("dashboard never came up: %w", err)
			}
		}
	}

	return m.Run(), nil
}

// setupDatabase (re)creates the e2e database and ensures the two product
// roles exist with the e2e password. Requires a DISPOSABLE postgres.
func setupDatabase(ctx context.Context, adminURL string) error {
	stmts := []string{
		fmt.Sprintf(`DO $$ BEGIN
			IF NOT EXISTS (SELECT FROM pg_roles WHERE rolname = 'uptime_api') THEN
				CREATE ROLE uptime_api LOGIN PASSWORD '%[1]s';
			ELSE
				ALTER ROLE uptime_api LOGIN PASSWORD '%[1]s';
			END IF;
		END $$;`, e2eDBPassword),
		fmt.Sprintf(`DO $$ BEGIN
			IF NOT EXISTS (SELECT FROM pg_roles WHERE rolname = 'uptime_scheduler') THEN
				CREATE ROLE uptime_scheduler LOGIN PASSWORD '%[1]s' BYPASSRLS;
			ELSE
				ALTER ROLE uptime_scheduler LOGIN PASSWORD '%[1]s' BYPASSRLS;
			END IF;
		END $$;`, e2eDBPassword),
		fmt.Sprintf(`DROP DATABASE IF EXISTS %s WITH (FORCE)`, e2eDBName),
		fmt.Sprintf(`CREATE DATABASE %s OWNER uptime_api`, e2eDBName),
	}
	for _, s := range stmts {
		if _, err := adminPool.Exec(ctx, s); err != nil {
			return fmt.Errorf("setup db: %w", err)
		}
	}
	return nil
}

func dbHostPort(adminURL string) (string, error) {
	u, err := url.Parse(adminURL)
	if err != nil {
		return "", fmt.Errorf("parse admin url: %w", err)
	}
	return u.Host, nil
}

func replaceDBName(adminURL, db string) string {
	u, _ := url.Parse(adminURL)
	u.Path = "/" + db
	return u.String()
}

// startJWKS serves a one-key JWKS document the way Auth0 would.
func startJWKS() (baseURL string, key *rsa.PrivateKey, stop func(), err error) {
	key, err = rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		return "", nil, nil, err
	}
	doc := map[string]any{
		"keys": []map[string]string{{
			"kty": "RSA",
			"kid": testKid,
			"use": "sig",
			"n":   base64.RawURLEncoding.EncodeToString(key.PublicKey.N.Bytes()),
			"e":   base64.RawURLEncoding.EncodeToString(big.NewInt(int64(key.PublicKey.E)).Bytes()),
		}},
	}
	mux := http.NewServeMux()
	mux.HandleFunc("/.well-known/jwks.json", func(w http.ResponseWriter, _ *http.Request) {
		_ = json.NewEncoder(w).Encode(doc)
	})
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		return "", nil, nil, err
	}
	srv := &http.Server{Handler: mux}
	go func() { _ = srv.Serve(ln) }()
	return "http://" + ln.Addr().String(), key, func() { _ = srv.Close() }, nil
}

// webhookSink records Slack-style {"text": ...} POSTs.
type webhookSink struct {
	srv *http.Server
	ln  net.Listener

	mu    sync.Mutex
	texts []string
}

func newWebhookSink() (*webhookSink, error) {
	s := &webhookSink{}
	mux := http.NewServeMux()
	mux.HandleFunc("/hook", func(w http.ResponseWriter, r *http.Request) {
		var body struct {
			Text string `json:"text"`
		}
		_ = json.NewDecoder(r.Body).Decode(&body)
		s.mu.Lock()
		s.texts = append(s.texts, body.Text)
		s.mu.Unlock()
		w.WriteHeader(http.StatusOK)
	})
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		return nil, err
	}
	s.ln = ln
	s.srv = &http.Server{Handler: mux}
	go func() { _ = s.srv.Serve(ln) }()
	return s, nil
}

func (s *webhookSink) url() string { return "http://" + s.ln.Addr().String() + "/hook" }
func (s *webhookSink) close()      { _ = s.srv.Close() }

func (s *webhookSink) waitFor(t *testing.T, substr string, timeout time.Duration) string {
	t.Helper()
	deadline := time.Now().Add(timeout)
	for time.Now().Before(deadline) {
		s.mu.Lock()
		for _, txt := range s.texts {
			if strings.Contains(txt, substr) {
				s.mu.Unlock()
				return txt
			}
		}
		s.mu.Unlock()
		time.Sleep(300 * time.Millisecond)
	}
	t.Fatalf("no slack message containing %q within %s", substr, timeout)
	return ""
}

// --- process helpers -------------------------------------------------------

func goBuild(dir, out string) error {
	cmd := exec.Command("go", "build", "-o", out, ".")
	cmd.Dir = dir
	cmd.Stdout = os.Stderr
	cmd.Stderr = os.Stderr
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("go build %s: %w", dir, err)
	}
	return nil
}

func startProcess(bin string, env []string) (*exec.Cmd, error) {
	cmd := exec.Command(bin)
	cmd.Env = append(os.Environ(), env...)
	cmd.Stdout = os.Stderr
	cmd.Stderr = os.Stderr
	if err := cmd.Start(); err != nil {
		return nil, fmt.Errorf("start %s: %w", bin, err)
	}
	return cmd, nil
}

func stopProcess(cmd *exec.Cmd) {
	if cmd == nil || cmd.Process == nil {
		return
	}
	_ = cmd.Process.Kill()
	_, _ = cmd.Process.Wait()
}

func freePort() (int, error) {
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		return 0, err
	}
	defer ln.Close()
	return ln.Addr().(*net.TCPAddr).Port, nil
}

func waitHTTP(url string, timeout time.Duration) error {
	deadline := time.Now().Add(timeout)
	client := &http.Client{Timeout: 2 * time.Second}
	var lastErr error
	for time.Now().Before(deadline) {
		resp, err := client.Get(url)
		if err == nil {
			resp.Body.Close()
			return nil // any HTTP response means the server is up
		}
		lastErr = err
		time.Sleep(300 * time.Millisecond)
	}
	return lastErr
}
