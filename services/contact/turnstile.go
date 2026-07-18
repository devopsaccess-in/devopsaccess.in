package main

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"strings"
	"time"
)

const turnstileEndpoint = "https://challenges.cloudflare.com/turnstile/v0/siteverify"

// turnstileVerifier validates Cloudflare Turnstile tokens. When the secret is
// empty it is "disabled" and verify() passes everything through, so the service
// works before Turnstile is configured.
type turnstileVerifier struct {
	secret   string
	endpoint string
	client   *http.Client
}

func newTurnstileVerifier(secret string) *turnstileVerifier {
	return &turnstileVerifier{
		secret:   secret,
		endpoint: turnstileEndpoint,
		client:   &http.Client{Timeout: 8 * time.Second},
	}
}

func (t *turnstileVerifier) enabled() bool { return t != nil && t.secret != "" }

// verify returns true when the token is valid (or when Turnstile is disabled).
// The []string is Cloudflare's "error-codes" on a rejection (e.g.
// invalid-input-secret, timeout-or-duplicate, missing-input-response) — logged
// by callers to diagnose failures. A non-nil error means verification could not
// be performed (network/system), distinct from a rejection.
func (t *turnstileVerifier) verify(ctx context.Context, token, ip string) (bool, []string, error) {
	if !t.enabled() {
		return true, nil, nil
	}
	if strings.TrimSpace(token) == "" {
		return false, []string{"missing-input-response"}, nil
	}

	form := url.Values{"secret": {t.secret}, "response": {token}}
	if ip != "" {
		form.Set("remoteip", ip)
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, t.endpoint, strings.NewReader(form.Encode()))
	if err != nil {
		return false, nil, fmt.Errorf("turnstile request: %w", err)
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	resp, err := t.client.Do(req)
	if err != nil {
		return false, nil, fmt.Errorf("turnstile verify: %w", err)
	}
	defer resp.Body.Close()

	var out struct {
		Success    bool     `json:"success"`
		ErrorCodes []string `json:"error-codes"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&out); err != nil {
		return false, nil, fmt.Errorf("turnstile decode: %w", err)
	}
	return out.Success, out.ErrorCodes, nil
}
