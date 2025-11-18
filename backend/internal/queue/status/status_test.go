package status

import (
	"context"
	"sync"
	"testing"
	"time"
)

func TestStreamTracker_RegisterAndUpdate(t *testing.T) {
	tracker := NewStreamTracker()

	key := StreamKey{
		TenantID: "tenant-1",
		SiteID:   "site-A",
		Metric:   "feeding.appetite",
		Source:   "cloud",
	}

	// Registrar stream
	tracker.RegisterStream(key)

	if tracker.Count() != 1 {
		t.Errorf("Expected 1 stream, got %d", tracker.Count())
	}

	// Actualizar con éxito
	tracker.UpdateSuccess(key, 150)

	kpi := tracker.GetKPIs(key)
	if kpi == nil {
		t.Fatal("Expected KPIs, got nil")
	}

	if kpi.LastSuccessTS == nil {
		t.Error("Expected LastSuccessTS to be set")
	}

	if kpi.LastLatencyMS == nil || *kpi.LastLatencyMS != 150 {
		t.Errorf("Expected latency 150, got %v", kpi.LastLatencyMS)
	}

	if kpi.ConsecutiveErrors != 0 {
		t.Errorf("Expected 0 consecutive errors, got %d", kpi.ConsecutiveErrors)
	}
}

func TestStreamTracker_UpdateError(t *testing.T) {
	tracker := NewStreamTracker()

	key := StreamKey{
		TenantID: "tenant-1",
		SiteID:   "site-A",
		Metric:   "climate.temperature",
		Source:   "processapi",
	}

	// Actualizar con error
	tracker.UpdateError(key, "connection timeout")

	kpi := tracker.GetKPIs(key)
	if kpi == nil {
		t.Fatal("Expected KPIs, got nil")
	}

	if kpi.LastErrorTS == nil {
		t.Error("Expected LastErrorTS to be set")
	}

	if kpi.LastErrorMsg == nil || *kpi.LastErrorMsg != "connection timeout" {
		t.Errorf("Expected error message 'connection timeout', got %v", kpi.LastErrorMsg)
	}

	if kpi.ConsecutiveErrors != 1 {
		t.Errorf("Expected 1 consecutive error, got %d", kpi.ConsecutiveErrors)
	}
}

func TestStreamTracker_ConsecutiveErrors(t *testing.T) {
	tracker := NewStreamTracker()

	key := StreamKey{
		TenantID: "tenant-1",
		SiteID:   "site-A",
		Metric:   "metric-1",
		Source:   "cloud",
	}

	// 3 errores consecutivos
	tracker.UpdateError(key, "error 1")
	tracker.UpdateError(key, "error 2")
	tracker.UpdateError(key, "error 3")

	kpi := tracker.GetKPIs(key)
	if kpi.ConsecutiveErrors != 3 {
		t.Errorf("Expected 3 consecutive errors, got %d", kpi.ConsecutiveErrors)
	}

	// Un éxito resetea los errores
	tracker.UpdateSuccess(key, 100)

	kpi = tracker.GetKPIs(key)
	if kpi.ConsecutiveErrors != 0 {
		t.Errorf("Expected 0 consecutive errors after success, got %d", kpi.ConsecutiveErrors)
	}

	if kpi.ConsecutiveSuccesses != 1 {
		t.Errorf("Expected 1 consecutive success, got %d", kpi.ConsecutiveSuccesses)
	}
}

func TestStreamTracker_InFlight(t *testing.T) {
	tracker := NewStreamTracker()

	key := StreamKey{
		TenantID: "tenant-1",
		SiteID:   "site-A",
		Metric:   "metric-1",
		Source:   "cloud",
	}

	// Marcar como in-flight
	tracker.MarkInFlight(key, true)

	kpi := tracker.GetKPIs(key)
	if !kpi.InFlight {
		t.Error("Expected InFlight to be true")
	}

	// Desmarcar
	tracker.MarkInFlight(key, false)

	kpi = tracker.GetKPIs(key)
	if kpi.InFlight {
		t.Error("Expected InFlight to be false")
	}
}

