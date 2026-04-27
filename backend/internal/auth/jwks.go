package auth

import (
	"context"
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rsa"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"math/big"
	"net/http"
	"sync"
	"time"
)

// JWKSCache fetches and caches a remote JSON Web Key Set, exposing keys
// by kid. It is safe for concurrent use.
//
// Refresh policy: on cache miss, the cache fetches fresh keys subject
// to MinRefreshInterval — back-to-back misses for unknown kids do not
// stampede the upstream.
type JWKSCache struct {
	URL                string
	HTTP               *http.Client
	MinRefreshInterval time.Duration
	now                func() time.Time

	mu        sync.RWMutex
	keys      map[string]any
	lastFetch time.Time
}

// NewJWKSCache returns a cache for the given JWKS URL. Defaults: a
// 10-second HTTP timeout and a 60-second min-refresh interval.
func NewJWKSCache(url string) *JWKSCache {
	return &JWKSCache{
		URL:                url,
		HTTP:               &http.Client{Timeout: 10 * time.Second},
		MinRefreshInterval: 60 * time.Second,
		now:                time.Now,
		keys:               map[string]any{},
	}
}

// Key returns the public key for kid. On miss it triggers a refresh
// (rate-limited by MinRefreshInterval) and retries once. If still
// unknown, an error is returned.
func (c *JWKSCache) Key(ctx context.Context, kid string) (any, error) {
	if k, ok := c.lookup(kid); ok {
		return k, nil
	}
	if err := c.maybeRefresh(ctx); err != nil {
		return nil, err
	}
	if k, ok := c.lookup(kid); ok {
		return k, nil
	}
	return nil, fmt.Errorf("auth: kid %q not in JWKS at %s", kid, c.URL)
}

func (c *JWKSCache) lookup(kid string) (any, bool) {
	c.mu.RLock()
	defer c.mu.RUnlock()
	k, ok := c.keys[kid]
	return k, ok
}

func (c *JWKSCache) maybeRefresh(ctx context.Context) error {
	c.mu.Lock()
	if c.now().Sub(c.lastFetch) < c.MinRefreshInterval && len(c.keys) > 0 {
		c.mu.Unlock()
		return nil
	}
	c.mu.Unlock()
	return c.refresh(ctx)
}

func (c *JWKSCache) refresh(ctx context.Context) error {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, c.URL, nil)
	if err != nil {
		return err
	}
	resp, err := c.HTTP.Do(req)
	if err != nil {
		return fmt.Errorf("auth: fetch JWKS: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("auth: JWKS %s: HTTP %d", c.URL, resp.StatusCode)
	}
	body, err := io.ReadAll(io.LimitReader(resp.Body, 1<<20))
	if err != nil {
		return fmt.Errorf("auth: read JWKS: %w", err)
	}
	var doc struct {
		Keys []jwk `json:"keys"`
	}
	if err := json.Unmarshal(body, &doc); err != nil {
		return fmt.Errorf("auth: parse JWKS: %w", err)
	}
	parsed := make(map[string]any, len(doc.Keys))
	for _, k := range doc.Keys {
		pub, err := k.publicKey()
		if err != nil {
			continue // skip unknown kty/crv silently — providers add new ones
		}
		if k.Kid != "" {
			parsed[k.Kid] = pub
		}
	}

	c.mu.Lock()
	c.keys = parsed
	c.lastFetch = c.now()
	c.mu.Unlock()
	return nil
}

// jwk is the subset of the JSON Web Key spec needed to parse RSA and EC
// signing keys. Other fields (use, alg, x5c, ...) are ignored.
type jwk struct {
	Kty string `json:"kty"`
	Kid string `json:"kid"`
	Alg string `json:"alg"`

	// RSA
	N string `json:"n,omitempty"`
	E string `json:"e,omitempty"`

	// EC
	Crv string `json:"crv,omitempty"`
	X   string `json:"x,omitempty"`
	Y   string `json:"y,omitempty"`
}

func (k jwk) publicKey() (any, error) {
	switch k.Kty {
	case "RSA":
		nBytes, err := base64.RawURLEncoding.DecodeString(k.N)
		if err != nil {
			return nil, fmt.Errorf("decode n: %w", err)
		}
		eBytes, err := base64.RawURLEncoding.DecodeString(k.E)
		if err != nil {
			return nil, fmt.Errorf("decode e: %w", err)
		}
		e := new(big.Int).SetBytes(eBytes).Int64()
		if e <= 0 || e > (1<<31)-1 {
			return nil, errors.New("rsa exponent out of range")
		}
		return &rsa.PublicKey{
			N: new(big.Int).SetBytes(nBytes),
			E: int(e),
		}, nil
	case "EC":
		var curve elliptic.Curve
		switch k.Crv {
		case "P-256":
			curve = elliptic.P256()
		case "P-384":
			curve = elliptic.P384()
		case "P-521":
			curve = elliptic.P521()
		default:
			return nil, fmt.Errorf("unsupported curve %q", k.Crv)
		}
		xBytes, err := base64.RawURLEncoding.DecodeString(k.X)
		if err != nil {
			return nil, fmt.Errorf("decode x: %w", err)
		}
		yBytes, err := base64.RawURLEncoding.DecodeString(k.Y)
		if err != nil {
			return nil, fmt.Errorf("decode y: %w", err)
		}
		return &ecdsa.PublicKey{
			Curve: curve,
			X:     new(big.Int).SetBytes(xBytes),
			Y:     new(big.Int).SetBytes(yBytes),
		}, nil
	default:
		return nil, fmt.Errorf("unsupported kty %q", k.Kty)
	}
}
