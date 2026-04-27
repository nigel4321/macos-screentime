package auth

import (
	"crypto/rand"
	"errors"
	"fmt"
	"math/big"
)

// PairingCodeTTL is the lifetime of a pairing code from issue to
// expiry; after that it cannot be redeemed.
const PairingCodeTTL = 10 * 60 // seconds — kept as int so callers can pass it to time.Duration math without importing this constant indirectly

// GeneratePairingCode returns a fresh, uniformly-distributed 6-digit
// numeric code suitable for human transcription. It uses crypto/rand —
// math/rand is unsafe for security tokens.
func GeneratePairingCode() (string, error) {
	n, err := rand.Int(rand.Reader, big.NewInt(1_000_000))
	if err != nil {
		return "", fmt.Errorf("auth: rand: %w", err)
	}
	return fmt.Sprintf("%06d", n.Int64()), nil
}

// Errors returned by pairing operations. They are exported so handlers
// can map them to HTTP status codes.
var (
	ErrPairingCodeNotFound = errors.New("auth: pairing code not found")
	ErrPairingCodeExpired  = errors.New("auth: pairing code expired")
	ErrPairingCodeConsumed = errors.New("auth: pairing code already consumed")
)
