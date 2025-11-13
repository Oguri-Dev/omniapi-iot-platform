package router

import (
	"time"

	"omniapi/internal/connectors"
	"omniapi/internal/domain"

	"go.mongodb.org/mongo-driver/bson/primitive"
)

// MessageType representa el tipo de mensaje
type MessageType string

const (
	MessageTypeData   MessageType = "DATA"   // Evento de datos
	MessageTypeStatus MessageType = "STATUS" // Heartbeat de estado
)

// SubscriptionFilter representa los criterios de filtrado para una suscripción
type SubscriptionFilter struct {
	// Filtros por StreamKey
	TenantID *primitive.ObjectID `json:"tenant_id,omitempty"`
	Kind     *domain.StreamKind  `json:"kind,omitempty"`
	FarmID   *string             `json:"farm_id,omitempty"`
	SiteID   *string             `json:"site_id,omitempty"`
	CageID   *string             `json:"cage_id,omitempty"`

	// Filtros adicionales
	Capabilities []domain.Capability `json:"capabilities,omitempty"`
	Sources      []string            `json:"sources,omitempty"`
	Tags         map[string]string   `json:"tags,omitempty"`
}

// Matches verifica si un CanonicalEvent coincide con este filtro
func (sf *SubscriptionFilter) Matches(event *connectors.CanonicalEvent) bool {
	streamKey := event.Envelope.Stream

	// Verificar TenantID
	if sf.TenantID != nil && *sf.TenantID != streamKey.TenantID {
		return false
	}

	// Verificar Kind
	if sf.Kind != nil && *sf.Kind != streamKey.Kind {
		return false
	}

	// Verificar FarmID
	if sf.FarmID != nil && *sf.FarmID != streamKey.FarmID {
		return false
	}

	// Verificar SiteID
	if sf.SiteID != nil && *sf.SiteID != streamKey.SiteID {
		return false
	}

	// Verificar CageID
	if sf.CageID != nil {
		if streamKey.CageID == nil {
			return false
		}
		if *sf.CageID != *streamKey.CageID {
			return false
		}
	}

	// Verificar Sources
	if len(sf.Sources) > 0 {
		found := false
		for _, source := range sf.Sources {
			if source == event.Envelope.Source {
				found = true
				break
			}
		}
		if !found {
			return false
		}
	}

	return true
}

// Subscription representa una suscripción activa de un cliente WebSocket
type Subscription struct {
	ID            string             `json:"id"`
	ClientID      string             `json:"client_id"`
	Filter        SubscriptionFilter `json:"filter"`
	IncludeStatus bool               `json:"include_status"` // Si true, recibe heartbeats de estado
	CreatedAt     time.Time          `json:"created_at"`
	LastEvent     *time.Time         `json:"last_event,omitempty"`
	EventCount    int64              `json:"event_count"`
}

// ThrottleConfig configura el comportamiento de throttle para un cliente
type ThrottleConfig struct {
	// ThrottleMs es el tiempo mínimo en ms entre eventos
	ThrottleMs int `json:"throttle_ms"`

	// MaxRate es el máximo número de eventos por segundo
	MaxRate float64 `json:"max_rate"`

	// BurstSize permite ráfagas de eventos hasta este límite
	BurstSize int `json:"burst_size"`

	// CoalescingEnabled activa el coalescing de eventos
	CoalescingEnabled bool `json:"coalescing_enabled"`

	// KeepLatest mantiene solo el último evento por stream ante backpressure
	KeepLatest bool `json:"keep_latest"`

	// BufferSize es el tamaño del buffer por stream
	BufferSize int `json:"buffer_size"`
}

// DefaultThrottleConfig retorna la configuración por defecto
func DefaultThrottleConfig() ThrottleConfig {
	return ThrottleConfig{
		ThrottleMs:        100,
		MaxRate:           10.0,
		BurstSize:         5,
		CoalescingEnabled: true,
		KeepLatest:        true,
		BufferSize:        100,
	}
}

// MultiConnectorPolicy define cómo manejar múltiples conectores para el mismo tipo de evento
type MultiConnectorPolicy string

