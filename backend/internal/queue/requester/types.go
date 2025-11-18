package requester

import (
	"context"
	"encoding/json"
	"time"
)

// Priority define la prioridad de una solicitud
type Priority string

const (
	PriorityHigh   Priority = "HIGH"
	PriorityNormal Priority = "NORMAL"
	PriorityLow    Priority = "LOW"
)

// Source define el origen de los datos
type Source string

const (
	SourceCloud      Source = "cloud"
	SourceProcessAPI Source = "processapi"
)

// TimeRange representa un rango de tiempo
type TimeRange struct {
	From time.Time `json:"from"`
	To   time.Time `json:"to"`
}

// Request representa una solicitud de datos
type Request struct {
	TenantID  string    `json:"tenant_id"`
	SiteID    string    `json:"site_id"`
	CageID    *string   `json:"cage_id,omitempty"`
	Metric    string    `json:"metric"` // e.g. feeding.appetite, feeding.feed_amount
	TimeRange TimeRange `json:"time_range"`
	Priority  Priority  `json:"priority"`
	Source    Source    `json:"source"` // cloud|processapi

	// Metadata interna
	EnqueuedAt time.Time `json:"enqueued_at"`
	RequestID  string    `json:"request_id"`
}

// Key genera una clave única para deduplicación y coalescing
func (r *Request) Key() string {
	cage := ""
	if r.CageID != nil {
		cage = *r.CageID
	}
	return r.TenantID + ":" + r.SiteID + ":" + cage + ":" + r.Metric + ":" + string(r.Source)
}

// Result representa el resultado de una solicitud
type Result struct {
	TenantID    string          `json:"tenant_id"`
	SiteID      string          `json:"site_id"`
	CageID      *string         `json:"cage_id,omitempty"`
	Metric      string          `json:"metric"`
	Source      Source          `json:"source"`
	LatencyMS   int64           `json:"latency_ms"`
	TsRange     TimeRange       `json:"ts_range"`
	Payload     json.RawMessage `json:"payload,omitempty"` // DATA canónica si llega algo
	Err         error           `json:"-"`
	ErrorMsg    string          `json:"error_msg,omitempty"`
	CompletedAt time.Time       `json:"completed_at"`
	RequestID   string          `json:"request_id"`
}

// IsSuccess indica si el resultado fue exitoso
func (r *Result) IsSuccess() bool {
	return r.Err == nil
}

// Strategy define la interfaz para ejecutar solicitudes concretas
type Strategy interface {
	// Execute ejecuta una solicitud y retorna el payload o error
	Execute(ctx context.Context, req Request) (json.RawMessage, error)

	// Name retorna el nombre de la estrategia
	Name() string

	// HealthCheck verifica si el servicio está disponible
	HealthCheck(ctx context.Context) error
}

// Requester es la interfaz principal de la cola de solicitudes
type Requester interface {
	// Enqueue agrega una solicitud a la cola
	Enqueue(req Request) error

	// Len retorna el número de solicitudes en cola
	Len() int

	// Start inicia el procesamiento con concurrencia=1
	Start(ctx context.Context) error

	// Stop detiene el procesamiento
	Stop() error

	// OnResult registra un callback para recibir resultados
	OnResult(callback func(Result))

	// GetMetrics retorna métricas actuales
	GetMetrics() Metrics

	// GetState retorna el estado actual del requester
	GetState() State
}

// State representa el estado actual del requester
type State string

const (
	StateIdle    State = "idle"
	StateRunning State = "running"
	StatePaused  State = "paused" // Circuit breaker activado
	StateStopped State = "stopped"
)

// Metrics representa métricas del requester
type Metrics struct {
	LastSuccessTS  time.Time  `json:"last_success_ts"`
	LastErrorTS    time.Time  `json:"last_error_ts"`
	LastLatencyMS  int64      `json:"last_latency_ms"`
	InFlight       bool       `json:"in_flight"`
	QueueLength    int        `json:"queue_length"`
	TotalProcessed int64      `json:"total_processed"`
	TotalErrors    int64      `json:"total_errors"`
	TotalSuccess   int64      `json:"total_success"`
	ConsecErrors   int        `json:"consecutive_errors"`
	AvgLatencyMS   float64    `json:"avg_latency_ms"`
	State          State      `json:"state"`
	CircuitOpen    bool       `json:"circuit_open"`
	NextRetryAt    *time.Time `json:"next_retry_at,omitempty"`
}

// Config representa la configuración del requester
type Config struct {
	// Timeouts
	RequestTimeout time.Duration `json:"request_timeout"`

	// Backoff exponencial
	BackoffInitial time.Duration `json:"backoff_initial"` // 1m
	BackoffStep2   time.Duration `json:"backoff_step2"`   // 2m
	BackoffStep3   time.Duration `json:"backoff_step3"`   // 5m

	// Circuit breaker
	MaxConsecutiveErrors int           `json:"max_consecutive_errors"` // N fallos para abrir
	CircuitPauseDuration time.Duration `json:"circuit_pause_duration"` // M minutos de pausa

	// Queue
	MaxQueueSize int `json:"max_queue_size"`

	// Coalescing
	CoalescingEnabled bool `json:"coalescing_enabled"`
}

// DefaultConfig retorna la configuración por defecto
func DefaultConfig() Config {
	return Config{
		RequestTimeout:       30 * time.Second,
		BackoffInitial:       1 * time.Minute,
		BackoffStep2:         2 * time.Minute,
		BackoffStep3:         5 * time.Minute,
		MaxConsecutiveErrors: 5,
		CircuitPauseDuration: 10 * time.Minute,
		MaxQueueSize:         1000,
		CoalescingEnabled:    true,
	}
}

// QueueStats representa estadísticas de la cola
type QueueStats struct {
	Size           int              `json:"size"`
	ByPriority     map[Priority]int `json:"by_priority"`
	OldestEnqueued *time.Time       `json:"oldest_enqueued,omitempty"`
	NewestEnqueued *time.Time       `json:"newest_enqueued,omitempty"`
}
