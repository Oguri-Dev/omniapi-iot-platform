package metrics

import (
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
)

// ═══════════════════════════════════════════════════════════
// Métricas de Requester (internal/queue/requester)
// ═══════════════════════════════════════════════════════════

var (
	// RequesterInFlight indica si hay un request en progreso
	// Labels: tenant, site, metric, source
	RequesterInFlight = promauto.NewGaugeVec(
		prometheus.GaugeOpts{
			Name: "omniapi_requester_in_flight",
			Help: "Indica si hay un request en progreso (0=idle, 1=in_flight)",
		},
		[]string{"tenant", "site", "metric", "source"},
	)

	// RequesterLastLatencyMS última latencia registrada en milisegundos
	// Labels: tenant, site, metric, source
	RequesterLastLatencyMS = promauto.NewGaugeVec(
		prometheus.GaugeOpts{
			Name: "omniapi_requester_last_latency_ms",
			Help: "Última latencia de request en milisegundos",
		},
		[]string{"tenant", "site", "metric", "source"},
	)

	// RequesterSuccessTotal contador de requests exitosos
	// Labels: tenant, site, metric, source
	RequesterSuccessTotal = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "omniapi_requester_success_total",
			Help: "Total de requests exitosos",
		},
		[]string{"tenant", "site", "metric", "source"},
	)

	// RequesterErrorTotal contador de requests fallidos
	// Labels: tenant, site, metric, source, code (error code o "unknown")
	RequesterErrorTotal = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "omniapi_requester_error_total",
			Help: "Total de requests fallidos",
		},
		[]string{"tenant", "site", "metric", "source", "code"},
	)

	// RequesterCircuitBreakerOpen indica si el circuit breaker está abierto
	// Labels: tenant, site, metric, source
	RequesterCircuitBreakerOpen = promauto.NewGaugeVec(
		prometheus.GaugeOpts{
			Name: "omniapi_requester_cb_open",
			Help: "Circuit breaker abierto (0=closed, 1=open)",
		},
		[]string{"tenant", "site", "metric", "source"},
	)

	// RequesterQueueLength longitud actual de la cola
	// Labels: tenant, site, metric, source
	RequesterQueueLength = promauto.NewGaugeVec(
		prometheus.GaugeOpts{
			Name: "omniapi_requester_queue_length",
			Help: "Número de requests en cola",
		},
		[]string{"tenant", "site", "metric", "source"},
	)
)

// ═══════════════════════════════════════════════════════════
// Métricas de Status (internal/queue/status)
// ═══════════════════════════════════════════════════════════

var (
	// StatusEmittedTotal contador de heartbeats de status emitidos
	// Labels: tenant, site, metric, source, state (ok|partial|degraded|failing|paused)
	StatusEmittedTotal = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "omniapi_status_emitted_total",
			Help: "Total de status heartbeats emitidos",
		},
		[]string{"tenant", "site", "metric", "source", "state"},
	)

	// StatusStalenessSeconds segundos desde el último éxito
	// Labels: tenant, site, metric, source
	StatusStalenessSeconds = promauto.NewGaugeVec(
		prometheus.GaugeOpts{
			Name: "omniapi_status_staleness_seconds",
			Help: "Segundos desde el último request exitoso",
		},
		[]string{"tenant", "site", "metric", "source"},
	)

	// StatusLastLatencyMS última latencia del stream
	// Labels: tenant, site, metric, source
	StatusLastLatencyMS = promauto.NewGaugeVec(
		prometheus.GaugeOpts{
			Name: "omniapi_status_last_latency_ms",
			Help: "Última latencia registrada del stream en milisegundos",
		},
		[]string{"tenant", "site", "metric", "source"},
	)
)

// ═══════════════════════════════════════════════════════════
// Métricas de Router (internal/router)
// ═══════════════════════════════════════════════════════════

var (
	// EventsDataOutTotal eventos DATA enviados a clientes
	// Labels: tenant, site, metric
	EventsDataOutTotal = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "omniapi_events_data_out_total",
			Help: "Total de eventos DATA enviados a clientes WebSocket",
		},
		[]string{"tenant", "site", "metric"},
	)

	// EventsStatusOutTotal eventos STATUS enviados a clientes
	// Labels: tenant, site, metric
	EventsStatusOutTotal = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "omniapi_events_status_out_total",
			Help: "Total de eventos STATUS enviados a clientes WebSocket",
		},
		[]string{"tenant", "site", "metric"},
	)

	// EventsDataInTotal eventos DATA recibidos del requester
	// Labels: tenant, site, metric
	EventsDataInTotal = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "omniapi_events_data_in_total",
			Help: "Total de eventos DATA recibidos de requesters",
		},
		[]string{"tenant", "site", "metric"},
	)

	// EventsDroppedTotal eventos descartados por buffers llenos
	EventsDroppedTotal = promauto.NewCounter(
		prometheus.CounterOpts{
			Name: "omniapi_events_dropped_total",
			Help: "Total de eventos descartados por buffers llenos",
		},
	)

	// RouterSubscriptionsActive suscripciones activas
	// Labels: tenant
	RouterSubscriptionsActive = promauto.NewGaugeVec(
		prometheus.GaugeOpts{
			Name: "omniapi_router_subscriptions_active",
			Help: "Número de suscripciones activas en el router",
		},
		[]string{"tenant"},
	)
)