const (
	// PolicyPriority usa el conector de mayor prioridad disponible
	PolicyPriority MultiConnectorPolicy = "priority"

	// PolicyFallback usa el primer conector disponible según orden de prioridad
	PolicyFallback MultiConnectorPolicy = "fallback"

	// PolicyMerge combina eventos de todos los conectores
	PolicyMerge MultiConnectorPolicy = "merge"

	// PolicyRoundRobin distribuye carga entre conectores
	PolicyRoundRobin MultiConnectorPolicy = "round_robin"
)

// ConnectorConfig configura un conector específico
type ConnectorConfig struct {
	ID          string `json:"id"`
	Type        string `json:"type"`
	Priority    int    `json:"priority"` // Mayor valor = mayor prioridad
	Weight      int    `json:"weight"`   // Para round-robin
	Enabled     bool   `json:"enabled"`
	Timeout     int    `json:"timeout_ms"`
	MaxRetries  int    `json:"max_retries"`
	CircuitOpen bool   `json:"circuit_open"` // Circuit breaker abierto
}

// MultiConnectorConfig configura políticas para múltiples conectores
type MultiConnectorConfig struct {
	TenantID   primitive.ObjectID     `json:"tenant_id"`
	Kind       domain.StreamKind      `json:"kind"`
	Policy     MultiConnectorPolicy   `json:"policy"`
	Connectors []ConnectorConfig      `json:"connectors"`
	Metadata   map[string]interface{} `json:"metadata,omitempty"`
}

// GetActiveConnectors retorna conectores activos ordenados por prioridad
func (mcc *MultiConnectorConfig) GetActiveConnectors() []ConnectorConfig {
	var active []ConnectorConfig
	for _, conn := range mcc.Connectors {
		if conn.Enabled && !conn.CircuitOpen {
			active = append(active, conn)
		}
	}

	// Ordenar por prioridad descendente
	for i := 0; i < len(active)-1; i++ {
		for j := i + 1; j < len(active); j++ {
			if active[j].Priority > active[i].Priority {
				active[i], active[j] = active[j], active[i]
			}
		}
	}

	return active
}

// ClientState mantiene el estado de un cliente WebSocket
type ClientState struct {
	ClientID       string                   `json:"client_id"`
	TenantID       primitive.ObjectID       `json:"tenant_id"`
	Subscriptions  []*Subscription          `json:"subscriptions"`
	ThrottleConfig ThrottleConfig           `json:"throttle_config"`
	Permissions    []domain.Capability      `json:"permissions"`
	Scopes         []domain.Scope           `json:"scopes"`
	LastEvent      time.Time                `json:"last_event"`
	Stats          ClientStats              `json:"stats"`
	StreamBuffers  map[string]*StreamBuffer `json:"-"` // Buffer por stream key
}

// ClientStats mantiene estadísticas por cliente
type ClientStats struct {
	EventsReceived int64     `json:"events_received"`
	EventsDropped  int64     `json:"events_dropped"`
	EventsSent     int64     `json:"events_sent"`
	Throttled      int64     `json:"throttled"`
	LastThrottle   time.Time `json:"last_throttle"`
	BytesSent      int64     `json:"bytes_sent"`
}

// StreamBuffer mantiene eventos pendientes por stream
type StreamBuffer struct {
	StreamKey  string                       `json:"stream_key"`
	Events     []*connectors.CanonicalEvent `json:"events"`
	MaxSize    int                          `json:"max_size"`
	KeepLatest bool                         `json:"keep_latest"`
}

// Push agrega un evento al buffer
func (sb *StreamBuffer) Push(event *connectors.CanonicalEvent) {
	if sb.KeepLatest && len(sb.Events) >= sb.MaxSize {
		// Reemplazar el último evento (keep-latest policy)
		sb.Events[len(sb.Events)-1] = event
	} else if len(sb.Events) < sb.MaxSize {
		sb.Events = append(sb.Events, event)
	}
	// Si está lleno y no es keep-latest, se descarta el evento
}

// Pop obtiene el siguiente evento del buffer
func (sb *StreamBuffer) Pop() *connectors.CanonicalEvent {
	if len(sb.Events) == 0 {
		return nil
	}
	event := sb.Events[0]
	sb.Events = sb.Events[1:]
	return event
}

// Len retorna el número de eventos en el buffer
func (sb *StreamBuffer) Len() int {
	return len(sb.Events)
}

