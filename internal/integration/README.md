# Integration Tests

Este paquete contiene pruebas de integración end-to-end que validan el flujo completo del sistema OmniAPI, desde los Requesters hasta el Router y WebSocket.

## 📋 Casos de Prueba

### Case 1: Requester Sequential Processing

**Objetivo:** Validar el procesamiento secuencial de requests con diferentes resultados (éxito/timeout).

**Escenario:**

- Procesa 3 requests secuenciales con un conector mock
- Request 1: Éxito lento (~200ms)
- Request 2: Éxito lento (~300ms)
- Request 3: Timeout (upstream demora 5s, timeout configurado 1s)

**Validaciones:**

- ✅ `in_flight` cambia correctamente durante el procesamiento
- ✅ `last_latency_ms` refleja los tiempos reales (200ms, 300ms, 1000ms)
- ✅ Se emite `Result` con `Err` en el timeout
- ✅ El requester continúa procesando tras el timeout
- ✅ Métricas finales: `TotalSuccess=2, TotalErrors=1, InFlight=false`

**Duración:** ~1.6s

---

### Case 2: StatusPusher Heartbeats

**Objetivo:** Validar que los heartbeats de estado reflejan correctamente la salud del stream.

**Escenario (3 fases):**

1. **Sin datos previos:** Heartbeats con `state=partial`, staleness crece
2. **Después de error:** `state=failing`, `LastErrorTS` y `LastErrorMsg` poblados
3. **Después de éxito:** `state=ok`, staleness baja, `LastSuccessTS` y `LastLatencyMS` correctos

**Validaciones:**

- ✅ Staleness crece cuando no hay éxitos recientes
- ✅ Estado pasa a `failing` tras errores
- ✅ Estado pasa a `ok` tras éxitos subsecuentes
- ✅ Heartbeats emitidos cada ~500ms (configurado para testing)

**Duración:** ~2.7s

---

### Case 3: Router Event Processing

**Objetivo:** Validar que el router acepta y procesa eventos DATA y STATUS.

**Escenario:**

- Enviar 1 evento DATA (via `OnRequesterResult`)
- Enviar 1 evento STATUS (via `OnStatusHeartbeat`)
- Configurar callback para capturar eventos

**Validaciones:**

- ✅ Router acepta ambos tipos de eventos sin errores
- ✅ Los eventos se procesan a través del pipeline
- ⚠️ Sin clientes subscritos, los eventos no se enrutan (esperado)

**Nota:** La distribución específica a clientes requiere que el Resolver tenga clientes registrados, lo cual está implementado en el WebSocket Hub pero no directamente en estos tests.

**Duración:** ~0.5s

---

### Case 4: WebSocket Backpressure Keep-Latest

**Objetivo:** Validar la política `keep-latest` para eventos STATUS ante backpressure.

**Escenario:**

- Canal de envío con buffer de 2 eventos
- Enviar 5 eventos STATUS rápidamente
- Simular cliente lento que no consume del canal

**Validaciones:**

- ✅ Los primeros 2 eventos llenan el buffer
- ✅ Los eventos 3-5 activan la política `keep-latest`
- ✅ Se descarta el evento más viejo y se guarda el más nuevo
- ✅ El cliente lento eventualmente recibe los 2 eventos más recientes (status-4, status-5)

**Resultado esperado:** Solo los eventos más recientes se preservan, evitando acumulación infinita de STATUS stale.

**Duración:** <0.1s

---

### Full Integration Test

**Objetivo:** Validar el flujo completo end-to-end de todos los componentes trabajando juntos.

**Arquitectura:**

```
Requester → Result → Router → (Callback)
    ↓                  ↑
StatusPusher → Status →
```

**Componentes:**

- **MockStrategy:** Simula upstream con latencias controladas
- **Requester:** Procesa 2 requests con timeout de 1s
- **StatusPusher:** Emite heartbeats cada 300ms
- **StreamTracker:** Rastrea estado de stream `tenant1:site1:temperature:cloud`
- **Router:** Procesa eventos DATA y STATUS

**Flujo:**

1. Requester procesa requests
2. Resultados actualizan StreamTracker
3. StreamTracker notifica estado al StatusPusher
4. StatusPusher emite heartbeats periódicos
5. Router procesa ambos tipos de eventos

**Validaciones:**

- ✅ Todos los componentes se inician sin errores
- ✅ Los callbacks se ejecutan correctamente
- ✅ La comunicación entre componentes funciona
- ⚠️ Sin clientes subscritos, no hay eventos enrutados (esperado)

**Duración:** ~2.1s

---

## 🚀 Ejecutar Tests

### Tests de integración únicamente

```powershell
go test ./internal/integration/... -v
```

### Todos los tests del proyecto

```powershell
go test ./... -v
```

### Con cobertura

```powershell
go test ./internal/integration/... -v -cover
```

### Test específico

```powershell
go test ./internal/integration/... -v -run TestCase1_RequesterSequentialProcessing
```

---

## 🏗️ MockStrategy

La clase `MockStrategy` permite simular un upstream con comportamiento controlado:

```go
strategy := NewMockStrategy("mock-upstream", []MockResponse{
    {Delay: 200*time.Millisecond, ShouldErr: false, Data: map[string]interface{}{"value": 42.5}},
    {Delay: 5*time.Second, ShouldErr: false, Data: map[string]interface{}{"value": 99.9}}, // Timeout
})
```

**Parámetros:**

- `Delay`: Latencia simulada
- `ShouldErr`: Si debe retornar error
- `ErrorMsg`: Mensaje de error personalizado
- `Data`: Payload de respuesta

Las respuestas se usan de forma circular (round-robin).

---

## 📊 Métricas de Prometheus

Los tests validan que las métricas de Prometheus se actualicen correctamente:

- **Requester:**

  - `omniapi_requester_in_flight`
  - `omniapi_requester_latency_ms`
  - `omniapi_requester_success_total`
  - `omniapi_requester_error_total`

- **Status:**

  - `omniapi_status_emitted_total{state="ok|partial|failing"}`
  - `omniapi_status_staleness_seconds`
  - `omniapi_status_last_latency_ms`

- **Router:**
  - `omniapi_events_data_in_total`
  - `omniapi_events_data_out_total`
  - `omniapi_events_status_out_total`
  - `omniapi_events_dropped_total`

---

## 🐛 Debugging

Para ver logs detallados durante los tests:

```powershell
go test ./internal/integration/... -v -count=1
```

El flag `-count=1` deshabilita el caché de tests, útil durante desarrollo.

---

## 🔄 Ciclo de Testing

1. **Unit Tests:** Componentes individuales (`internal/queue/`, `internal/router/`)
2. **Integration Tests:** Este paquete - flujo end-to-end
3. **System Tests:** Servidor completo con MongoDB y WebSocket real (manual)

---

## ✅ Checklist de Validación

Antes de hacer merge, asegurarse que:

- [ ] `go test ./... -v` pasa 100%
- [ ] No hay warnings de compilación
- [ ] Métricas de Prometheus se actualizan correctamente
- [ ] Los 4 casos de prueba + test full pasan
- [ ] Duración total <10 segundos

---

## 📚 Referencias

- [Requester README](../queue/requester/README.md)
- [Status README](../queue/status/README.md)
- [Router README](../router/README.md)
- [Prometheus Metrics](../../PROMETHEUS_METRICS.md)

---

**Última actualización:** Noviembre 2025
