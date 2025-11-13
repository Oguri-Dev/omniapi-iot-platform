# OmniAPI Prometheus Metrics

## Overview

Este documento describe las métricas de Prometheus expuestas por OmniAPI, sus etiquetas (labels), y las mejores prácticas para evitar cardinalidad explosiva.

## Endpoint de Métricas

Las métricas están disponibles en:

```
http://localhost:8080/metrics
```

## Métricas Disponibles

### 1. Métricas de Requester (`internal/queue/requester`)

#### `omniapi_requester_in_flight`

**Tipo**: Gauge  
**Descripción**: Indica si hay un request en progreso (0=idle, 1=in_flight)  
**Labels**:

- `tenant` - ID del tenant (sanitizado)
- `site` - ID del site (sanitizado)
- `metric` - Categoría de métrica (feeding, biometric, climate, etc.)
- `source` - Fuente de datos (cloud, processapi)

**Ejemplo**:

```
omniapi_requester_in_flight{tenant="655f1c2e8c4b2a1234567890",site="greenhouse-1",metric="feeding",source="cloud"} 1
```

#### `omniapi_requester_last_latency_ms`

**Tipo**: Gauge  
**Descripción**: Última latencia de request en milisegundos  
**Labels**: `tenant`, `site`, `metric`, `source`

**Ejemplo**:

```
omniapi_requester_last_latency_ms{tenant="655f1c2e8c4b2a1234567890",site="greenhouse-1",metric="feeding",source="cloud"} 245.5
```

#### `omniapi_requester_success_total`

**Tipo**: Counter  
**Descripción**: Total de requests exitosos  
**Labels**: `tenant`, `site`, `metric`, `source`

**Ejemplo**:

```
omniapi_requester_success_total{tenant="655f1c2e8c4b2a1234567890",site="greenhouse-1",metric="feeding",source="cloud"} 1523
```

#### `omniapi_requester_error_total`

**Tipo**: Counter  
**Descripción**: Total de requests fallidos  
**Labels**:

- `tenant`, `site`, `metric`, `source`
- `code` - Categoría de error (timeout, client_error, server_error, other)

**Ejemplo**:

```
omniapi_requester_error_total{tenant="655f1c2e8c4b2a1234567890",site="greenhouse-1",metric="feeding",source="cloud",code="timeout"} 12
```

#### `omniapi_requester_cb_open`

**Tipo**: Gauge  
**Descripción**: Circuit breaker abierto (0=closed, 1=open)  
**Labels**: `tenant`, `site`, `metric`, `source`

**Ejemplo**:

```
omniapi_requester_cb_open{tenant="655f1c2e8c4b2a1234567890",site="greenhouse-1",metric="feeding",source="cloud"} 0
```

#### `omniapi_requester_queue_length`

**Tipo**: Gauge  
**Descripción**: Número de requests en cola  
**Labels**: `tenant`, `site`, `metric`, `source`

**Ejemplo**:

```
omniapi_requester_queue_length{tenant="655f1c2e8c4b2a1234567890",site="greenhouse-1",metric="feeding",source="cloud"} 3
```

---

### 2. Métricas de Status (`internal/queue/status`)

#### `omniapi_status_emitted_total`

**Tipo**: Counter  
**Descripción**: Total de status heartbeats emitidos  
**Labels**:

- `tenant`, `site`, `metric`, `source`
- `state` - Estado del stream (ok, partial, degraded, failing, paused)

**Ejemplo**:

```
omniapi_status_emitted_total{tenant="655f1c2e8c4b2a1234567890",site="greenhouse-1",metric="feeding",source="cloud",state="ok"} 456
```

#### `omniapi_status_staleness_seconds`

**Tipo**: Gauge  
**Descripción**: Segundos desde el último request exitoso  
**Labels**: `tenant`, `site`, `metric`, `source`

**Ejemplo**:

```
omniapi_status_staleness_seconds{tenant="655f1c2e8c4b2a1234567890",site="greenhouse-1",metric="feeding",source="cloud"} 15
```

#### `omniapi_status_last_latency_ms`

**Tipo**: Gauge  
**Descripción**: Última latencia registrada del stream en milisegundos  
**Labels**: `tenant`, `site`, `metric`, `source`

**Ejemplo**:

```
omniapi_status_last_latency_ms{tenant="655f1c2e8c4b2a1234567890",site="greenhouse-1",metric="feeding",source="cloud"} 234.2
```

---

### 3. Métricas de Router (`internal/router`)

#### `omniapi_events_data_in_total`

**Tipo**: Counter  
**Descripción**: Total de eventos DATA recibidos de requesters  
**Labels**: `tenant`, `site`, `metric`

**Ejemplo**:

```
omniapi_events_data_in_total{tenant="655f1c2e8c4b2a1234567890",site="greenhouse-1",metric="feeding"} 2345
```

