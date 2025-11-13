# Sequential Requester Module

## Descripción

El módulo `requester` implementa una cola de consultas **SECUENCIAL** (concurrencia=1) para hacer requests a proveedores de datos externos (ScaleAQ Cloud, ProcessAPI) con las siguientes características:

- **Concurrencia=1**: Solo una solicitud activa por proveedor-sitio
- **Cola de prioridades**: Alta, Normal, Baja
- **Coalescing**: Requests duplicados con la misma key se fusionan
- **Circuit Breaker**: Pausa automática tras N errores consecutivos
- **Backoff exponencial**: 1m → 2m → 5m ante errores
- **Métricas**: Latencias, tasas de éxito/error, estado

## Arquitectura

```
┌──────────────────────────────────────────────────────────┐
│                   SequentialRequester                     │
│  ┌────────────┐  ┌──────────────┐  ┌────────────────┐   │
│  │ Priority   │→ │ Circuit      │→ │ Strategy       │   │
│  │ Queue      │  │ Breaker      │  │ (ScaleAQ/API)  │   │
│  └────────────┘  └──────────────┘  └────────────────┘   │
│       ↓                 ↓                   ↓            │
│  ┌────────────────────────────────────────────────┐     │
│  │         Process Loop (concurrency=1)           │     │
│  └────────────────────────────────────────────────┘     │
│       ↓                                                   │
│  ┌────────────────────────────────────────────────┐     │
│  │      Metrics Collector & Result Callback       │     │
│  └────────────────────────────────────────────────┘     │
└──────────────────────────────────────────────────────────┘
```

## Componentes

### 1. RequestQueue (queue.go)

Cola de prioridades con heap y coalescing:

```go
queue := NewRequestQueue(maxSize)

req := Request{
    TenantID:  "tenant-1",
    SiteID:    "site-A",
    Metric:    "feeding.appetite",
    TimeRange: TimeRange{From: start, To: end},
    Priority:  PriorityHigh,
    Source:    SourceCloud,
}

// Enqueue con coalescing
queue.Enqueue(req, true)

// Dequeue por prioridad (High > Normal > Low)
item, ok := queue.Dequeue()
```

**Coalescing**: Requests con la misma `Key()` (tenant+site+metric+source) se fusionan, manteniendo solo la más reciente.

### 2. CircuitBreaker (circuit_breaker.go)

Protección ante fallos consecutivos:

```go
cb := NewCircuitBreaker(config)

// Registrar resultados
cb.RecordSuccess()   // Resetea errores consecutivos
cb.RecordFailure()   // Incrementa contador

// Verificar estado
if cb.IsOpen() {
    // Esperar hasta nextRetryAt
    next := cb.GetNextRetryAt()
    time.Sleep(time.Until(*next))
}
```

**Estados**:

- **Closed**: Operación normal
- **Open**: Pausado tras MaxConsecutiveErrors
- **Half-Open**: Permite 1 reintento después de CircuitPauseDuration

### 3. Strategy (strategy.go)

Interfaz para proveedores de datos:

```go
type Strategy interface {
    Execute(ctx context.Context, req Request) (json.RawMessage, error)
    Name() string
    HealthCheck(ctx context.Context) error
}
```

**Estrategias disponibles**:

- `MockStrategy`: Para testing
- `NoOpStrategy`: Placeholder sin operación
- `ScaleAQCloudStrategy`: TODO - Conectar a `/time-series/retrieve`
- `ProcessAPIStrategy`: TODO - Conectar a API local

**Ejemplo de implementación personalizada**:

```go
type MyStrategy struct {
    client *http.Client
    endpoint string
}

func (s *MyStrategy) Execute(ctx context.Context, req Request) (json.RawMessage, error) {
    // Construir URL con req.Metric, req.TimeRange, etc.
    url := fmt.Sprintf("%s?metric=%s&from=%d&to=%d",
        s.endpoint, req.Metric, req.TimeRange.From.Unix(), req.TimeRange.To.Unix())

    // Hacer HTTP request
    resp, err := s.client.Get(url)
    if err != nil {
        return nil, err
    }
    defer resp.Body.Close()

    // Leer payload
    payload, err := io.ReadAll(resp.Body)
    return json.RawMessage(payload), err
}

func (s *MyStrategy) Name() string {
    return "my-custom-strategy"
}

func (s *MyStrategy) HealthCheck(ctx context.Context) error {
    resp, err := s.client.Head(s.endpoint)
    if err != nil {
        return err
    }
    if resp.StatusCode != 200 {
        return fmt.Errorf("unhealthy: %d", resp.StatusCode)
    }
    return nil
}
```

