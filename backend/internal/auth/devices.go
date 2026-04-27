package auth

import (
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"errors"
	"fmt"
)

// Platform identifies which client a device row belongs to. The set
// here must stay in sync with the CHECK constraint on device.platform
// in migration 00002_device.sql.
const (
	PlatformMacOS   = "macos"
	PlatformAndroid = "android"
)

// IsValidPlatform reports whether p is one of the recognised platform
// strings accepted by the schema.
func IsValidPlatform(p string) bool {
	return p == PlatformMacOS || p == PlatformAndroid
}

// deviceTokenBytes is the size of the random secret backing a device
// token. 32 bytes => 256 bits, comfortably above any plausible online
// brute-force budget.
const deviceTokenBytes = 32

// GenerateDeviceToken returns a fresh, base64url-encoded device
// token. The plaintext is returned to the client exactly once at
// registration time; the server retains only its SHA-256 hash.
func GenerateDeviceToken() (string, error) {
	buf := make([]byte, deviceTokenBytes)
	if _, err := rand.Read(buf); err != nil {
		return "", fmt.Errorf("auth: rand: %w", err)
	}
	return base64.RawURLEncoding.EncodeToString(buf), nil
}

// HashDeviceToken returns the SHA-256 digest used as the lookup key
// for a device token. SHA-256 (not bcrypt/argon2) is sufficient here
// because the input is server-generated 32-byte random — there is no
// human-chosen entropy to defend against.
func HashDeviceToken(token string) []byte {
	sum := sha256.Sum256([]byte(token))
	return sum[:]
}

// ErrUnknownDevice is returned by DeviceResolver implementations when
// the provided token does not resolve to a device for the given
// account.
var ErrUnknownDevice = errors.New("auth: unknown device token")
