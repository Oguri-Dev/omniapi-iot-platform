package status

import (
	"context"
	"time"
)

// Status representa el estado de un stream de datos en un momento dado
type Status struct {
	TenantID string
	SiteID   string
	CageID   *string // Opcional

	Metric string
	Source string // "cloud"|"processapi"|"derived"

	// KPIs de salud
	LastSuccessTS *time.Time // Último éxito
	LastErrorTS   *time.Time // Último error
	LastErrorMsg  *string    // Mensaje del último error
	LastLatencyMS *int64     // Latencia del último request (ms)

	InFlight bool // Request en progreso

	// Métricas calculadas
	StalenessSec int64  // Segundos desde last_success (0 si nunca tuvo éxito)
	State        string // "ok"|"partial"|"degraded"|"failing"|"paused"
	Notes        *string

	EmittedAt time.Time // Timestamp de emisión del heartbeat
}

// StreamState representa los posibles estados de un stream
type StreamState string

const (
	StateOK       StreamState = "ok"       // Funcionando correctamente
	StatePartial  StreamState = "partial"  // Algunos errores, pero funcionando
	StateDegraded StreamState = "degraded" // Errores frecuentes
	StateFailing  StreamState = "failing"  // Fallas consecutivas
	StatePaused   StreamState = "paused"   // Circuit breaker abierto
)

// String convierte StreamState a string
func (s StreamState) String() string {
	return string(s)
}

// StatusPusher emite heartbeats de estado periódicamente
type StatusPusher interface {
	// Start inicia la emisión periódica de heartbeats
	Start(ctx context.Context) error

	// Stop detiene la emisión
	Stop() error

	// OnEmit registra un callback que se ejecuta cada vez que se emite un Status
	OnEmit(callback func(Status))

	// GetCurrentStatus retorna el estado actual de todos los streams conocidos
	GetCurrentStatus() []Status
}

// Config contiene la configuración del StatusPusher
type Config struct {
	// HeartbeatInterval es la frecuencia de emisión de heartbeats
	HeartbeatInterval time.Duration

	// StaleThresholdOK es el umbral de staleness para considerar el stream OK (segundos)
	StaleThresholdOK int64

	// StaleThresholdDegraded es el umbral para considerar el stream degraded (segundos)
	StaleThresholdDegraded int64

	// MaxConsecutiveErrors es el número de errores consecutivos antes de marcar como failing
	MaxConsecutiveErrors int
}

// DefaultConfig retorna la configuración por defecto
func DefaultConfig() Config {
	return Config{
		HeartbeatInterval:      10 * time.Second, // Heartbeat cada 10s
		StaleThresholdOK:       60,               // <1min es OK
		StaleThresholdDegraded: 300,              // 1-5min es degraded
		MaxConsecutiveErrors:   3,                // 3 errores consecutivos = failing
	}
}

// StreamKey identifica únicamente un stream
type StreamKey struct {
	TenantID string
	SiteID   string
	CageID   *string
	Metric   string
	Source   string
}

// Key genera una clave única para el stream
func (sk StreamKey) Key() string {
	cageID := ""
	if sk.CageID != nil {
		cageID = *sk.CageID
	}
	return sk.TenantID + ":" + sk.SiteID + ":" + cageID + ":" + sk.Metric + ":" + sk.Source
}

// StreamKPIs almacena los KPIs de un stream
type StreamKPIs struct {
	LastSuccessTS        *time.Time
	LastErrorTS          *time.Time
	LastErrorMsg         *string
	LastLatencyMS        *int64
	InFlight             bool
	ConsecutiveErrors    int
	ConsecutiveSuccesses int
	CircuitBreakerOpen   bool
	Notes                *string
}

// Clone crea una copia profunda de los KPIs
func (kpi *StreamKPIs) Clone() StreamKPIs {
	clone := StreamKPIs{
		InFlight:             kpi.InFlight,
		ConsecutiveErrors:    kpi.ConsecutiveErrors,
		ConsecutiveSuccesses: kpi.ConsecutiveSuccesses,
		CircuitBreakerOpen:   kpi.CircuitBreakerOpen,
	}

	if kpi.LastSuccessTS != nil {
		ts := *kpi.LastSuccessTS
		clone.LastSuccessTS = &ts
	}

	if kpi.LastErrorTS != nil {
		ts := *kpi.LastErrorTS
		clone.LastErrorTS = &ts
	}

	if kpi.LastErrorMsg != nil {
		msg := *kpi.LastErrorMsg
		clone.LastErrorMsg = &msg
	}

	if kpi.LastLatencyMS != nil {
		lat := *kpi.LastLatencyMS
		clone.LastLatencyMS = &lat
	}

	if kpi.Notes != nil {
		note := *kpi.Notes
		clone.Notes = &note
	}

	return clone
}
