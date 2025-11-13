package adapters

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"

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

	// Endpoint de autenticación ScaleAQ
	authURL := fmt.Sprintf("%s/auth/token", strings.TrimSuffix(service.BaseURL, "/"))

	// Preparar payload
	payload := map[string]string{
		"username": service.Credentials.Username,
		"password": service.Credentials.Password,
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

	// Parsear respuesta
	var authResp struct {
		Token     string `json:"token"`
		ExpiresIn int    `json:"expires_in"`
	}

	if err := json.Unmarshal(body, &authResp); err != nil {
		return nil, fmt.Errorf("error parseando respuesta: %w", err)
	}

	// Construir TokenResponse
	tokenResp := &TokenResponse{
		AccessToken: authResp.Token,
		TokenType:   "Bearer",
		ExpiresIn:   authResp.ExpiresIn,
		ExpiresAt:   time.Now().Add(time.Duration(authResp.ExpiresIn) * time.Second),
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

	// Endpoint de autenticación Innovex (OAuth2)
	authURL := fmt.Sprintf("%s/api_register/token/", strings.TrimSuffix(service.BaseURL, "/"))

	// Preparar form data (application/x-www-form-urlencoded)
	formData := url.Values{}
	formData.Set("grant_type", "password")
	formData.Set("client_id", service.Credentials.ClientID)
	formData.Set("client_secret", service.Credentials.ClientSecret)
	formData.Set("username", service.Credentials.Username)
	formData.Set("password", service.Credentials.Password)

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