func TestStreamTracker_CircuitBreaker(t *testing.T) {
	tracker := NewStreamTracker()

	key := StreamKey{
		TenantID: "tenant-1",
		SiteID:   "site-A",
		Metric:   "metric-1",
		Source:   "cloud",
	}

	// Abrir circuit breaker
	tracker.SetCircuitBreaker(key, true)

	kpi := tracker.GetKPIs(key)
	if !kpi.CircuitBreakerOpen {
		t.Error("Expected CircuitBreakerOpen to be true")
	}

	// Cerrar circuit breaker
	tracker.SetCircuitBreaker(key, false)

	kpi = tracker.GetKPIs(key)
	if kpi.CircuitBreakerOpen {
		t.Error("Expected CircuitBreakerOpen to be false")
	}
}

func TestStreamTracker_GetAllStreams(t *testing.T) {
	tracker := NewStreamTracker()

	keys := []StreamKey{
		{TenantID: "tenant-1", SiteID: "site-A", Metric: "metric-1", Source: "cloud"},
		{TenantID: "tenant-1", SiteID: "site-B", Metric: "metric-2", Source: "processapi"},
		{TenantID: "tenant-2", SiteID: "site-C", Metric: "metric-3", Source: "derived"},
	}

	for _, key := range keys {
		tracker.RegisterStream(key)
	}

	allStreams := tracker.GetAllStreams()
	if len(allStreams) != 3 {
		t.Errorf("Expected 3 streams, got %d", len(allStreams))
	}
}

func TestStreamTracker_RemoveStream(t *testing.T) {
	tracker := NewStreamTracker()

	key := StreamKey{
		TenantID: "tenant-1",
		SiteID:   "site-A",
		Metric:   "metric-1",
		Source:   "cloud",
	}

	tracker.RegisterStream(key)
	if tracker.Count() != 1 {
		t.Errorf("Expected 1 stream, got %d", tracker.Count())
	}

	tracker.RemoveStream(key)
	if tracker.Count() != 0 {
		t.Errorf("Expected 0 streams after removal, got %d", tracker.Count())
	}
}

func TestStatusPusher_HeartbeatFrequency(t *testing.T) {
	config := DefaultConfig()
	config.HeartbeatInterval = 100 * time.Millisecond

	tracker := NewStreamTracker()
	pusher := NewStatusPusher(config, tracker)

	// Registrar un stream
	key := StreamKey{
		TenantID: "tenant-1",
		SiteID:   "site-A",
		Metric:   "feeding.appetite",
		Source:   "cloud",
	}
	tracker.RegisterStream(key)
	tracker.UpdateSuccess(key, 100)

	// Contar emisiones
	emitCount := 0
	var mu sync.Mutex

	pusher.OnEmit(func(s Status) {
		mu.Lock()
		emitCount++
		mu.Unlock()
	})

	// Iniciar
	ctx := context.Background()
	pusher.Start(ctx)
	defer pusher.Stop()

	// Esperar ~350ms (debería emitir ~3-4 veces: inicio + 3 ticks)
	time.Sleep(350 * time.Millisecond)

	mu.Lock()
	count := emitCount
	mu.Unlock()

	if count < 3 || count > 5 {
		t.Errorf("Expected 3-5 emissions in 350ms with 100ms interval, got %d", count)
	}
}

