package requester

import (
	"sync"
	"time"
)

// CircuitBreaker implementa un circuit breaker con backoff exponencial
type CircuitBreaker struct {
	config            Config
	consecutiveErrors int
	isOpen            bool
	openedAt          time.Time
	nextRetryAt       time.Time
	mu                sync.RWMutex
}

// NewCircuitBreaker crea un nuevo circuit breaker
func NewCircuitBreaker(config Config) *CircuitBreaker {
	return &CircuitBreaker{
		config: config,
	}
}

// RecordSuccess registra un éxito
func (cb *CircuitBreaker) RecordSuccess() {
	cb.mu.Lock()
	defer cb.mu.Unlock()

	cb.consecutiveErrors = 0
	if cb.isOpen {
		cb.isOpen = false
		cb.openedAt = time.Time{}
		cb.nextRetryAt = time.Time{}
	}
}

// RecordFailure registra un fallo y aplica backoff
func (cb *CircuitBreaker) RecordFailure() {
	cb.mu.Lock()
	defer cb.mu.Unlock()

	cb.consecutiveErrors++

	// Si alcanzamos el máximo de errores consecutivos, abrir el circuit breaker
	if cb.consecutiveErrors >= cb.config.MaxConsecutiveErrors {
		cb.isOpen = true
		cb.openedAt = time.Now()
		cb.nextRetryAt = time.Now().Add(cb.config.CircuitPauseDuration)
	}
}

// IsOpen verifica si el circuit breaker está abierto
func (cb *CircuitBreaker) IsOpen() bool {
	cb.mu.RLock()
	defer cb.mu.RUnlock()

	if !cb.isOpen {
		return false
	}

	// Verificar si es tiempo de reintentar
	if time.Now().After(cb.nextRetryAt) {
		return false // Permitir un intento
	}

	return true
}

// GetConsecutiveErrors retorna el número de errores consecutivos
func (cb *CircuitBreaker) GetConsecutiveErrors() int {
	cb.mu.RLock()
	defer cb.mu.RUnlock()
	return cb.consecutiveErrors
}

// GetNextRetryAt retorna cuándo se puede reintentar
func (cb *CircuitBreaker) GetNextRetryAt() *time.Time {
	cb.mu.RLock()
	defer cb.mu.RUnlock()

	if cb.isOpen && !cb.nextRetryAt.IsZero() {
		t := cb.nextRetryAt
		return &t
	}
	return nil
}

// Reset reinicia el circuit breaker
func (cb *CircuitBreaker) Reset() {
	cb.mu.Lock()
	defer cb.mu.Unlock()

	cb.consecutiveErrors = 0
	cb.isOpen = false
	cb.openedAt = time.Time{}
	cb.nextRetryAt = time.Time{}
}

// GetState retorna el estado actual del circuit breaker
func (cb *CircuitBreaker) GetState() CircuitBreakerState {
	cb.mu.RLock()
	defer cb.mu.RUnlock()

	if cb.isOpen {
		return CircuitBreakerStateOpen
	}
	if cb.consecutiveErrors > 0 {
		return CircuitBreakerStateHalfOpen
	}
	return CircuitBreakerStateClosed
}

// CircuitBreakerState representa el estado del circuit breaker
type CircuitBreakerState string

const (
	CircuitBreakerStateClosed   CircuitBreakerState = "closed"
	CircuitBreakerStateOpen     CircuitBreakerState = "open"
	CircuitBreakerStateHalfOpen CircuitBreakerState = "half_open"
)

// BackoffCalculator calcula el tiempo de espera con backoff exponencial
type BackoffCalculator struct {
	config Config
}

// NewBackoffCalculator crea un nuevo calculador de backoff
func NewBackoffCalculator(config Config) *BackoffCalculator {
	return &BackoffCalculator{
		config: config,
	}
}

// CalculateBackoff calcula el tiempo de backoff basado en el número de errores
func (bc *BackoffCalculator) CalculateBackoff(errorCount int) time.Duration {
	if errorCount == 0 {
		return 0
	}

	// Backoff exponencial: 1m, 2m, 5m+
	switch {
	case errorCount == 1:
		return bc.config.BackoffInitial
	case errorCount == 2:
		return bc.config.BackoffStep2
	case errorCount >= 3:
		return bc.config.BackoffStep3
	default:
		return bc.config.BackoffInitial
	}
}

// GetNextRetryTime calcula cuándo se debe reintentar
func (bc *BackoffCalculator) GetNextRetryTime(errorCount int) time.Time {
	backoff := bc.CalculateBackoff(errorCount)
	return time.Now().Add(backoff)
}