### 4. SequentialRequester (requester.go)

Orquestador principal:

```go
config := DefaultConfig()
config.RequestTimeout = 30 * time.Second
config.MaxConsecutiveErrors = 3
config.CircuitPauseDuration = 5 * time.Minute

strategy := NewScaleAQCloudStrategy(endpoint, apiKey)
requester := NewSequentialRequester(config, strategy)

// Iniciar procesamiento
ctx := context.Background()
requester.Start(ctx)
defer requester.Stop()

// Registrar callback para resultados
requester.OnResult(func(result Result) {
    if result.IsSuccess() {
        log.Printf("✓ %s: %d bytes, %dms", result.Metric, len(result.Payload), result.LatencyMS)
    } else {
        log.Printf("✗ %s: %v", result.Metric, result.Err)
    }
})

// Enqueue requests
req := Request{
    TenantID:  "tenant-1",
    SiteID:    "site-A",
    Metric:    "climate.temperature",
    TimeRange: TimeRange{
        From: time.Now().Add(-24 * time.Hour),
        To:   time.Now(),
    },
    Priority: PriorityHigh,
    Source:   SourceCloud,
}

if err := requester.Enqueue(req); err != nil {
    log.Printf("Enqueue failed: %v", err)
}
```

## Configuración

```go
type Config struct {
    QueueSize              int           // Tamaño máximo de cola (default: 1000)
    RequestTimeout         time.Duration // Timeout por request (default: 30s)
    MaxConsecutiveErrors   int           // Errores antes de pausar (default: 5)
    CircuitPauseDuration   time.Duration // Duración de pausa (default: 5min)
    BackoffInitial         time.Duration // Backoff inicial (default: 1min)
    BackoffStep2           time.Duration // Backoff paso 2 (default: 2min)
    BackoffStep3           time.Duration // Backoff paso 3 (default: 5min)
}

config := DefaultConfig() // Valores por defecto
```

## Métricas

```go
metrics := requester.GetMetrics()

fmt.Printf(`
Queue Size:       %d
Total Processed:  %d
Total Success:    %d
Total Errors:     %d
Consec. Errors:   %d
Avg Latency:      %.1fms
Last Latency:     %dms
State:            %s
Circuit Open:     %v
Next Retry:       %v
`,
    metrics.QueueSize,
    metrics.TotalProcessed,
    metrics.TotalSuccess,
    metrics.TotalErrors,
    metrics.ConsecErrors,
    metrics.AvgLatencyMS,
    metrics.LastLatencyMS,
    metrics.State,
    metrics.CircuitOpen,
    metrics.NextRetryAt,
)
```

**Estados posibles**:

- `idle`: Sin actividad
- `running`: Procesando requests
- `paused`: Circuit breaker abierto

## Prioridades

```go
const (
    PriorityLow    Priority = "low"     // Backfill histórico
    PriorityNormal Priority = "normal"  // Consultas regulares
    PriorityHigh   Priority = "high"    // Alertas, dashboards
)
```

La cola procesa siempre High → Normal → Low (FIFO dentro de cada nivel).

## Coalescing y Deduplicación

Requests con la misma **key** se fusionan:

```go
func (r Request) Key() string {
    return fmt.Sprintf("%s:%s:%s:%s", r.TenantID, r.SiteID, r.Metric, r.Source)
}
```

**Ejemplo**:

```go
// Request 1 (enqueued)
{TenantID: "tenant-1", SiteID: "site-A", Metric: "feeding.appetite", TimeRange: [08:00-09:00]}

// Request 2 (enqueued 5s después)
{TenantID: "tenant-1", SiteID: "site-A", Metric: "feeding.appetite", TimeRange: [08:00-10:00]}

// Resultado: Solo Request 2 se procesa (coalesced)
```

Esto previene saturar el proveedor con requests duplicados mientras uno está pendiente.

## Testing

```bash
go test -v ./internal/queue/requester/...
```

**Tests disponibles**:

- `TestRequestQueue_Enqueue`: Enqueue básico
- `TestRequestQueue_Coalescing`: Fusión de requests duplicados
- `TestRequestQueue_Priority`: Orden High > Normal > Low
- `TestRequestQueue_FullQueue`: Límite de cola
- `TestCircuitBreaker_Open`: Apertura tras N errores
- `TestCircuitBreaker_Recovery`: Reseteo tras éxito
- `TestSequentialRequester_Basic`: Flujo completo básico
- `TestSequentialRequester_Sequential`: Concurrencia=1 verificada
- `TestSequentialRequester_CircuitBreaker`: Pausa automática
- `TestMetricsCollector_Tracking`: Latencias y contadores
- `TestBackoffCalculator`: Cálculo de backoff

## Backoff Exponencial

Ante errores consecutivos:

| Errores | Backoff |
| ------- | ------- |
| 1       | 1 min   |
| 2       | 2 min   |
| 3+      | 5 min   |

Después de `CircuitPauseDuration`, se permite **1 reintento** (half-open). Si falla, vuelve a pausar. Si tiene éxito, resetea los errores.

## Ejemplo Completo

```go
package main

import (
    "context"
    "log"
    "time"
    "omniapi/internal/queue/requester"
)

func main() {
    // Configurar
    config := requester.DefaultConfig()
    config.MaxConsecutiveErrors = 3
    config.CircuitPauseDuration = 2 * time.Minute

    // Crear estrategia (por ahora mock, luego ScaleAQ)
    strategy := requester.NewMockStrategy("test-provider")

    // Crear requester
    req := requester.NewSequentialRequester(config, strategy)

    // Iniciar
    ctx := context.Background()
    req.Start(ctx)
    defer req.Stop()

    // Callback para resultados
    req.OnResult(func(result requester.Result) {
        if result.IsSuccess() {
            log.Printf("✓ %s/%s: %d bytes", result.TenantID, result.Metric, len(result.Payload))
        } else {
            log.Printf("✗ %s/%s: %v", result.TenantID, result.Metric, result.Err)
        }
    })

    // Enqueue requests
    requests := []requester.Request{
        {
            TenantID:  "tenant-1",
            SiteID:    "site-A",
            Metric:    "feeding.appetite",
            TimeRange: requester.TimeRange{
                From: time.Now().Add(-1 * time.Hour),
                To:   time.Now(),
            },
            Priority: requester.PriorityHigh,
            Source:   requester.SourceCloud,
        },
        {
            TenantID:  "tenant-1",
            SiteID:    "site-B",
            Metric:    "climate.temperature",
            TimeRange: requester.TimeRange{
                From: time.Now().Add(-24 * time.Hour),
                To:   time.Now(),
            },
            Priority: requester.PriorityNormal,
            Source:   requester.SourceProcessAPI,
        },
    }

    for _, r := range requests {
        if err := req.Enqueue(r); err != nil {
            log.Printf("Failed to enqueue: %v", err)
        }
    }

    // Esperar procesamiento
    time.Sleep(5 * time.Second)

    // Imprimir métricas
    metrics := req.GetMetrics()
    log.Printf("Metrics: processed=%d, errors=%d, avg_latency=%.1fms",
        metrics.TotalProcessed, metrics.TotalErrors, metrics.AvgLatencyMS)
}
```

## TODOs

- [ ] Implementar `ScaleAQCloudStrategy.Execute()` con llamada real a `/time-series/retrieve`
- [ ] Implementar `ProcessAPIStrategy.Execute()` con llamada a API local
- [ ] Agregar reintentos con exponential backoff dentro de cada request (antes de fallar)
- [ ] Persistir cola en disco para sobrevivir reinicios
- [ ] Métricas Prometheus (histogramas de latencia, gauges de queue size, etc.)
- [ ] Health check endpoint HTTP para monitoreo

## Integración con Router

El módulo `requester` se puede integrar con `internal/router` para:

1. **Backfill on-demand**: Cuando un cliente se suscribe a una métrica que no está en caché
2. **Retry de eventos perdidos**: Si el conector local falla
3. **Validación cruzada**: Comparar datos locales vs cloud

```go
// En router, cuando no hay datos en caché:
if !hasLocalData(metric) {
    req := requester.Request{
        TenantID:  subscription.TenantID,
        SiteID:    subscription.SiteID,
        Metric:    metric,
        TimeRange: requester.TimeRange{From: time.Now().Add(-1*time.Hour), To: time.Now()},
        Priority:  requester.PriorityHigh,
        Source:    requester.SourceCloud,
    }
    cloudRequester.Enqueue(req)
}
```
