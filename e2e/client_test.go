//go:build e2e

package e2e

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"testing"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

// mintToken signs an access token exactly the way the fake Auth0 would.
func mintToken(t *testing.T, sub, email, name string) string {
	t.Helper()
	claims := jwt.MapClaims{
		"iss": issuer,
		"aud": audience,
		"sub": sub,
		"exp": time.Now().Add(time.Hour).Unix(),
		"iat": time.Now().Add(-time.Minute).Unix(),
	}
	if email != "" {
		claims["https://devopsaccess.in/email"] = email
	}
	if name != "" {
		claims["https://devopsaccess.in/name"] = name
	}
	tok := jwt.NewWithClaims(jwt.SigningMethodRS256, claims)
	tok.Header["kid"] = testKid
	signed, err := tok.SignedString(signKey)
	if err != nil {
		t.Fatalf("sign token: %v", err)
	}
	return signed
}

// apiClient is one authenticated user's view of the API.
type apiClient struct {
	token string
	http  *http.Client
}

func newClient(t *testing.T, sub, email, name string) *apiClient {
	return &apiClient{
		token: mintToken(t, sub, email, name),
		http:  &http.Client{Timeout: 10 * time.Second},
	}
}

// do sends a request and decodes JSON into out (out may be nil). Returns the
// HTTP status.
func (c *apiClient) do(t *testing.T, method, path string, body, out any) int {
	t.Helper()
	var buf *bytes.Reader
	if body != nil {
		b, err := json.Marshal(body)
		if err != nil {
			t.Fatalf("marshal body: %v", err)
		}
		buf = bytes.NewReader(b)
	} else {
		buf = bytes.NewReader(nil)
	}
	req, err := http.NewRequest(method, apiBase+path, buf)
	if err != nil {
		t.Fatalf("build request: %v", err)
	}
	if c.token != "" {
		req.Header.Set("Authorization", "Bearer "+c.token)
	}
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	resp, err := c.http.Do(req)
	if err != nil {
		t.Fatalf("%s %s: %v", method, path, err)
	}
	defer resp.Body.Close()
	if out != nil {
		data, _ := io.ReadAll(resp.Body)
		if len(bytes.TrimSpace(data)) > 0 {
			if err := json.Unmarshal(data, out); err != nil && resp.StatusCode < 300 {
				t.Fatalf("%s %s: decode response: %v (body %.200s)", method, path, err, data)
			}
		}
	}
	return resp.StatusCode
}

// mustDo asserts the expected status and fails with the response body.
func (c *apiClient) mustDo(t *testing.T, method, path string, body, out any, wantStatus int) {
	t.Helper()
	var raw json.RawMessage
	status := c.do(t, method, path, body, &raw)
	if status != wantStatus {
		t.Fatalf("%s %s: status %d, want %d (body %s)", method, path, status, wantStatus, raw)
	}
	if out != nil && len(raw) > 0 {
		if err := json.Unmarshal(raw, out); err != nil {
			t.Fatalf("%s %s: decode: %v", method, path, err)
		}
	}
}

// expedite rewinds a monitor's last_checked_at so the next 1s scheduler tick
// claims it immediately — collapses the 60s minimum interval for tests.
func expedite(t *testing.T, monitorID string) {
	t.Helper()
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if _, err := adminPool.Exec(ctx,
		`UPDATE monitors SET last_checked_at = now() - interval '10 minutes' WHERE id = $1`,
		monitorID); err != nil {
		t.Fatalf("expedite: %v", err)
	}
}

// stalePing backdates a heartbeat's last ping so it is overdue, standing in
// for the passage of time (a test can't wait out a real period+grace window).
func stalePing(t *testing.T, monitorID string, age time.Duration) {
	t.Helper()
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if _, err := adminPool.Exec(ctx,
		`UPDATE monitors SET last_ping_at = now() - $2::interval, created_at = now() - $2::interval
		 WHERE id = $1`,
		monitorID, fmt.Sprintf("%d seconds", int(age.Seconds()))); err != nil {
		t.Fatalf("stale ping: %v", err)
	}
}

// waitForState polls a monitor (expediting between polls) until it reaches
// the wanted state.
func waitForState(t *testing.T, c *apiClient, id, want string, timeout time.Duration) {
	t.Helper()
	deadline := time.Now().Add(timeout)
	last := ""
	for time.Now().Before(deadline) {
		var m struct {
			State string `json:"state"`
		}
		c.mustDo(t, "GET", "/api/monitors/"+id, nil, &m, http.StatusOK)
		if m.State == want {
			return
		}
		last = m.State
		expedite(t, id)
		time.Sleep(1200 * time.Millisecond)
	}
	t.Fatalf("monitor %s never reached state %q (last %q)", id, want, last)
}

func monitorPayload(name, url string, threshold int) map[string]any {
	return map[string]any{
		"name":              name,
		"url":               url,
		"method":            "GET",
		"interval_seconds":  60,
		"timeout_ms":        2000,
		"expected_status":   200,
		"failure_threshold": threshold,
	}
}

func uniqueSub(prefix string) string {
	return fmt.Sprintf("auth0|%s-%d", prefix, time.Now().UnixNano())
}
