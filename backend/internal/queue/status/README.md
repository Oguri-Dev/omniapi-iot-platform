# Status Heartbeat Module

## Descripción

El módulo `status` emite **heartbeats de estado periódicos** para cada stream de datos conocido, independientemente de si hubo datos nuevos o no. Esto permite al frontend/dashboard monitorear la salud de los streams en tiempo real.

## Características

- ✅ **Heartbeats periódicos**: Emite estado cada N segundos (configurable, default 10s)
- ✅ **Sin llamadas upstream**: Solo empaqueta y emite el estado actual
- ✅ **Estados calculados**: `ok`, `partial`, `degraded`, `failing`, `paused`
- ✅ **Staleness tracking**: Mide cuánto tiempo sin datos exitosos
- ✅ **KPIs por stream**: Latencia, errores, in-flight, circuit breaker
- ✅ **Callback pattern**: Entrega hacia router/broker vía `OnEmit`

## Arquitectura

```
┌──────────────────────────────────────────────────┐
│              StatusPusher                        │
│  ┌──────────────┐      ┌──────────────────┐     │
│  │ Ticker       │  →   │ StreamTracker    │     │
│  │ (10s)        │      │ (KPIs por stream)│     │
│  └──────────────┘      └──────────────────┘     │
│         ↓                       ↓                │
│  ┌──────────────────────────────────────┐       │
│  │  buildStatus()                        │       │
│  │  - Calcular staleness                 │       │
│  │  - Determinar estado                  │       │
│  │  - Empaquetar Status                  │       │
│  └──────────────────────────────────────┘       │
│         ↓                                        │
│  ┌──────────────────────────────────────┐       │
│  │  OnEmit callback                      │       │
│  │  → Router/Broker → WebSocket          │       │
│  └──────────────────────────────────────┘       │
└──────────────────────────────────────────────────┘
```

## Tipos Principales

### Status

Representa el estado de un stream en un momento dado:

```go
type Status struct {
    TenantID string
    SiteID   string
    CageID   *string // Opcional

    Metric string
    Source string // "cloud"|"processapi"|"derived"

    // KPIs de salud
    LastSuccessTS *time.Time // Último éxito
    LastErrorTS   *time.Time // Último error
    LastErrorMsg  *string    // Mensaje del último error
    LastLatencyMS *int64     // Latencia del último request (ms)

    InFlight bool // Request en progreso

    // Métricas calculadas
    StalenessSec int64  // Segundos desde last_success
    State        string // "ok"|"partial"|"degraded"|"failing"|"paused"
    Notes        *string

    EmittedAt time.Time // Timestamp de emisión
}
```

### Estados Posibles

| Estado     | Descripción                            | Criterio                                    |
| ---------- | -------------------------------------- | ------------------------------------------- |
| `ok`       | Funcionando correctamente              | Datos frescos (<60s), sin errores recientes |
| `partial`  | Funcionando pero con problemas menores | Datos frescos pero con algunos errores      |
| `degraded` | Datos viejos o errores frecuentes      | Staleness >60s y <300s, o errores múltiples |
| `failing`  | Fallas consecutivas                    | ≥3 errores consecutivos                     |
| `paused`   | Circuit breaker abierto                | Circuit breaker activo                      |

## Uso Básico

### 1. Crear y Configurar

```go
package main

import (
    "context"
    "log"
    "time"
    "omniapi/internal/queue/status"
)

func main() {
    // Configurar
    config := status.DefaultConfig()
    config.HeartbeatInterval = 5 * time.Second  // Heartbeat cada 5s
    config.StaleThresholdOK = 60                // <60s es OK
    config.StaleThresholdDegraded = 300         // >300s es degraded
    config.MaxConsecutiveErrors = 3

    // Crear tracker y pusher
    tracker := status.NewStreamTracker()
    pusher := status.NewStatusPusher(config, tracker)

    // Registrar callback
    pusher.OnEmit(func(s status.Status) {
        log.Printf("[HEARTBEAT] %s/%s/%s: state=%s, staleness=%ds, in_flight=%v",
            s.TenantID, s.SiteID, s.Metric, s.State, s.StalenessSec, s.InFlight)
    })

    // Iniciar emisión
    ctx := context.Background()
    pusher.Start(ctx)
    defer pusher.Stop()

    // Tu aplicación sigue corriendo...
    select {}
}
```

