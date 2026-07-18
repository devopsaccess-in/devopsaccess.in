package auth

import (
	"crypto/rand"
	"crypto/rsa"
	"encoding/base64"
	"encoding/json"
	"math/big"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

const (
	testIssuer   = "https://test.auth0.local/"
	testAudience = "https://api.devopsaccess.in"
	testKid      = "test-key-1"
)

// jwksServer serves a JWKS document for the given public key.
func jwksServer(t *testing.T, pub *rsa.PublicKey, kid string) *httptest.Server {
	t.Helper()
	doc := map[string]any{
		"keys": []map[string]string{{
			"kty": "RSA",
			"kid": kid,
			"use": "sig",
			"n":   base64.RawURLEncoding.EncodeToString(pub.N.Bytes()),
			"e":   base64.RawURLEncoding.EncodeToString(big.NewInt(int64(pub.E)).Bytes()),
		}},
	}
	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		_ = json.NewEncoder(w).Encode(doc)
	}))
}

func signToken(t *testing.T, key *rsa.PrivateKey, kid string, method jwt.SigningMethod, claims jwt.MapClaims) string {
	t.Helper()
	tok := jwt.NewWithClaims(method, claims)
	tok.Header["kid"] = kid
	var signed string
	var err error
	if method == jwt.SigningMethodHS256 {
		signed, err = tok.SignedString([]byte("hs256-secret"))
	} else {
		signed, err = tok.SignedString(key)
	}
	if err != nil {
		t.Fatalf("sign token: %v", err)
	}
	return signed
}

func validClaims() jwt.MapClaims {
	return jwt.MapClaims{
		"iss": testIssuer,
		"aud": testAudience,
		"sub": "auth0|user123",
		"exp": time.Now().Add(time.Hour).Unix(),
		"iat": time.Now().Add(-time.Minute).Unix(),
	}
}

func TestMiddleware(t *testing.T) {
	key, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		t.Fatalf("generate key: %v", err)
	}
	srv := jwksServer(t, &key.PublicKey, testKid)
	defer srv.Close()

	newVerifier := func() *Verifier {
		return &Verifier{
			issuer:   testIssuer,
			audience: testAudience,
			cache:    newKeyCache(srv.URL),
		}
	}

	expired := validClaims()
	expired["exp"] = time.Now().Add(-time.Hour).Unix()
	wrongAud := validClaims()
	wrongAud["aud"] = "https://other-api.example.com"
	wrongIss := validClaims()
	wrongIss["iss"] = "https://evil.example.com/"
	noSub := validClaims()
	delete(noSub, "sub")
	noExp := validClaims()
	delete(noExp, "exp")

	tests := []struct {
		name       string
		authHeader string
		wantStatus int
		wantSub    string
	}{
		{"valid token", "Bearer " + signToken(t, key, testKid, jwt.SigningMethodRS256, validClaims()), http.StatusOK, "auth0|user123"},
		{"missing header", "", http.StatusUnauthorized, ""},
		{"not a bearer", "Basic abc123", http.StatusUnauthorized, ""},
		{"garbage token", "Bearer not.a.jwt", http.StatusUnauthorized, ""},
		{"expired", "Bearer " + signToken(t, key, testKid, jwt.SigningMethodRS256, expired), http.StatusUnauthorized, ""},
		{"no exp claim", "Bearer " + signToken(t, key, testKid, jwt.SigningMethodRS256, noExp), http.StatusUnauthorized, ""},
		{"wrong audience", "Bearer " + signToken(t, key, testKid, jwt.SigningMethodRS256, wrongAud), http.StatusUnauthorized, ""},
		{"wrong issuer", "Bearer " + signToken(t, key, testKid, jwt.SigningMethodRS256, wrongIss), http.StatusUnauthorized, ""},
		{"HS256 rejected", "Bearer " + signToken(t, key, testKid, jwt.SigningMethodHS256, validClaims()), http.StatusUnauthorized, ""},
		{"unknown kid", "Bearer " + signToken(t, key, "other-kid", jwt.SigningMethodRS256, validClaims()), http.StatusUnauthorized, ""},
		{"no subject", "Bearer " + signToken(t, key, testKid, jwt.SigningMethodRS256, noSub), http.StatusUnauthorized, ""},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var gotSub string
			handler := newVerifier().Middleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				gotSub, _ = Sub(r.Context())
				w.WriteHeader(http.StatusOK)
			}))

			req := httptest.NewRequest(http.MethodGet, "/api/me", nil)
			if tt.authHeader != "" {
				req.Header.Set("Authorization", tt.authHeader)
			}
			rec := httptest.NewRecorder()
			handler.ServeHTTP(rec, req)

			if rec.Code != tt.wantStatus {
				t.Fatalf("status = %d, want %d (body %s)", rec.Code, tt.wantStatus, rec.Body.String())
			}
			if tt.wantSub != "" && gotSub != tt.wantSub {
				t.Fatalf("sub = %q, want %q", gotSub, tt.wantSub)
			}
		})
	}
}

func TestKeyCacheRefetchOnUnknownKid(t *testing.T) {
	key, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		t.Fatalf("generate key: %v", err)
	}
	srv := jwksServer(t, &key.PublicKey, "rotated-kid")
	defer srv.Close()

	c := newKeyCache(srv.URL)
	// Seed the cache as if an old key set had been fetched, then ask for a
	// kid that only the "rotated" JWKS has. minInterval must not block the
	// very first refetch for the unknown kid.
	now := time.Now()
	c.now = func() time.Time { return now }
	c.keys = map[string]*rsa.PublicKey{"old-kid": &key.PublicKey}
	c.lastFetch = now.Add(-2 * time.Minute)

	if _, err := c.key("rotated-kid"); err != nil {
		t.Fatalf("expected refetch to find rotated key, got error: %v", err)
	}
	// A second unknown kid within minInterval must NOT refetch: it fails fast.
	c.lastFetch = now
	if _, err := c.key("never-existed"); err == nil {
		t.Fatal("expected unknown kid to fail without hammering the JWKS endpoint")
	}
}

func TestStringClaim(t *testing.T) {
	c := jwt.MapClaims{"email": "v@example.com", "empty": "", "num": 42}
	tests := []struct {
		name string
		keys []string
		want string
	}{
		{"first match", []string{"email"}, "v@example.com"},
		{"fallback past empty", []string{"empty", "email"}, "v@example.com"},
		{"non-string skipped", []string{"num", "email"}, "v@example.com"},
		{"no match", []string{"missing"}, ""},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := StringClaim(c, tt.keys...); got != tt.want {
				t.Fatalf("StringClaim(%v) = %q, want %q", tt.keys, got, tt.want)
			}
		})
	}
}
