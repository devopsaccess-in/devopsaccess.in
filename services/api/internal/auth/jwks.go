package auth

import (
	"crypto/rsa"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"math/big"
	"net/http"
	"sync"
	"time"
)

// jwksDoc is the JSON Web Key Set document Auth0 serves at
// /.well-known/jwks.json. Only the RSA fields we need.
type jwksDoc struct {
	Keys []struct {
		Kty string `json:"kty"`
		Kid string `json:"kid"`
		Use string `json:"use"`
		N   string `json:"n"`
		E   string `json:"e"`
	} `json:"keys"`
}

// keyCache fetches and caches RSA public keys by kid. An unknown kid triggers
// a refetch (key rotation), rate-limited to minInterval so a flood of
// bad-kid tokens cannot hammer Auth0. Cached keys are refreshed after ttl.
type keyCache struct {
	url    string
	client *http.Client

	mu          sync.Mutex
	keys        map[string]*rsa.PublicKey
	lastFetch   time.Time
	minInterval time.Duration
	ttl         time.Duration
	now         func() time.Time
}

func newKeyCache(url string) *keyCache {
	return &keyCache{
		url:         url,
		client:      &http.Client{Timeout: 10 * time.Second},
		minInterval: time.Minute,
		ttl:         12 * time.Hour,
		now:         time.Now,
	}
}

func (c *keyCache) key(kid string) (*rsa.PublicKey, error) {
	c.mu.Lock()
	defer c.mu.Unlock()

	fresh := c.now().Sub(c.lastFetch) <= c.ttl
	if k, ok := c.keys[kid]; ok && fresh {
		return k, nil
	}
	if c.keys == nil || c.now().Sub(c.lastFetch) >= c.minInterval {
		if err := c.fetchLocked(); err != nil {
			// Keep serving a known key when the refresh fails — better a
			// slightly stale key than every request failing auth.
			if k, ok := c.keys[kid]; ok {
				return k, nil
			}
			return nil, err
		}
	}
	if k, ok := c.keys[kid]; ok {
		return k, nil
	}
	return nil, fmt.Errorf("no key %q in JWKS", kid)
}

// fetchLocked downloads and parses the key set. Caller holds c.mu.
func (c *keyCache) fetchLocked() error {
	resp, err := c.client.Get(c.url)
	if err != nil {
		return fmt.Errorf("fetch jwks: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("fetch jwks: status %d", resp.StatusCode)
	}

	var doc jwksDoc
	if err := json.NewDecoder(resp.Body).Decode(&doc); err != nil {
		return fmt.Errorf("decode jwks: %w", err)
	}

	keys := make(map[string]*rsa.PublicKey, len(doc.Keys))
	for _, k := range doc.Keys {
		if k.Kty != "RSA" || (k.Use != "" && k.Use != "sig") || k.Kid == "" {
			continue
		}
		pub, err := rsaKey(k.N, k.E)
		if err != nil {
			return fmt.Errorf("parse jwk %q: %w", k.Kid, err)
		}
		keys[k.Kid] = pub
	}
	if len(keys) == 0 {
		return fmt.Errorf("jwks at %s contains no RSA signing keys", c.url)
	}
	c.keys = keys
	c.lastFetch = c.now()
	return nil
}

func rsaKey(n64, e64 string) (*rsa.PublicKey, error) {
	nb, err := base64.RawURLEncoding.DecodeString(n64)
	if err != nil {
		return nil, fmt.Errorf("modulus: %w", err)
	}
	eb, err := base64.RawURLEncoding.DecodeString(e64)
	if err != nil {
		return nil, fmt.Errorf("exponent: %w", err)
	}
	e := 0
	for _, b := range eb {
		e = e<<8 | int(b)
	}
	if e <= 1 {
		return nil, fmt.Errorf("invalid exponent")
	}
	return &rsa.PublicKey{N: new(big.Int).SetBytes(nb), E: e}, nil
}
