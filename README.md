# OmniAPI 🚀

[![Go Version](https://img.shields.io/badge/Go-1.24+-00ADD8?style=for-the-badge&logo=go)](https://golang.org/)
[![License](https://img.shields.io/badge/license-MIT-blue.svg?style=for-the-badge)](LICENSE)
[![Docker](https://img.shields.io/badge/Docker-Ready-2496ED?style=for-the-badge&logo=docker)](Dockerfile)
[![MongoDB](https://img.shields.io/badge/MongoDB-7.0+-47A248?style=for-the-badge&logo=mongodb)](https://www.mongodb.com/)
[![WebSocket](https://img.shields.io/badge/WebSocket-Enabled-FF6B6B?style=for-the-badge)](websocket/)
[![MQTT](https://img.shields.io/badge/MQTT-Supported-660066?style=for-the-badge)](internal/connectors/adapters/mqttfeed/)
[![Prometheus](https://img.shields.io/badge/Prometheus-Ready-E6522C?style=for-the-badge&logo=prometheus)](PROMETHEUS_METRICS.md)

Un sistema IoT avanzado desarrollado en Go con arquitectura de conectores, validación de esquemas, WebSockets y API REST completa para integración multi-tenant de datos agrícolas y acuícolas.

> 🌟 **Proyecto destacado**: Plataforma completa para IoT agrícola con framework de conectores extensible, validación automática de esquemas, arquitectura multi-tenant y sistema de **doble cola** (consultas secuenciales + heartbeats de estado).

## 📑 Tabla de Contenidos

- [Arquitectura del Sistema](#️-arquitectura-del-sistema)
  - [Flujo de Doble Cola](#flujo-de-doble-cola)
- [Características Principales](#-características-principales)
- [Quick Start](#-quick-start)
- [Contrato WebSocket](#-contrato-websocket)
  - [Eventos DATA y STATUS](#eventos-data-datos-del-upstream)
  - [Política Keep-Latest](#política-keep-latest-para-status)
- [Métricas Prometheus](#-métricas-prometheus)
  - [Queries para Evidenciar Demoras](#-queries-para-evidenciar-demoras-del-upstream)
  - [Alertas Recomendadas](#alertas-recomendadas)
- [Configuración Rápida](#️-configuración-rápida)
  - [Parámetros Clave](#parámetros-clave-del-sistema)
  - [Ejemplos por Escenario](#ejemplos-de-configuración-por-escenario)
- [Estructura del Proyecto](#-estructura-del-proyecto)
- [API Endpoints](#-api-endpoints)
- [Testing](#-testing)
- [Troubleshooting](#-troubleshooting)
- [Documentación Adicional](#-documentación-adicional)

## 🏗️ Arquitectura del Sistema

### Flujo de Doble Cola

OmniAPI implementa un sistema de **doble cola** para optimizar la comunicación con sistemas upstream:

```
┌─────────────────────────────────────────────────────────────┐
│                    FLUJO DE DOBLE COLA                       │
├─────────────────────────────────────────────────────────────┤
│                                                               │
│  🔵 Cola 1: REQUESTER (Consultas Secuenciales)              │
│  ┌──────────────────────────────────────────────────────┐   │
│  │ • Procesa requests uno por uno (sequential)          │   │
│  │ • Timeout configurable (default: 10s)                │   │
│  │ • Circuit breaker ante errores consecutivos          │   │
│  │ • Backoff exponencial: 1s → 2s → 5s                 │   │
│  │ • Coalescing: evita requests duplicados              │   │
│  │ • Prioridad: URGENT > HIGH > NORMAL > LOW           │   │
│  └──────────────────────────────────────────────────────┘   │
│                          ↓                                    │
│                    [Result Events]                           │
│                          ↓                                    │
│  🟢 Cola 2: STATUS PUSHER (Heartbeats)                      │
│  ┌──────────────────────────────────────────────────────┐   │
│  │ • Emite heartbeats periódicos cada 30s               │   │
│  │ • Rastrea staleness (tiempo sin datos)              │   │
│  │ • Estados: ok | partial | failing | paused           │   │
│  │ • Detecta degradación ante errores                   │   │
│  │ • Tracking de latencias y errores consecutivos       │   │
│  └──────────────────────────────────────────────────────┘   │
│                          ↓                                    │
│                   [Status Events]                            │
│                          ↓                                    │
│              🔀 ROUTER (Event Distribution)                  │
│  ┌──────────────────────────────────────────────────────┐   │
│  │ • Filtra eventos por tenant/site/metric              │   │
│  │ • Throttling configurable por cliente                │   │
│  │ • Keep-latest policy para STATUS                     │   │
│  │ • Métricas Prometheus integradas                     │   │
│  └──────────────────────────────────────────────────────┘   │
│                          ↓                                    │
│              📡 WebSocket Hub (Clientes)                     │
│  ┌──────────────────────────────────────────────────────┐   │
│  │ • Suscripciones con includeStatus flag               │   │
│  │ • Envío asíncrono de eventos DATA y STATUS           │   │
│  │ • Backpressure handling con buffers                  │   │
│  └──────────────────────────────────────────────────────┘   │
└─────────────────────────────────────────────────────────────┘
```

**Ventajas del diseño:**

- ✅ **Separation of concerns**: Consultas (DATA) separadas de salud (STATUS)
- ✅ **Observabilidad**: Clientes ven tanto datos como estado del upstream
- ✅ **Resiliencia**: Circuit breaker y backoff evitan saturar upstreams lentos
- ✅ **Eficiencia**: Coalescing evita consultas redundantes
- ✅ **Transparencia**: Staleness evidencia demoras del upstream en tiempo real

## 🎯 Características Principales

### 🏗️ Arquitectura de Conectores

- ✅ Framework de conectores extensible con catálogo global
- ✅ Conectores MQTT Feed para datos de alimentación
- ✅ Conectores REST Climate para datos climáticos
- ✅ Conector dummy para testing y demostración
- ✅ Sistema de mappings configurable proveedor→canónico

### � Sistema de Doble Cola

- ✅ **Cola 1 (Requester)**: Consultas secuenciales con circuit breaker
- ✅ **Cola 2 (StatusPusher)**: Heartbeats periódicos de salud
- ✅ Backoff exponencial ante errores (1s → 2s → 5s)
- ✅ Coalescing automático de requests duplicados
- ✅ Priorización: URGENT > HIGH > NORMAL > LOW

### �🔍 Validación y Esquemas

- ✅ Validación automática con JSON Schema
- ✅ Esquemas versionados (feeding.v1, climate.v1, biometric.v1)
- ✅ Backward compatibility y evolución de esquemas
- ✅ API de validación REST

### 🌐 WebSocket en Tiempo Real

- ✅ Suscripciones con filtros (tenant/site/metric)
- ✅ **includeStatus flag**: Controla recepción de heartbeats
- ✅ Eventos DATA: Datos del upstream con latencia
- ✅ Eventos STATUS: Salud del stream (staleness, state)
- ✅ Throttling y backpressure handling
- ✅ Keep-latest policy para STATUS

### 📊 Observabilidad con Prometheus

- ✅ 20 métricas end-to-end del flujo de datos
- ✅ Latencias P50/P95/P99 de upstream
- ✅ Staleness evidencia demoras en tiempo real
- ✅ Circuit breaker y error tracking
- ✅ Métricas por tenant/site/metric
- ✅ Endpoint `/metrics` listo para scraping

### ⚙️ Configuración y Deployment

- ✅ Configuración YAML multi-archivo
- ✅ Gestión de secretos con variables de entorno
- ✅ Hot-reload de configuración
- ✅ Docker ready y production ready

## 🚀 Quick Start

### Opción 1: Desarrollo Local

```bash
# Clonar repositorio
git clone https://github.com/TM-Opera-O/omniapi-iot-platform.git
cd omniapi-iot-platform

# Setup automático (Windows)
.\setup.bat

# Setup automático (Linux/Mac)
./setup.sh

# O manual:
cp .env.example .env  # Editar con tus valores
go mod tidy
go run main.go
```

### Opción 2: Docker (Recomendado)

```bash
# Stack completo con MongoDB + MQTT
docker-compose up -d

# Solo la aplicación
docker build -t omniapi .
docker run -p 3000:3000 --env-file .env omniapi
```

### 🌐 URLs después del setup

- **🏠 Aplicación**: http://localhost:3000
- **🏥 Health Check**: http://localhost:3000/api/health
- **📊 API Info**: http://localhost:3000/api/info
- **🔗 WebSocket**: ws://localhost:3000/ws
- **🧪 WS Test**: http://localhost:3000/ws/test
- **📈 Prometheus Metrics**: http://localhost:8080/metrics

## 📡 Contrato WebSocket

### Conexión y Autenticación

```javascript
const ws = new WebSocket('ws://localhost:3000/ws')

// Autenticar (enviar token)
ws.send(
  JSON.stringify({
    type: 'AUTH',
    token: 'your-jwt-token',
  })
)
```

### Suscripción con `includeStatus`

El flag `includeStatus` controla si recibes **heartbeats de estado** además de datos:

```javascript
// Suscripción CON heartbeats (recomendado para monitoreo)
ws.send(
  JSON.stringify({
    type: 'SUBSCRIBE',
    filter: {
      kind: 'feeding',
      site_id: 'site-123',
      metric: 'appetite',
    },
    includeStatus: true, // 🟢 Recibirás DATA + STATUS
  })
)

// Suscripción SIN heartbeats (solo datos)
ws.send(
  JSON.stringify({
    type: 'SUBSCRIBE',
    filter: {
      kind: 'climate',
      farm_id: 'farm-456',
    },
    includeStatus: false, // 🔵 Solo DATA
  })
)
```

### Eventos DATA (Datos del Upstream)

Eventos recibidos cuando el Requester obtiene datos exitosamente:

```json
{
  "type": "DATA",
  "kind": "feeding.appetite.v1",
  "timestamp": "2025-11-10T14:30:00Z",
  "envelope": {
    "stream": {
      "tenant_id": "507f1f77bcf86cd799439011",
      "kind": "feeding",
      "farm_id": "farm-001",
      "site_id": "site-123",
      "cage_id": "cage-A1"
    },
    "source": "mqtt-feed-connector",
    "timestamp": "2025-11-10T14:30:00Z"
  },
  "data": {
    "appetite": 85.5,
    "consumption_rate": 12.3,
    "timestamp": "2025-11-10T14:29:55Z"
  },
  "metadata": {
    "latency_ms": 245, // ⏱️ Latencia del upstream
    "source": "cloud",
    "priority": "normal"
  }
}
```

**Campos clave:**

- `latency_ms`: Tiempo que tardó el upstream en responder
- `envelope.stream`: Identifica el stream de datos (tenant/site/metric)
- `data`: Payload canónico validado con schema

### Eventos STATUS (Heartbeats de Salud)

Eventos emitidos periódicamente por el StatusPusher (cada 30s por defecto):

```json
{
  "type": "STATUS",
  "kind": "status.heartbeat.v1",
  "timestamp": "2025-11-10T14:30:30Z",
  "envelope": {
    "stream": {
      "tenant_id": "507f1f77bcf86cd799439011",
      "kind": "status",
      "farm_id": "farm-001",
      "site_id": "site-123"
    },
    "source": "status-pusher",
    "timestamp": "2025-11-10T14:30:30Z"
  },
  "status": {
    "state": "ok", // ok | partial | failing | paused
    "staleness_sec": 5, // 🕐 Segundos sin datos frescos
    "in_flight": false, // ¿Request en proceso?
    "last_success_ts": "2025-11-10T14:30:25Z",
    "last_latency_ms": 245, // ⏱️ Última latencia observada
    "consecutive_errors": 0,
    "circuit_open": false,
    "last_error_ts": null,
    "last_error_msg": null
  }
}
```

**Estados posibles:**

- `ok`: Datos frescos, sin errores recientes
- `partial`: Datos obsoletos (staleness > 1s) pero sin errores
- `failing`: Errores consecutivos (≥2) o circuit breaker abierto
- `paused`: Stream pausado manualmente

**Campos clave para evidenciar demoras:**

- `staleness_sec`: **Indicador principal** de demora del upstream
  - `< 5s`: Upstream respondiendo normalmente
  - `5-60s`: Upstream lento (degradado)
  - `> 60s`: Upstream muy lento o sin responder
- `last_latency_ms`: Última latencia observada del upstream
- `circuit_open`: Si está `true`, el upstream falló repetidamente

### Ejemplo: Detectar Upstream Lento

```javascript
ws.onmessage = (event) => {
  const msg = JSON.parse(event.data)

  if (msg.type === 'STATUS') {
    const staleness = msg.status.staleness_sec
    const latency = msg.status.last_latency_ms

    if (staleness > 60) {
      console.warn(`⚠️ Upstream muy lento: ${staleness}s sin datos`)
    } else if (latency > 5000) {
      console.warn(`⚠️ Alta latencia: ${latency}ms`)
    } else {
      console.log(`✅ Stream saludable: staleness=${staleness}s, latency=${latency}ms`)
    }
  }

  if (msg.type === 'DATA') {
    console.log(`📦 Datos recibidos: latency=${msg.metadata.latency_ms}ms`)
  }
}
```

### Política Keep-Latest para STATUS

Ante **backpressure** (cliente lento), OmniAPI aplica **keep-latest** para eventos STATUS:

- ✅ **DATA**: Se descarta si buffer lleno (no queremos datos obsoletos)
- ✅ **STATUS**: Se reemplaza el viejo con el nuevo (siempre el heartbeat más reciente)

Esto garantiza que siempre recibes el **estado actual** del stream, incluso si el cliente está sobrecargado.

## 🛠️ Requisitos Previos

- Go 1.24.0 o superior
- MongoDB 4.4 o superior (opcional, para persistencia)
- VS Code con extensión de Go (recomendado)
- Prometheus 2.x (opcional, para métricas)

## 📊 Métricas Prometheus

OmniAPI expone **20 métricas** en el endpoint `/metrics` para observabilidad completa del flujo de datos.

### Endpoint de Métricas

```bash
curl http://localhost:8080/metrics
```

### Métricas del Requester (Cola 1)

```promql
# Requests en vuelo (gauge)
omniapi_requester_in_flight{tenant="tenant1",site="site-123",metric="feeding",source="cloud"}

# Latencia del upstream (gauge, en millisegundos)
omniapi_requester_latency_ms{tenant="tenant1",site="site-123",metric="feeding",source="cloud"}

# Requests exitosos (counter)
omniapi_requester_success_total{tenant="tenant1",site="site-123",metric="feeding",source="cloud"}

# Requests con error (counter)
omniapi_requester_error_total{tenant="tenant1",site="site-123",metric="feeding",source="cloud",error_code="timeout"}

# Circuit breaker abierto (gauge, 0 o 1)
omniapi_requester_circuit_breaker_open{tenant="tenant1",site="site-123",metric="feeding",source="cloud"}

# Tamaño de la cola (gauge)
omniapi_requester_queue_length{tenant="tenant1",site="site-123",metric="feeding",source="cloud"}
```

### Métricas del StatusPusher (Cola 2)

```promql
# Heartbeats emitidos (counter)
omniapi_status_emitted_total{state="ok"}  # ok | partial | failing | paused

# Staleness del stream (gauge, en segundos) 🕐
omniapi_status_staleness_seconds{tenant="tenant1",site="site-123",metric="feeding",source="cloud"}

# Última latencia observada (gauge, en millisegundos)
omniapi_status_last_latency_ms{tenant="tenant1",site="site-123",metric="feeding",source="cloud"}
```

### Métricas del Router

```promql
# Eventos DATA entrantes (counter)
omniapi_events_data_in_total{tenant="tenant1",site="site-123",metric="feeding"}

# Eventos DATA distribuidos (counter)
omniapi_events_data_out_total{tenant="tenant1",site="site-123",metric="feeding"}

# Eventos STATUS distribuidos (counter)
omniapi_events_status_out_total{tenant="tenant1",site="site-123",metric="feeding"}

# Eventos descartados (counter)
omniapi_events_dropped_total{tenant="tenant1",site="site-123",metric="feeding"}

# Suscripciones activas (gauge)
omniapi_router_subscriptions_active
```

### Métricas del WebSocket Hub

```promql
# Conexiones activas (gauge)
omniapi_ws_connections_active

# Conexiones totales (counter)
omniapi_ws_connections_total

# Mensajes enviados (counter)
omniapi_ws_messages_out_total

# Latencia de delivery (histogram con P50/P95/P99) ⏱️
omniapi_ws_delivery_latency_ms_bucket{le="10"}
omniapi_ws_delivery_latency_ms_bucket{le="50"}
omniapi_ws_delivery_latency_ms_bucket{le="100"}

# Eventos afectados por backpressure (counter)
omniapi_ws_event_backpressure_total
```

### 🎯 Queries para Evidenciar Demoras del Upstream

#### 1. Latencia P95 del Upstream (últimos 5 minutos)

```promql
histogram_quantile(0.95,
  rate(omniapi_ws_delivery_latency_ms_bucket[5m])
)
```

#### 2. Staleness Promedio por Site

```promql
avg by (site) (
  omniapi_status_staleness_seconds
)
```

**Interpretación:**

- `< 5s`: Upstream respondiendo rápido ✅
- `5-30s`: Upstream lento, posible congestión ⚠️
- `> 60s`: Upstream muy lento o sin responder ❌

#### 3. Tasa de Errores del Requester

```promql
rate(omniapi_requester_error_total[5m])
```

#### 4. Streams con Circuit Breaker Abierto

```promql
omniapi_requester_circuit_breaker_open == 1
```

**Alerta:** Si un stream tiene `circuit_breaker_open=1`, significa que falló ≥3 veces consecutivas y está pausado.

#### 5. Latencia Upstream vs. Latency de Delivery WS

```promql
# Latencia del upstream (tiempo que tarda en responder)
omniapi_requester_latency_ms

# vs.

# Latencia de delivery (tiempo desde upstream hasta cliente WS)
histogram_quantile(0.95, rate(omniapi_ws_delivery_latency_ms_bucket[5m]))
```

**Diferencia:**

- `requester_latency_ms`: Tiempo del upstream en responder (fuera de nuestro control)
- `ws_delivery_latency_ms`: Tiempo interno desde recepción hasta envío al cliente (debe ser <50ms)

#### 6. Dashboard Grafana de Ejemplo

```promql
# Panel 1: Staleness por Stream (gauge)
omniapi_status_staleness_seconds

# Panel 2: Latencia P95 (graph)
histogram_quantile(0.95, rate(omniapi_requester_latency_ms[5m]))

# Panel 3: Tasa de Errores (graph)
sum by (error_code) (rate(omniapi_requester_error_total[5m]))

# Panel 4: Circuit Breakers Abiertos (stat)
sum(omniapi_requester_circuit_breaker_open)

# Panel 5: Throughput (graph)
rate(omniapi_events_data_out_total[5m])
```

### Alertas Recomendadas

```yaml
# alerts.yml
groups:
  - name: omniapi_upstream
    interval: 30s
    rules:
      # Upstream lento (staleness > 60s)
      - alert: UpstreamHighStaleness
        expr: omniapi_status_staleness_seconds > 60
        for: 2m
        labels:
          severity: warning
        annotations:
          summary: 'Upstream lento en {{ $labels.site }}'
          description: 'Staleness de {{ $value }}s (> 60s threshold)'

      # Alta latencia del upstream (> 5s)
      - alert: UpstreamHighLatency
        expr: omniapi_requester_latency_ms > 5000
        for: 1m
        labels:
          severity: warning
        annotations:
          summary: 'Alta latencia en {{ $labels.site }}'
          description: 'Latencia de {{ $value }}ms (> 5000ms threshold)'

      # Circuit breaker abierto
      - alert: CircuitBreakerOpen
        expr: omniapi_requester_circuit_breaker_open == 1
        for: 30s
        labels:
          severity: critical
        annotations:
          summary: 'Circuit breaker abierto en {{ $labels.site }}'
          description: 'Stream {{ $labels.metric }} pausado por errores consecutivos'
```

### Cardinality Control

Las métricas usan **sanitización de labels** para prevenir explosión de cardinalidad:

- `tenant_id`: Truncado a 24 caracteres
- `site_id`: Truncado a 32 caracteres
- `metric`: Agrupado en 6 categorías (feeding, biometric, climate, ops, status, other)
- `error_code`: Agrupado en 5 categorías (timeout, connection, 4xx, 5xx, other)

**Cardinalidad estimada:** ~8M series para 100 tenants × 1000 sites × 6 metrics

## ⚙️ Configuración Rápida

### Parámetros Clave del Sistema

OmniAPI usa configuración YAML multi-archivo. Los archivos principales están en `configs/`:

#### 1. Configuración del Requester (Cola 1)

**Archivo:** `configs/app.yaml` (sección `requester`)

```yaml
requester:
  # Timeout para requests al upstream (en segundos)
  request_timeout: 10

  # Circuit breaker: máximo de errores consecutivos antes de pausar
  max_consecutive_errors: 3

  # Duración de pausa del circuit breaker (cuando se abre)
  circuit_pause_duration: 60 # 1 minuto

  # Backoff exponencial ante errores
  backoff_initial: 1 # 1 segundo (primer retry)
  backoff_step2: 2 # 2 segundos (segundo retry)
  backoff_step3: 5 # 5 segundos (tercer retry)

  # Tamaño máximo de la cola de requests
  max_queue_size: 1000

  # Coalescing: evita requests duplicados en ventana de tiempo
  coalescing_enabled: true
  coalescing_window: 5 # 5 segundos
```

**Ajustes recomendados:**

| Escenario           | `request_timeout` | `max_consecutive_errors` | `circuit_pause_duration` |
| ------------------- | ----------------- | ------------------------ | ------------------------ |
| **Upstream rápido** | 5s                | 3                        | 30s                      |
| **Upstream normal** | 10s               | 3                        | 60s                      |
| **Upstream lento**  | 30s               | 5                        | 300s (5min)              |
| **Testing/Dev**     | 2s                | 2                        | 10s                      |

#### 2. Configuración del StatusPusher (Cola 2)

**Archivo:** `configs/app.yaml` (sección `status`)

```yaml
status:
  # Intervalo de heartbeats (en segundos)
  heartbeat_interval: 30

  # Umbral de staleness para estado "ok" (en segundos)
  stale_threshold_ok: 1

  # Umbral de staleness para estado "degraded" (en segundos)
  stale_threshold_degraded: 60

  # Máximo de errores consecutivos antes de state=failing
  max_consecutive_errors: 2
```

**Ajustes recomendados:**

| Escenario           | `heartbeat_interval` | `stale_threshold_ok` | `stale_threshold_degraded` |
| ------------------- | -------------------- | -------------------- | -------------------------- |
| **Alta frecuencia** | 10s                  | 1s                   | 30s                        |
| **Normal**          | 30s                  | 5s                   | 60s                        |
| **Baja frecuencia** | 60s                  | 10s                  | 300s                       |
| **Testing/Dev**     | 5s                   | 1s                   | 10s                        |

#### 3. Configuración del Router y Throttling

**Archivo:** `configs/app.yaml` (sección `router`)

```yaml
router:
  # Tamaño del buffer de eventos
  event_buffer_size: 1000

  # Throttling por cliente (en millisegundos)
  throttle_ms: 100 # Mínimo 100ms entre eventos

  # Tasa máxima de eventos por segundo
  max_rate: 10.0

  # Burst size (ráfagas permitidas)
  burst_size: 5

  # Coalescing de eventos
  coalescing_enabled: true

  # Keep-latest policy para STATUS
  keep_latest: true

  # Tamaño del buffer por stream
  buffer_size: 100
```

#### 4. Configuración de WebSocket

**Archivo:** `configs/app.yaml` (sección `websocket`)

```yaml
websocket:
  # Puerto del servidor WebSocket
  port: 3000

  # Tamaño del buffer de envío por cliente
  send_buffer_size: 256

  # Timeout de lectura/escritura
  read_timeout: 60 # 60 segundos
  write_timeout: 10 # 10 segundos

  # Ping interval (keep-alive)
  ping_interval: 30 # 30 segundos

  # Máximo tamaño de mensaje
  max_message_size: 1048576 # 1 MB
```

### Ejemplos de Configuración por Escenario

#### Escenario 1: Desarrollo Local (Fast Feedback)

```yaml
requester:
  request_timeout: 2
  max_consecutive_errors: 2
  circuit_pause_duration: 10
  backoff_initial: 1
  backoff_step2: 2
  backoff_step3: 3
  max_queue_size: 100
  coalescing_enabled: true
  coalescing_window: 2

status:
  heartbeat_interval: 5
  stale_threshold_ok: 1
  stale_threshold_degraded: 10
  max_consecutive_errors: 2

router:
  event_buffer_size: 100
  throttle_ms: 50
  max_rate: 20.0
  burst_size: 10
```

#### Escenario 2: Producción con Upstream Estable

```yaml
requester:
  request_timeout: 10
  max_consecutive_errors: 3
  circuit_pause_duration: 60
  backoff_initial: 1
  backoff_step2: 2
  backoff_step3: 5
  max_queue_size: 1000
  coalescing_enabled: true
  coalescing_window: 5

status:
  heartbeat_interval: 30
  stale_threshold_ok: 5
  stale_threshold_degraded: 60
  max_consecutive_errors: 2

router:
  event_buffer_size: 1000
  throttle_ms: 100
  max_rate: 10.0
  burst_size: 5
```

#### Escenario 3: Upstream Lento/Inestable

```yaml
requester:
  request_timeout: 30 # Más tolerante
  max_consecutive_errors: 5 # Más intentos antes de abrir circuit
  circuit_pause_duration: 300 # 5 minutos de pausa
  backoff_initial: 2
  backoff_step2: 5
  backoff_step3: 10
  max_queue_size: 2000 # Cola más grande
  coalescing_enabled: true
  coalescing_window: 10 # Ventana más amplia

status:
  heartbeat_interval: 60 # Menos frecuente
  stale_threshold_ok: 10
  stale_threshold_degraded: 300 # 5 minutos
  max_consecutive_errors: 3

router:
  event_buffer_size: 2000
  throttle_ms: 200 # Más conservador
  max_rate: 5.0
  burst_size: 3
```

### Variables de Entorno

Algunas configuraciones críticas se pueden sobrescribir con variables de entorno:

```bash
# .env
OMNIAPI_PORT=8080
OMNIAPI_LOG_LEVEL=info                    # debug | info | warn | error
OMNIAPI_REQUESTER_TIMEOUT=10
OMNIAPI_STATUS_INTERVAL=30
OMNIAPI_CIRCUIT_MAX_ERRORS=3
OMNIAPI_MONGODB_URI=mongodb://localhost:27017
OMNIAPI_MONGODB_DATABASE=omniapi
```

### Aplicar Cambios de Configuración

```bash
# Opción 1: Reiniciar el servidor (recomendado)
go run main.go

# Opción 2: Hot-reload (si está habilitado)
# Enviar señal SIGHUP
kill -HUP <pid>

# Opción 3: Docker
docker-compose restart omniapi
```

### Validar Configuración

```bash
# Ver configuración activa
curl http://localhost:8080/api/info

# Ver métricas de configuración
curl http://localhost:8080/metrics | grep "omniapi_config"
```

### Troubleshooting: Ajustar Configuración

#### Problema: "Muchos timeouts del upstream"

```yaml
# Solución: Aumentar timeout y backoff
requester:
  request_timeout: 30 # Era: 10
  backoff_initial: 2 # Era: 1
  backoff_step2: 5 # Era: 2
  backoff_step3: 10 # Era: 5
```

#### Problema: "Circuit breaker se abre muy rápido"

```yaml
# Solución: Aumentar tolerancia a errores
requester:
  max_consecutive_errors: 5 # Era: 3
  circuit_pause_duration: 300 # Era: 60
```

#### Problema: "Staleness muy alto en streams"

```yaml
# Solución: Heartbeats más frecuentes y thresholds ajustados
status:
  heartbeat_interval: 15 # Era: 30
  stale_threshold_degraded: 120 # Era: 60
```

#### Problema: "Clientes WebSocket se desconectan"

```yaml
# Solución: Timeouts y ping más largos
websocket:
  read_timeout: 120 # Era: 60
  ping_interval: 45 # Era: 30
```

#### Problema: "Alta latencia de delivery WS"

```yaml
# Solución: Buffers más grandes y throttling más permisivo
router:
  event_buffer_size: 2000 # Era: 1000
  throttle_ms: 50 # Era: 100
  max_rate: 20.0 # Era: 10.0
```

## 📁 Estructura del Proyecto

```
omniapi-iot-platform/
├── main.go                              # Punto de entrada
├── go.mod / go.sum                      # Dependencias Go
├── docker-compose.yml                   # Stack completo (app + MongoDB + MQTT)
├── Dockerfile                           # Imagen Docker
│
├── configs/                             # ⚙️ Configuración YAML
│   ├── app.yaml                         # Configuración principal
│   ├── connections.yaml                 # Conexiones a upstreams
│   ├── tenants.yaml                     # Configuración multi-tenant
│   └── mappings/                        # Mapeos proveedor→canónico
│       ├── feeding-mqtt.yaml
│       ├── climate-standard.yaml
│       └── biometric-standard.yaml
│
├── internal/                            # 🏗️ Core del sistema
│   ├── connectors/                      # Framework de conectores
│   │   ├── catalog.go                   # Catálogo global de conectores
│   │   ├── types.go                     # Interfaces y tipos
│   │   └── adapters/                    # Implementaciones
│   │       ├── mqttfeed/               # Conector MQTT para feeding
│   │       └── restclimate/            # Conector REST para climate
│   │
│   ├── queue/                           # 🔄 Sistema de doble cola
│   │   ├── requester/                   # Cola 1: Consultas secuenciales
│   │   │   ├── requester.go            # Requester principal
│   │   │   ├── circuit_breaker.go      # Circuit breaker
│   │   │   ├── backoff.go              # Backoff exponencial
│   │   │   └── requester_test.go       # Tests
│   │   └── status/                      # Cola 2: Heartbeats
│   │       ├── status_pusher.go        # Emisor de heartbeats
│   │       ├── tracker.go              # Tracking de estado
│   │       └── status_test.go          # Tests
│   │
│   ├── router/                          # 📡 Event routing
│   │   ├── router.go                    # Router principal
│   │   ├── resolver.go                  # Resolución de suscripciones
│   │   ├── throttler.go                 # Throttling y backpressure
│   │   ├── types.go                     # SubscriptionFilter, etc.
│   │   └── integration_test.go          # Tests de integración
│   │
│   ├── metrics/                         # 📊 Prometheus metrics
│   │   ├── metrics.go                   # 20 métricas del sistema
│   │   └── metrics_test.go              # Tests de sanitización
│   │
│   ├── schema/                          # 🔍 Validación de esquemas
│   │   ├── schema.go                    # Manager de esquemas
│   │   └── schema_test.go               # Tests
│   │
│   ├── domain/                          # 🎯 Modelos de dominio
│   │   ├── stream_key.go                # Identificadores de streams
│   │   ├── tenant.go                    # Multi-tenancy
│   │   ├── capability.go                # Permisos
│   │   └── access_control.go            # Control de acceso
│   │
│   └── integration/                     # 🧪 Integration tests
│       ├── integration_test.go          # 5 casos de prueba
│       └── README.md                    # Documentación de tests
│
├── websocket/                           # 📡 WebSocket Hub
│   ├── hub.go                           # Gestión de conexiones
│   └── handlers.go                      # Handlers WS
│
├── handlers/                            # 🌐 HTTP Handlers
│   ├── handlers.go                      # Endpoints REST
│   ├── mongodb_handlers.go              # Handlers con MongoDB
│   └── schema_handlers.go               # Validación de esquemas
│
├── database/                            # 💾 MongoDB
│   └── mongodb.go                       # Cliente MongoDB
│
├── adapters/                            # 🔌 Conectores legacy
│   └── dummy/                           # Conector dummy para testing
│
└── docs/                                # 📚 Documentación
    ├── PROMETHEUS_METRICS.md            # Referencia completa de métricas
    ├── PROMETHEUS_TESTING.md            # Guía de testing con Prometheus
    ├── WEBSOCKET_README.md              # Contrato WebSocket completo
    ├── MONGODB_README.md                # Configuración MongoDB
    └── INTEGRATION_TESTS_SUMMARY.md     # Resumen de tests
```

### Documentación Detallada

- **[PROMETHEUS_METRICS.md](PROMETHEUS_METRICS.md)**: Referencia completa de las 20 métricas, queries PromQL, alertas y dashboards Grafana
- **[WEBSOCKET_README.md](WEBSOCKET_README.md)**: Contrato WebSocket completo con ejemplos de suscripciones, eventos DATA/STATUS y manejo de errores
- **[Integration Tests](internal/integration/README.md)**: Documentación de los 5 casos de prueba de integración
- **[MONGODB_README.md](MONGODB_README.md)**: Configuración de MongoDB, índices y consultas
- **[CONTRIBUTING.md](CONTRIBUTING.md)**: Guía para contribuir al proyecto

## 🔧 Desarrollo

### Agregar nuevas rutas

Para agregar una nueva ruta, modifica el archivo `main.go`:

```go
func main() {
    // Agregar nueva ruta
    http.HandleFunc("/nueva-ruta", nuevoHandler)

    // ... resto del código
}

func nuevoHandler(w http.ResponseWriter, r *http.Request) {
    fmt.Fprint(w, "¡Nueva funcionalidad!")
}
```

### Ejecutar en modo desarrollo

Para desarrollo con recarga automática, puedes usar:

```bash
# Instalar air para hot reload (opcional)
go install github.com/cosmtrek/air@latest

# Ejecutar con hot reload
air
```

## 🧪 Testing

OmniAPI incluye tests completos en todos los niveles:

### Ejecutar todos los tests

```bash
go test ./... -v
```

### Tests por paquete

```bash
# Unit tests del Requester
go test ./internal/queue/requester/... -v

# Unit tests del StatusPusher
go test ./internal/queue/status/... -v

# Unit tests del Router
go test ./internal/router/... -v

# Integration tests (5 casos completos)
go test ./internal/integration/... -v

# Tests de métricas Prometheus
go test ./internal/metrics/... -v
```

### Cobertura de tests

```bash
# Cobertura global
go test ./... -cover

# Cobertura detallada con HTML
go test ./... -coverprofile=coverage.out
go tool cover -html=coverage.out
```

### Tests de integración

Los **5 casos de prueba** validan el flujo completo:

1. **Requester Sequential Processing**: 3 requests (2 éxitos + 1 timeout)
2. **StatusPusher Heartbeats**: Transiciones de estado (partial → failing → ok)
3. **Router Event Processing**: Procesamiento de DATA y STATUS
4. **WebSocket Backpressure**: Política keep-latest para STATUS
5. **Full Integration**: Flujo completo Requester → StatusPusher → Router

Ver detalles en [internal/integration/README.md](internal/integration/README.md)

### Testing con Prometheus

```bash
# 1. Iniciar el servidor
go run main.go

# 2. Verificar métricas
curl http://localhost:8080/metrics

# 3. Buscar métrica específica
curl http://localhost:8080/metrics | grep "omniapi_requester_latency"

# 4. Testing con Prometheus local
# Ver guía completa en docs/PROMETHEUS_TESTING.md
```

## 🐳 Docker (Opcional)

Crear un `Dockerfile` para containerización:

```dockerfile
FROM golang:1.21-alpine AS builder
WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN go build -o main .

FROM alpine:latest
RUN apk --no-cache add ca-certificates
WORKDIR /root/
COPY --from=builder /app/main .
EXPOSE 8080
CMD ["./main"]
```

## 📊 API Endpoints

| Método | Endpoint                      | Descripción                         |
| ------ | ----------------------------- | ----------------------------------- |
| GET    | `/`                           | Página principal con interfaz       |
| GET    | `/api/health`                 | Estado del servidor (JSON)          |
| GET    | `/api/info`                   | Información del sistema y versión   |
| GET    | `/metrics`                    | Métricas Prometheus (scraping)      |
| WS     | `/ws`                         | WebSocket para streaming de eventos |
| GET    | `/ws/test`                    | Interfaz de testing WebSocket       |
| POST   | `/api/validate`               | Validar payload contra schema       |
| GET    | `/api/schemas`                | Listar schemas disponibles          |
| GET    | `/api/schemas/:kind/:version` | Obtener schema específico           |

### Ejemplo de respuesta `/api/health`:

```json
{
  "status": "ok",
  "message": "El servidor está funcionando correctamente",
  "timestamp": "2025-11-10T14:30:00Z",
  "version": "1.0.0",
  "components": {
    "requester": "running",
    "status_pusher": "running",
    "router": "running",
    "websocket": "running",
    "mongodb": "connected"
  }
}
```

### Ejemplo de respuesta `/metrics`:

```prometheus
# HELP omniapi_requester_latency_ms Latencia del upstream en millisegundos
# TYPE omniapi_requester_latency_ms gauge
omniapi_requester_latency_ms{tenant="tenant1",site="site-123",metric="feeding",source="cloud"} 245

# HELP omniapi_status_staleness_seconds Segundos desde último éxito
# TYPE omniapi_status_staleness_seconds gauge
omniapi_status_staleness_seconds{tenant="tenant1",site="site-123",metric="feeding",source="cloud"} 5

# HELP omniapi_ws_connections_active Conexiones WebSocket activas
# TYPE omniapi_ws_connections_active gauge
omniapi_ws_connections_active 12
```

## 🔄 Próximos Pasos

- [x] Sistema de doble cola (Requester + StatusPusher)
- [x] Circuit breaker y backoff exponencial
- [x] WebSocket con eventos DATA y STATUS
- [x] Métricas Prometheus completas (20 métricas)
- [x] Tests de integración (5 casos)
- [x] Throttling y backpressure handling
- [ ] Dashboard Grafana pre-configurado
- [ ] Alertas Prometheus configurables
- [ ] API REST completa para gestión de tenants
- [ ] Autenticación JWT para WebSocket
- [ ] Rate limiting por tenant
- [ ] Logging estructurado con correlación IDs

## 📝 Comandos Útiles

```bash
# Desarrollo
go run main.go                    # Ejecutar en modo desarrollo
go build -o omniapi.exe .         # Compilar binario
go fmt ./...                      # Formatear código
go vet ./...                      # Verificar problemas

# Testing
go test ./... -v                  # Todos los tests
go test ./... -cover              # Con cobertura
go test ./... -count=1            # Sin caché (para debugging)
go test -run TestCase1 ./...      # Test específico

# Dependencias
go mod tidy                       # Limpiar dependencias
go get -u ./...                   # Actualizar dependencias
go list -m all                    # Listar módulos

# Métricas y Observabilidad
curl http://localhost:8080/metrics                       # Ver todas las métricas
curl http://localhost:8080/metrics | grep staleness      # Buscar staleness
curl http://localhost:8080/metrics | grep latency        # Buscar latencias

# WebSocket Testing
# Ver interfaz de testing en: http://localhost:3000/ws/test
wscat -c ws://localhost:3000/ws   # Cliente CLI (requiere wscat)

# Docker
docker-compose up -d              # Stack completo
docker-compose logs -f omniapi    # Ver logs
docker-compose restart omniapi    # Reiniciar app
docker-compose down               # Detener todo

# Prometheus + Grafana (si configurado)
docker-compose up -d prometheus grafana
# Prometheus: http://localhost:9090
# Grafana: http://localhost:3001 (admin/admin)
```

## 🚨 Troubleshooting

### Problema: Circuit Breaker se abre constantemente

**Síntoma:** Métrica `omniapi_requester_circuit_breaker_open = 1`

**Solución:**

```yaml
# Ajustar en configs/app.yaml
requester:
  max_consecutive_errors: 5 # Aumentar tolerancia
  circuit_pause_duration: 300 # Pausar más tiempo (5min)
  request_timeout: 30 # Más tiempo para upstream lento
```

### Problema: Alta staleness en streams

**Síntoma:** `omniapi_status_staleness_seconds > 60`

**Causas posibles:**

1. Upstream lento → Aumentar `request_timeout`
2. Heartbeats poco frecuentes → Reducir `heartbeat_interval`
3. Errores consecutivos → Revisar logs del upstream

**Solución:**

```yaml
status:
  heartbeat_interval: 15 # Más frecuente
  stale_threshold_degraded: 120 # Más tolerante
```

### Problema: Clientes WebSocket se desconectan

**Síntoma:** `omniapi_ws_connections_active` decae rápidamente

**Solución:**

```yaml
websocket:
  read_timeout: 120 # Más tiempo
  ping_interval: 45 # Keep-alive más frecuente
```

### Problema: Eventos descartados (backpressure)

**Síntoma:** `omniapi_ws_event_backpressure_total` incrementa

**Solución:**

```yaml
router:
  event_buffer_size: 2000 # Buffer más grande
  throttle_ms: 50 # Menos restrictivo
  max_rate: 20.0 # Mayor throughput
```

### Ver logs detallados

```bash
# Configurar log level
export OMNIAPI_LOG_LEVEL=debug
go run main.go

# Ver logs de componente específico
# Los logs incluyen: [REQUESTER], [STATUS], [ROUTER], [WS]
```

## 🤝 Contribución

¡Las contribuciones son bienvenidas! Por favor:

1. Fork el proyecto
2. Crea una rama para tu feature (`git checkout -b feature/nueva-funcionalidad`)
3. Asegúrate que los tests pasen (`go test ./... -v`)
4. Commit tus cambios (`git commit -am 'feat: agregar nueva funcionalidad'`)
5. Push a la rama (`git push origin feature/nueva-funcionalidad`)
6. Crea un Pull Request

### Convenciones de Commits

Seguimos [Conventional Commits](https://www.conventionalcommits.org/):

- `feat:` Nueva funcionalidad
- `fix:` Corrección de bug
- `docs:` Cambios en documentación
- `test:` Agregar o modificar tests
- `refactor:` Refactorización sin cambiar funcionalidad
- `perf:` Mejoras de rendimiento
- `chore:` Mantenimiento (dependencias, configuración)

Ver más detalles en [CONTRIBUTING.md](CONTRIBUTING.md)

## 📚 Documentación Adicional

- **[PROMETHEUS_METRICS.md](PROMETHEUS_METRICS.md)** - Referencia completa de las 20 métricas con queries PromQL, alertas y dashboards Grafana
- **[WEBSOCKET_README.md](WEBSOCKET_README.md)** - Contrato WebSocket completo con ejemplos de eventos DATA/STATUS
- **[MONGODB_README.md](MONGODB_README.md)** - Configuración de MongoDB, índices y consultas
- **[Integration Tests](internal/integration/README.md)** - Documentación de los 5 casos de prueba
- **[PROMETHEUS_TESTING.md](docs/PROMETHEUS_TESTING.md)** - Guía para testing con Prometheus y Grafana
- **[INTEGRATION_TESTS_SUMMARY.md](INTEGRATION_TESTS_SUMMARY.md)** - Resumen ejecutivo de tests

## 🏆 Features Destacadas

### 1. Sistema de Doble Cola Único

OmniAPI separa **consultas de datos** (Requester) de **heartbeats de salud** (StatusPusher), proporcionando:

- Visibilidad del estado del upstream en tiempo real
- Detección temprana de degradación
- Circuit breaker inteligente

### 2. Observabilidad End-to-End

Con **20 métricas Prometheus**, puedes:

- Medir latencia P95 del upstream
- Detectar staleness (tiempo sin datos)
- Monitorear circuit breakers y errores
- Tracking de throughput y backpressure

### 3. WebSocket con Eventos DATA + STATUS

Los clientes reciben:

- **DATA**: Datos del upstream con latencia
- **STATUS**: Heartbeats periódicos con staleness y estado
- **Keep-latest** para STATUS ante backpressure

### 4. Testing Completo

- 5 casos de integración end-to-end
- Unit tests en todos los componentes
- MockStrategy para simular upstreams
- 100% de tests pasando

## 📄 Licencia

Este proyecto está bajo la Licencia MIT. Ver el archivo [LICENSE](LICENSE) para más detalles.

## 👨‍💻 Autor

**OmniAPI IoT Platform**  
Desarrollado con ❤️ usando Go y las mejores prácticas de:

- Clean Architecture
- Domain-Driven Design
- Observability-First approach
- Test-Driven Development

---

## 🌟 Estrella el Proyecto

Si este proyecto te resulta útil, ¡dale una ⭐ en GitHub!

---

**Versión:** 1.0.0  
**Última actualización:** Noviembre 2025  
**Go Version:** 1.24+  
**Estado:** ✅ Production Ready
