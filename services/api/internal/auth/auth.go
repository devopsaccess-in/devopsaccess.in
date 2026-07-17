// Package auth validates Auth0-issued RS256 access tokens (JWKS fetched and
// cached from the tenant's well-known endpoint) and exposes the verified
// claims to handlers via the request context.
package auth

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"

	"github.com/golang-jwt/jwt/v5"
)

type ctxKey int

const claimsKey ctxKey = 0

// Verifier validates bearer tokens against an Auth0 tenant.
type Verifier struct {
	issuer   string
	audience string
	cache    *keyCache
}

// NewVerifier builds a Verifier for an Auth0 domain (e.g.
// "devopsaccess.eu.auth0.com") and API audience. A domain given with an
// explicit http(s):// scheme is used as the issuer base URL verbatim — that
// form exists for the E2E harness, which serves a local JWKS; production
// config always passes a bare domain.
func NewVerifier(domain, audience string) *Verifier {
	issuer := "https://" + domain + "/"
	if strings.HasPrefix(domain, "http://") || strings.HasPrefix(domain, "https://") {
		issuer = strings.TrimSuffix(domain, "/") + "/"
	}
	return &Verifier{
		issuer:   issuer,
		audience: audience,
		cache:    newKeyCache(issuer + ".well-known/jwks.json"),
	}
}

// Middleware rejects requests without a valid RS256 bearer token for our
// issuer + audience, and stores the verified claims in the request context.
func (v *Verifier) Middleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		raw, ok := bearerToken(r.Header.Get("Authorization"))
		if !ok {
			unauthorized(w, "missing bearer token")
			return
		}
		claims := jwt.MapClaims{}
		if _, err := jwt.ParseWithClaims(raw, claims, v.keyFunc,
			jwt.WithValidMethods([]string{"RS256"}),
			jwt.WithIssuer(v.issuer),
			jwt.WithAudience(v.audience),
			jwt.WithExpirationRequired(),
		); err != nil {
			unauthorized(w, "invalid token")
			return
		}
		if sub, _ := claims.GetSubject(); sub == "" {
			unauthorized(w, "token has no subject")
			return
		}
		next.ServeHTTP(w, r.WithContext(context.WithValue(r.Context(), claimsKey, claims)))
	})
}

func (v *Verifier) keyFunc(t *jwt.Token) (any, error) {
	kid, _ := t.Header["kid"].(string)
	if kid == "" {
		return nil, fmt.Errorf("token has no kid")
	}
	return v.cache.key(kid)
}

// Claims returns the verified token claims, if the request passed Middleware.
func Claims(ctx context.Context) (jwt.MapClaims, bool) {
	c, ok := ctx.Value(claimsKey).(jwt.MapClaims)
	return c, ok
}

// Sub returns the verified token subject (the Auth0 user id).
func Sub(ctx context.Context) (string, bool) {
	c, ok := Claims(ctx)
	if !ok {
		return "", false
	}
	sub, err := c.GetSubject()
	if err != nil || sub == "" {
		return "", false
	}
	return sub, true
}

// StringClaim returns the first non-empty string claim among keys — used for
// optional profile claims (email, name) that Auth0 Actions add to the token.
func StringClaim(c jwt.MapClaims, keys ...string) string {
	for _, k := range keys {
		if v, ok := c[k].(string); ok && v != "" {
			return v
		}
	}
	return ""
}

func bearerToken(header string) (string, bool) {
	const prefix = "Bearer "
	if len(header) > len(prefix) && strings.EqualFold(header[:len(prefix)], prefix) {
		return strings.TrimSpace(header[len(prefix):]), true
	}
	return "", false
}

func unauthorized(w http.ResponseWriter, msg string) {
	w.Header().Set("WWW-Authenticate", `Bearer realm="api"`)
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusUnauthorized)
	_ = json.NewEncoder(w).Encode(map[string]string{"error": msg})
}
