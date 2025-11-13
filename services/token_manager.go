package services

import (
	"fmt"
	"sync"
	"time"

	"omniapi/adapters"
	"omniapi/models"

	"go.mongodb.org/mongo-driver/bson/primitive"
)

// TokenCache entrada de cache para tokens
type TokenCache struct {
	Token        string
	TokenType    string
	ExpiresAt    time.Time
	ServiceID    primitive.ObjectID
	ServiceType  string
	RefreshToken string // Para OAuth2 que soporte refresh
}

// IsExpired verifica si el token ya expiró
func (tc *TokenCache) IsExpired() bool {
	// Considera expirado si faltan menos de 5 minutos
	return time.Now().Add(5 * time.Minute).After(tc.ExpiresAt)
}

// TokenManager gestor de tokens en memoria (NO SE PERSISTEN EN DB)
type TokenManager struct {
	cache map[string]*TokenCache // Key: serviceID.Hex()
	mu    sync.RWMutex
}

var (
	tokenManagerInstance *TokenManager
	tokenManagerOnce     sync.Once
)

// GetTokenManager obtiene la instancia singleton del Token Manager
func GetTokenManager() *TokenManager {
	tokenManagerOnce.Do(func() {
		tokenManagerInstance = &TokenManager{
			cache: make(map[string]*TokenCache),
		}
		// Iniciar goroutine de limpieza
		go tokenManagerInstance.cleanupExpiredTokens()
	})
	return tokenManagerInstance
}

// GetToken obtiene un token válido del cache o autentica nuevamente
func (tm *TokenManager) GetToken(service *models.ExternalService) (string, error) {
	serviceKey := service.ID.Hex()

	// Verificar cache
	tm.mu.RLock()
	cachedToken, exists := tm.cache[serviceKey]
	tm.mu.RUnlock()

	// Si existe y no está expirado, retornar
	if exists && !cachedToken.IsExpired() {
		return cachedToken.Token, nil
	}

	// Token expirado o no existe, autenticar nuevamente
	return tm.authenticateAndCache(service)
}

// authenticateAndCache autentica y guarda en cache
func (tm *TokenManager) authenticateAndCache(service *models.ExternalService) (string, error) {
	// Obtener adaptador de autenticación
	authAdapter, err := adapters.GetAuthAdapter(service.ServiceType)
	if err != nil {
		return "", fmt.Errorf("error obteniendo adaptador: %w", err)
	}

	// Autenticar
	tokenResp, err := authAdapter.Authenticate(service)
	if err != nil {
		return "", fmt.Errorf("error en autenticación: %w", err)
	}

	// Guardar en cache (SOLO EN MEMORIA)
	tm.mu.Lock()
	tm.cache[service.ID.Hex()] = &TokenCache{
		Token:        tokenResp.AccessToken,
		TokenType:    tokenResp.TokenType,
		ExpiresAt:    tokenResp.ExpiresAt,
		ServiceID:    service.ID,
		ServiceType:  service.ServiceType,
		RefreshToken: tokenResp.RefreshToken,
	}
	tm.mu.Unlock()

	return tokenResp.AccessToken, nil
}

// TestConnection prueba la conexión sin guardar en cache (solo para testing)
func (tm *TokenManager) TestConnection(service *models.ExternalService) (*adapters.TokenResponse, error) {
	authAdapter, err := adapters.GetAuthAdapter(service.ServiceType)
	if err != nil {
		return nil, fmt.Errorf("error obteniendo adaptador: %w", err)
	}

	// Autenticar pero NO guardar en cache
	tokenResp, err := authAdapter.Authenticate(service)
	if err != nil {
		return nil, fmt.Errorf("error en autenticación: %w", err)
	}

	return tokenResp, nil
}

// InvalidateToken invalida un token en cache (fuerza re-autenticación)
func (tm *TokenManager) InvalidateToken(serviceID primitive.ObjectID) {
	tm.mu.Lock()
	delete(tm.cache, serviceID.Hex())
	tm.mu.Unlock()
}

// GetCachedToken obtiene un token del cache sin re-autenticar
func (tm *TokenManager) GetCachedToken(serviceID primitive.ObjectID) (*TokenCache, bool) {
	tm.mu.RLock()
	defer tm.mu.RUnlock()

	token, exists := tm.cache[serviceID.Hex()]
	return token, exists
}

// cleanupExpiredTokens limpia tokens expirados cada 10 minutos
func (tm *TokenManager) cleanupExpiredTokens() {
	ticker := time.NewTicker(10 * time.Minute)
	defer ticker.Stop()

	for range ticker.C {
		tm.mu.Lock()
		now := time.Now()
		for key, token := range tm.cache {
			if now.After(token.ExpiresAt) {
				delete(tm.cache, key)
			}
		}
		tm.mu.Unlock()
	}
}

// GetCacheStats obtiene estadísticas del cache (para debugging)
func (tm *TokenManager) GetCacheStats() map[string]interface{} {
	tm.mu.RLock()
	defer tm.mu.RUnlock()

	stats := map[string]interface{}{
		"total_tokens":   len(tm.cache),
		"expired_tokens": 0,
		"valid_tokens":   0,
	}

	now := time.Now()
	for _, token := range tm.cache {
		if now.After(token.ExpiresAt) {
			stats["expired_tokens"] = stats["expired_tokens"].(int) + 1
		} else {
			stats["valid_tokens"] = stats["valid_tokens"].(int) + 1
		}
	}

	return stats
}

// ClearCache limpia completamente el cache (para testing o mantenimiento)
func (tm *TokenManager) ClearCache() {
	tm.mu.Lock()
	tm.cache = make(map[string]*TokenCache)
	tm.mu.Unlock()
}
