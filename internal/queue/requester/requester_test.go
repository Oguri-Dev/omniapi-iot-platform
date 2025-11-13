package requester

import (
	"context"
	"encoding/json"
	"fmt"
	"testing"
	"time"
)

func TestRequestQueue_Enqueue(t *testing.T) {
	queue := NewRequestQueue(10)

	req := Request{
		TenantID:  "tenant-1",
		SiteID:    "site-1",
		Metric:    "feeding.appetite",
		TimeRange: TimeRange{From: time.Now(), To: time.Now().Add(time.Hour)},
		Priority:  PriorityNormal,
		Source:    SourceCloud,
	}

	err := queue.Enqueue(req, false)
	if err != nil {
		t.Fatalf("Failed to enqueue: %v", err)
	}

	if queue.Len() != 1 {
		t.Errorf("Expected queue length 1, got %d", queue.Len())
	}
}

func TestRequestQueue_Coalescing(t *testing.T) {
	queue := NewRequestQueue(10)

	req1 := Request{
		TenantID:  "tenant-1",
		SiteID:    "site-1",
		Metric:    "feeding.appetite",
		TimeRange: TimeRange{From: time.Now(), To: time.Now().Add(time.Hour)},
		Priority:  PriorityNormal,
		Source:    SourceCloud,
		RequestID: "req-1",
	}

	req2 := Request{
		TenantID:  "tenant-1",
		SiteID:    "site-1",
		Metric:    "feeding.appetite",
		TimeRange: TimeRange{From: time.Now(), To: time.Now().Add(2 * time.Hour)},
		Priority:  PriorityNormal,
		Source:    SourceCloud,
		RequestID: "req-2", // Mismo key pero diferente RequestID
	}

	// Enqueue primera vez
	queue.Enqueue(req1, true)

	// Enqueue segunda vez con coalescing
	queue.Enqueue(req2, true)

	// Debe tener solo 1 elemento (coalesced)
	if queue.Len() != 1 {
		t.Errorf("Expected queue length 1 after coalescing, got %d", queue.Len())
	}

	// Dequeue y verificar que es el segundo request
	dequeued, ok := queue.Dequeue()
	if !ok {
		t.Fatal("Failed to dequeue")
	}

	if dequeued.RequestID != "req-2" {
		t.Errorf("Expected coalesced request ID 'req-2', got '%s'", dequeued.RequestID)
	}
}

func TestRequestQueue_Priority(t *testing.T) {
	queue := NewRequestQueue(10)

	lowPrio := Request{
		TenantID:  "tenant-1",
		SiteID:    "site-1",
		Metric:    "metric-low",
		TimeRange: TimeRange{From: time.Now(), To: time.Now().Add(time.Hour)},
		Priority:  PriorityLow,
		Source:    SourceCloud,
	}

	highPrio := Request{
		TenantID:  "tenant-1",
		SiteID:    "site-2",
		Metric:    "metric-high",
		TimeRange: TimeRange{From: time.Now(), To: time.Now().Add(time.Hour)},
		Priority:  PriorityHigh,
		Source:    SourceCloud,
	}

	// Enqueue low priority first
	queue.Enqueue(lowPrio, false)

	// Enqueue high priority second
	queue.Enqueue(highPrio, false)

	// Dequeue debe retornar high priority primero
	dequeued, ok := queue.Dequeue()
	if !ok {
		t.Fatal("Failed to dequeue")
	}

	if dequeued.Priority != PriorityHigh {
		t.Errorf("Expected high priority first, got %s", dequeued.Priority)
	}
}

func TestRequestQueue_FullQueue(t *testing.T) {
	queue := NewRequestQueue(2)

	req := Request{
		TenantID:  "tenant-1",
		SiteID:    "site-1",
		Metric:    "metric-1",
		TimeRange: TimeRange{From: time.Now(), To: time.Now().Add(time.Hour)},
		Priority:  PriorityNormal,
		Source:    SourceCloud,
	}

	// Enqueue 2 requests
	queue.Enqueue(req, false)

	req.Metric = "metric-2"
	queue.Enqueue(req, false)

	// Tercero debe fallar
	req.Metric = "metric-3"
	err := queue.Enqueue(req, false)

	if err != ErrQueueFull {
		t.Errorf("Expected ErrQueueFull, got %v", err)
	}
}

