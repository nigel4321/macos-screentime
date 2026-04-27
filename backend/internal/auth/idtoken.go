package auth

import (
	"context"
	"errors"
	"fmt"
	"slices"

	"github.com/golang-jwt/jwt/v5"
)

// Identity describes a verified third-party identity claim used to
// drive backend account linkage.
type Identity struct {
	Provider string // "apple" | "google"
	Subject  string // sub claim — opaque identifier from the IdP
	Email    string // optional, may be empty
}

// IDTokenVerifier verifies a third-party identity token (Apple/Google)
// against a JWKS, with provider-specific iss/aud checks.
type IDTokenVerifier struct {
	provider     string
	jwks         *JWKSCache
	allowedIss   []string
	audience     string
	allowedAlgs  []string
}

// NewAppleVerifier wraps a JWKS cache pointed at Apple's keys URL with
// the iss/aud checks Apple requires. audience is the bundle id of the
// macOS app (e.g. "dev.nigel.MacAgent").
func NewAppleVerifier(jwks *JWKSCache, audience string) *IDTokenVerifier {
	return &IDTokenVerifier{
		provider:    "apple",
		jwks:        jwks,
		allowedIss:  []string{"https://appleid.apple.com"},
		audience:    audience,
		allowedAlgs: []string{"RS256"},
	}
}

// NewGoogleVerifier wraps a JWKS cache pointed at Google's keys URL
// with the iss/aud checks Google requires. audience is the OAuth client
// id used for Sign-in with Google.
func NewGoogleVerifier(jwks *JWKSCache, audience string) *IDTokenVerifier {
	return &IDTokenVerifier{
		provider:    "google",
		jwks:        jwks,
		allowedIss:  []string{"accounts.google.com", "https://accounts.google.com"},
		audience:    audience,
		allowedAlgs: []string{"RS256", "ES256"},
	}
}

type idTokenClaims struct {
	jwt.RegisteredClaims
	Email string `json:"email"`
}

// Verify parses and validates the raw ID token, returning the verified
// identity on success. Errors fall into two categories: caller bugs
// (parse failures, missing kid, JWKS unreachable) and authentication
// failures (signature, exp, iss, aud). Both shape into a non-nil error.
func (v *IDTokenVerifier) Verify(ctx context.Context, raw string) (*Identity, error) {
	out := &idTokenClaims{}
	_, err := jwt.ParseWithClaims(
		raw, out,
		func(t *jwt.Token) (any, error) {
			kid, _ := t.Header["kid"].(string)
			if kid == "" {
				return nil, errors.New("auth: id token missing kid")
			}
			return v.jwks.Key(ctx, kid)
		},
		jwt.WithValidMethods(v.allowedAlgs),
		jwt.WithExpirationRequired(),
		jwt.WithAudience(v.audience),
	)
	if err != nil {
		return nil, fmt.Errorf("auth: verify %s id token: %w", v.provider, err)
	}
	if !slices.Contains(v.allowedIss, out.Issuer) {
		return nil, fmt.Errorf("auth: %s id token has unexpected iss %q", v.provider, out.Issuer)
	}
	if out.Subject == "" {
		return nil, fmt.Errorf("auth: %s id token missing sub", v.provider)
	}
	return &Identity{
		Provider: v.provider,
		Subject:  out.Subject,
		Email:    out.Email,
	}, nil
}
