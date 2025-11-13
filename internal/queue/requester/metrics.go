package requester

import (
	"sync"
	"time"
)

// MetricsCollector recopila métricas del requester
type MetricsCollector struct {
	lastSuccessTS  time.Time
	lastErrorTS    time.Time
	lastLatencyMS  int64
	inFlight       bool
	totalProcessed int64
	totalErrors    int64
	totalSuccess   int64
	latencies      []int64 // Para calcular promedio
	maxLatencies   int     // Máximo de latencias a mantener
	mu             sync.RWMutex
}

// NewMetricsCollector crea un nuevo colector de métricas
func NewMetricsCollector() *MetricsCollector {
	return &MetricsCollector{
		latencies:    make([]int64, 0, 100),
		maxLatencies: 100,
	}
}

// RecordStart marca el inicio de una solicitud
func (mc *MetricsCollector) RecordStart() {
	mc.mu.Lock()
	defer mc.mu.Unlock()
	mc.inFlight = true
}

// RecordSuccess registra un éxito con su latencia
func (mc *MetricsCollector) RecordSuccess(latencyMS int64) {
	mc.mu.Lock()
	defer mc.mu.Unlock()

	mc.inFlight = false
	mc.lastSuccessTS = time.Now()
	mc.lastLatencyMS = latencyMS
	mc.totalProcessed++
	mc.totalSuccess++

	// Agregar latencia para promedio
	mc.latencies = append(mc.latencies, latencyMS)
	if len(mc.latencies) > mc.maxLatencies {
		mc.latencies = mc.latencies[1:]
	}
}

// RecordError registra un error
func (mc *MetricsCollector) RecordError() {
	mc.mu.Lock()
	defer mc.mu.Unlock()

	mc.inFlight = false
	mc.lastErrorTS = time.Now()
	mc.totalProcessed++
	mc.totalErrors++
}

// RecordEnd marca el fin de una solicitud (sin éxito ni error)
func (mc *MetricsCollector) RecordEnd() {
	mc.mu.Lock()
	defer mc.mu.Unlock()
	mc.inFlight = false
}

// GetMetrics retorna las métricas actuales
func (mc *MetricsCollector) GetMetrics() MetricsSnapshot {
	mc.mu.RLock()
	defer mc.mu.RUnlock()

	return MetricsSnapshot{
		LastSuccessTS:  mc.lastSuccessTS,
		LastErrorTS:    mc.lastErrorTS,
		LastLatencyMS:  mc.lastLatencyMS,
		InFlight:       mc.inFlight,
		TotalProcessed: mc.totalProcessed,
		TotalErrors:    mc.totalErrors,
		TotalSuccess:   mc.totalSuccess,
		AvgLatencyMS:   mc.calculateAvgLatency(),
	}
}

// calculateAvgLatency calcula la latencia promedio
func (mc *MetricsCollector) calculateAvgLatency() float64 {
	if len(mc.latencies) == 0 {
		return 0
	}

	var sum int64
	for _, lat := range mc.latencies {
		sum += lat
	}

	return float64(sum) / float64(len(mc.latencies))
}

// IsInFlight indica si hay una solicitud en curso
func (mc *MetricsCollector) IsInFlight() bool {
	mc.mu.RLock()
	defer mc.mu.RUnlock()
	return mc.inFlight
}

// Reset reinicia las métricas
func (mc *MetricsCollector) Reset() {
	mc.mu.Lock()
	defer mc.mu.Unlock()

	mc.lastSuccessTS = time.Time{}
	mc.lastErrorTS = time.Time{}
	mc.lastLatencyMS = 0
	mc.inFlight = false
	mc.totalProcessed = 0
	mc.totalErrors = 0
	mc.totalSuccess = 0
	mc.latencies = make([]int64, 0, mc.maxLatencies)
}

// MetricsSnapshot es una instantánea de las métricas
type MetricsSnapshot struct {
	LastSuccessTS  time.Time
	LastErrorTS    time.Time
	LastLatencyMS  int64
	InFlight       bool
	TotalProcessed int64
	TotalErrors    int64
	TotalSuccess   int64
	AvgLatencyMS   float64
}

// GetSuccessRate retorna la tasa de éxito
func (ms *MetricsSnapshot) GetSuccessRate() float64 {
	if ms.TotalProcessed == 0 {
		return 0
	}
	return float64(ms.TotalSuccess) / float64(ms.TotalProcessed) * 100
}

// GetErrorRate retorna la tasa de error
func (ms *MetricsSnapshot) GetErrorRate() float64 {
	if ms.TotalProcessed == 0 {
		return 0
	}
	return float64(ms.TotalErrors) / float64(ms.TotalProcessed) * 100
}