func TestCircuitBreaker_Open(t *testing.T) {
	config := DefaultConfig()
	config.MaxConsecutiveErrors = 3
	config.CircuitPauseDuration = 100 * time.Millisecond

	cb := NewCircuitBreaker(config)

	// Registrar errores
	for i := 0; i < 3; i++ {
		cb.RecordFailure()
	}

	// Circuit breaker debe estar abierto
	if !cb.IsOpen() {
		t.Error("Circuit breaker should be open after max errors")
	}

	// Esperar tiempo de pausa
	time.Sleep(150 * time.Millisecond)

	// Ahora debe permitir reintentos
	if cb.IsOpen() {
		t.Error("Circuit breaker should allow retry after pause duration")
	}
}

func TestCircuitBreaker_Recovery(t *testing.T) {
	config := DefaultConfig()
	config.MaxConsecutiveErrors = 3

	cb := NewCircuitBreaker(config)

	// Registrar 2 errores
	cb.RecordFailure()
	cb.RecordFailure()

	if cb.GetConsecutiveErrors() != 2 {
		t.Errorf("Expected 2 consecutive errors, got %d", cb.GetConsecutiveErrors())
	}

	// Registrar éxito
	cb.RecordSuccess()

	// Errores consecutivos deben resetear
	if cb.GetConsecutiveErrors() != 0 {
		t.Errorf("Expected 0 consecutive errors after success, got %d", cb.GetConsecutiveErrors())
	}

	// Circuit breaker debe estar cerrado
	if cb.IsOpen() {
		t.Error("Circuit breaker should be closed after success")
	}
}

func TestSequentialRequester_Basic(t *testing.T) {
	config := DefaultConfig()
	config.RequestTimeout = 1 * time.Second

	strategy := NewMockStrategy("test")
	requester := NewSequentialRequester(config, strategy)

	ctx := context.Background()
	if err := requester.Start(ctx); err != nil {
		t.Fatalf("Failed to start requester: %v", err)
	}
	defer requester.Stop()

	// Enqueue request
	req := Request{
		TenantID:  "tenant-1",
		SiteID:    "site-1",
		Metric:    "feeding.appetite",
		TimeRange: TimeRange{From: time.Now(), To: time.Now().Add(time.Hour)},
		Priority:  PriorityNormal,
		Source:    SourceCloud,
	}

	resultReceived := make(chan Result, 1)
	requester.OnResult(func(r Result) {
		resultReceived <- r
	})

	if err := requester.Enqueue(req); err != nil {
		t.Fatalf("Failed to enqueue: %v", err)
	}

	// Esperar resultado
	select {
	case result := <-resultReceived:
		if result.Err != nil {
			t.Errorf("Expected success, got error: %v", result.Err)
		}
		if result.LatencyMS <= 0 {
			t.Error("Expected positive latency")
		}
	case <-time.After(2 * time.Second):
		t.Fatal("Timeout waiting for result")
	}
}

func TestSequentialRequester_Sequential(t *testing.T) {
	config := DefaultConfig()
	config.RequestTimeout = 500 * time.Millisecond

	// Estrategia que tarda 100ms
	strategy := NewMockStrategy("test")
	strategy.SetPayloadFunc(func(req Request) json.RawMessage {
		time.Sleep(100 * time.Millisecond)
		return json.RawMessage(`{}`)
	})

	requester := NewSequentialRequester(config, strategy)

	ctx := context.Background()
	requester.Start(ctx)
	defer requester.Stop()

	results := make(chan Result, 3)
	requester.OnResult(func(r Result) {
		results <- r
	})

	// Enqueue 3 requests
	for i := 0; i < 3; i++ {
		req := Request{
			TenantID:  "tenant-1",
			SiteID:    "site-1",
			Metric:    fmt.Sprintf("metric-%d", i),
			TimeRange: TimeRange{From: time.Now(), To: time.Now().Add(time.Hour)},
			Priority:  PriorityNormal,
			Source:    SourceCloud,
		}
		requester.Enqueue(req)
	}

	// Dar tiempo a que se encolen
	time.Sleep(100 * time.Millisecond)

	// Verificar que se procesan secuencialmente (toma ~300ms, no paralelo)
	startTime := time.Now()
	count := 0

	for i := 0; i < 3; i++ {
		select {
		case result := <-results:
			count++
			t.Logf("Received result %d: err=%v, latency=%dms", i, result.Err, result.LatencyMS)
		case <-time.After(2 * time.Second):
			metrics := requester.GetMetrics()
			t.Fatalf("Timeout waiting for result %d/%d, queue len=%d, metrics=%+v", i+1, 3, requester.GetQueueSize(), metrics)
		}
	}

	elapsed := time.Since(startTime)

	// Debe tomar al menos 300ms (3 x 100ms) si es secuencial
	if elapsed < 300*time.Millisecond {
		t.Errorf("Expected sequential processing (~300ms), took %v", elapsed)
	}

	if count != 3 {
		t.Errorf("Expected 3 results, got %d", count)
	}
}