func TestStatusPusher_BuildStatus(t *testing.T) {
	config := DefaultConfig()
	tracker := NewStreamTracker()

	key := StreamKey{
		TenantID: "tenant-1",
		SiteID:   "site-A",
		Metric:   "feeding.appetite",
		Source:   "cloud",
	}

	// Actualizar con éxito
	tracker.UpdateSuccess(key, 150)

	pusher := NewStatusPusher(config, tracker)

	statuses := pusher.GetCurrentStatus()
	if len(statuses) != 1 {
		t.Fatalf("Expected 1 status, got %d", len(statuses))
	}

	status := statuses[0]

	if status.TenantID != "tenant-1" {
		t.Errorf("Expected TenantID 'tenant-1', got '%s'", status.TenantID)
	}

	if status.SiteID != "site-A" {
		t.Errorf("Expected SiteID 'site-A', got '%s'", status.SiteID)
	}

	if status.Metric != "feeding.appetite" {
		t.Errorf("Expected Metric 'feeding.appetite', got '%s'", status.Metric)
	}

	if status.Source != "cloud" {
		t.Errorf("Expected Source 'cloud', got '%s'", status.Source)
	}

	if status.LastSuccessTS == nil {
		t.Error("Expected LastSuccessTS to be set")
	}

	if status.LastLatencyMS == nil || *status.LastLatencyMS != 150 {
		t.Errorf("Expected latency 150, got %v", status.LastLatencyMS)
	}

	if status.State != StateOK.String() {
		t.Errorf("Expected state 'ok', got '%s'", status.State)
	}
}

func TestStatusPusher_DetermineState_OK(t *testing.T) {
	config := DefaultConfig()
	config.StaleThresholdOK = 60
	tracker := NewStreamTracker()

	key := StreamKey{
		TenantID: "tenant-1",
		SiteID:   "site-A",
		Metric:   "metric-1",
		Source:   "cloud",
	}

	// Éxito reciente (< 60s)
	tracker.UpdateSuccess(key, 100)

	pusher := NewStatusPusher(config, tracker)
	statuses := pusher.GetCurrentStatus()

	if len(statuses) != 1 {
		t.Fatalf("Expected 1 status, got %d", len(statuses))
	}

	if statuses[0].State != StateOK.String() {
		t.Errorf("Expected state 'ok', got '%s'", statuses[0].State)
	}

	if statuses[0].StalenessSec > 5 {
		t.Errorf("Expected staleness < 5s, got %d", statuses[0].StalenessSec)
	}
}

func TestStatusPusher_DetermineState_Partial(t *testing.T) {
	config := DefaultConfig()
	config.StaleThresholdOK = 60
	tracker := NewStreamTracker()

	key := StreamKey{
		TenantID: "tenant-1",
		SiteID:   "site-A",
		Metric:   "metric-1",
		Source:   "cloud",
	}

	// Éxito reciente pero con un error después
	tracker.UpdateSuccess(key, 100)
	time.Sleep(10 * time.Millisecond)
	tracker.UpdateError(key, "minor error")

	pusher := NewStatusPusher(config, tracker)
	statuses := pusher.GetCurrentStatus()

	if len(statuses) != 1 {
		t.Fatalf("Expected 1 status, got %d", len(statuses))
	}

	// Debería ser partial porque tiene errores consecutivos pero datos frescos
	if statuses[0].State != StatePartial.String() {
		t.Errorf("Expected state 'partial', got '%s'", statuses[0].State)
	}
}

func TestStatusPusher_DetermineState_Failing(t *testing.T) {
	config := DefaultConfig()
	config.MaxConsecutiveErrors = 3
	tracker := NewStreamTracker()

	key := StreamKey{
		TenantID: "tenant-1",
		SiteID:   "site-A",
		Metric:   "metric-1",
		Source:   "cloud",
	}

	// 3 errores consecutivos
	tracker.UpdateError(key, "error 1")
	tracker.UpdateError(key, "error 2")
	tracker.UpdateError(key, "error 3")

	pusher := NewStatusPusher(config, tracker)
	statuses := pusher.GetCurrentStatus()

	if len(statuses) != 1 {
		t.Fatalf("Expected 1 status, got %d", len(statuses))
	}

	if statuses[0].State != StateFailing.String() {
		t.Errorf("Expected state 'failing', got '%s'", statuses[0].State)
	}
}