// ═══════════════════════════════════════════════════════════
// Métricas de WebSocket (websocket/)
// ═══════════════════════════════════════════════════════════

var (
	// WSConnectionsActive conexiones WebSocket activas
	WSConnectionsActive = promauto.NewGauge(
		prometheus.GaugeOpts{
			Name: "omniapi_ws_connections_active",
			Help: "Número de conexiones WebSocket activas",
		},
	)

	// WSConnectionsTotal total de conexiones establecidas
	WSConnectionsTotal = promauto.NewCounter(
		prometheus.CounterOpts{
			Name: "omniapi_ws_connections_total",
			Help: "Total de conexiones WebSocket establecidas",
		},
	)

	// WSMessagesInTotal mensajes recibidos de clientes
	// Labels: type (SUB|UNSUB|PING)
	WSMessagesInTotal = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "omniapi_ws_messages_in_total",
			Help: "Total de mensajes recibidos de clientes WebSocket",
		},
		[]string{"type"},
	)

	// WSMessagesOutTotal mensajes enviados a clientes
	// Labels: type (ACK|ERROR|PONG|DATA|STATUS)
	WSMessagesOutTotal = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "omniapi_ws_messages_out_total",
			Help: "Total de mensajes enviados a clientes WebSocket",
		},
		[]string{"type"},
	)

	// WSDeliveryLatencyMS latencia de entrega de eventos (histograma)
	WSDeliveryLatencyMS = promauto.NewHistogram(
		prometheus.HistogramOpts{
			Name:    "omniapi_ws_delivery_latency_ms",
			Help:    "Latencia de entrega de eventos WebSocket en milisegundos",
			Buckets: []float64{1, 5, 10, 25, 50, 100, 250, 500, 1000, 2500, 5000},
		},
	)

	// WSEventBackpressureTotal eventos descartados por backpressure
	// Labels: type (DATA|STATUS)
	WSEventBackpressureTotal = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "omniapi_ws_event_backpressure_total",
			Help: "Total de eventos descartados por backpressure en WebSocket",
		},
		[]string{"type"},
	)

	// WSSubscriptionsActive suscripciones activas por cliente
	WSSubscriptionsActive = promauto.NewGauge(
		prometheus.GaugeOpts{
			Name: "omniapi_ws_subscriptions_active",
			Help: "Número total de suscripciones activas en WebSocket",
		},
	)
)

// ═══════════════════════════════════════════════════════════
// Helpers para evitar cardinalidad explosiva
// ═══════════════════════════════════════════════════════════

// SanitizeTenantID limita el tenant ID a valores conocidos
// Para evitar cardinalidad explosiva en caso de tenant_ids arbitrarios
func SanitizeTenantID(tenantID string) string {
	if tenantID == "" {
		return "unknown"
	}
	// Truncar si es muy largo (ej. más de 24 caracteres)
	if len(tenantID) > 24 {
		return tenantID[:24]
	}
	return tenantID
}

// SanitizeMetric mapea métricas a categorías conocidas
func SanitizeMetric(metric string) string {
	// Extraer prefijo (feeding, biometric, climate, etc.)
	for _, prefix := range []string{"feeding", "biometric", "climate", "water", "ops", "status"} {
		if len(metric) >= len(prefix) && metric[:len(prefix)] == prefix {
			return prefix
		}
	}
	return "other"
}

// SanitizeSiteID trunca site IDs muy largos
func SanitizeSiteID(siteID string) string {
	if siteID == "" {
		return "unknown"
	}
	if len(siteID) > 32 {
		return siteID[:32]
	}
	return siteID
}

// SanitizeErrorCode mapea códigos de error a categorías
func SanitizeErrorCode(code string) string {
	if code == "" {
		return "unknown"
	}
	// Limitar a códigos HTTP estándar o categorías
	switch {
	case code == "timeout":
		return "timeout"
	case code == "connection_refused":
		return "connection_refused"
	case code == "400", code == "401", code == "403", code == "404":
		return "client_error"
	case code == "500", code == "502", code == "503", code == "504":
		return "server_error"
	default:
		return "other"
	}
}
