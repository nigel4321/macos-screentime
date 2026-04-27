// Package auth implements identity-token verification (Apple, Google),
// backend JWT issuance and verification, account pairing, and the HTTP
// middleware that ties them together.
package auth

import (
	"crypto/ecdsa"
	"crypto/sha256"
	"crypto/x509"
	"encoding/base64"
	"encoding/pem"
	"errors"
	"fmt"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

// Issuer is the iss claim the backend stamps on every token it issues.
const Issuer = "screentime-backend"

// signingMethod is locked to ES256 to prevent algorithm-confusion attacks
// where an attacker submits an HS256 token signed with the public key.
var signingMethod = jwt.SigningMethodES256

// Claims is the backend JWT payload. AccountID maps to the standard
// `sub` claim; the embedded RegisteredClaims carries iss/exp/iat.
type Claims struct {
	AccountID string `json:"sub"`
	jwt.RegisteredClaims
}

// Signer issues new tokens using a single ES256 private key. The
// matching public key participates in verification via Verifier.
type Signer struct {
	priv *ecdsa.PrivateKey
	kid  string
	now  func() time.Time
	ttl  time.Duration
}

// NewSigner parses a PEM-encoded EC P-256 private key and returns a
// signer with a default 1h token TTL.
func NewSigner(pemBytes []byte) (*Signer, error) {
	priv, err := parseECPrivateKeyPEM(pemBytes)
	if err != nil {
		return nil, err
	}
	kid, err := KeyID(&priv.PublicKey)
	if err != nil {
		return nil, err
	}
	return &Signer{
		priv: priv,
		kid:  kid,
		now:  time.Now,
		ttl:  time.Hour,
	}, nil
}

// PublicKey returns the signer's verification key. Pair this with
// Verifier so freshly-issued tokens can be verified locally.
func (s *Signer) PublicKey() *ecdsa.PublicKey { return &s.priv.PublicKey }

// KID returns the key identifier embedded in token headers.
func (s *Signer) KID() string { return s.kid }

// Issue creates a signed JWT for the given account id. The token has
// iss=Issuer, sub=accountID, iat=now, exp=now+ttl, and a kid header.
func (s *Signer) Issue(accountID string) (string, error) {
	now := s.now()
	claims := Claims{
		AccountID: accountID,
		RegisteredClaims: jwt.RegisteredClaims{
			Issuer:    Issuer,
			IssuedAt:  jwt.NewNumericDate(now),
			ExpiresAt: jwt.NewNumericDate(now.Add(s.ttl)),
		},
	}
	tok := jwt.NewWithClaims(signingMethod, claims)
	tok.Header["kid"] = s.kid
	return tok.SignedString(s.priv)
}

// Verifier validates tokens against a fixed set of public keys keyed by
// kid. Adding additional keys lets the backend accept tokens signed by
// rotated-out keys during a deprecation window.
type Verifier struct {
	keys map[string]*ecdsa.PublicKey
}

// NewVerifier builds a verifier from one or more EC P-256 public keys.
// At least one key is required.
func NewVerifier(keys ...*ecdsa.PublicKey) (*Verifier, error) {
	if len(keys) == 0 {
		return nil, errors.New("auth: NewVerifier needs at least one public key")
	}
	v := &Verifier{keys: make(map[string]*ecdsa.PublicKey, len(keys))}
	for _, k := range keys {
		kid, err := KeyID(k)
		if err != nil {
			return nil, err
		}
		v.keys[kid] = k
	}
	return v, nil
}

// Parse validates the token signature, algorithm, and standard claims,
// returning the parsed Claims on success.
func (v *Verifier) Parse(raw string) (*Claims, error) {
	out := &Claims{}
	_, err := jwt.ParseWithClaims(
		raw, out,
		func(t *jwt.Token) (any, error) {
			kid, _ := t.Header["kid"].(string)
			if kid == "" {
				return nil, errors.New("auth: token missing kid header")
			}
			k, ok := v.keys[kid]
			if !ok {
				return nil, fmt.Errorf("auth: unknown kid %q", kid)
			}
			return k, nil
		},
		jwt.WithValidMethods([]string{signingMethod.Alg()}),
		jwt.WithIssuer(Issuer),
		jwt.WithExpirationRequired(),
	)
	if err != nil {
		return nil, err
	}
	return out, nil
}

// KeyID returns a deterministic, URL-safe identifier for an EC public
// key: the URL-safe base64 of SHA-256 over the SubjectPublicKeyInfo
// (DER) encoding. Two callers with the same public key always derive
// the same kid.
func KeyID(pub *ecdsa.PublicKey) (string, error) {
	der, err := x509.MarshalPKIXPublicKey(pub)
	if err != nil {
		return "", fmt.Errorf("auth: marshal public key: %w", err)
	}
	sum := sha256.Sum256(der)
	return base64.RawURLEncoding.EncodeToString(sum[:]), nil
}

func parseECPrivateKeyPEM(pemBytes []byte) (*ecdsa.PrivateKey, error) {
	block, _ := pem.Decode(pemBytes)
	if block == nil {
		return nil, errors.New("auth: no PEM data found")
	}
	switch block.Type {
	case "EC PRIVATE KEY":
		return x509.ParseECPrivateKey(block.Bytes)
	case "PRIVATE KEY":
		k, err := x509.ParsePKCS8PrivateKey(block.Bytes)
		if err != nil {
			return nil, err
		}
		ec, ok := k.(*ecdsa.PrivateKey)
		if !ok {
			return nil, errors.New("auth: PKCS#8 key is not ECDSA")
		}
		return ec, nil
	default:
		return nil, fmt.Errorf("auth: unsupported PEM block %q", block.Type)
	}
}
