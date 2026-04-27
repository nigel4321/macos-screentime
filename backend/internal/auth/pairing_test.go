package auth

import (
	"regexp"
	"testing"
)

func TestGeneratePairingCode_Format(t *testing.T) {
	re := regexp.MustCompile(`^\d{6}$`)
	for i := 0; i < 100; i++ {
		c, err := GeneratePairingCode()
		if err != nil {
			t.Fatalf("GeneratePairingCode: %v", err)
		}
		if !re.MatchString(c) {
			t.Errorf("code %q does not match ^\\d{6}$", c)
		}
	}
}

func TestGeneratePairingCode_Distribution(t *testing.T) {
	// Sanity check: 200 generations should not all collide on a small
	// prefix, otherwise rand.Int has been miswired.
	seen := map[string]struct{}{}
	for i := 0; i < 200; i++ {
		c, _ := GeneratePairingCode()
		seen[c[:2]] = struct{}{}
	}
	if len(seen) < 20 {
		t.Errorf("only %d distinct 2-digit prefixes across 200 codes — distribution looks bad", len(seen))
	}
}