### 2. Registrar Streams

```go
// Registrar un stream nuevo
key := status.StreamKey{
    TenantID: "tenant-1",
    SiteID:   "site-A",
    Metric:   "feeding.appetite",
    Source:   "cloud",
}
tracker.RegisterStream(key)
```

### 3. Actualizar KPIs

```go
// Cuando un request tiene éxito
tracker.UpdateSuccess(key, 150) // 150ms de latencia

// Cuando un request falla
tracker.UpdateError(key, "connection timeout")

// Marcar como procesando
tracker.MarkInFlight(key, true)

// Actualizar circuit breaker
tracker.SetCircuitBreaker(key, true) // Abierto

// Agregar notas
tracker.SetNotes(key, "Migrating to new API")
```

### 4. Obtener Estado Actual

```go
// Obtener estado de todos los streams
statuses := pusher.GetCurrentStatus()

for _, s := range statuses {
    fmt.Printf("%s/%s: %s (staleness: %ds)\n",
        s.TenantID, s.Metric, s.State, s.StalenessSec)
}
```

## Integración con Requester

El `StatusPusher` puede leer KPIs del `SequentialRequester` para actualizar el estado:

```go
// En el callback del requester
requester.OnResult(func(result requester.Result) {
    key := status.StreamKey{
        TenantID: result.TenantID,
        SiteID:   result.SiteID,
        Metric:   result.Metric,
        Source:   result.Source,
    }

    if result.IsSuccess() {
        tracker.UpdateSuccess(key, result.LatencyMS)
    } else {
        tracker.UpdateError(key, result.ErrorMsg)
    }

    // Actualizar circuit breaker
    metrics := requester.GetMetrics()
    tracker.SetCircuitBreaker(key, metrics.CircuitOpen)
})
```

## Integración con Router

El `StatusPusher` entrega heartbeats al router vía callback:

```go
// En el router
pusher.OnEmit(func(s status.Status) {
    // Construir evento de status
    event := connectors.CanonicalEvent{
        EventType: "status.heartbeat",
        Timestamp: s.EmittedAt,
        TenantID:  s.TenantID,
        SiteID:    s.SiteID,
        Kind:      "status",
        Data: map[string]interface{}{
            "metric":         s.Metric,
            "source":         s.Source,
            "state":          s.State,
            "staleness_sec":  s.StalenessSec,
            "in_flight":      s.InFlight,
            "last_success":   s.LastSuccessTS,
            "last_error":     s.LastErrorTS,
            "last_error_msg": s.LastErrorMsg,
            "last_latency":   s.LastLatencyMS,
        },
    }

    // Enrutar a clientes suscritos
    router.RouteEvent(event)
})
```

## Configuración

```go
type Config struct {
    // Frecuencia de emisión de heartbeats
    HeartbeatInterval time.Duration // Default: 10s

    // Umbral de staleness para estado OK (segundos)
    StaleThresholdOK int64 // Default: 60s

    // Umbral de staleness para estado degraded (segundos)
    StaleThresholdDegraded int64 // Default: 300s (5min)

    // Errores consecutivos antes de marcar como failing
    MaxConsecutiveErrors int // Default: 3
}

// Usar valores por defecto
config := status.DefaultConfig()

// O personalizar
config := status.Config{
    HeartbeatInterval:      5 * time.Second,
    StaleThresholdOK:       30,
    StaleThresholdDegraded: 180,
    MaxConsecutiveErrors:   5,
}
```

## Determinación de Estado

