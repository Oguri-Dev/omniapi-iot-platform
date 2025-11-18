package router

import "time"

// IntegrationConfig configura la integración con requester y status
type IntegrationConfig struct {
	// Status heartbeat configuration
	StatusHeartbeatInterval time.Duration `json:"status_heartbeat_interval"` // Frecuencia de heartbeats (default: 10s)
	StaleThresholdOK        int64         `json:"stale_threshold_ok"`        // Umbral para considerar OK en segundos (default: 60)
	StaleThresholdDegraded  int64         `json:"stale_threshold_degraded"`  // Umbral para considerar degraded en segundos (default: 300)

	// Requester configuration
	RequesterTimeout    time.Duration `json:"requester_timeout"`     // Timeout por request (default: 30s)
	RequesterQueueSize  int           `json:"requester_queue_size"`  // Tamaño de cola (default: 1000)
	RequesterMaxRetries int           `json:"requester_max_retries"` // Máximo de reintentos (default: 3)

	// Circuit breaker configuration
	CircuitBreakerMaxErrors     int           `json:"circuit_breaker_max_errors"`     // Errores consecutivos antes de abrir (default: 5)
	CircuitBreakerPauseDuration time.Duration `json:"circuit_breaker_pause_duration"` // Duración de pausa cuando se abre (default: 5min)

	// Backoff configuration
	BackoffInitial time.Duration `json:"backoff_initial"` // Backoff inicial (default: 1min)
	BackoffStep2   time.Duration `json:"backoff_step2"`   // Backoff paso 2 (default: 2min)
	BackoffStep3   time.Duration `json:"backoff_step3"`   // Backoff paso 3 (default: 5min)

	// General
	EnableStatusHeartbeats bool `json:"enable_status_heartbeats"` // Habilitar heartbeats de estado (default: true)
	EnableRequester        bool `json:"enable_requester"`         // Habilitar requester (default: true)
}

// DefaultIntegrationConfig retorna la configuración por defecto
func DefaultIntegrationConfig() IntegrationConfig {
	return IntegrationConfig{
		// Status
		StatusHeartbeatInterval: 10 * time.Second,
		StaleThresholdOK:        60,
		StaleThresholdDegraded:  300,

		// Requester
		RequesterTimeout:    30 * time.Second,
		RequesterQueueSize:  1000,
		RequesterMaxRetries: 3,

		// Circuit breaker
		CircuitBreakerMaxErrors:     5,
		CircuitBreakerPauseDuration: 5 * time.Minute,

		// Backoff
		BackoffInitial: 1 * time.Minute,
		BackoffStep2:   2 * time.Minute,
		BackoffStep3:   5 * time.Minute,

		// General
		EnableStatusHeartbeats: true,
		EnableRequester:        true,
	}
}
