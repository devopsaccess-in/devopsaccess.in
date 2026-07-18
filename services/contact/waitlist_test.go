package main

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/rs/zerolog"
)

// Both paths below return before touching the store, so a nil store is fine.

func TestWaitlistHoneypotDropsSilently(t *testing.T) {
	h := &waitlistHandler{limiter: newIPLimiter(100, 100), log: zerolog.Nop()}
	req := httptest.NewRequest(http.MethodPost, "/api/waitlist",
		strings.NewReader(`{"email":"a@b.com","website":"http://spam"}`))
	rec := httptest.NewRecorder()
	h.handle(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("honeypot status = %d, want 200", rec.Code)
	}
}

func TestWaitlistRejectsInvalidEmail(t *testing.T) {
	h := &waitlistHandler{limiter: newIPLimiter(100, 100), log: zerolog.Nop()}
	req := httptest.NewRequest(http.MethodPost, "/api/waitlist", strings.NewReader(`{"email":"nope"}`))
	rec := httptest.NewRecorder()
	h.handle(rec, req)
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("invalid email status = %d, want 400", rec.Code)
	}
}
