package status

import (
	"context"
	"sync"
	"time"

	"omniapi/internal/metrics"
)

// DefaultStatusPusher implementa StatusPusher
type DefaultStatusPusher struct {
	config   Config
	tracker  *StreamTracker
	callback func(Status)
	ctx      context.Context
	cancel   context.CancelFunc
	wg       sync.WaitGroup
	mu       sync.RWMutex
	started  bool
}

// NewStatusPusher crea un nuevo StatusPusher
func NewStatusPusher(config Config, tracker *StreamTracker) StatusPusher {
	return &DefaultStatusPusher{
		config:  config,
		tracker: tracker,
	}
}

// Start inicia la emisión periódica de heartbeats
func (sp *DefaultStatusPusher) Start(ctx context.Context) error {
	sp.mu.Lock()
	defer sp.mu.Unlock()

	if sp.started {
		return nil // Ya iniciado
	}

	sp.ctx, sp.cancel = context.WithCancel(ctx)
	sp.started = true

	sp.wg.Add(1)
	go sp.emitLoop()

	return nil
}

// Stop detiene la emisión
func (sp *DefaultStatusPusher) Stop() error {
	sp.mu.Lock()
	defer sp.mu.Unlock()

	if !sp.started {
		return nil
	}

	sp.cancel()
	sp.wg.Wait()
	sp.started = false

	return nil
}

// OnEmit registra un callback que se ejecuta cada vez que se emite un Status
func (sp *DefaultStatusPusher) OnEmit(callback func(Status)) {
	sp.mu.Lock()
	defer sp.mu.Unlock()

	sp.callback = callback
}

// GetCurrentStatus retorna el estado actual de todos los streams conocidos
func (sp *DefaultStatusPusher) GetCurrentStatus() []Status {
	streams := sp.tracker.GetAllStreams()
	kpis := sp.tracker.GetAllKPIs()

	statuses := make([]Status, 0, len(streams))
	now := time.Now()

	for k, streamKey := range streams {
		kpi := kpis[k]
		status := sp.buildStatus(streamKey, kpi, now)
		statuses = append(statuses, status)
	}

	return statuses
}

// emitLoop es el loop principal que emite heartbeats periódicamente
func (sp *DefaultStatusPusher) emitLoop() {
	defer sp.wg.Done()

	ticker := time.NewTicker(sp.config.HeartbeatInterval)
	defer ticker.Stop()

	// Emitir inmediatamente al inicio
	sp.emitHeartbeats()

	for {
		select {
		case <-sp.ctx.Done():
			return
		case <-ticker.C:
			sp.emitHeartbeats()
		}
	}
}

// emitHeartbeats emite heartbeats para todos los streams conocidos
func (sp *DefaultStatusPusher) emitHeartbeats() {
	statuses := sp.GetCurrentStatus()

	sp.mu.RLock()
	callback := sp.callback
	sp.mu.RUnlock()

	if callback != nil {
		for _, status := range statuses {
			// Actualizar métricas de Prometheus
			sp.updatePrometheusMetrics(status)

			// Ejecutar callback
			callback(status)
		}
	}
}

// updatePrometheusMetrics actualiza las métricas de Prometheus para un status
func (sp *DefaultStatusPusher) updatePrometheusMetrics(status Status) {
	// Sanitizar labels
	tenant := metrics.SanitizeTenantID(status.TenantID)
	site := metrics.SanitizeSiteID(status.SiteID)
	metric := metrics.SanitizeMetric(status.Metric)
	source := status.Source

	// Incrementar contador de status emitidos
	metrics.StatusEmittedTotal.WithLabelValues(
		tenant, site, metric, source, status.State,
	).Inc()

	// Actualizar staleness
	metrics.StatusStalenessSeconds.WithLabelValues(
		tenant, site, metric, source,
	).Set(float64(status.StalenessSec))

	// Actualizar última latencia si está disponible
	if status.LastLatencyMS != nil {
		metrics.StatusLastLatencyMS.WithLabelValues(
			tenant, site, metric, source,
		).Set(float64(*status.LastLatencyMS))
	}
}

// buildStatus construye un Status a partir de un StreamKey y sus KPIs
func (sp *DefaultStatusPusher) buildStatus(key StreamKey, kpi StreamKPIs, now time.Time) Status {
	status := Status{
		TenantID:      key.TenantID,
		SiteID:        key.SiteID,
		CageID:        key.CageID,
		Metric:        key.Metric,
		Source:        key.Source,
		LastSuccessTS: kpi.LastSuccessTS,
		LastErrorTS:   kpi.LastErrorTS,
		LastErrorMsg:  kpi.LastErrorMsg,
		LastLatencyMS: kpi.LastLatencyMS,
		InFlight:      kpi.InFlight,
		Notes:         kpi.Notes,
		EmittedAt:     now,
	}

	// Calcular staleness
	status.StalenessSec = sp.calculateStaleness(kpi.LastSuccessTS, now)

	// Determinar estado
	status.State = sp.determineState(kpi, status.StalenessSec)

	return status
}

// calculateStaleness calcula los segundos desde el último éxito
func (sp *DefaultStatusPusher) calculateStaleness(lastSuccess *time.Time, now time.Time) int64 {
	if lastSuccess == nil {
		return 0 // Nunca tuvo éxito
	}

	return int64(now.Sub(*lastSuccess).Seconds())
}

// determineState determina el estado del stream basado en los KPIs
func (sp *DefaultStatusPusher) determineState(kpi StreamKPIs, stalenessSec int64) string {
	// Si el circuit breaker está abierto, está pausado
	if kpi.CircuitBreakerOpen {
		return StatePaused.String()
	}

	// Si tiene muchos errores consecutivos, está failing
	if kpi.ConsecutiveErrors >= sp.config.MaxConsecutiveErrors {
		return StateFailing.String()
	}

	// Si nunca tuvo éxito y tiene errores, está failing
	if kpi.LastSuccessTS == nil && kpi.LastErrorTS != nil {
		return StateFailing.String()
	}

	// Si nunca tuvo éxito y no tiene errores, está en estado desconocido (partial)
	if kpi.LastSuccessTS == nil && kpi.LastErrorTS == nil {
		return StatePartial.String()
	}

	// Evaluar staleness
	if stalenessSec > sp.config.StaleThresholdDegraded {
		// Muy viejo, degraded
		return StateDegraded.String()
	}

	if stalenessSec > sp.config.StaleThresholdOK {
		// Algo viejo, pero no tanto
		if kpi.ConsecutiveErrors > 0 {
			return StateDegraded.String()
		}
		return StatePartial.String()
	}

	// Datos frescos
	if kpi.ConsecutiveErrors > 0 {
		return StatePartial.String()
	}

	return StateOK.String()
}

// GetTracker retorna el StreamTracker (útil para testing)
func (sp *DefaultStatusPusher) GetTracker() *StreamTracker {
	return sp.tracker
}
