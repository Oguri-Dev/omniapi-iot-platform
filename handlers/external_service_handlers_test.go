package handlers

import (
	"testing"

	"omniapi/crypto"
	"omniapi/models"
)

func TestEncryptServiceCredentials(t *testing.T) {
	key := "12345678901234567890123456789012"
	if err := crypto.InitService(key); err != nil {
		t.Fatalf("failed to init crypto service: %v", err)
	}

	cryptoService, err := crypto.GetService()
	if err != nil {
		t.Fatalf("failed to get crypto service: %v", err)
	}

	credentials := &models.ServiceCredentials{
		Username:     "demo",
		Password:     "super-secret",
		ClientSecret: "client-secret",
		APIKey:       "api-key",
	}

	if err := encryptServiceCredentials(credentials, cryptoService); err != nil {
		t.Fatalf("encryptServiceCredentials returned error: %v", err)
	}

	assertEncryptedAndDecrypted(t, cryptoService, credentials.Password, "super-secret")
	assertEncryptedAndDecrypted(t, cryptoService, credentials.ClientSecret, "client-secret")
	assertEncryptedAndDecrypted(t, cryptoService, credentials.APIKey, "api-key")

	// Ejecutar nuevamente para asegurar idempotencia
	if err := encryptServiceCredentials(credentials, cryptoService); err != nil {
		t.Fatalf("second encryptServiceCredentials returned error: %v", err)
	}
}

func assertEncryptedAndDecrypted(t *testing.T, cryptoService crypto.CryptoService, encrypted, expectedPlain string) {
	t.Helper()

	if encrypted == "" {
		t.Fatalf("expected encrypted value, got empty string")
	}

	if encrypted == expectedPlain {
		t.Fatalf("value was not encrypted, got %s", encrypted)
	}

	decrypted, err := cryptoService.Decrypt(encrypted)
	if err != nil {
		t.Fatalf("failed to decrypt value: %v", err)
	}

	if decrypted != expectedPlain {
		t.Fatalf("decrypted value mismatch. got %s, want %s", decrypted, expectedPlain)
	}
}
