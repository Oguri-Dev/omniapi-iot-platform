package router

import (
	"testing"
	"time"

	"omniapi/internal/connectors"
	"omniapi/internal/domain"

	"go.mongodb.org/mongo-driver/bson/primitive"
)

func TestThrottler_ShouldSend_ThrottleMs(t *testing.T) {
	throttler := NewThrottler()
	clientID := "client-1"

	config := ThrottleConfig{
		ThrottleMs:        100, // 100ms mínimo entre eventos
		MaxRate:           0,   // Sin límite de rate
		BurstSize:         0,   // Sin burst control
		CoalescingEnabled: false,
		KeepLatest:        false,
		BufferSize:        10,
	}

	throttler.RegisterClient(clientID, config)

	event := createTestEvent(primitive.NewObjectID(), domain.StreamKindFeeding, "farm-1", "site-1", nil)

	// Primer evento debe pasar
	if !throttler.ShouldSend(clientID, event) {
		t.Error("First event should be sent")
	}

	// Segundo evento inmediatamente después debe ser bloqueado
	if throttler.ShouldSend(clientID, event) {
		t.Error("Second event should be throttled")
	}

	// Esperar el tiempo de throttle
	time.Sleep(110 * time.Millisecond)

	// Ahora debe pasar
	if !throttler.ShouldSend(clientID, event) {
		t.Error("Event after throttle period should be sent")
	}
}

func TestThrottler_ShouldSend_MaxRate(t *testing.T) {
	throttler := NewThrottler()
	clientID := "client-1"

	config := ThrottleConfig{
		ThrottleMs:        0,   // Sin throttle de tiempo
		MaxRate:           5.0, // 5 eventos por segundo
		BurstSize:         0,   // Sin burst control
		CoalescingEnabled: false,
		KeepLatest:        false,
		BufferSize:        10,
	}

	throttler.RegisterClient(clientID, config)

	event := createTestEvent(primitive.NewObjectID(), domain.StreamKindFeeding, "farm-1", "site-1", nil)

	// Enviar 5 eventos (debe pasar todos)
	sentCount := 0
	for i := 0; i < 5; i++ {
		if throttler.ShouldSend(clientID, event) {
			sentCount++
		}
	}

	if sentCount != 5 {
		t.Errorf("Expected 5 events to be sent, got %d", sentCount)
	}

	// El sexto evento debe ser bloqueado
	if throttler.ShouldSend(clientID, event) {
		t.Error("Sixth event should be throttled (exceeds MaxRate)")
	}

	// Esperar un segundo para resetear el contador
	time.Sleep(1100 * time.Millisecond)

	// Ahora debe pasar nuevamente
	if !throttler.ShouldSend(clientID, event) {
		t.Error("Event after rate reset should be sent")
	}
}

func TestThrottler_ShouldSend_BurstSize(t *testing.T) {
	throttler := NewThrottler()
	clientID := "client-1"

	config := ThrottleConfig{
		ThrottleMs:        0,
		MaxRate:           10.0, // 10 eventos por segundo
		BurstSize:         3,    // Máximo 3 eventos en burst
		CoalescingEnabled: false,
		KeepLatest:        false,
		BufferSize:        10,
	}

	throttler.RegisterClient(clientID, config)

	event := createTestEvent(primitive.NewObjectID(), domain.StreamKindFeeding, "farm-1", "site-1", nil)

	// Enviar 3 eventos rápidamente (burst)
	sentCount := 0
	for i := 0; i < 3; i++ {
		if throttler.ShouldSend(clientID, event) {
			sentCount++
		}
	}

	if sentCount != 3 {
		t.Errorf("Expected 3 events in burst, got %d", sentCount)
	}

	// El cuarto evento debe ser bloqueado (bucket vacío)
	if throttler.ShouldSend(clientID, event) {
		t.Error("Fourth event should be throttled (bucket empty)")
	}

	// Esperar para que se rellene el bucket
	time.Sleep(150 * time.Millisecond)

	// Ahora debe pasar (bucket recargado)
	if !throttler.ShouldSend(clientID, event) {
		t.Error("Event after bucket refill should be sent")
	}
}

func TestStreamBuffer_KeepLatest(t *testing.T) {
	tenantID := primitive.NewObjectID()
	buffer := NewStreamBuffer("test-stream", 3, true) // keep-latest enabled

	// Agregar 5 eventos (más que el máximo)
	for i := 0; i < 5; i++ {
		event := createTestEvent(tenantID, domain.StreamKindFeeding, "farm-1", "site-1", nil)
		event.Envelope.Sequence = uint64(i)
		buffer.Push(event)
	}

	// Debe tener solo 3 eventos
	if buffer.Len() != 3 {
		t.Errorf("Expected 3 events in buffer, got %d", buffer.Len())
	}

	// El último evento debe ser el más reciente (sequence 4)
	events := buffer.Events
	lastEvent := events[len(events)-1]
	if lastEvent.Envelope.Sequence != 4 {
		t.Errorf("Expected last event sequence 4, got %d", lastEvent.Envelope.Sequence)
	}
}