#### `omniapi_events_data_out_total`

**Tipo**: Counter  
**Descripción**: Total de eventos DATA enviados a clientes WebSocket  
**Labels**: `tenant`, `site`, `metric`

**Ejemplo**:

```
omniapi_events_data_out_total{tenant="655f1c2e8c4b2a1234567890",site="greenhouse-1",metric="feeding"} 4690
```

#### `omniapi_events_status_out_total`

**Tipo**: Counter  
**Descripción**: Total de eventos STATUS enviados a clientes WebSocket  
**Labels**: `tenant`, `site`, `metric`

**Ejemplo**:

```
omniapi_events_status_out_total{tenant="655f1c2e8c4b2a1234567890",site="greenhouse-1",metric="feeding"} 789
```

#### `omniapi_events_dropped_total`

**Tipo**: Counter  
**Descripción**: Total de eventos descartados por buffers llenos  
**Labels**: (ninguna)

**Ejemplo**:

```
omniapi_events_dropped_total 23
```

#### `omniapi_router_subscriptions_active`

**Tipo**: Gauge  
**Descripción**: Número de suscripciones activas en el router  
**Labels**: `tenant`

**Ejemplo**:

```
omniapi_router_subscriptions_active{tenant="655f1c2e8c4b2a1234567890"} 12
```

---

### 4. Métricas de WebSocket (`websocket/`)

#### `omniapi_ws_connections_active`

**Tipo**: Gauge  
**Descripción**: Número de conexiones WebSocket activas  
**Labels**: (ninguna)

**Ejemplo**:

```
omniapi_ws_connections_active 7
```

#### `omniapi_ws_connections_total`

**Tipo**: Counter  
**Descripción**: Total de conexiones WebSocket establecidas  
**Labels**: (ninguna)

**Ejemplo**:

```
omniapi_ws_connections_total 143
```

#### `omniapi_ws_messages_in_total`

**Tipo**: Counter  
**Descripción**: Total de mensajes recibidos de clientes WebSocket  
**Labels**: `type` (SUB, UNSUB, PING)

**Ejemplo**:

```
omniapi_ws_messages_in_total{type="SUB"} 89
omniapi_ws_messages_in_total{type="PING"} 1234
```

#### `omniapi_ws_messages_out_total`

**Tipo**: Counter  
**Descripción**: Total de mensajes enviados a clientes WebSocket  
**Labels**: `type` (ACK, ERROR, PONG, DATA, STATUS)

**Ejemplo**:

```
omniapi_ws_messages_out_total{type="DATA"} 5678
omniapi_ws_messages_out_total{type="STATUS"} 234
omniapi_ws_messages_out_total{type="PONG"} 1234
```

#### `omniapi_ws_delivery_latency_ms`

**Tipo**: Histogram  
**Descripción**: Latencia de entrega de eventos WebSocket en milisegundos  
**Buckets**: 1, 5, 10, 25, 50, 100, 250, 500, 1000, 2500, 5000  
**Labels**: (ninguna)

**Ejemplo**:

```
omniapi_ws_delivery_latency_ms_bucket{le="5"} 1234
omniapi_ws_delivery_latency_ms_bucket{le="10"} 2345
omniapi_ws_delivery_latency_ms_bucket{le="25"} 3456
omniapi_ws_delivery_latency_ms_sum 123456.78
omniapi_ws_delivery_latency_ms_count 5000
```

**P95 Calculation**:

```promql
histogram_quantile(0.95, rate(omniapi_ws_delivery_latency_ms_bucket[5m]))
```

#### `omniapi_ws_event_backpressure_total`

**Tipo**: Counter  
**Descripción**: Total de eventos descartados por backpressure en WebSocket  
**Labels**: `type` (DATA, STATUS)

**Ejemplo**:

```
omniapi_ws_event_backpressure_total{type="DATA"} 45
omniapi_ws_event_backpressure_total{type="STATUS"} 12
```

#### `omniapi_ws_subscriptions_active`

**Tipo**: Gauge  
**Descripción**: Número total de suscripciones activas en WebSocket  
**Labels**: (ninguna)

**Ejemplo**:

```
omniapi_ws_subscriptions_active 23
```

---

## Prevención de Cardinalidad Explosiva

### 🚨 Problema

La cardinalidad excesiva ocurre cuando las etiquetas (labels) de métricas generan demasiadas combinaciones únicas, causando:

- **Alto uso de memoria** en Prometheus
- **Queries lentas**
- **Degradación del rendimiento**
- **Posible out-of-memory**

### ✅ Estrategias Implementadas

#### 1. **Sanitización de Labels**

Se aplican funciones de sanitización en `internal/metrics/metrics.go`:

##### `SanitizeTenantID(tenantID string) string`