// EventEnvelope encapsula un evento con su routing info
type EventEnvelope struct {
	Event       *connectors.CanonicalEvent `json:"event"`
	TargetCount int                        `json:"target_count"`
	Timestamp   time.Time                  `json:"timestamp"`
}

// RoutingDecision representa la decisión de routing para un evento
type RoutingDecision struct {
	Event       *connectors.CanonicalEvent `json:"event"`
	Clients     []string                   `json:"clients"`
	Reason      string                     `json:"reason,omitempty"`
	Timestamp   time.Time                  `json:"timestamp"`
	ProcessedIn time.Duration              `json:"processed_in"`
}

// RouterStats mantiene estadísticas del router
type RouterStats struct {
	EventsRouted        int64            `json:"events_routed"`
	EventsDropped       int64            `json:"events_dropped"`
	EventsDataOut       int64            `json:"events_data_out"`   // Eventos DATA enviados
	EventsStatusOut     int64            `json:"events_status_out"` // Eventos STATUS enviados
	ActiveClients       int              `json:"active_clients"`
	ActiveSubscriptions int              `json:"active_subscriptions"`
	TotalBytesRouted    int64            `json:"total_bytes_routed"`
	AvgRoutingTimeMs    float64          `json:"avg_routing_time_ms"`
	RouteP95Ms          float64          `json:"route_p95_ms"` // Percentil 95 de tiempo de routing
	EventsByKind        map[string]int64 `json:"events_by_kind"`
	ClientsByTenant     map[string]int   `json:"clients_by_tenant"`
	routingTimeSamples  []float64        // Buffer para calcular P95 (no exportado)
	maxSamples          int              // Límite de muestras (no exportado)
}

// RecordRoutingTime registra un tiempo de routing para métricas
func (rs *RouterStats) RecordRoutingTime(durationMs float64) {
	if rs.routingTimeSamples == nil {
		rs.routingTimeSamples = make([]float64, 0, 1000)
		rs.maxSamples = 1000
	}

	rs.routingTimeSamples = append(rs.routingTimeSamples, durationMs)

	// Limitar tamaño del buffer
	if len(rs.routingTimeSamples) > rs.maxSamples {
		rs.routingTimeSamples = rs.routingTimeSamples[len(rs.routingTimeSamples)-rs.maxSamples:]
	}

	// Recalcular promedio
	sum := 0.0
	for _, t := range rs.routingTimeSamples {
		sum += t
	}
	rs.AvgRoutingTimeMs = sum / float64(len(rs.routingTimeSamples))

	// Calcular P95
	rs.RouteP95Ms = rs.calculateP95()
}

// calculateP95 calcula el percentil 95
func (rs *RouterStats) calculateP95() float64 {
	if len(rs.routingTimeSamples) == 0 {
		return 0
	}

	// Copiar y ordenar
	samples := make([]float64, len(rs.routingTimeSamples))
	copy(samples, rs.routingTimeSamples)

	// Bubble sort simple (suficiente para 1000 elementos)
	for i := 0; i < len(samples)-1; i++ {
		for j := i + 1; j < len(samples); j++ {
			if samples[j] < samples[i] {
				samples[i], samples[j] = samples[j], samples[i]
			}
		}
	}

	// P95 es el elemento en la posición 95% del array
	idx := int(float64(len(samples)) * 0.95)
	if idx >= len(samples) {
		idx = len(samples) - 1
	}

	return samples[idx]
}

// NewStreamBuffer crea un nuevo buffer de stream
func NewStreamBuffer(streamKey string, maxSize int, keepLatest bool) *StreamBuffer {
	return &StreamBuffer{
		StreamKey:  streamKey,
		Events:     make([]*connectors.CanonicalEvent, 0, maxSize),
		MaxSize:    maxSize,
		KeepLatest: keepLatest,
	}
}

// NewClientState crea un nuevo estado de cliente
func NewClientState(clientID string, tenantID primitive.ObjectID) *ClientState {
	return &ClientState{
		ClientID:       clientID,
		TenantID:       tenantID,
		Subscriptions:  make([]*Subscription, 0),
		ThrottleConfig: DefaultThrottleConfig(),
		Permissions:    make([]domain.Capability, 0),
		Scopes:         make([]domain.Scope, 0),
		StreamBuffers:  make(map[string]*StreamBuffer),
		Stats:          ClientStats{},
	}
}
