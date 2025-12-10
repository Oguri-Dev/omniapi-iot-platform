package polling

import (
	"time"

	"go.mongodb.org/mongo-driver/bson/primitive"
)

// EndpointInstance representa una instancia de un endpoint configurado para polling
// Permite múltiples instancias del mismo endpoint con diferentes parámetros
type EndpointInstance struct {
	InstanceID  string            `bson:"instance_id" json:"instance_id"`                     // ID único de la instancia
	EndpointID  string            `bson:"endpoint_id" json:"endpoint_id"`                     // Referencia al catálogo de endpoints
	Label       string            `bson:"label" json:"label"`                                 // Nombre descriptivo (ej: "Oxígeno Jaula 1")
	Method      string            `bson:"method" json:"method"`                               // GET, POST
	Path        string            `bson:"path" json:"path"`                                   // Path del endpoint con placeholders
	TargetBlock string            `bson:"target_block" json:"target_block"`                   // snapshots, timeseries, kpis, assets
	Params      map[string]string `bson:"params,omitempty" json:"params,omitempty"`           // Parámetros (monitor_id, sensor_id, etc.)
	Enabled     bool              `bson:"enabled" json:"enabled"`                             // Si está activo para polling
	IntervalMS  int64             `bson:"interval_ms,omitempty" json:"interval_ms,omitempty"` // Intervalo en ms (0 = usar global, min: 1000ms)
}

// PollingConfig configuración de polling para un site/provider
type PollingConfig struct {
	ID         primitive.ObjectID `bson:"_id,omitempty" json:"id,omitempty"`
	Provider   string             `bson:"provider" json:"provider"`                 // innovex, scaleaq
	SiteID     string             `bson:"site_id" json:"site_id"`                   // ID del site
	SiteCode   string             `bson:"site_code" json:"site_code"`               // Código del site
	SiteName   string             `bson:"site_name" json:"site_name"`               // Nombre del site
	TenantID   string             `bson:"tenant_id" json:"tenant_id"`               // ID del tenant
	TenantCode string             `bson:"tenant_code" json:"tenant_code"`           // Código del tenant
	ServiceID  string             `bson:"service_id" json:"service_id"`             // ID del ExternalService para auth
	Endpoints  []EndpointInstance `bson:"endpoints" json:"endpoints"`               // Instancias de endpoints
	IntervalMS int64              `bson:"interval_ms" json:"interval_ms"`           // Intervalo de polling en ms (default: 2000)
	AutoStart  bool               `bson:"auto_start" json:"auto_start"`             // Si debe iniciar automáticamente al arrancar el servidor
	Status     string             `bson:"status" json:"status"`                     // active, paused, stopped
	Output     *OutputConfig      `bson:"output,omitempty" json:"output,omitempty"` // Configuración de salida MQTT
	CreatedAt  time.Time          `bson:"created_at" json:"created_at"`
	UpdatedAt  time.Time          `bson:"updated_at" json:"updated_at"`
	CreatedBy  string             `bson:"created_by,omitempty" json:"created_by,omitempty"`
}

// PollingResult resultado de una ejecución de polling
type PollingResult struct {
	InstanceID   string            `json:"instance_id"`
	EndpointID   string            `json:"endpoint_id"`
	Label        string            `json:"label"`
	Provider     string            `json:"provider"`
	SiteID       string            `json:"site_id"`
	TenantID     string            `json:"tenant_id"`
	Path         string            `json:"path"`
	FullURL      string            `json:"full_url"`
	Method       string            `json:"method"`
	Params       map[string]string `json:"params,omitempty"`
	StatusCode   int               `json:"status_code"`
	Success      bool              `json:"success"`
	Data         interface{}       `json:"data,omitempty"`
	Error        string            `json:"error,omitempty"`
	LatencyMS    int64             `json:"latency_ms"`
	PolledAt     time.Time         `json:"polled_at"`
	ResponseSize int               `json:"response_size"`
}

// WorkerStatus estado de un worker de polling
type WorkerStatus struct {
	InstanceID      string    `json:"instance_id"`
	EndpointID      string    `json:"endpoint_id"`
	Label           string    `json:"label"`
	Status          string    `json:"status"`      // running, paused, error, stopped
	IntervalMS      int64     `json:"interval_ms"` // Intervalo efectivo de este worker
	LastPollAt      time.Time `json:"last_poll_at,omitempty"`
	LastSuccessAt   time.Time `json:"last_success_at,omitempty"`
	LastErrorAt     time.Time `json:"last_error_at,omitempty"`
	LastError       string    `json:"last_error,omitempty"`
	TotalPolls      int64     `json:"total_polls"`
	TotalSuccess    int64     `json:"total_success"`
	TotalErrors     int64     `json:"total_errors"`
	AvgLatencyMS    float64   `json:"avg_latency_ms"`
	ConsecutiveErrs int       `json:"consecutive_errors"`
}

// EngineStatus estado general del polling engine
type EngineStatus struct {
	Status        string                  `json:"status"` // running, stopped
	ActiveWorkers int                     `json:"active_workers"`
	TotalConfigs  int                     `json:"total_configs"`
	Workers       map[string]WorkerStatus `json:"workers"` // Key: configID:instanceID
	StartedAt     time.Time               `json:"started_at,omitempty"`
}

// StartPollingRequest request para iniciar polling
type StartPollingRequest struct {
	Provider   string             `json:"provider"`
	SiteID     string             `json:"site_id"`
	SiteCode   string             `json:"site_code"`
	SiteName   string             `json:"site_name"`
	TenantID   string             `json:"tenant_id"`
	TenantCode string             `json:"tenant_code"`
	ServiceID  string             `json:"service_id"` // ID del ExternalService
	Endpoints  []EndpointInstance `json:"endpoints"`
	IntervalMS int64              `json:"interval_ms,omitempty"` // Default: 2000
	AutoStart  bool               `json:"auto_start,omitempty"`  // Si true, inicia polling automáticamente al guardar
	Output     *OutputConfig      `json:"output,omitempty"`      // Configuración de salida MQTT
}

// OutputConfig configuración de salida para los datos polleados
type OutputConfig struct {
	BrokerID      string `bson:"broker_id,omitempty" json:"broker_id,omitempty"`           // ID del broker a usar
	TopicTemplate string `bson:"topic_template,omitempty" json:"topic_template,omitempty"` // Template del topic
	Enabled       bool   `bson:"enabled" json:"enabled"`                                   // Si está habilitado el envío
}

// StopPollingRequest request para detener polling
type StopPollingRequest struct {
	ConfigID   string `json:"config_id,omitempty"`   // Detener config específica
	SiteID     string `json:"site_id,omitempty"`     // Detener todas las de un site
	Provider   string `json:"provider,omitempty"`    // Filtrar por provider
	InstanceID string `json:"instance_id,omitempty"` // Detener instancia específica
}
