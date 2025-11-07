package connectors

import (
	"context"
	"encoding/json"
	"time"

	"omniapi/internal/domain"
)

// HealthStatus representa el estado de salud de un conector
type HealthStatus string

const (
	HealthStatusHealthy   HealthStatus = "healthy"
	HealthStatusDegraded  HealthStatus = "degraded"
	HealthStatusUnhealthy HealthStatus = "unhealthy"
	HealthStatusUnknown   HealthStatus = "unknown"
)

// HealthInfo contiene información del estado de salud
type HealthInfo struct {
	Status     HealthStatus           `json:"status"`
	Message    string                 `json:"message,omitempty"`
	LastCheck  time.Time              `json:"last_check"`
	Metrics    map[string]interface{} `json:"metrics,omitempty"`
	ErrorCount int                    `json:"error_count"`
	Uptime     time.Duration          `json:"uptime"`
}

// EventFilter representa filtros para suscripciones a eventos
type EventFilter struct {
	StreamKey    *domain.StreamKey   `json:"stream_key,omitempty"`
	Capabilities []domain.Capability `json:"capabilities,omitempty"`
	Sources      []string            `json:"sources,omitempty"`
	Tags         map[string]string   `json:"tags,omitempty"`
}

// EventFlags representa flags especiales para eventos
type EventFlags int

const (
	EventFlagNone      EventFlags = 0
	EventFlagRetry     EventFlags = 1 << iota // Evento en retry
	EventFlagDuplicate                        // Posible duplicado
	EventFlagLate                             // Evento tardío
	EventFlagSynthetic                        // Evento sintético/calculado
)

// Envelope contiene metadatos del evento
type Envelope struct {
	Version       string           `json:"version"`                  // Versión del envelope
	Timestamp     time.Time        `json:"timestamp"`                // Timestamp del evento
	Stream        domain.StreamKey `json:"stream"`                   // StreamKey asociada
	Source        string           `json:"source"`                   // Identificador del origen
	Sequence      uint64           `json:"sequence"`                 // Número de secuencia
	Flags         EventFlags       `json:"flags"`                    // Flags especiales
	TraceID       string           `json:"trace_id,omitempty"`       // ID de trazabilidad
	CorrelationID string           `json:"correlation_id,omitempty"` // ID de correlación
}

// CanonicalEvent representa un evento en formato canónico
type CanonicalEvent struct {
	Envelope      Envelope        `json:"envelope"`
	Payload       json.RawMessage `json:"payload"`
	Kind          string          `json:"kind"`           // Tipo de evento (feeding, biometric, etc.)
	SchemaVersion string          `json:"schema_version"` // Versión del schema utilizado
}

// Connector define la interfaz que deben implementar todos los conectores
type Connector interface {
	// Start inicia el conector
	Start(ctx context.Context) error

	// Stop detiene el conector
	Stop() error

	// Capabilities retorna las capabilities soportadas por este conector
	Capabilities() []domain.Capability

	// Subscribe configura filtros para eventos
	Subscribe(filters ...EventFilter) error

	// OnEvent configura el canal donde se enviarán los eventos
	OnEvent(eventChan chan<- CanonicalEvent)

	// Health retorna información del estado de salud
	Health() HealthInfo

	// ID retorna el ID único de esta instancia del conector
	ID() string

	// Type retorna el tipo de conector
	Type() string

	// Config retorna la configuración actual
	Config() map[string]interface{}
}

// ConnectorFactory es una función que crea una instancia de conector
type ConnectorFactory func(config map[string]interface{}) (Connector, error)

// ConnectorRegistration contiene información de registro de un conector
type ConnectorRegistration struct {
	Type         string                 `json:"type"`
	Version      string                 `json:"version"`
	Factory      ConnectorFactory       `json:"-"`
	Capabilities []domain.Capability    `json:"capabilities"`
	ConfigSchema map[string]interface{} `json:"config_schema,omitempty"`
	Description  string                 `json:"description,omitempty"`
}

// EventSubscription representa una suscripción activa
type EventSubscription struct {
	ID      string        `json:"id"`
	Filters []EventFilter `json:"filters"`
	Active  bool          `json:"active"`
	Created time.Time     `json:"created"`
}
