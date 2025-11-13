package router

import (
	"sync"
	"time"

	"omniapi/internal/connectors"
)

// Throttler gestiona el rate limiting y coalescing de eventos
type Throttler struct {
	clients map[string]*ClientThrottleState
	mu      sync.RWMutex
}

// ClientThrottleState mantiene el estado de throttle de un cliente
type ClientThrottleState struct {
	ClientID         string
	Config           ThrottleConfig
	LastSent         time.Time
	TokenBucket      int // Para rate limiting con token bucket
	LastTokenFill    time.Time
	EventsThisSecond int
	SecondStart      time.Time
}

// NewThrottler crea un nuevo throttler
func NewThrottler() *Throttler {
	return &Throttler{
		clients: make(map[string]*ClientThrottleState),
	}
}

// RegisterClient registra un cliente en el throttler
func (t *Throttler) RegisterClient(clientID string, config ThrottleConfig) {
	t.mu.Lock()
	defer t.mu.Unlock()

	now := time.Now()
	t.clients[clientID] = &ClientThrottleState{
		ClientID:         clientID,
		Config:           config,
		LastSent:         time.Time{}, // Zero time para permitir primer evento inmediatamente
		TokenBucket:      config.BurstSize,
		LastTokenFill:    now,
		EventsThisSecond: 0,
		SecondStart:      now,
	}
}

// UnregisterClient elimina un cliente del throttler
func (t *Throttler) UnregisterClient(clientID string) {
	t.mu.Lock()
	defer t.mu.Unlock()

	delete(t.clients, clientID)
}

// UpdateConfig actualiza la configuración de throttle de un cliente
func (t *Throttler) UpdateConfig(clientID string, config ThrottleConfig) error {
	t.mu.Lock()
	defer t.mu.Unlock()

	state, exists := t.clients[clientID]
	if !exists {
		// Si no existe, crear nuevo estado
		now := time.Now()
		t.clients[clientID] = &ClientThrottleState{
			ClientID:         clientID,
			Config:           config,
			LastSent:         time.Time{}, // Zero time para permitir primer evento inmediatamente
			TokenBucket:      config.BurstSize,
			LastTokenFill:    now,
			EventsThisSecond: 0,
			SecondStart:      now,
		}
		return nil
	}

	state.Config = config
	return nil
}

// ShouldSend determina si un evento debe ser enviado basándose en throttling
func (t *Throttler) ShouldSend(clientID string, event *connectors.CanonicalEvent) bool {
	t.mu.Lock()
	defer t.mu.Unlock()

	state, exists := t.clients[clientID]
	if !exists {
		// Si no hay configuración, permitir por defecto
		return true
	}

	now := time.Now()

	// Rellenar token bucket
	t.refillTokenBucket(state, now)

	// Verificar ThrottleMs (tiempo mínimo entre eventos)
	if state.Config.ThrottleMs > 0 {
		minInterval := time.Duration(state.Config.ThrottleMs) * time.Millisecond
		if now.Sub(state.LastSent) < minInterval {
			return false
		}
	}

	// Verificar MaxRate (eventos por segundo)
	if state.Config.MaxRate > 0 {
		// Resetear contador si cambió el segundo
		if now.Sub(state.SecondStart) >= time.Second {
			state.EventsThisSecond = 0
			state.SecondStart = now
		}

		if float64(state.EventsThisSecond) >= state.Config.MaxRate {
			return false
		}
	}

	// Verificar token bucket para burst control
	if state.Config.BurstSize > 0 {
		if state.TokenBucket <= 0 {
			return false
		}
		state.TokenBucket--
	}

	// Actualizar contadores
	state.LastSent = now
	state.EventsThisSecond++

	return true
}

// refillTokenBucket rellena el token bucket basándose en el tiempo transcurrido
func (t *Throttler) refillTokenBucket(state *ClientThrottleState, now time.Time) {
	if state.Config.MaxRate <= 0 {
		return
	}

	elapsed := now.Sub(state.LastTokenFill)
	tokensToAdd := int(elapsed.Seconds() * state.Config.MaxRate)

	if tokensToAdd > 0 {
		state.TokenBucket += tokensToAdd
		if state.TokenBucket > state.Config.BurstSize {
			state.TokenBucket = state.Config.BurstSize
		}
		state.LastTokenFill = now
	}
}

// BufferEvent agrega un evento al buffer de un cliente
func (t *Throttler) BufferEvent(clientID string, event *connectors.CanonicalEvent, client *ClientState) {
	streamKey := event.Envelope.Stream.String()

	// Obtener o crear buffer para este stream
	buffer, exists := client.StreamBuffers[streamKey]
	if !exists {
		buffer = NewStreamBuffer(
			streamKey,
			client.ThrottleConfig.BufferSize,
			client.ThrottleConfig.KeepLatest,
		)
		client.StreamBuffers[streamKey] = buffer
	}

	buffer.Push(event)
}

