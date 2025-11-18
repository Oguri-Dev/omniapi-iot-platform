package crypto

import (
	"strings"
	"testing"
)

const testKey = "12345678901234567890123456789012"

func setupCryptoService(t *testing.T) CryptoService {
	t.Helper()
	if err := InitService(testKey); err != nil {
		t.Fatalf("failed to init crypto service: %v", err)
	}

	svc, err := GetService()
	if err != nil {
		t.Fatalf("failed to get crypto service: %v", err)
	}

	return svc
}

func TestIsEncryptedDetectsEncryptedValues(t *testing.T) {
	svc := setupCryptoService(t)

	encrypted, err := svc.Encrypt("super-secret")
	if err != nil {
		t.Fatalf("encrypt failed: %v", err)
	}

	if !svc.IsEncrypted(encrypted) {
		t.Fatalf("expected value to be detected as encrypted")
	}

	decrypted, err := svc.Decrypt(encrypted)
	if err != nil {
		t.Fatalf("decrypt failed: %v", err)
	}

	if decrypted != "super-secret" {
		t.Fatalf("unexpected decrypted value: %s", decrypted)
	}
}

func TestIsEncryptedIgnoresPlainBase64Secrets(t *testing.T) {
	svc := setupCryptoService(t)

	plainBase64 := "CxN4QpQGMjwbrlM1PALCvB9XK9HO0KRap5Xp2ANQwoQLZT9ujhWS0PTZX6VvtdIDXfGCa0RlWPA2y40dhfrTelGNhqP0am99Q8smC0oV1Iu42E4BPWLXp3onbbnEegYD"
	if svc.IsEncrypted(plainBase64) {
		t.Fatalf("plain base64 string should not be detected as encrypted")
	}

	encrypted, err := svc.Encrypt(plainBase64)
	if err != nil {
		t.Fatalf("encrypt failed: %v", err)
	}

	if !svc.IsEncrypted(encrypted) {
		t.Fatalf("encrypted value should be detected as encrypted")
	}
}

func TestDecryptSupportsLegacyValuesWithoutPrefix(t *testing.T) {
	svc := setupCryptoService(t)

	encrypted, err := svc.Encrypt("legacy-secret")
	if err != nil {
		t.Fatalf("encrypt failed: %v", err)
	}

	legacyFormat := strings.TrimPrefix(encrypted, encryptedPrefix)
	decrypted, err := svc.Decrypt(legacyFormat)
	if err != nil {
		t.Fatalf("decrypt failed for legacy value: %v", err)
	}

	if decrypted != "legacy-secret" {
		t.Fatalf("unexpected decrypted value: %s", decrypted)
	}
}
