package utils

import (
	"strings"
	"testing"
)

// TestEncryptMFASecretRoundTrip confirms a TOTP secret can be encrypted and
// decrypted cleanly with the MFA key — independent of the OAuth key so the
// two domains can rotate separately.
func TestEncryptMFASecretRoundTrip(t *testing.T) {
	t.Setenv("MFA_SECRET_ENCRYPTION_KEY", strings.Repeat("11", 32))
	t.Setenv("OAUTH_TOKEN_ENCRYPTION_KEY", strings.Repeat("22", 32))

	plaintext := "JBSWY3DPEHPK3PXP" // a sample TOTP base32 secret
	cipher, err := EncryptMFASecret(plaintext)
	if err != nil {
		t.Fatalf("encrypt: %v", err)
	}
	if cipher == plaintext {
		t.Fatalf("ciphertext must differ from plaintext")
	}
	back, err := DecryptMFASecret(cipher)
	if err != nil {
		t.Fatalf("decrypt: %v", err)
	}
	if back != plaintext {
		t.Fatalf("round trip mismatch: got %q want %q", back, plaintext)
	}
}

// TestMFAKeyFallsBackToOAuthKey confirms the dev-friendly fallback: if the
// dedicated MFA key is not set, the OAuth key is used. Keeps local setups
// working with a single env var.
func TestMFAKeyFallsBackToOAuthKey(t *testing.T) {
	t.Setenv("MFA_SECRET_ENCRYPTION_KEY", "")
	t.Setenv("OAUTH_TOKEN_ENCRYPTION_KEY", strings.Repeat("33", 32))

	cipher, err := EncryptMFASecret("hello")
	if err != nil {
		t.Fatalf("encrypt: %v", err)
	}
	back, err := DecryptMFASecret(cipher)
	if err != nil {
		t.Fatalf("decrypt: %v", err)
	}
	if back != "hello" {
		t.Fatalf("fallback round trip mismatch: %q", back)
	}
}
