package requester

import (
	"omniapi/internal/metrics"
)

// RequesterMetrics wrapper para instrumentar un Requester con métricas de Prometheus
type RequesterMetrics struct {
	requester Requester
	tenant    string
	site      string
	metric    string
	source    string
}

// NewRequesterWithMetrics crea un requester instrumentado con métricas de Prometheus
func NewRequesterWithMetrics(req Requester, tenant, site, metric, source string) *RequesterMetrics {
	// Sanitizar labels para evitar cardinalidad explosiva
	return &RequesterMetrics{
		requester: req,
		tenant:    metrics.SanitizeTenantID(tenant),
		site:      metrics.SanitizeSiteID(site),
		metric:    metrics.SanitizeMetric(metric),
		source:    source,
	}
}

// UpdateMetrics actualiza las métricas de Prometheus basándose en el estado actual
func (rm *RequesterMetrics) UpdateMetrics() {
	m := rm.requester.GetMetrics()

	// InFlight
	inFlightValue := 0.0
	if m.InFlight {
		inFlightValue = 1.0
	}
	metrics.RequesterInFlight.WithLabelValues(
		rm.tenant, rm.site, rm.metric, rm.source,
	).Set(inFlightValue)

	// LastLatencyMS
	metrics.RequesterLastLatencyMS.WithLabelValues(
		rm.tenant, rm.site, rm.metric, rm.source,
	).Set(float64(m.LastLatencyMS))

	// Circuit Breaker
	cbOpen := 0.0
	if m.CircuitOpen {
		cbOpen = 1.0
	}
	metrics.RequesterCircuitBreakerOpen.WithLabelValues(
		rm.tenant, rm.site, rm.metric, rm.source,
	).Set(cbOpen)

	// Queue Length
	metrics.RequesterQueueLength.WithLabelValues(
		rm.tenant, rm.site, rm.metric, rm.source,
	).Set(float64(m.QueueLength))

	// Note: Counters (success/error) se actualizan en el callback OnResult
	// para evitar problemas con counters que solo deben incrementarse
}

// RecordSuccess registra un éxito en Prometheus
func (rm *RequesterMetrics) RecordSuccess() {
	metrics.RequesterSuccessTotal.WithLabelValues(
		rm.tenant, rm.site, rm.metric, rm.source,
	).Inc()
}

// RecordError registra un error en Prometheus
func (rm *RequesterMetrics) RecordError(errorCode string) {
	code := metrics.SanitizeErrorCode(errorCode)
	metrics.RequesterErrorTotal.WithLabelValues(
		rm.tenant, rm.site, rm.metric, rm.source, code,
	).Inc()
}

// GetWrappedRequester retorna el requester subyacente
func (rm *RequesterMetrics) GetWrappedRequester() Requester {
	return rm.requester
}
