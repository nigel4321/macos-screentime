package auth

import (
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/x509"
	"encoding/pem"
	"strings"
	"testing"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

func mustGenerateP256(t *testing.T) (*ecdsa.PrivateKey, []byte) {
	t.Helper()
	priv, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	if err != nil {
		t.Fatalf("generate key: %v", err)
	}
	der, err := x509.MarshalECPrivateKey(priv)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	pemBytes := pem.EncodeToMemory(&pem.Block{Type: "EC PRIVATE KEY", Bytes: der})
	return priv, pemBytes
}

func TestSigner_IssueAndVerifyRoundtrip(t *testing.T) {
	_, pemBytes := mustGenerateP256(t)
	signer, err := NewSigner(pemBytes)
	if err != nil {
		t.Fatalf("NewSigner: %v", err)
	}
	verifier, err := NewVerifier(signer.PublicKey())
	if err != nil {
		t.Fatalf("NewVerifier: %v", err)
	}

	tok, err := signer.Issue("acct-123")
	if err != nil {
		t.Fatalf("Issue: %v", err)
	}

	claims, err := verifier.Parse(tok)
	if err != nil {
		t.Fatalf("Parse: %v", err)
	}
	if claims.AccountID != "acct-123" {
		t.Errorf("AccountID: got %q, want acct-123", claims.AccountID)
	}
	if claims.Issuer != Issuer {
		t.Errorf("Issuer: got %q, want %q", claims.Issuer, Issuer)
	}
}

func TestSigner_PKCS8KeyAccepted(t *testing.T) {
	priv, _ := mustGenerateP256(t)
	der, err := x509.MarshalPKCS8PrivateKey(priv)
	if err != nil {
		t.Fatalf("marshal pkcs8: %v", err)
	}
	pemBytes := pem.EncodeToMemory(&pem.Block{Type: "PRIVATE KEY", Bytes: der})
	if _, err := NewSigner(pemBytes); err != nil {
		t.Fatalf("NewSigner with PKCS#8: %v", err)
	}
}

func TestVerifier_RejectsExpiredToken(t *testing.T) {
	_, pemBytes := mustGenerateP256(t)
	signer, _ := NewSigner(pemBytes)
	signer.now = func() time.Time { return time.Now().Add(-2 * time.Hour) }
	signer.ttl = time.Hour

	verifier, _ := NewVerifier(signer.PublicKey())
	tok, _ := signer.Issue("acct-1")

	if _, err := verifier.Parse(tok); err == nil {
		t.Fatal("expected expired token to be rejected")
	}
}

func TestVerifier_RejectsUnknownKID(t *testing.T) {
	_, pemA := mustGenerateP256(t)
	_, pemB := mustGenerateP256(t)
	signerA, _ := NewSigner(pemA)
	signerB, _ := NewSigner(pemB)

	// Verifier knows only signerA's key.
	verifier, _ := NewVerifier(signerA.PublicKey())

	tok, _ := signerB.Issue("acct-1")
	_, err := verifier.Parse(tok)
	if err == nil || !strings.Contains(err.Error(), "unknown kid") {
		t.Fatalf("expected unknown-kid error, got %v", err)
	}
}

func TestVerifier_RejectsHS256AlgorithmConfusion(t *testing.T) {
	_, pemBytes := mustGenerateP256(t)
	signer, _ := NewSigner(pemBytes)
	verifier, _ := NewVerifier(signer.PublicKey())

	// Forge an HS256 token signed with the EC public key bytes — the
	// classic algorithm-confusion attack against permissive verifiers.
	pubDER, _ := x509.MarshalPKIXPublicKey(signer.PublicKey())
	tok := jwt.NewWithClaims(jwt.SigningMethodHS256, Claims{
		AccountID:        "attacker",
		RegisteredClaims: jwt.RegisteredClaims{Issuer: Issuer, ExpiresAt: jwt.NewNumericDate(time.Now().Add(time.Hour))},
	})
	tok.Header["kid"] = signer.KID()
	signed, err := tok.SignedString(pubDER)
	if err != nil {
		t.Fatalf("forge: %v", err)
	}

	if _, err := verifier.Parse(signed); err == nil {
		t.Fatal("verifier accepted HS256 token — algorithm-confusion attack succeeded")
	}
}

func TestVerifier_RejectsTamperedSignature(t *testing.T) {
	_, pemBytes := mustGenerateP256(t)
	signer, _ := NewSigner(pemBytes)
	verifier, _ := NewVerifier(signer.PublicKey())

	tok, _ := signer.Issue("acct-1")
	// Flip a middle character of the signature so we don't accidentally
	// land on base64 padding bits that would round-trip to the same
	// signature bytes.
	parts := strings.Split(tok, ".")
	if len(parts) != 3 {
		t.Fatalf("unexpected token structure: %q", tok)
	}
	mid := len(parts[2]) / 2
	flipped := byte('A')
	if parts[2][mid] == 'A' {
		flipped = 'B'
	}
	tampered := parts[0] + "." + parts[1] + "." + parts[2][:mid] + string(flipped) + parts[2][mid+1:]
	if _, err := verifier.Parse(tampered); err == nil {
		t.Fatal("expected tampered signature to be rejected")
	}
}

func TestKeyID_Deterministic(t *testing.T) {
	priv, _ := mustGenerateP256(t)
	a, _ := KeyID(&priv.PublicKey)
	b, _ := KeyID(&priv.PublicKey)
	if a != b {
		t.Errorf("KeyID not deterministic: %q vs %q", a, b)
	}
	if a == "" {
		t.Error("KeyID returned empty string")
	}
}

func TestKeyID_DifferentKeysProduceDifferentIDs(t *testing.T) {
	privA, _ := mustGenerateP256(t)
	privB, _ := mustGenerateP256(t)
	a, _ := KeyID(&privA.PublicKey)
	b, _ := KeyID(&privB.PublicKey)
	if a == b {
		t.Error("KeyID collision across distinct keys")
	}
}

func TestVerifier_AcceptsRotatedKey(t *testing.T) {
	// Rotation scenario: token issued by oldSigner, verifier knows
	// both old and new public keys.
	_, oldPEM := mustGenerateP256(t)
	_, newPEM := mustGenerateP256(t)
	oldSigner, _ := NewSigner(oldPEM)
	newSigner, _ := NewSigner(newPEM)

	verifier, _ := NewVerifier(oldSigner.PublicKey(), newSigner.PublicKey())
	tok, _ := oldSigner.Issue("acct-1")

	claims, err := verifier.Parse(tok)
	if err != nil {
		t.Fatalf("Parse: %v", err)
	}
	if claims.AccountID != "acct-1" {
		t.Errorf("AccountID: got %q", claims.AccountID)
	}
}
