package main

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/rs/zerolog"
)

func sign(body, secret string) string {
	mac := hmac.New(sha256.New, []byte(secret))
	mac.Write([]byte(body))
	return hex.EncodeToString(mac.Sum(nil))
}

func TestVerifySignature(t *testing.T) {
	body := `{"event":"payment.captured"}`
	secret := "whsec_test"
	if !verifySignature([]byte(body), sign(body, secret), secret) {
		t.Fatal("valid signature should verify")
	}
	if verifySignature([]byte(body), "deadbeef", secret) {
		t.Fatal("invalid signature must not verify")
	}
	if verifySignature([]byte(body), sign(body, "other"), secret) {
		t.Fatal("signature from a different secret must not verify")
	}
}

func TestRazorpayRejectsBadSignature(t *testing.T) {
	h := &razorpayHandler{secret: "whsec_test", log: zerolog.Nop()}
	req := httptest.NewRequest(http.MethodPost, "/api/webhooks/razorpay", strings.NewReader(`{"event":"payment.captured"}`))
	req.Header.Set("X-Razorpay-Signature", "bad")
	rec := httptest.NewRecorder()
	h.handle(rec, req)
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("bad signature status = %d, want 400", rec.Code)
	}
}

func TestRazorpayDisabledWithoutSecret(t *testing.T) {
	h := &razorpayHandler{secret: "", log: zerolog.Nop()}
	req := httptest.NewRequest(http.MethodPost, "/api/webhooks/razorpay", strings.NewReader(`{}`))
	rec := httptest.NewRecorder()
	h.handle(rec, req)
	if rec.Code != http.StatusServiceUnavailable {
		t.Fatalf("disabled status = %d, want 503", rec.Code)
	}
}
