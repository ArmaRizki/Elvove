package httpapi

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/sha256"
	"encoding/base64"
	"testing"
)

func TestSMTPPasswordEncryptionMatchesJavaGCMEnvelope(t *testing.T) {
	key := []byte("01234567890123456789012345678901")
	for _, plaintext := range []string{"mail-password", "compatibility"} {
		encoded, err := encryptSMTPPassword(key, plaintext)
		if err != nil {
			t.Fatal(err)
		}
		got, err := decryptSMTPPassword(key, encoded)
		if err != nil || got != plaintext {
			t.Fatalf("round trip = %q, %v", got, err)
		}
	}
	encoded, _ := encryptSMTPPassword(key, "wrong-key")
	if _, err := decryptSMTPPassword([]byte("abcdef0123456789abcdef0123456789"), encoded); err == nil {
		t.Fatal("expected wrong key to fail")
	}
}

func TestSMTPPasswordDecryptsJavaGeneratedGCMEnvelope(t *testing.T) {
	// Fixed AES-GCM vector using Java's SHA-256 key derivation and nonce||ciphertext||tag Base64 envelope.
	const encoded = "AAECAwQFBgcICQoLpIase8/017V0rDrfSvUqvl+4hlZHvsn9g4BWdt7rsw=="
	key := []byte("01234567890123456789012345678901")
	got, err := decryptSMTPPassword(key, encoded)
	if err != nil || got != "java-compatible" {
		t.Fatalf("decrypted %q, %v", got, err)
	}
	derived := sha256.Sum256(key)
	block, err := aes.NewCipher(derived[:])
	if err != nil {
		t.Fatal(err)
	}
	gcm, err := cipher.NewGCM(block)
	if err != nil {
		t.Fatal(err)
	}
	if _, err = base64.StdEncoding.DecodeString(encoded); err != nil || gcm.NonceSize() != 12 {
		t.Fatal("invalid Java fixture")
	}
}
