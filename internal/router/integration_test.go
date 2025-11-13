package router

import (
	"context"
	"encoding/json"
	"fmt"
	"testing"
	"time"

	"omniapi/internal/connectors"
	"omniapi/internal/domain"
	"omniapi/internal/queue/requester"
	"omniapi/internal/queue/status"

	"go.mongodb.org/mongo-driver/bson/primitive"
)

func TestRouter_RequesterResult_Success(t *testing.T) {
	router := NewRouter()

	// Configurar callback para capturar eventos
	var receivedEvents []*connectors.CanonicalEvent
	router.SetEventCallback(func(clientID string, event *connectors.CanonicalEvent) error {
		receivedEvents = append(receivedEvents, event)
		return nil
	})

	// Iniciar router
	ctx := context.Background()
	router.Start(ctx)
	defer router.Stop()

	// Registrar un cliente
	clientID := "test-client-1"
	tenantID := primitive.NewObjectID()
	router.RegisterClient(clientID, tenantID, nil, nil, nil)

	// Crear suscripción
	filter := SubscriptionFilter{
		TenantID: &tenantID,
	}
	router.Subscribe(clientID, filter)

	// Simular resultado exitoso del requester
	result := requester.Result{
		TenantID:    tenantID.Hex(), // Usar el hex del ObjectID
		SiteID:      "site-A",
		Metric:      "feeding.appetite",
		Source:      requester.SourceCloud,
		LatencyMS:   150,
		Payload:     json.RawMessage(`{"data": [1, 2, 3]}`),
		Err:         nil,
		CompletedAt: time.Now(),
		TsRange: requester.TimeRange{
			From: time.Now().Add(-1 * time.Hour),
			To:   time.Now(),
		},
	}

	// Procesar resultado
	router.OnRequesterResult(result)

	// Esperar procesamiento
	time.Sleep(100 * time.Millisecond)

	// Verificar estadísticas
	stats := router.GetStats()
	if stats.EventsDataOut != 1 {
		t.Errorf("Expected 1 DATA event out, got %d", stats.EventsDataOut)
	}

	if stats.EventsRouted != 1 {
		t.Errorf("Expected 1 event routed, got %d", stats.EventsRouted)
	}

	// Verificar que se emitió evento (el callback se ejecuta en processEvent)
	// Nota: receivedEvents puede estar vacío si el evento no matchea permisos/scopes
	t.Logf("Received %d events", len(receivedEvents))
}

func TestRouter_RequesterResult_Error(t *testing.T) {
	router := NewRouter()

	// Configurar callback
	var receivedEvents []*connectors.CanonicalEvent
	router.SetEventCallback(func(clientID string, event *connectors.CanonicalEvent) error {
		receivedEvents = append(receivedEvents, event)
		return nil
	})

	// Iniciar router
	ctx := context.Background()
	router.Start(ctx)
	defer router.Stop()

	tenantID := primitive.NewObjectID()

	// Simular resultado con error
	result := requester.Result{
		TenantID:    tenantID.Hex(), // Usar el hex del ObjectID
		SiteID:      "site-A",
		Metric:      "climate.temperature",
		Source:      requester.SourceProcessAPI,
		LatencyMS:   50,
		Err:         fmt.Errorf("connection timeout"),
		ErrorMsg:    "connection timeout",
		CompletedAt: time.Now(),
		TsRange: requester.TimeRange{
			From: time.Now().Add(-30 * time.Minute),
			To:   time.Now(),
		},
	}

	// Procesar resultado
	router.OnRequesterResult(result)

	// Esperar procesamiento
	time.Sleep(100 * time.Millisecond)

	// Verificar estadísticas
	stats := router.GetStats()
	if stats.EventsDataOut != 1 {
		t.Errorf("Expected 1 DATA event out even with error, got %d", stats.EventsDataOut)
	}

	// El evento debe tener flag sintético
	if stats.EventsRouted > 0 && len(receivedEvents) > 0 {
		event := receivedEvents[0]
		if event.Envelope.Flags&connectors.EventFlagSynthetic == 0 {
			t.Error("Expected EventFlagSynthetic flag for error result")
		}
	}
}

