package crypto

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"encoding/base64"
	"fmt"
	"io"
	"os"
	"strings"
)

// CryptoService interfaz para servicios de encriptación
type CryptoService interface {
	Encrypt(plaintext string) (string, error)
	Decrypt(ciphertext string) (string, error)
	IsEncrypted(text string) bool
}

// AESCryptoService implementación AES-GCM
type AESCryptoService struct {
	key []byte
}

// NewAESCryptoService crear nuevo servicio de encriptación AES
func NewAESCryptoService(key string) (*AESCryptoService, error) {
	// La clave debe ser de 32 bytes para AES-256
	keyBytes := []byte(key)
	if len(keyBytes) != 32 {
		return nil, fmt.Errorf("la clave debe ser de exactamente 32 bytes, recibida: %d bytes", len(keyBytes))
	}

	return &AESCryptoService{
		key: keyBytes,
	}, nil
}

// Encrypt encriptar texto usando AES-GCM
func (s *AESCryptoService) Encrypt(plaintext string) (string, error) {
	if plaintext == "" {
		return "", nil
	}

	// Crear cipher
	block, err := aes.NewCipher(s.key)
	if err != nil {
		return "", fmt.Errorf("error creando cipher: %w", err)
	}

	// Crear GCM
	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return "", fmt.Errorf("error creando GCM: %w", err)
	}

	// Generar nonce aleatorio
	nonce := make([]byte, gcm.NonceSize())
	if _, err := io.ReadFull(rand.Reader, nonce); err != nil {
		return "", fmt.Errorf("error generando nonce: %w", err)
	}

	// Encriptar
	ciphertext := gcm.Seal(nonce, nonce, []byte(plaintext), nil)

	// Codificar en base64
	return base64.StdEncoding.EncodeToString(ciphertext), nil
}

// Decrypt desencriptar texto usando AES-GCM
func (s *AESCryptoService) Decrypt(ciphertext string) (string, error) {
	if ciphertext == "" {
		return "", nil
	}

	// Decodificar base64
	data, err := base64.StdEncoding.DecodeString(ciphertext)
	if err != nil {
		return "", fmt.Errorf("error decodificando base64: %w", err)
	}

	// Crear cipher
	block, err := aes.NewCipher(s.key)
	if err != nil {
		return "", fmt.Errorf("error creando cipher: %w", err)
	}

	// Crear GCM
	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return "", fmt.Errorf("error creando GCM: %w", err)
	}

	// Verificar tamaño mínimo
	nonceSize := gcm.NonceSize()
	if len(data) < nonceSize {
		return "", fmt.Errorf("ciphertext demasiado corto")
	}

	// Extraer nonce y ciphertext
	nonce, ciphertextBytes := data[:nonceSize], data[nonceSize:]

	// Desencriptar
	plaintext, err := gcm.Open(nil, nonce, ciphertextBytes, nil)
	if err != nil {
		return "", fmt.Errorf("error desencriptando: %w", err)
	}

	return string(plaintext), nil
}

// IsEncrypted verificar si un texto está encriptado (heurística simple)
func (s *AESCryptoService) IsEncrypted(text string) bool {
	if text == "" {
		return false
	}

	// Un texto encriptado en base64 típicamente:
	// 1. Tiene longitud múltiple de 4 (padding base64)
	// 2. Solo contiene caracteres base64 válidos
	// 3. Tiene longitud suficiente (mínimo ~20 chars para AES-GCM)

	if len(text) < 20 {
		return false
	}

	// Verificar que sea base64 válido
	_, err := base64.StdEncoding.DecodeString(text)
	if err != nil {
		return false
	}

	// Verificar que no contenga espacios ni caracteres típicos de texto plano
	if strings.Contains(text, " ") || strings.Contains(text, "@") || strings.Contains(text, ".") {
		// Excepciones: algunos emails o URLs pueden estar encriptados
		// Pero si contiene patrones típicos de texto plano, probablemente no esté encriptado
		if !strings.HasSuffix(text, "=") && len(text) < 50 {
			return false
		}
	}

	return true
}

// Instancia global del servicio
var globalCryptoService CryptoService

// GetService obtener instancia del servicio de encriptación
func GetService() (CryptoService, error) {
	if globalCryptoService != nil {
		return globalCryptoService, nil
	}

	// Obtener clave de encriptación desde variable de entorno
	encryptionKey := os.Getenv("ENCRYPTION_KEY")
	if encryptionKey == "" {
		return nil, fmt.Errorf("ENCRYPTION_KEY no configurada en variables de entorno")
	}

	// Crear servicio AES
	service, err := NewAESCryptoService(encryptionKey)
	if err != nil {
		return nil, fmt.Errorf("error inicializando servicio de encriptación: %w", err)
	}

	globalCryptoService = service
	return globalCryptoService, nil
}

// InitService inicializar servicio con clave específica (para testing)
func InitService(key string) error {
	service, err := NewAESCryptoService(key)
	if err != nil {
		return err
	}
	globalCryptoService = service
	return nil
}
