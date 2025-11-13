package requester

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/google/uuid"
)

// SequentialRequester implementa Requester con concurrencia=1
type SequentialRequester struct {
	config         Config
	queue          *RequestQueue
	strategy       Strategy
	circuitBreaker *CircuitBreaker
	metrics        *MetricsCollector
	backoff        *BackoffCalculator

	state          State
	resultCallback func(Result)

	ctx    context.Context
	cancel context.CancelFunc
	wg     sync.WaitGroup
	mu     sync.RWMutex

	currentRequest *Request
}

// NewSequentialRequester crea un nuevo requester secuencial
func NewSequentialRequester(config Config, strategy Strategy) *SequentialRequester {
	return &SequentialRequester{
		config:         config,
		queue:          NewRequestQueue(config.MaxQueueSize),
		strategy:       strategy,
		circuitBreaker: NewCircuitBreaker(config),
		metrics:        NewMetricsCollector(),
		backoff:        NewBackoffCalculator(config),
		state:          StateIdle,
	}
}

// Enqueue agrega una solicitud a la cola
func (sr *SequentialRequester) Enqueue(req Request) error {
	sr.mu.RLock()
	state := sr.state
	sr.mu.RUnlock()

	// Verificar estado
	if state == StateStopped {
		return ErrRequesterStopped
	}

	// Validar request
	if err := sr.validateRequest(req); err != nil {
		return err
	}

	// Generar ID si no tiene
	if req.RequestID == "" {
		req.RequestID = uuid.New().String()
	}

	// Timestamp de enqueue
	req.EnqueuedAt = time.Now()

	// Encolar con coalescing según configuración
	return sr.queue.Enqueue(req, sr.config.CoalescingEnabled)
}

// Len retorna el número de solicitudes en cola
func (sr *SequentialRequester) Len() int {
	return sr.queue.Len()
}

// Start inicia el procesamiento con concurrencia=1
func (sr *SequentialRequester) Start(ctx context.Context) error {
	sr.mu.Lock()
	defer sr.mu.Unlock()

	if sr.state == StateRunning {
		return fmt.Errorf("requester already running")
	}

	if sr.strategy == nil {
		return ErrNoStrategy
	}

	sr.ctx, sr.cancel = context.WithCancel(ctx)
	sr.state = StateRunning

	sr.wg.Add(1)
	go sr.processLoop()

	return nil
}

// Stop detiene el procesamiento
func (sr *SequentialRequester) Stop() error {
	sr.mu.Lock()
	defer sr.mu.Unlock()

	if sr.state == StateStopped {
		return nil
	}

	if sr.cancel != nil {
		sr.cancel()
	}

	sr.state = StateStopped
	sr.wg.Wait()

	return nil
}

// OnResult registra un callback para recibir resultados
func (sr *SequentialRequester) OnResult(callback func(Result)) {
	sr.mu.Lock()
	defer sr.mu.Unlock()
	sr.resultCallback = callback
}

// GetMetrics retorna métricas actuales
func (sr *SequentialRequester) GetMetrics() Metrics {
	sr.mu.RLock()
	defer sr.mu.RUnlock()

	snapshot := sr.metrics.GetMetrics()
	queueStats := sr.queue.GetStats()

	return Metrics{
		LastSuccessTS:  snapshot.LastSuccessTS,
		LastErrorTS:    snapshot.LastErrorTS,
		LastLatencyMS:  snapshot.LastLatencyMS,
		InFlight:       snapshot.InFlight,
		QueueLength:    queueStats.Size,
		TotalProcessed: snapshot.TotalProcessed,
		TotalErrors:    snapshot.TotalErrors,
		TotalSuccess:   snapshot.TotalSuccess,
		ConsecErrors:   sr.circuitBreaker.GetConsecutiveErrors(),
		AvgLatencyMS:   snapshot.AvgLatencyMS,
		State:          sr.state,
		CircuitOpen:    sr.circuitBreaker.IsOpen(),
		NextRetryAt:    sr.circuitBreaker.GetNextRetryAt(),
	}
}

// GetState retorna el estado actual
func (sr *SequentialRequester) GetState() State {
	sr.mu.RLock()
	defer sr.mu.RUnlock()
	return sr.state
}

// processLoop es el loop principal de procesamiento (concurrencia=1)
func (sr *SequentialRequester) processLoop() {
	defer sr.wg.Done()

	ticker := time.NewTicker(10 * time.Millisecond) // Poll cada 10ms
	defer ticker.Stop()

	for {
		select {
		case <-sr.ctx.Done():
			return

		case <-ticker.C:
			// Procesar si hay elementos en la cola
			if sr.queue.Len() > 0 {
				sr.processNext()
			}
		}
	}
}