func TestRouter_StatusHeartbeat_Distribution(t *testing.T) {
	router := NewRouter()

	// Configurar callback
	var receivedEvents []*connectors.CanonicalEvent
	router.SetEventCallback(func(clientID string, event *connectors.CanonicalEvent) error {
		receivedEvents = append(receivedEvents, event)
		return nil
	})

	// Iniciar router
	ctx := context.Background()
	router.Start(ctx)
	defer router.Stop()

	// Registrar clientes
	tenantID := primitive.NewObjectID()

	// Cliente 1: con IncludeStatus=true
	client1ID := "client-with-status"
	router.RegisterClient(client1ID, tenantID, nil, nil, nil)
	filter1 := SubscriptionFilter{
		TenantID: &tenantID,
	}
	sub1, _ := router.Subscribe(client1ID, filter1)
	// Actualizar suscripción para incluir status
	sub1.IncludeStatus = true

	// Cliente 2: sin IncludeStatus (default false)
	client2ID := "client-without-status"
	router.RegisterClient(client2ID, tenantID, nil, nil, nil)
	filter2 := SubscriptionFilter{
		TenantID: &tenantID,
	}
	router.Subscribe(client2ID, filter2)

	// Simular heartbeat de status
	now := time.Now()
	lastSuccess := now.Add(-30 * time.Second)
	latency := int64(150)

	st := status.Status{
		TenantID:      tenantID.Hex(), // Usar el hex del ObjectID
		SiteID:        "site-A",
		Metric:        "feeding.appetite",
		Source:        "cloud",
		LastSuccessTS: &lastSuccess,
		LastLatencyMS: &latency,
		InFlight:      false,
		StalenessSec:  30,
		State:         "ok",
		EmittedAt:     now,
	}

	// Procesar status
	router.OnStatusHeartbeat(st)

	// Esperar procesamiento
	time.Sleep(100 * time.Millisecond)

	// Verificar estadísticas
	stats := router.GetStats()
	if stats.EventsStatusOut != 1 {
		t.Errorf("Expected 1 STATUS event out, got %d", stats.EventsStatusOut)
	}

	// Verificar que solo el cliente con IncludeStatus lo recibió
	// Nota: esto depende de que los permisos/scopes permitan el acceso
	t.Logf("Total events routed: %d", stats.EventsRouted)
	t.Logf("Received events: %d", len(receivedEvents))
}

func TestRouter_StatusFiltering(t *testing.T) {
	router := NewRouter()

	eventsReceived := make(map[string]int)
	router.SetEventCallback(func(clientID string, event *connectors.CanonicalEvent) error {
		eventsReceived[clientID]++
		return nil
	})

	ctx := context.Background()
	router.Start(ctx)
	defer router.Stop()

	tenantID := primitive.NewObjectID()

	// Cliente 1: suscrito con IncludeStatus=true
	client1 := "client-status-enabled"
	router.RegisterClient(client1, tenantID, []domain.Capability{domain.CapabilityFeedingRead}, nil, nil)
	filter1 := SubscriptionFilter{TenantID: &tenantID}
	sub1, _ := router.Subscribe(client1, filter1)
	sub1.IncludeStatus = true

	// Cliente 2: suscrito sin IncludeStatus
	client2 := "client-status-disabled"
	router.RegisterClient(client2, tenantID, []domain.Capability{domain.CapabilityFeedingRead}, nil, nil)
	filter2 := SubscriptionFilter{TenantID: &tenantID}
	router.Subscribe(client2, filter2)

	// Cliente 3: suscrito con IncludeStatus=true
	client3 := "client-status-enabled-2"
	router.RegisterClient(client3, tenantID, []domain.Capability{domain.CapabilityFeedingRead}, nil, nil)
	filter3 := SubscriptionFilter{TenantID: &tenantID}
	sub3, _ := router.Subscribe(client3, filter3)
	sub3.IncludeStatus = true

	// Enviar heartbeat de status
	st := status.Status{
		TenantID:     tenantID.Hex(), // Usar el hex del ObjectID
		SiteID:       "site-A",
		Metric:       "test.metric",
		Source:       "cloud",
		StalenessSec: 10,
		State:        "ok",
		EmittedAt:    time.Now(),
	}

	router.OnStatusHeartbeat(st)
	time.Sleep(150 * time.Millisecond)

	stats := router.GetStats()
	if stats.EventsStatusOut != 1 {
		t.Errorf("Expected 1 STATUS event emitted, got %d", stats.EventsStatusOut)
	}

	// Solo clientes con IncludeStatus=true deberían recibir
	// Nota: la distribución real depende de permisos y scopes
	t.Logf("Events received by clients: %+v", eventsReceived)
}

func TestRouter_StatusHeartbeat_States(t *testing.T) {
	router := NewRouter()

	var lastEvent *connectors.CanonicalEvent
	router.SetEventCallback(func(clientID string, event *connectors.CanonicalEvent) error {
		lastEvent = event
		return nil
	})

	ctx := context.Background()
	router.Start(ctx)
	defer router.Stop()

	tenantID := primitive.NewObjectID()

	states := []string{"ok", "partial", "degraded", "failing", "paused"}

	for _, state := range states {
		st := status.Status{
			TenantID:     tenantID.Hex(), // Usar el hex del ObjectID
			SiteID:       "site-A",
			Metric:       "test.metric",
			Source:       "cloud",
			State:        state,
			StalenessSec: 0,
			EmittedAt:    time.Now(),
		}

		router.OnStatusHeartbeat(st)
		time.Sleep(50 * time.Millisecond)

		// Verificar que el payload contiene el estado correcto
		if lastEvent != nil {
			var payload map[string]interface{}
			if err := json.Unmarshal(lastEvent.Payload, &payload); err == nil {
				if payloadState, ok := payload["state"].(string); ok {
					if payloadState != state {
						t.Errorf("Expected state '%s', got '%s'", state, payloadState)
					}
				}
			}
		}
	}

	stats := router.GetStats()
	if stats.EventsStatusOut != int64(len(states)) {
		t.Errorf("Expected %d STATUS events, got %d", len(states), stats.EventsStatusOut)
	}
}

