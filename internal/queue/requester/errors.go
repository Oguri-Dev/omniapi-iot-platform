package requester

import "errors"

var (
	// ErrQueueFull indica que la cola está llena
	ErrQueueFull = errors.New("queue is full")

	// ErrRequesterStopped indica que el requester está detenido
	ErrRequesterStopped = errors.New("requester is stopped")

	// ErrRequesterPaused indica que el requester está pausado (circuit breaker)
	ErrRequesterPaused = errors.New("requester is paused due to circuit breaker")

	// ErrInvalidRequest indica que la solicitud no es válida
	ErrInvalidRequest = errors.New("invalid request")

	// ErrNoStrategy indica que no hay estrategia configurada
	ErrNoStrategy = errors.New("no strategy configured")

	// ErrTimeout indica que la solicitud excedió el timeout
	ErrTimeout = errors.New("request timeout")

	// ErrCircuitOpen indica que el circuit breaker está abierto
	ErrCircuitOpen = errors.New("circuit breaker is open")
)
