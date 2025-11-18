package adapters

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"

	"omniapi/crypto"
	"omniapi/models"
)

// TokenResponse respuesta estándar de autenticación
type TokenResponse struct {
	AccessToken  string    `json:"access_token"`
	TokenType    string    `json:"token_type"`
	ExpiresIn    int       `json:"expires_in"`
	RefreshToken string    `json:"refresh_token,omitempty"`
	ExpiresAt    time.Time `json:"-"` // Calculado internamente
}

// AuthAdapter interfaz para adaptadores de autenticación
type AuthAdapter interface {
	Authenticate(service *models.ExternalService) (*TokenResponse, error)
	GetServiceType() string
}

// ============================================================
// ScaleAQ Auth Adapter (Bearer Token)
// ============================================================

type ScaleAQAuthAdapter struct{}

func NewScaleAQAuthAdapter() *ScaleAQAuthAdapter {
	return &ScaleAQAuthAdapter{}
}

func (a *ScaleAQAuthAdapter) GetServiceType() string {
	return "scaleaq"
}

func (a *ScaleAQAuthAdapter) Authenticate(service *models.ExternalService) (*TokenResponse, error) {
	if service.Credentials == nil {
		return nil, fmt.Errorf("credenciales no configuradas")
	}

	// Desencriptar credenciales para uso en API externa
	cryptoService, err := crypto.GetService()
	if err != nil {
		return nil, fmt.Errorf("error inicializando crypto service: %w", err)
	}

	// Desencriptar password
	decryptedPassword := service.Credentials.Password
	if cryptoService.IsEncrypted(service.Credentials.Password) {
		decryptedPassword, err = cryptoService.Decrypt(service.Credentials.Password)
		if err != nil {
			return nil, fmt.Errorf("error desencriptando password: %w", err)
		}
	}

	if isBcryptHash(decryptedPassword) {
		return nil, fmt.Errorf("el password de ScaleAQ fue almacenado con un formato anterior no reversible; vuelve a guardar el servicio con la clave actual")
	}

	// Endpoint de autenticación ScaleAQ
	authURL := fmt.Sprintf("%s/auth/token", strings.TrimSuffix(service.BaseURL, "/"))

	// Preparar payload con credenciales desencriptadas
	payload := map[string]string{
		"username": service.Credentials.Username,
		"password": decryptedPassword,
	}

	jsonData, err := json.Marshal(payload)
	if err != nil {
		return nil, fmt.Errorf("error creando payload: %w", err)
	}

	// Crear request
	req, err := http.NewRequest("POST", authURL, bytes.NewBuffer(jsonData))
	if err != nil {
		return nil, fmt.Errorf("error creando request: %w", err)
	}

	// Headers específicos de ScaleAQ
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Scale-Version", "2025-01-01") // Header customizado de ScaleAQ

	// Headers adicionales de configuración
	if service.Credentials.CustomHeaders != nil {
		for key, value := range service.Credentials.CustomHeaders {
			req.Header.Set(key, value)
		}
	}

	// Ejecutar request con timeout
	client := &http.Client{
		Timeout: 30 * time.Second,
	}

	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("error en request de autenticación: %w", err)
	}
	defer resp.Body.Close()

	// Leer respuesta
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("error leyendo respuesta: %w", err)
	}

	// Verificar status code
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("autenticación fallida (HTTP %d): %s", resp.StatusCode, string(body))
	}

	// Parsear respuesta - ScaleAQ devuelve expires_in como string
	var authResp struct {
		AccessToken string `json:"access_token"`
		TokenType   string `json:"token_type"`
		ExpiresIn   string `json:"expires_in"` // ← Cambiado a string
	}

	if err := json.Unmarshal(body, &authResp); err != nil {
		return nil, fmt.Errorf("error parseando respuesta: %w", err)
	}

	// Convertir expires_in de string a int
	expiresInInt := 3600 // Default 1 hora
	if authResp.ExpiresIn != "" {
		if parsed, err := strconv.Atoi(authResp.ExpiresIn); err == nil {
			expiresInInt = parsed
		}
	}

	// Construir TokenResponse
	tokenResp := &TokenResponse{
		AccessToken: authResp.AccessToken,
		TokenType:   authResp.TokenType,
		ExpiresIn:   expiresInInt, // Usar el valor convertido a int
		ExpiresAt:   time.Now().Add(time.Duration(expiresInInt) * time.Second),
	}

	return tokenResp, nil
}

// ============================================================
// Innovex Auth Adapter (OAuth2 Password Grant)
// ============================================================

type InnovexAuthAdapter struct{}

func NewInnovexAuthAdapter() *InnovexAuthAdapter {
	return &InnovexAuthAdapter{}
}

func (a *InnovexAuthAdapter) GetServiceType() string {
	return "innovex"
}