El algoritmo para determinar el estado es:

```go
func determineState(kpi StreamKPIs, stalenessSec int64) string {
    // 1. Circuit breaker abierto → paused
    if kpi.CircuitBreakerOpen {
        return "paused"
    }

    // 2. Muchos errores consecutivos → failing
    if kpi.ConsecutiveErrors >= MaxConsecutiveErrors {
        return "failing"
    }

    // 3. Nunca tuvo éxito pero tiene errores → failing
    if kpi.LastSuccessTS == nil && kpi.LastErrorTS != nil {
        return "failing"
    }

    // 4. Nunca tuvo éxito ni errores → partial
    if kpi.LastSuccessTS == nil && kpi.LastErrorTS == nil {
        return "partial"
    }

    // 5. Datos muy viejos (>StaleThresholdDegraded) → degraded
    if stalenessSec > StaleThresholdDegraded {
        return "degraded"
    }

    // 6. Datos algo viejos (>StaleThresholdOK)
    if stalenessSec > StaleThresholdOK {
        if kpi.ConsecutiveErrors > 0 {
            return "degraded"
        }
        return "partial"
    }

    // 7. Datos frescos con errores → partial
    if kpi.ConsecutiveErrors > 0 {
        return "partial"
    }

    // 8. Todo OK
    return "ok"
}
```

## Ejemplo Frontend (WebSocket)

```javascript
// Cliente WebSocket recibiendo heartbeats
ws.onmessage = (event) => {
  const msg = JSON.parse(event.data)

  if (msg.event_type === 'status.heartbeat') {
    const data = msg.data

    // Actualizar UI con indicador de estado
    updateStreamStatus(data.metric, {
      state: data.state, // "ok"|"partial"|"degraded"|"failing"|"paused"
      staleness: data.staleness_sec,
      lastSuccess: data.last_success,
      lastError: data.last_error_msg,
      inFlight: data.in_flight,
    })

    // Cambiar color según estado
    const colors = {
      ok: 'green',
      partial: 'yellow',
      degraded: 'orange',
      failing: 'red',
      paused: 'gray',
    }

    setIndicatorColor(data.metric, colors[data.state])
  }
}
```

## Testing

```bash
# Ejecutar todos los tests
go test -v ./internal/queue/status/...

# Con cobertura
go test -cover ./internal/queue/status/...
```

**Tests disponibles**:

- `TestStreamTracker_RegisterAndUpdate`: Registro y actualización de streams
- `TestStreamTracker_UpdateError`: Manejo de errores
- `TestStreamTracker_ConsecutiveErrors`: Conteo de errores consecutivos
- `TestStreamTracker_InFlight`: Tracking de requests en vuelo
- `TestStreamTracker_CircuitBreaker`: Estado de circuit breaker
- `TestStreamTracker_GetAllStreams`: Listado de streams
- `TestStreamTracker_RemoveStream`: Eliminación de streams
- `TestStatusPusher_HeartbeatFrequency`: Frecuencia de emisión
- `TestStatusPusher_BuildStatus`: Construcción de Status
- `TestStatusPusher_DetermineState_*`: Determinación de estados (OK, Partial, Failing, Paused)
- `TestStatusPusher_Staleness`: Cálculo de staleness
- `TestStatusPusher_StopAndRestart`: Start/Stop del pusher

## Métricas y Monitoreo

El módulo proporciona información útil para monitoreo:

```go
// Obtener todos los streams activos
streams := tracker.GetAllStreams()
fmt.Printf("Streams activos: %d\n", len(streams))

// Obtener KPIs de un stream específico
kpi := tracker.GetKPIs(key)
if kpi != nil {
    fmt.Printf("Errores consecutivos: %d\n", kpi.ConsecutiveErrors)
    fmt.Printf("Éxitos consecutivos: %d\n", kpi.ConsecutiveSuccesses)
}

// Contar streams por estado
statuses := pusher.GetCurrentStatus()
counts := make(map[string]int)
for _, s := range statuses {
    counts[s.State]++
}

fmt.Printf("OK: %d, Partial: %d, Degraded: %d, Failing: %d, Paused: %d\n",
    counts["ok"], counts["partial"], counts["degraded"],
    counts["failing"], counts["paused"])
```