- Limita el tenant ID a 24 caracteres
- Convierte IDs vacíos a "unknown"
- **Ejemplo**: `655f1c2e8c4b2a1234567890abcdefgh` → `655f1c2e8c4b2a1234567890`

##### `SanitizeMetric(metric string) string`

- Agrupa métricas por prefijo (feeding, biometric, climate, water, ops, status)
- Previene métricas con nombres arbitrarios
- **Ejemplo**: `feeding.appetite` → `feeding`, `climate.temperature` → `climate`

##### `SanitizeSiteID(siteID string) string`

- Limita site IDs a 32 caracteres
- Convierte IDs vacíos a "unknown"
- **Ejemplo**: `greenhouse-complex-section-A-subsection-1-unit-alpha` → `greenhouse-complex-section-A-su`

##### `SanitizeErrorCode(code string) string`

- Agrupa errores en categorías (timeout, client_error, server_error, connection_refused, other)
- Previene explosión por códigos de error únicos
- **Ejemplo**: `500` → `server_error`, `timeout_connection_refused` → `timeout`

#### 2. **Limitación de Cardinalidad por Dimensión**

| Label    | Cardinalidad Esperada | Estrategia                                                     |
| -------- | --------------------- | -------------------------------------------------------------- |
| `tenant` | 10-100 tenants        | Truncar a 24 chars                                             |
| `site`   | 100-1000 sites        | Truncar a 32 chars                                             |
| `metric` | 6 categorías          | Mapear a prefijos conocidos                                    |
| `source` | 2-3 fuentes           | cloud, processapi, derived                                     |
| `state`  | 5 estados             | ok, partial, degraded, failing, paused                         |
| `code`   | 5 categorías          | timeout, client_error, server_error, connection_refused, other |
| `type`   | 8 tipos               | SUB, UNSUB, PING, ACK, ERROR, PONG, DATA, STATUS               |

**Cardinalidad Total Estimada**:

```
Requester: 100 tenants × 1000 sites × 6 metrics × 2 sources = 1.2M series
Status: 100 tenants × 1000 sites × 6 metrics × 2 sources × 5 states = 6M series
Router: 100 tenants × 1000 sites × 6 metrics = 600K series
WebSocket: ~20 series (sin labels de cardinalidad alta)
```

**Total**: ~8M series (manejable para Prometheus)

#### 3. **Métricas Sin Labels de Alta Cardinalidad**

Las siguientes métricas NO tienen labels de alta cardinalidad:

- `omniapi_events_dropped_total`
- `omniapi_ws_connections_active`
- `omniapi_ws_connections_total`
- `omniapi_ws_delivery_latency_ms`
- `omniapi_ws_subscriptions_active`

Esto previene la explosión al evitar cruzar dimensiones innecesarias.

#### 4. **Uso de Histogramas en lugar de Gauges**

Para latencia de WebSocket, usamos `Histogram` en lugar de múltiples `Gauge` con percentiles pre-calculados:

```
✅ omniapi_ws_delivery_latency_ms (Histogram)
❌ omniapi_ws_delivery_p50_ms, omniapi_ws_delivery_p95_ms, omniapi_ws_delivery_p99_ms (3 Gauges)
```

Esto reduce las series y permite calcular percentiles dinámicamente en queries.

---

## Queries Útiles (PromQL)

### Requester Performance

**Tasa de éxito por site:**

```promql
rate(omniapi_requester_success_total[5m])
```

**Tasa de error por categoría:**

```promql
sum(rate(omniapi_requester_error_total[5m])) by (code)
```

**Latencia promedio por métrica:**

```promql
avg(omniapi_requester_last_latency_ms) by (metric)
```

**Circuit breaker activo:**

```promql
sum(omniapi_requester_cb_open) by (tenant, site, metric)
```

### Status Health

**Streams con staleness alto (>2 minutos):**

```promql
omniapi_status_staleness_seconds > 120
```

**Distribución de estados:**

```promql
sum(rate(omniapi_status_emitted_total[5m])) by (state)
```

### Router Throughput

**Eventos DATA por segundo (in vs out):**

```promql
rate(omniapi_events_data_in_total[1m])
rate(omniapi_events_data_out_total[1m])
```

**Eventos descartados:**

```promql
rate(omniapi_events_dropped_total[5m])
```

### WebSocket Performance

**Conexiones activas:**

```promql
omniapi_ws_connections_active
```

**P95 latency de entrega:**

```promql
histogram_quantile(0.95, rate(omniapi_ws_delivery_latency_ms_bucket[5m]))
```

**Backpressure rate:**

```promql
rate(omniapi_ws_event_backpressure_total[5m])
```

**Mensajes por tipo:**

```promql
sum(rate(omniapi_ws_messages_out_total[1m])) by (type)
```

---