func (a *InnovexAuthAdapter) Authenticate(service *models.ExternalService) (*TokenResponse, error) {
	if service.Credentials == nil {
		return nil, fmt.Errorf("credenciales no configuradas")
	}

	// Desencriptar credenciales para uso en API externa
	cryptoService, err := crypto.GetService()
	if err != nil {
		return nil, fmt.Errorf("error inicializando crypto service: %w", err)
	}

	// Desencriptar campos sensibles
	decryptedPassword := service.Credentials.Password
	decryptedClientSecret := service.Credentials.ClientSecret

	fmt.Printf("🔐 Password IsEncrypted: %v (length: %d)\n", cryptoService.IsEncrypted(service.Credentials.Password), len(service.Credentials.Password))
	fmt.Printf("🔐 ClientSecret IsEncrypted: %v (length: %d)\n", cryptoService.IsEncrypted(service.Credentials.ClientSecret), len(service.Credentials.ClientSecret))

	if cryptoService.IsEncrypted(service.Credentials.Password) {
		decryptedPassword, err = cryptoService.Decrypt(service.Credentials.Password)
		if err != nil {
			fmt.Printf("❌ Error desencriptando password: %v\n", err)
			fmt.Printf("⚠️  Usando password como texto plano como fallback\n")
			// Fallback: usar el password como texto plano
			decryptedPassword = service.Credentials.Password
		} else {
			fmt.Printf("✅ Password desencriptado correctamente\n")
		}
	}

	if cryptoService.IsEncrypted(service.Credentials.ClientSecret) {
		decryptedClientSecret, err = cryptoService.Decrypt(service.Credentials.ClientSecret)
		if err != nil {
			fmt.Printf("❌ Error desencriptando client secret: %v\n", err)
			fmt.Printf("⚠️  Usando client_secret como texto plano como fallback\n")
			// Fallback: usar el client_secret como texto plano
			decryptedClientSecret = service.Credentials.ClientSecret
		} else {
			fmt.Printf("✅ Client secret desencriptado correctamente\n")
		}
	}

	if isBcryptHash(decryptedPassword) {
		return nil, fmt.Errorf("el password de Innovex fue almacenado con un formato anterior no reversible; vuelve a guardar el servicio con la clave actual")
	}

	if isBcryptHash(decryptedClientSecret) {
		return nil, fmt.Errorf("el client secret de Innovex fue almacenado con un formato anterior no reversible; vuelve a guardar el servicio con la clave actual")
	}

	// Endpoint de autenticación Innovex - evitar duplicar path si ya está en base_url
	authURL := service.BaseURL
	if !strings.Contains(service.BaseURL, "/api_register/token") {
		authURL = fmt.Sprintf("%s/api_register/token/", strings.TrimSuffix(service.BaseURL, "/"))
	}

	fmt.Printf("🔗 Innovex Auth URL: %s\n", authURL)

	// Preparar form data (application/x-www-form-urlencoded)
	formData := url.Values{}
	formData.Set("grant_type", "password")
	formData.Set("client_id", service.Credentials.ClientID)
	formData.Set("client_secret", decryptedClientSecret)
	formData.Set("username", service.Credentials.Username)
	formData.Set("password", decryptedPassword)

	// Crear request
	req, err := http.NewRequest("POST", authURL, strings.NewReader(formData.Encode()))
	if err != nil {
		return nil, fmt.Errorf("error creando request: %w", err)
	}

	// Headers OAuth2
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	// Headers adicionales
	if service.Credentials.CustomHeaders != nil {
		for key, value := range service.Credentials.CustomHeaders {
			req.Header.Set(key, value)
		}
	}

	// Ejecutar request con timeout
	client := &http.Client{
		Timeout: 30 * time.Second,
	}

	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("error en request de autenticación: %w", err)
	}
	defer resp.Body.Close()

	// Leer respuesta
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("error leyendo respuesta: %w", err)
	}

	fmt.Printf("📋 Innovex Response Status: %d\n", resp.StatusCode)
	if len(body) > 500 {
		fmt.Printf("📋 Innovex Response Body (first 500 chars): %s\n", string(body[:500]))
	} else {
		fmt.Printf("📋 Innovex Response Body: %s\n", string(body))
	}

	// Verificar status code
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("autenticación fallida (HTTP %d): %s", resp.StatusCode, string(body))
	}

	// Parsear respuesta OAuth2
	var tokenResp TokenResponse
	if err := json.Unmarshal(body, &tokenResp); err != nil {
		return nil, fmt.Errorf("error parseando respuesta: %w", err)
	}

	// Calcular fecha de expiración
	if tokenResp.ExpiresIn > 0 {
		tokenResp.ExpiresAt = time.Now().Add(time.Duration(tokenResp.ExpiresIn) * time.Second)
	}

	return &tokenResp, nil
}

// ============================================================
// API Key Auth Adapter (genérico)
// ============================================================

type APIKeyAuthAdapter struct{}

func NewAPIKeyAuthAdapter() *APIKeyAuthAdapter {
	return &APIKeyAuthAdapter{}
}

func (a *APIKeyAuthAdapter) GetServiceType() string {
	return "apikey"
}

func (a *APIKeyAuthAdapter) Authenticate(service *models.ExternalService) (*TokenResponse, error) {
	if service.Credentials == nil || service.Credentials.APIKey == "" {
		return nil, fmt.Errorf("API Key no configurada")
	}

	// Para API Key, el "token" es la key misma (no caduca)
	tokenResp := &TokenResponse{
		AccessToken: service.Credentials.APIKey,
		TokenType:   "ApiKey",
		ExpiresIn:   0,                                          // No expira
		ExpiresAt:   time.Now().Add(100 * 365 * 24 * time.Hour), // 100 años
	}

	return tokenResp, nil
}

// ============================================================
// Factory para crear adaptadores
// ============================================================

func GetAuthAdapter(serviceType string) (AuthAdapter, error) {
	switch strings.ToLower(serviceType) {
	case "scaleaq":
		return NewScaleAQAuthAdapter(), nil
	case "innovex":
		return NewInnovexAuthAdapter(), nil
	case "apikey":
		return NewAPIKeyAuthAdapter(), nil
	default:
		return nil, fmt.Errorf("tipo de servicio no soportado: %s", serviceType)
	}
}

func isBcryptHash(value string) bool {
	return strings.HasPrefix(value, "$2a$") || strings.HasPrefix(value, "$2b$") || strings.HasPrefix(value, "$2y$")
}