func TestStatusPusher_DetermineState_Paused(t *testing.T) {
	config := DefaultConfig()
	tracker := NewStreamTracker()

	key := StreamKey{
		TenantID: "tenant-1",
		SiteID:   "site-A",
		Metric:   "metric-1",
		Source:   "cloud",
	}

	// Circuit breaker abierto
	tracker.SetCircuitBreaker(key, true)

	pusher := NewStatusPusher(config, tracker)
	statuses := pusher.GetCurrentStatus()

	if len(statuses) != 1 {
		t.Fatalf("Expected 1 status, got %d", len(statuses))
	}

	if statuses[0].State != StatePaused.String() {
		t.Errorf("Expected state 'paused', got '%s'", statuses[0].State)
	}
}

func TestStatusPusher_Staleness(t *testing.T) {
	config := DefaultConfig()
	config.StaleThresholdOK = 2       // 2 segundos
	config.StaleThresholdDegraded = 5 // 5 segundos

	tracker := NewStreamTracker()

	key := StreamKey{
		TenantID: "tenant-1",
		SiteID:   "site-A",
		Metric:   "metric-1",
		Source:   "cloud",
	}

	// Simular éxito hace 3 segundos
	tracker.RegisterStream(key)
	pastTime := time.Now().Add(-3 * time.Second)
	kpi := tracker.GetKPIs(key)
	if kpi == nil {
		t.Fatal("Expected KPIs, got nil")
	}

	// Actualizar manualmente el timestamp
	tracker.UpdateSuccess(key, 100)
	// Modificar el KPI directamente para testing
	tracker.mu.Lock()
	tracker.streams[key.Key()].LastSuccessTS = &pastTime
	tracker.mu.Unlock()

	pusher := NewStatusPusher(config, tracker)
	statuses := pusher.GetCurrentStatus()

	if len(statuses) != 1 {
		t.Fatalf("Expected 1 status, got %d", len(statuses))
	}

	// Staleness debería ser ~3 segundos
	if statuses[0].StalenessSec < 2 || statuses[0].StalenessSec > 4 {
		t.Errorf("Expected staleness ~3s, got %d", statuses[0].StalenessSec)
	}

	// Estado debería ser partial (entre 2 y 5 segundos)
	if statuses[0].State != StatePartial.String() {
		t.Errorf("Expected state 'partial' for 3s staleness, got '%s'", statuses[0].State)
	}
}

func TestStatusPusher_StopAndRestart(t *testing.T) {
	config := DefaultConfig()
	config.HeartbeatInterval = 50 * time.Millisecond

	tracker := NewStreamTracker()

	// Registrar un stream para que haya algo que emitir
	key := StreamKey{
		TenantID: "tenant-1",
		SiteID:   "site-A",
		Metric:   "metric-1",
		Source:   "cloud",
	}
	tracker.RegisterStream(key)

	pusher := NewStatusPusher(config, tracker)

	emitCount := 0
	var mu sync.Mutex

	pusher.OnEmit(func(s Status) {
		mu.Lock()
		emitCount++
		mu.Unlock()
	})

	// Iniciar
	ctx := context.Background()
	pusher.Start(ctx)

	// Esperar un poco
	time.Sleep(120 * time.Millisecond)

	// Detener
	pusher.Stop()

	mu.Lock()
	countAfterStop := emitCount
	mu.Unlock()

	if countAfterStop == 0 {
		t.Error("Expected at least one emission before stop")
	}

	// Esperar más tiempo
	time.Sleep(120 * time.Millisecond)

	mu.Lock()
	countAfterWait := emitCount
	mu.Unlock()

	// No debería haber más emisiones después de Stop
	if countAfterWait != countAfterStop {
		t.Errorf("Expected no emissions after Stop, got %d new emissions", countAfterWait-countAfterStop)
	}
}