// processNext procesa la siguiente solicitud de la cola
func (sr *SequentialRequester) processNext() {
	// Verificar circuit breaker
	if sr.circuitBreaker.IsOpen() {
		sr.updateState(StatePaused)

		// Emitir resultado de estado "paused"
		sr.emitPausedResult()

		// Esperar hasta que se pueda reintentar
		nextRetry := sr.circuitBreaker.GetNextRetryAt()
		if nextRetry != nil {
			time.Sleep(time.Until(*nextRetry))
		}

		sr.updateState(StateRunning)
		return
	}

	// Dequeue siguiente request
	req, ok := sr.queue.Dequeue()
	if !ok {
		return // Cola vacía
	}

	// Marcar como in-flight
	sr.mu.Lock()
	sr.currentRequest = req
	sr.mu.Unlock()

	sr.metrics.RecordStart()

	// Procesar request
	result := sr.executeRequest(*req)

	// Emitir resultado
	sr.emitResult(result)

	// Actualizar métricas y circuit breaker
	if result.IsSuccess() {
		sr.metrics.RecordSuccess(result.LatencyMS)
		sr.circuitBreaker.RecordSuccess()
	} else {
		sr.metrics.RecordError()
		sr.circuitBreaker.RecordFailure()
	}

	// Limpiar current request
	sr.mu.Lock()
	sr.currentRequest = nil
	sr.mu.Unlock()
}

// executeRequest ejecuta una solicitud usando la estrategia
func (sr *SequentialRequester) executeRequest(req Request) Result {
	startTime := time.Now()

	// Crear contexto con timeout
	ctx, cancel := context.WithTimeout(sr.ctx, sr.config.RequestTimeout)
	defer cancel()

	// Ejecutar estrategia
	payload, err := sr.strategy.Execute(ctx, req)

	latencyMS := time.Since(startTime).Milliseconds()

	result := Result{
		TenantID:    req.TenantID,
		SiteID:      req.SiteID,
		CageID:      req.CageID,
		Metric:      req.Metric,
		Source:      req.Source,
		LatencyMS:   latencyMS,
		TsRange:     req.TimeRange,
		Payload:     payload,
		Err:         err,
		CompletedAt: time.Now(),
		RequestID:   req.RequestID,
	}

	if err != nil {
		result.ErrorMsg = err.Error()
	}

	return result
}

// emitResult emite un resultado a través del callback
func (sr *SequentialRequester) emitResult(result Result) {
	sr.mu.RLock()
	callback := sr.resultCallback
	sr.mu.RUnlock()

	if callback != nil {
		// Ejecutar callback en goroutine para no bloquear el loop
		go callback(result)
	}
}

// emitPausedResult emite un resultado indicando que el requester está pausado
func (sr *SequentialRequester) emitPausedResult() {
	sr.mu.RLock()
	callback := sr.resultCallback
	sr.mu.RUnlock()

	if callback != nil {
		result := Result{
			Err:         ErrRequesterPaused,
			ErrorMsg:    "requester paused due to circuit breaker",
			CompletedAt: time.Now(),
		}
		go callback(result)
	}
}

// updateState actualiza el estado del requester
func (sr *SequentialRequester) updateState(newState State) {
	sr.mu.Lock()
	defer sr.mu.Unlock()
	sr.state = newState
}

// validateRequest valida una solicitud
func (sr *SequentialRequester) validateRequest(req Request) error {
	if req.TenantID == "" {
		return fmt.Errorf("%w: tenant_id is required", ErrInvalidRequest)
	}
	if req.SiteID == "" {
		return fmt.Errorf("%w: site_id is required", ErrInvalidRequest)
	}
	if req.Metric == "" {
		return fmt.Errorf("%w: metric is required", ErrInvalidRequest)
	}
	if req.TimeRange.From.IsZero() || req.TimeRange.To.IsZero() {
		return fmt.Errorf("%w: time_range is required", ErrInvalidRequest)
	}
	if req.TimeRange.From.After(req.TimeRange.To) {
		return fmt.Errorf("%w: time_range.from must be before time_range.to", ErrInvalidRequest)
	}
	return nil
}

// SetStrategy cambia la estrategia en runtime
func (sr *SequentialRequester) SetStrategy(strategy Strategy) {
	sr.mu.Lock()
	defer sr.mu.Unlock()
	sr.strategy = strategy
}

// GetQueueStats retorna estadísticas de la cola
func (sr *SequentialRequester) GetQueueStats() QueueStats {
	return sr.queue.GetStats()
}

// GetCurrentRequest retorna la solicitud actual en procesamiento
func (sr *SequentialRequester) GetCurrentRequest() *Request {
	sr.mu.RLock()
	defer sr.mu.RUnlock()
	return sr.currentRequest
}

// Clear limpia la cola (útil para testing)
func (sr *SequentialRequester) Clear() {
	sr.queue.Clear()
}

// ResetCircuitBreaker reinicia el circuit breaker manualmente
func (sr *SequentialRequester) ResetCircuitBreaker() {
	sr.circuitBreaker.Reset()
	sr.updateState(StateRunning)
}

// GetQueueSize retorna el tamaño actual de la cola
func (sr *SequentialRequester) GetQueueSize() int {
	return sr.queue.Len()
}