func TestStreamBuffer_NoKeepLatest(t *testing.T) {
	tenantID := primitive.NewObjectID()
	buffer := NewStreamBuffer("test-stream", 3, false) // keep-latest disabled

	// Agregar 3 eventos
	for i := 0; i < 3; i++ {
		event := createTestEvent(tenantID, domain.StreamKindFeeding, "farm-1", "site-1", nil)
		event.Envelope.Sequence = uint64(i)
		buffer.Push(event)
	}

	if buffer.Len() != 3 {
		t.Errorf("Expected 3 events in buffer, got %d", buffer.Len())
	}

	// Intentar agregar un cuarto evento (debe ser descartado)
	event := createTestEvent(tenantID, domain.StreamKindFeeding, "farm-1", "site-1", nil)
	event.Envelope.Sequence = 100
	buffer.Push(event)

	// Debe seguir teniendo 3 eventos
	if buffer.Len() != 3 {
		t.Errorf("Expected 3 events (no growth), got %d", buffer.Len())
	}

	// El último evento debe ser el tercero original (no el nuevo)
	lastEvent := buffer.Events[len(buffer.Events)-1]
	if lastEvent.Envelope.Sequence == 100 {
		t.Error("New event should have been dropped (keep-latest is false)")
	}
}

func TestStreamBuffer_PopOrder(t *testing.T) {
	tenantID := primitive.NewObjectID()
	buffer := NewStreamBuffer("test-stream", 5, false)

	// Agregar eventos con secuencias conocidas
	for i := 0; i < 3; i++ {
		event := createTestEvent(tenantID, domain.StreamKindFeeding, "farm-1", "site-1", nil)
		event.Envelope.Sequence = uint64(i)
		buffer.Push(event)
	}

	// Pop debe retornar en orden FIFO
	for i := 0; i < 3; i++ {
		event := buffer.Pop()
		if event == nil {
			t.Fatal("Pop returned nil unexpectedly")
		}
		if event.Envelope.Sequence != uint64(i) {
			t.Errorf("Expected sequence %d, got %d", i, event.Envelope.Sequence)
		}
	}

	// Buffer debe estar vacío
	if buffer.Len() != 0 {
		t.Errorf("Expected empty buffer, got %d events", buffer.Len())
	}

	// Pop en buffer vacío debe retornar nil
	if buffer.Pop() != nil {
		t.Error("Pop on empty buffer should return nil")
	}
}

func TestThrottler_CoalesceEvents(t *testing.T) {
	throttler := NewThrottler()
	tenantID := primitive.NewObjectID()

	// Crear eventos del mismo stream
	events := make([]*connectors.CanonicalEvent, 0)
	for i := 0; i < 5; i++ {
		event := createTestEvent(tenantID, domain.StreamKindFeeding, "farm-1", "site-1", nil)
		event.Envelope.Sequence = uint64(i)
		event.Envelope.Timestamp = time.Now().Add(time.Duration(i) * time.Millisecond)
		events = append(events, event)
	}

	// Coalescing debe retornar solo el último evento
	coalesced := throttler.CoalesceEvents(events)

	if len(coalesced) != 1 {
		t.Errorf("Expected 1 coalesced event, got %d", len(coalesced))
	}

	// Debe ser el último evento (sequence 4)
	if coalesced[0].Envelope.Sequence != 4 {
		t.Errorf("Expected sequence 4, got %d", coalesced[0].Envelope.Sequence)
	}
}

func TestThrottler_CoalesceEvents_MultipleStreams(t *testing.T) {
	throttler := NewThrottler()
	tenantID := primitive.NewObjectID()
	cageID1 := "cage-1"
	cageID2 := "cage-2"

	// Crear eventos de diferentes streams
	events := make([]*connectors.CanonicalEvent, 0)

	// 3 eventos de cage-1
	for i := 0; i < 3; i++ {
		event := createTestEvent(tenantID, domain.StreamKindFeeding, "farm-1", "site-1", &cageID1)
		event.Envelope.Sequence = uint64(i)
		event.Envelope.Timestamp = time.Now().Add(time.Duration(i) * time.Millisecond)
		events = append(events, event)
	}

	// 2 eventos de cage-2
	for i := 0; i < 2; i++ {
		event := createTestEvent(tenantID, domain.StreamKindFeeding, "farm-1", "site-1", &cageID2)
		event.Envelope.Sequence = uint64(i + 100)
		event.Envelope.Timestamp = time.Now().Add(time.Duration(i) * time.Millisecond)
		events = append(events, event)
	}

	// Coalescing debe retornar 2 eventos (uno por stream)
	coalesced := throttler.CoalesceEvents(events)

	if len(coalesced) != 2 {
		t.Errorf("Expected 2 coalesced events, got %d", len(coalesced))
	}
}

