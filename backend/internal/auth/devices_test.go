package auth

import (
	"crypto/sha256"
	"encoding/base64"
	"testing"
)

func TestGenerateDeviceToken_DecodesTo32Bytes(t *testing.T) {
	tok, err := GenerateDeviceToken()
	if err != nil {
		t.Fatalf("GenerateDeviceToken: %v", err)
	}
	raw, err := base64.RawURLEncoding.DecodeString(tok)
	if err != nil {
		t.Fatalf("base64 decode %q: %v", tok, err)
	}
	if len(raw) != 32 {
		t.Errorf("token length: got %d bytes, want 32", len(raw))
	}
}

func TestGenerateDeviceToken_NotPredictable(t *testing.T) {
	seen := make(map[string]struct{}, 64)
	for i := 0; i < 64; i++ {
		tok, err := GenerateDeviceToken()
		if err != nil {
			t.Fatalf("GenerateDeviceToken: %v", err)
		}
		if _, dup := seen[tok]; dup {
			t.Fatalf("collision on attempt %d: %q", i, tok)
		}
		seen[tok] = struct{}{}
	}
}

func TestHashDeviceToken_MatchesSHA256(t *testing.T) {
	got := HashDeviceToken("hello")
	want := sha256.Sum256([]byte("hello"))
	if string(got) != string(want[:]) {
		t.Errorf("hash mismatch")
	}
}

func TestIsValidPlatform(t *testing.T) {
	for _, ok := range []string{"macos", "android"} {
		if !IsValidPlatform(ok) {
			t.Errorf("IsValidPlatform(%q) = false, want true", ok)
		}
	}
	for _, bad := range []string{"", "MACOS", "ios", "linux", "windows"} {
		if IsValidPlatform(bad) {
			t.Errorf("IsValidPlatform(%q) = true, want false", bad)
		}
	}
}
