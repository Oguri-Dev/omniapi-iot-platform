package status

import (
	"sync"
	"time"
)

// StreamTracker mantiene el estado de todos los streams conocidos
type StreamTracker struct {
	streams map[string]*StreamKPIs // key -> KPIs
	keys    map[string]StreamKey   // key -> StreamKey
	mu      sync.RWMutex
}

// NewStreamTracker crea un nuevo tracker de streams
func NewStreamTracker() *StreamTracker {
	return &StreamTracker{
		streams: make(map[string]*StreamKPIs),
		keys:    make(map[string]StreamKey),
	}
}

// RegisterStream registra un nuevo stream
func (st *StreamTracker) RegisterStream(key StreamKey) {
	st.mu.Lock()
	defer st.mu.Unlock()

	k := key.Key()
	if _, exists := st.streams[k]; !exists {
		st.streams[k] = &StreamKPIs{}
		st.keys[k] = key
	}
}

// UpdateSuccess actualiza el stream con un resultado exitoso
func (st *StreamTracker) UpdateSuccess(key StreamKey, latencyMS int64) {
	st.mu.Lock()
	defer st.mu.Unlock()

	k := key.Key()

	// Registrar si no existe
	if _, exists := st.streams[k]; !exists {
		st.streams[k] = &StreamKPIs{}
		st.keys[k] = key
	}

	kpi := st.streams[k]
	now := time.Now()
	kpi.LastSuccessTS = &now
	kpi.LastLatencyMS = &latencyMS
	kpi.ConsecutiveErrors = 0
	kpi.ConsecutiveSuccesses++
	kpi.InFlight = false
	kpi.CircuitBreakerOpen = false
	kpi.LastErrorMsg = nil // Limpiar error anterior
}

// UpdateError actualiza el stream con un error
func (st *StreamTracker) UpdateError(key StreamKey, errorMsg string) {
	st.mu.Lock()
	defer st.mu.Unlock()

	k := key.Key()

	// Registrar si no existe
	if _, exists := st.streams[k]; !exists {
		st.streams[k] = &StreamKPIs{}
		st.keys[k] = key
	}

	kpi := st.streams[k]
	now := time.Now()
	kpi.LastErrorTS = &now
	kpi.LastErrorMsg = &errorMsg
	kpi.ConsecutiveErrors++
	kpi.ConsecutiveSuccesses = 0
	kpi.InFlight = false
}

// MarkInFlight marca un stream como procesando un request
func (st *StreamTracker) MarkInFlight(key StreamKey, inFlight bool) {
	st.mu.Lock()
	defer st.mu.Unlock()

	k := key.Key()

	// Registrar si no existe
	if _, exists := st.streams[k]; !exists {
		st.streams[k] = &StreamKPIs{}
		st.keys[k] = key
	}

	st.streams[k].InFlight = inFlight
}

// SetCircuitBreaker actualiza el estado del circuit breaker
func (st *StreamTracker) SetCircuitBreaker(key StreamKey, isOpen bool) {
	st.mu.Lock()
	defer st.mu.Unlock()

	k := key.Key()

	// Registrar si no existe
	if _, exists := st.streams[k]; !exists {
		st.streams[k] = &StreamKPIs{}
		st.keys[k] = key
	}

	st.streams[k].CircuitBreakerOpen = isOpen
}

// SetNotes actualiza las notas de un stream
func (st *StreamTracker) SetNotes(key StreamKey, notes string) {
	st.mu.Lock()
	defer st.mu.Unlock()

	k := key.Key()

	// Registrar si no existe
	if _, exists := st.streams[k]; !exists {
		st.streams[k] = &StreamKPIs{}
		st.keys[k] = key
	}

	if notes == "" {
		st.streams[k].Notes = nil
	} else {
		st.streams[k].Notes = &notes
	}
}

// GetKPIs retorna una copia de los KPIs de un stream
func (st *StreamTracker) GetKPIs(key StreamKey) *StreamKPIs {
	st.mu.RLock()
	defer st.mu.RUnlock()

	k := key.Key()
	kpi, exists := st.streams[k]
	if !exists {
		return nil
	}

	clone := kpi.Clone()
	return &clone
}

// GetAllStreams retorna todos los streams conocidos
func (st *StreamTracker) GetAllStreams() map[string]StreamKey {
	st.mu.RLock()
	defer st.mu.RUnlock()

	// Clonar el mapa
	result := make(map[string]StreamKey, len(st.keys))
	for k, v := range st.keys {
		result[k] = v
	}

	return result
}

// GetAllKPIs retorna todos los KPIs de todos los streams
func (st *StreamTracker) GetAllKPIs() map[string]StreamKPIs {
	st.mu.RLock()
	defer st.mu.RUnlock()

	result := make(map[string]StreamKPIs, len(st.streams))
	for k, kpi := range st.streams {
		result[k] = kpi.Clone()
	}

	return result
}

// RemoveStream elimina un stream del tracker
func (st *StreamTracker) RemoveStream(key StreamKey) {
	st.mu.Lock()
	defer st.mu.Unlock()

	k := key.Key()
	delete(st.streams, k)
	delete(st.keys, k)
}

// Clear limpia todos los streams
func (st *StreamTracker) Clear() {
	st.mu.Lock()
	defer st.mu.Unlock()

	st.streams = make(map[string]*StreamKPIs)
	st.keys = make(map[string]StreamKey)
}

// Count retorna el número de streams registrados
func (st *StreamTracker) Count() int {
	st.mu.RLock()
	defer st.mu.RUnlock()

	return len(st.streams)
}