## Casos de Uso

### 1. Dashboard de Salud

Mostrar indicadores visuales del estado de todos los streams:

```
┌─────────────────────────────────────┐
│  Stream Health Dashboard            │
├─────────────────────────────────────┤
│ ● feeding.appetite    OK     (2s)   │
│ ● climate.temp        PARTIAL (45s) │
│ ● biometric.weight    DEGRADED (4m) │
│ ⏸ water.quality       PAUSED        │
│ ✗ oxygen.level        FAILING       │
└─────────────────────────────────────┘
```

### 2. Alertas Proactivas

Detectar problemas antes de que afecten a usuarios:

```go
pusher.OnEmit(func(s status.Status) {
    if s.State == status.StateFailing.String() {
        alerting.Send(fmt.Sprintf(
            "Stream %s/%s is FAILING: %s",
            s.TenantID, s.Metric, *s.LastErrorMsg))
    }

    if s.StalenessSec > 600 { // >10 minutos
        alerting.Send(fmt.Sprintf(
            "Stream %s/%s is stale (%ds without data)",
            s.TenantID, s.Metric, s.StalenessSec))
    }
})
```

### 3. Métricas de SLA

Calcular disponibilidad por stream:

```go
statuses := pusher.GetCurrentStatus()
okCount := 0
totalCount := len(statuses)

for _, s := range statuses {
    if s.State == status.StateOK.String() {
        okCount++
    }
}

availability := float64(okCount) / float64(totalCount) * 100
fmt.Printf("Availability: %.2f%%\n", availability)
```

## TODOs

- [ ] Persistir estado de streams en disco para sobrevivir reinicios
- [ ] Agregar métricas Prometheus (gauge de staleness, counter de heartbeats, etc.)
- [ ] Soportar múltiples callbacks registrados simultáneamente
- [ ] Health check HTTP endpoint para monitoreo externo
- [ ] Compresión de heartbeats cuando hay muchos streams (batch envío)
- [ ] TTL para streams inactivos (auto-remover después de N horas sin updates)

## Arquitectura Completa

```
┌────────────────────────────────────────────────────────┐
│                   OmniAPI Platform                      │
│                                                          │
│  ┌───────────┐    ┌──────────────┐    ┌─────────────┐ │
│  │ Requester │ →  │ StatusPusher │ →  │   Router    │ │
│  │ (Upstream)│    │ (Heartbeats) │    │ (WebSocket) │ │
│  └───────────┘    └──────────────┘    └─────────────┘ │
│        ↓                 ↓                     ↓        │
│   [Result]          [Status]             [Event]       │
│        ↓                 ↓                     ↓        │
│  ┌───────────────────────────────────────────────────┐ │
│  │          StreamTracker (KPIs)                     │ │
│  │  - last_success: 2024-11-10 15:30:00              │ │
│  │  - last_error: null                               │ │
│  │  - latency: 150ms                                 │ │
│  │  - consecutive_errors: 0                          │ │
│  │  - circuit_breaker: closed                        │ │
│  └───────────────────────────────────────────────────┘ │
│                          ↓                             │
│  ┌───────────────────────────────────────────────────┐ │
│  │          WebSocket Clients                        │ │
│  │  {                                                │ │
│  │    "event_type": "status.heartbeat",              │ │
│  │    "data": {                                      │ │
│  │      "metric": "feeding.appetite",                │ │
│  │      "state": "ok",                               │ │
│  │      "staleness_sec": 5                           │ │
│  │    }                                              │ │
│  │  }                                                │ │
│  └───────────────────────────────────────────────────┘ │
└────────────────────────────────────────────────────────┘
```