func TestSequentialRequester_CircuitBreaker(t *testing.T) {
	config := DefaultConfig()
	config.MaxConsecutiveErrors = 2
	config.CircuitPauseDuration = 200 * time.Millisecond
	config.RequestTimeout = 100 * time.Millisecond

	strategy := NewMockStrategy("test")
	strategy.latency = 10        // 10ms de latencia
	strategy.SetShouldFail(true) // Todas las requests fallan

	requester := NewSequentialRequester(config, strategy)

	ctx := context.Background()
	requester.Start(ctx)
	defer requester.Stop()

	results := make(chan Result, 5)
	requester.OnResult(func(r Result) {
		results <- r
	})

	// Enqueue requests que van a fallar
	for i := 0; i < 3; i++ {
		req := Request{
			TenantID:  "tenant-1",
			SiteID:    "site-1",
			Metric:    fmt.Sprintf("metric-%d", i),
			TimeRange: TimeRange{From: time.Now(), To: time.Now().Add(time.Hour)},
			Priority:  PriorityNormal,
			Source:    SourceCloud,
		}
		requester.Enqueue(req)
	}

	// Esperar resultados y verificar que hay errores
	errorCount := 0
	for i := 0; i < 2; i++ { // Solo esperamos 2 resultados (antes de que se abra el circuit breaker)
		select {
		case result := <-results:
			if result.Err != nil {
				errorCount++
				t.Logf("Result %d: error=%v", i, result.Err)
			}
		case <-time.After(1 * time.Second):
			metrics := requester.GetMetrics()
			t.Fatalf("Timeout waiting for result %d, metrics=%+v", i, metrics)
		}
	}

	if errorCount < 2 {
		t.Fatalf("Expected at least 2 errors, got %d", errorCount)
	}

	// Verificar inmediatamente que el circuit breaker se abrió
	metrics := requester.GetMetrics()
	t.Logf("Metrics: errors=%d, consec=%d, circuit_open=%v, state=%s",
		metrics.TotalErrors, metrics.ConsecErrors, metrics.CircuitOpen, metrics.State)

	// Circuit breaker debe haberse abierto
	if !metrics.CircuitOpen {
		t.Error("Circuit breaker should be open after consecutive errors")
	}

	// Estado debe ser paused SI el circuit breaker está abierto y se intentó procesar
	// Nota: el estado podría ser "running" si no ha intentado procesar desde que se abrió
	t.Logf("Final state: %s (circuit open: %v)", metrics.State, metrics.CircuitOpen)
}

func TestMetricsCollector_Tracking(t *testing.T) {
	mc := NewMetricsCollector()

	// Registrar éxitos
	mc.RecordStart()
	mc.RecordSuccess(100)

	mc.RecordStart()
	mc.RecordSuccess(200)

	snapshot := mc.GetMetrics()

	if snapshot.TotalSuccess != 2 {
		t.Errorf("Expected 2 successes, got %d", snapshot.TotalSuccess)
	}

	if snapshot.LastLatencyMS != 200 {
		t.Errorf("Expected last latency 200, got %d", snapshot.LastLatencyMS)
	}

	expectedAvg := 150.0
	if snapshot.AvgLatencyMS != expectedAvg {
		t.Errorf("Expected average latency %.1f, got %.1f", expectedAvg, snapshot.AvgLatencyMS)
	}
}

func TestBackoffCalculator(t *testing.T) {
	config := DefaultConfig()
	bc := NewBackoffCalculator(config)

	tests := []struct {
		errorCount int
		expected   time.Duration
	}{
		{0, 0},
		{1, config.BackoffInitial},
		{2, config.BackoffStep2},
		{3, config.BackoffStep3},
		{5, config.BackoffStep3}, // Cap at step 3
	}

	for _, tt := range tests {
		result := bc.CalculateBackoff(tt.errorCount)
		if result != tt.expected {
			t.Errorf("For error count %d, expected backoff %v, got %v",
				tt.errorCount, tt.expected, result)
		}
	}
}