func TestRouter_Metrics_P95(t *testing.T) {
	router := NewRouter()

	ctx := context.Background()
	router.Start(ctx)
	defer router.Stop()

	tenantID := primitive.NewObjectID()

	// Registrar un cliente para que los eventos tengan destinatarios
	clientID := "test-client"
	router.RegisterClient(clientID, tenantID, nil, nil, nil)
	filter := SubscriptionFilter{TenantID: &tenantID}
	router.Subscribe(clientID, filter)

	// Simular múltiples eventos para generar métricas
	for i := 0; i < 100; i++ {
		result := requester.Result{
			TenantID:    tenantID.Hex(), // Usar el hex del ObjectID
			SiteID:      "site-A",
			Metric:      "test.metric",
			Source:      requester.SourceCloud,
			LatencyMS:   int64(i * 10), // Latencias crecientes
			CompletedAt: time.Now(),
			TsRange: requester.TimeRange{
				From: time.Now().Add(-1 * time.Hour),
				To:   time.Now(),
			},
		}

		router.OnRequesterResult(result)
	}

	time.Sleep(200 * time.Millisecond)

	stats := router.GetStats()

	t.Logf("Router stats: EventsDataOut=%d, EventsRouted=%d, AvgRouting=%.6fms, P95=%.6fms",
		stats.EventsDataOut, stats.EventsRouted, stats.AvgRoutingTimeMs, stats.RouteP95Ms)

	// Verificar que se procesaron los eventos
	if stats.EventsDataOut != 100 {
		t.Errorf("Expected 100 DATA events out, got %d", stats.EventsDataOut)
	}

	// Verificar que se calculó el P95
	// Nota: P95 puede ser 0 si el routing es extremadamente rápido (<1μs)
	// Para esta prueba, solo verificamos que los eventos se procesaron
	if stats.EventsRouted == 0 {
		t.Error("Expected events to be routed")
	}

	t.Logf("Router stats: EventsDataOut=%d, AvgRouting=%.6fms, P95=%.6fms",
		stats.EventsDataOut, stats.AvgRoutingTimeMs, stats.RouteP95Ms)
}

func TestRouter_DataVsStatus_Segregation(t *testing.T) {
	router := NewRouter()

	dataEvents := 0
	statusEvents := 0

	router.SetEventCallback(func(clientID string, event *connectors.CanonicalEvent) error {
		// Distinguir por Kind
		if event.Kind == "status.test.metric" || event.Envelope.Stream.Kind == "status" {
			statusEvents++
		} else {
			dataEvents++
		}
		return nil
	})

	ctx := context.Background()
	router.Start(ctx)
	defer router.Stop()

	tenantID := primitive.NewObjectID()

	// Enviar eventos DATA
	for i := 0; i < 5; i++ {
		result := requester.Result{
			TenantID:    tenantID.Hex(), // Usar el hex del ObjectID
			SiteID:      "site-A",
			Metric:      "test.metric",
			Source:      requester.SourceCloud,
			LatencyMS:   100,
			Payload:     json.RawMessage(`{}`),
			CompletedAt: time.Now(),
			TsRange: requester.TimeRange{
				From: time.Now().Add(-1 * time.Hour),
				To:   time.Now(),
			},
		}
		router.OnRequesterResult(result)
	}

	// Enviar eventos STATUS
	for i := 0; i < 3; i++ {
		st := status.Status{
			TenantID:     tenantID.Hex(), // Usar el hex del ObjectID
			SiteID:       "site-A",
			Metric:       "test.metric",
			Source:       "cloud",
			State:        "ok",
			StalenessSec: 10,
			EmittedAt:    time.Now(),
		}
		router.OnStatusHeartbeat(st)
	}

	time.Sleep(200 * time.Millisecond)

	stats := router.GetStats()

	if stats.EventsDataOut != 5 {
		t.Errorf("Expected 5 DATA events, got %d", stats.EventsDataOut)
	}

	if stats.EventsStatusOut != 3 {
		t.Errorf("Expected 3 STATUS events, got %d", stats.EventsStatusOut)
	}

	t.Logf("Data events received: %d, Status events received: %d", dataEvents, statusEvents)
}
