package requester

import (
	"context"
	"encoding/json"
	"fmt"
	"time"
)

// MockStrategy es una estrategia de prueba para testing
type MockStrategy struct {
	name        string
	shouldFail  bool
	latency     int64
	payloadFunc func(Request) json.RawMessage
}

// NewMockStrategy crea una nueva estrategia mock
func NewMockStrategy(name string) *MockStrategy {
	return &MockStrategy{
		name:       name,
		shouldFail: false,
		latency:    100,
		payloadFunc: func(req Request) json.RawMessage {
			return json.RawMessage(fmt.Sprintf(`{"metric":"%s","data":[]}`, req.Metric))
		},
	}
}

// Execute ejecuta la estrategia mock
func (ms *MockStrategy) Execute(ctx context.Context, req Request) (json.RawMessage, error) {
	if ms.shouldFail {
		return nil, fmt.Errorf("mock strategy error")
	}

	// Simular latency si está configurada
	if ms.latency > 0 {
		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		case <-time.After(time.Duration(ms.latency) * time.Millisecond):
		}
	}

	if ms.payloadFunc != nil {
		return ms.payloadFunc(req), nil
	}

	return json.RawMessage(`{}`), nil
}

// Name retorna el nombre de la estrategia
func (ms *MockStrategy) Name() string {
	return ms.name
}

// HealthCheck verifica la salud de la estrategia
func (ms *MockStrategy) HealthCheck(ctx context.Context) error {
	if ms.shouldFail {
		return fmt.Errorf("mock strategy unhealthy")
	}
	return nil
}

// SetShouldFail configura si la estrategia debe fallar
func (ms *MockStrategy) SetShouldFail(fail bool) {
	ms.shouldFail = fail
}

// SetPayloadFunc configura la función que genera el payload
func (ms *MockStrategy) SetPayloadFunc(fn func(Request) json.RawMessage) {
	ms.payloadFunc = fn
}

// NoOpStrategy es una estrategia que no hace nada (placeholder)
type NoOpStrategy struct{}

// NewNoOpStrategy crea una nueva estrategia no-op
func NewNoOpStrategy() *NoOpStrategy {
	return &NoOpStrategy{}
}

// Execute no hace nada
func (nos *NoOpStrategy) Execute(ctx context.Context, req Request) (json.RawMessage, error) {
	return json.RawMessage(`{"status":"noop"}`), nil
}

// Name retorna el nombre
func (nos *NoOpStrategy) Name() string {
	return "noop"
}

// HealthCheck siempre retorna OK
func (nos *NoOpStrategy) HealthCheck(ctx context.Context) error {
	return nil
}

// ScaleAQCloudStrategy es un placeholder para la estrategia de ScaleAQ Cloud
// Implementación real debe hacerse en el futuro
type ScaleAQCloudStrategy struct {
	endpoint string
	apiKey   string
}

// NewScaleAQCloudStrategy crea una estrategia para ScaleAQ Cloud
func NewScaleAQCloudStrategy(endpoint, apiKey string) *ScaleAQCloudStrategy {
	return &ScaleAQCloudStrategy{
		endpoint: endpoint,
		apiKey:   apiKey,
	}
}

// Execute ejecuta la solicitud contra ScaleAQ Cloud
func (scs *ScaleAQCloudStrategy) Execute(ctx context.Context, req Request) (json.RawMessage, error) {
	// TODO: Implementar llamada real a /time-series/retrieve
	// Por ahora es un placeholder
	return nil, fmt.Errorf("ScaleAQ Cloud strategy not implemented yet")
}

// Name retorna el nombre
func (scs *ScaleAQCloudStrategy) Name() string {
	return "scaleaq-cloud"
}

// HealthCheck verifica conectividad con ScaleAQ Cloud
func (scs *ScaleAQCloudStrategy) HealthCheck(ctx context.Context) error {
	// TODO: Implementar health check real
	return fmt.Errorf("ScaleAQ Cloud health check not implemented yet")
}

// ProcessAPIStrategy es un placeholder para la estrategia de ProcessAPI local
type ProcessAPIStrategy struct {
	endpoint string
}

// NewProcessAPIStrategy crea una estrategia para ProcessAPI
func NewProcessAPIStrategy(endpoint string) *ProcessAPIStrategy {
	return &ProcessAPIStrategy{
		endpoint: endpoint,
	}
}

// Execute ejecuta la solicitud contra ProcessAPI local
func (pas *ProcessAPIStrategy) Execute(ctx context.Context, req Request) (json.RawMessage, error) {
	// TODO: Implementar llamada real a ProcessAPI
	return nil, fmt.Errorf("ProcessAPI strategy not implemented yet")
}

// Name retorna el nombre
func (pas *ProcessAPIStrategy) Name() string {
	return "processapi"
}

// HealthCheck verifica conectividad con ProcessAPI
func (pas *ProcessAPIStrategy) HealthCheck(ctx context.Context) error {
	// TODO: Implementar health check real
	return fmt.Errorf("ProcessAPI health check not implemented yet")
}