// GetPendingEvents obtiene eventos pendientes de un cliente que pueden ser enviados
func (t *Throttler) GetPendingEvents(clientID string, client *ClientState) []*connectors.CanonicalEvent {
	events := make([]*connectors.CanonicalEvent, 0)

	// Iterar por todos los buffers del cliente
	for _, buffer := range client.StreamBuffers {
		for buffer.Len() > 0 {
			event := buffer.Pop()
			if event != nil && t.ShouldSend(clientID, event) {
				events = append(events, event)
			} else if event != nil {
				// Si no se puede enviar, devolverlo al buffer
				// Crear un nuevo slice para evitar modificar durante iteración
				newEvents := make([]*connectors.CanonicalEvent, 0, buffer.Len()+1)
				newEvents = append(newEvents, event)
				for buffer.Len() > 0 {
					if e := buffer.Pop(); e != nil {
						newEvents = append(newEvents, e)
					}
				}
				buffer.Events = newEvents
				break
			}
		}
	}

	return events
}

// CoalesceEvents combina eventos similares del mismo stream
func (t *Throttler) CoalesceEvents(events []*connectors.CanonicalEvent) []*connectors.CanonicalEvent {
	if len(events) <= 1 {
		return events
	}

	// Mapa por stream key para coalescing
	byStream := make(map[string]*connectors.CanonicalEvent)

	for _, event := range events {
		streamKey := event.Envelope.Stream.String()

		// Mantener solo el último evento por stream
		existing, exists := byStream[streamKey]
		if !exists || event.Envelope.Timestamp.After(existing.Envelope.Timestamp) {
			byStream[streamKey] = event
		}
	}

	// Convertir mapa a slice
	coalesced := make([]*connectors.CanonicalEvent, 0, len(byStream))
	for _, event := range byStream {
		coalesced = append(coalesced, event)
	}

	return coalesced
}

// ProcessEvent procesa un evento para un cliente con throttling y buffering
func (t *Throttler) ProcessEvent(clientID string, event *connectors.CanonicalEvent, client *ClientState) (bool, string) {
	t.mu.Lock()
	defer t.mu.Unlock()

	state, exists := t.clients[clientID]
	if !exists {
		// Sin configuración de throttle, enviar directamente
		return true, "no_throttle"
	}

	now := time.Now()
	t.refillTokenBucket(state, now)

	// Verificar si se puede enviar inmediatamente
	if t.canSendNow(state, now) {
		t.updateSendMetrics(state, now)
		return true, "sent"
	}

	// Si no se puede enviar, buffear
	if state.Config.CoalescingEnabled {
		t.BufferEvent(clientID, event, client)
		client.Stats.Throttled++
		return false, "buffered"
	}

	// Si buffering no está habilitado, descartar
	client.Stats.EventsDropped++
	return false, "dropped"
}

// canSendNow verifica si un evento puede ser enviado ahora
func (t *Throttler) canSendNow(state *ClientThrottleState, now time.Time) bool {
	// Verificar ThrottleMs
	if state.Config.ThrottleMs > 0 {
		minInterval := time.Duration(state.Config.ThrottleMs) * time.Millisecond
		if now.Sub(state.LastSent) < minInterval {
			return false
		}
	}

	// Verificar MaxRate
	if state.Config.MaxRate > 0 {
		if now.Sub(state.SecondStart) >= time.Second {
			state.EventsThisSecond = 0
			state.SecondStart = now
		}

		if float64(state.EventsThisSecond) >= state.Config.MaxRate {
			return false
		}
	}

	// Verificar token bucket
	if state.Config.BurstSize > 0 && state.TokenBucket <= 0 {
		return false
	}

	return true
}

// updateSendMetrics actualiza las métricas después de enviar
func (t *Throttler) updateSendMetrics(state *ClientThrottleState, now time.Time) {
	state.LastSent = now
	state.EventsThisSecond++

	if state.Config.BurstSize > 0 {
		state.TokenBucket--
	}
}

// GetStats retorna estadísticas del throttler
func (t *Throttler) GetStats() map[string]interface{} {
	t.mu.RLock()
	defer t.mu.RUnlock()

	stats := map[string]interface{}{
		"active_clients": len(t.clients),
	}

	totalTokens := 0
	for _, state := range t.clients {
		totalTokens += state.TokenBucket
	}

	if len(t.clients) > 0 {
		stats["avg_tokens_available"] = float64(totalTokens) / float64(len(t.clients))
	}

	return stats
}

// GetClientStats retorna estadísticas de un cliente específico
func (t *Throttler) GetClientStats(clientID string) map[string]interface{} {
	t.mu.RLock()
	defer t.mu.RUnlock()

	state, exists := t.clients[clientID]
	if !exists {
		return map[string]interface{}{
			"exists": false,
		}
	}

	return map[string]interface{}{
		"exists":             true,
		"token_bucket":       state.TokenBucket,
		"events_this_second": state.EventsThisSecond,
		"last_sent":          state.LastSent,
		"config":             state.Config,
	}
}