## Alertas Recomendadas

### Alta Tasa de Errores (Requester)

```yaml
- alert: HighRequesterErrorRate
  expr: |
    rate(omniapi_requester_error_total[5m]) / 
    (rate(omniapi_requester_success_total[5m]) + rate(omniapi_requester_error_total[5m])) 
    > 0.1
  for: 5m
  labels:
    severity: warning
  annotations:
    summary: 'Alta tasa de errores en requester (>10%)'
```

### Circuit Breaker Abierto

```yaml
- alert: CircuitBreakerOpen
  expr: omniapi_requester_cb_open == 1
  for: 2m
  labels:
    severity: critical
  annotations:
    summary: 'Circuit breaker abierto en {{ $labels.tenant }}/{{ $labels.site }}/{{ $labels.metric }}'
```

### Staleness Alto

```yaml
- alert: HighStreamStaleness
  expr: omniapi_status_staleness_seconds > 300
  for: 5m
  labels:
    severity: warning
  annotations:
    summary: 'Stream {{ $labels.metric }} en {{ $labels.site }} no ha recibido datos en 5+ minutos'
```

### WebSocket Backpressure

```yaml
- alert: HighWebSocketBackpressure
  expr: rate(omniapi_ws_event_backpressure_total[5m]) > 10
  for: 5m
  labels:
    severity: warning
  annotations:
    summary: 'Alto backpressure en WebSocket (>10 eventos/s descartados)'
```

### Alta Latencia de Delivery

```yaml
- alert: HighWebSocketLatency
  expr: |
    histogram_quantile(0.95, rate(omniapi_ws_delivery_latency_ms_bucket[5m])) 
    > 100
  for: 5m
  labels:
    severity: warning
  annotations:
    summary: 'P95 de latencia WebSocket >100ms'
```

---

## Dashboards de Grafana

### Panel 1: Requester Overview

- **Connections Gauge**: `omniapi_requester_in_flight`
- **Success Rate**: `rate(omniapi_requester_success_total[5m])`
- **Error Rate by Code**: `sum(rate(omniapi_requester_error_total[5m])) by (code)`
- **Latency Heatmap**: `omniapi_requester_last_latency_ms`

### Panel 2: Status Health

- **Staleness by Site**: `omniapi_status_staleness_seconds`
- **State Distribution**: `sum(omniapi_status_emitted_total) by (state)`
- **Circuit Breaker Status**: `omniapi_requester_cb_open`

### Panel 3: WebSocket Performance

- **Active Connections**: `omniapi_ws_connections_active`
- **Message Rate**: `rate(omniapi_ws_messages_out_total[1m])`
- **P50/P95/P99 Latency**: `histogram_quantile(...)`
- **Backpressure**: `rate(omniapi_ws_event_backpressure_total[5m])`

### Panel 4: Router Throughput

- **Events In/Out**: `rate(omniapi_events_data_in_total[1m])` vs `rate(omniapi_events_data_out_total[1m])`
- **Dropped Events**: `rate(omniapi_events_dropped_total[5m])`
- **Active Subscriptions**: `omniapi_router_subscriptions_active`

---

## Mejores Prácticas

### ✅ DO

- Usar las funciones de sanitización para todos los labels dinámicos
- Limitar el número de sites/tenants únicos en queries
- Usar `rate()` para counters
- Usar `histogram_quantile()` para percentiles
- Agregar queries con `sum()` y `avg()` cuando sea posible

### ❌ DON'T

- No agregar labels con valores únicos por request (request_id, timestamp, etc.)
- No crear métricas con cage_id como label (puede tener miles de valores)
- No usar `increase()` en lugar de `rate()` para alertas
- No hacer queries sin límite de tiempo (`[:]`)
- No crear métricas personalizadas sin sanitización

---

## Troubleshooting

### Cardinalidad Alta Detectada

**Verificar series activas:**

```promql
count({__name__=~"omniapi_.*"})
```

**Top 10 métricas por cardinalidad:**

```sh
curl -s http://localhost:8080/metrics | grep "omniapi_" | wc -l
```

**Identificar labels problemáticos:**

```promql
count by (__name__, tenant) ({__name__=~"omniapi_.*"})
```

### Métricas Faltantes

**Verificar que los componentes estén reportando:**

```sh
curl http://localhost:8080/metrics | grep omniapi_requester_success_total
```

Si no aparecen, verificar:

1. El requester está iniciado correctamente
2. Los callbacks están registrados
3. El collector de métricas está corriendo

---

## Referencias

- [Prometheus Best Practices](https://prometheus.io/docs/practices/naming/)
- [Avoiding High Cardinality](https://prometheus.io/docs/practices/instrumentation/#do-not-overuse-labels)
- [Histogram vs Summary](https://prometheus.io/docs/practices/histograms/)