func TestThrottler_ProcessEvent(t *testing.T) {
	throttler := NewThrottler()
	clientID := "client-1"
	tenantID := primitive.NewObjectID()

	config := ThrottleConfig{
		ThrottleMs:        50,
		MaxRate:           10.0,
		BurstSize:         3,
		CoalescingEnabled: true,
		KeepLatest:        true,
		BufferSize:        10,
	}

	throttler.RegisterClient(clientID, config)
	client := NewClientState(clientID, tenantID)
	client.ThrottleConfig = config

	event := createTestEvent(tenantID, domain.StreamKindFeeding, "farm-1", "site-1", nil)

	// Primer evento debe ser enviado
	sent, reason := throttler.ProcessEvent(clientID, event, client)
	if !sent || reason != "sent" {
		t.Errorf("First event should be sent, got: sent=%v, reason=%s", sent, reason)
	}

	// Segundo evento inmediato debe ser buffereado
	sent, reason = throttler.ProcessEvent(clientID, event, client)
	if sent || reason != "buffered" {
		t.Errorf("Second event should be buffered, got: sent=%v, reason=%s", sent, reason)
	}

	// Verificar que se incrementó el contador de throttled
	if client.Stats.Throttled != 1 {
		t.Errorf("Expected throttled count 1, got %d", client.Stats.Throttled)
	}
}

func TestThrottler_UpdateConfig(t *testing.T) {
	throttler := NewThrottler()
	clientID := "client-1"

	// Configuración inicial
	config1 := ThrottleConfig{
		ThrottleMs: 100,
		MaxRate:    5.0,
		BurstSize:  3,
	}

	throttler.RegisterClient(clientID, config1)

	// Verificar configuración inicial
	stats := throttler.GetClientStats(clientID)
	if !stats["exists"].(bool) {
		t.Error("Client should exist")
	}

	// Actualizar configuración
	config2 := ThrottleConfig{
		ThrottleMs: 200,
		MaxRate:    10.0,
		BurstSize:  5,
	}

	err := throttler.UpdateConfig(clientID, config2)
	if err != nil {
		t.Errorf("Failed to update config: %v", err)
	}

	// Verificar que se actualizó
	state := throttler.clients[clientID]
	if state.Config.ThrottleMs != 200 {
		t.Errorf("Expected ThrottleMs 200, got %d", state.Config.ThrottleMs)
	}
	if state.Config.MaxRate != 10.0 {
		t.Errorf("Expected MaxRate 10.0, got %f", state.Config.MaxRate)
	}
}

func TestThrottler_GetPendingEvents(t *testing.T) {
	throttler := NewThrottler()
	clientID := "client-1"
	tenantID := primitive.NewObjectID()

	config := ThrottleConfig{
		ThrottleMs:        100,
		MaxRate:           5.0,
		BurstSize:         5,
		CoalescingEnabled: true,
		KeepLatest:        true,
		BufferSize:        10,
	}

	throttler.RegisterClient(clientID, config)
	client := NewClientState(clientID, tenantID)
	client.ThrottleConfig = config

	// Buffear varios eventos
	for i := 0; i < 3; i++ {
		event := createTestEvent(tenantID, domain.StreamKindFeeding, "farm-1", "site-1", nil)
		event.Envelope.Sequence = uint64(i)
		throttler.BufferEvent(clientID, event, client)
	}

	// Verificar que hay eventos en el buffer
	streamKey := createTestEvent(tenantID, domain.StreamKindFeeding, "farm-1", "site-1", nil).Envelope.Stream.String()
	if buffer, exists := client.StreamBuffers[streamKey]; exists {
		if buffer.Len() != 3 {
			t.Errorf("Expected 3 buffered events, got %d", buffer.Len())
		}
	} else {
		t.Error("Stream buffer should exist")
	}

	// Esperar para que pase el throttle
	time.Sleep(150 * time.Millisecond)

	// Obtener eventos pendientes
	pending := throttler.GetPendingEvents(clientID, client)

	if len(pending) == 0 {
		t.Error("Expected pending events after throttle period")
	}
}

func TestDefaultThrottleConfig(t *testing.T) {
	config := DefaultThrottleConfig()

	if config.ThrottleMs != 100 {
		t.Errorf("Expected default ThrottleMs 100, got %d", config.ThrottleMs)
	}

	if config.MaxRate != 10.0 {
		t.Errorf("Expected default MaxRate 10.0, got %f", config.MaxRate)
	}

	if config.BurstSize != 5 {
		t.Errorf("Expected default BurstSize 5, got %d", config.BurstSize)
	}

	if !config.CoalescingEnabled {
		t.Error("Expected CoalescingEnabled to be true by default")
	}

	if !config.KeepLatest {
		t.Error("Expected KeepLatest to be true by default")
	}

	if config.BufferSize != 100 {
		t.Errorf("Expected default BufferSize 100, got %d", config.BufferSize)
	}
}
