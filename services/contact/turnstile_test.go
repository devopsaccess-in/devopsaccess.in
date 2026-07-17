package main

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestTurnstileDisabledPassesThrough(t *testing.T) {
	v := newTurnstileVerifier("")
	ok, _, err := v.verify(context.Background(), "", "1.2.3.4")
	if err != nil || !ok {
		t.Fatalf("disabled verifier should pass: ok=%v err=%v", ok, err)
	}
}

func TestTurnstileVerify(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_ = r.ParseForm()
		if r.FormValue("response") == "good" {
			_, _ = w.Write([]byte(`{"success":true}`))
		} else {
			_, _ = w.Write([]byte(`{"success":false}`))
		}
	}))
	defer srv.Close()

	v := newTurnstileVerifier("secret")
	v.endpoint = srv.URL

	if ok, _, err := v.verify(context.Background(), "good", "1.2.3.4"); err != nil || !ok {
		t.Fatalf("valid token: ok=%v err=%v", ok, err)
	}
	if ok, _, _ := v.verify(context.Background(), "bad", ""); ok {
		t.Fatal("invalid token must fail")
	}
	if ok, codes, _ := v.verify(context.Background(), "", ""); ok || len(codes) == 0 {
		t.Fatal("missing token must fail with error codes when enabled")
	}
}

func TestGlobalLimiter(t *testing.T) {
	g := newGlobalLimiter(1, 1)
	if !g.Allow() {
		t.Fatal("first request should be allowed")
	}
	if g.Allow() {
		t.Fatal("second request should be blocked by the global cap")
	}
}
