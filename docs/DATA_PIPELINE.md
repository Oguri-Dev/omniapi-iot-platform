# OmniAPI Data Normalization & Streaming Plan

## 1. Objetivo

Construir un flujo único que tome los datos crudos de ScaleAQ e Innovex, los normalice en estructuras comunes y los publique tanto al **router → WebSocket Hub** como al **broker MQTT** sin duplicar lógica. El resultado debe ser un `payload` consistente para cualquier dashboard o consumidor en tiempo real.

## 2. Componentes del flujo

| Paso                  | Responsable                                                                 | Descripción                                                                                                                                                                                      |
| --------------------- | --------------------------------------------------------------------------- | ------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------ |
| 1. Descubrimiento     | `frontend/src/pages/Services.tsx` + mocks                                   | Determina qué endpoints necesita cada proveedor y qué headers/credenciales se deben usar.                                                                                                        |
| 2. Collectors         | `internal/providers/{scaleaq,innovex}` (nuevo paquete)                      | Clientes HTTP que ejecutan las llamadas (`/time-series/retrieve`, `/api_dataweb/all_monitors`, etc.), manejan auth y devuelven structs tipados (`ScaleAQRawResponse`, `InnovexMonitorResponse`). |
| 3. Normalizer         | `internal/ingest/normalizer` (nuevo)                                        | Convierte respuestas heterogéneas en un `NormalizedPayload` común; agrega metadatos de site, proveedor y ventana temporal.                                                                       |
| 4. Requester          | `internal/queue/requester`                                                  | Orquesta el polling por sitio/metric, aplica circuit breaker/backoff y emite `requester.Result` con el payload normalizado.                                                                      |
| 5. Router + WebSocket | `internal/router`, `internal/websocket`                                     | Ya implementados. Reciben `CanonicalEvent` vía `OnRequesterResult` y los distribuyen a los clientes suscritos.                                                                                   |
| 6. Broker Publisher   | `internal/connectors/adapters/mqttfeed` o nuevo `internal/broker/publisher` | Toma el mismo `NormalizedPayload` y lo publica en topics MQTT (`tenant/site/metric`). Permite que sistemas legados consuman sin WebSocket.                                                       |

> **Nota:** El Requester sigue siendo el punto de ensamblaje: los collectors/normalizer viven debajo de `Strategy.Execute`, de modo que `result.Payload` ya salga en formato canónico.

## 3. Canonical payload schema

```ts
interface NormalizedPayload {
  context: {
    provider: 'scaleaq' | 'innovex'
    tenantId: string
    siteId: string
    siteName?: string
    generatedAt: string
    window: { from: string; to: string }
    sourceEndpoints: string[]
  }
  assets?: {
    monitors?: MonitorSummary[]
    sensors?: SensorSummary[]
  }
  snapshots?: Record<string, SnapshotBlock>
  timeseries?: Record<string, TimeseriesBlock>
  kpis?: Record<string, KPIBlock>
  alerts?: AlertBlock[]
}
```

- **SnapshotBlock**: valores actuales por sensor/metric (`oxygen`, `temperature`, `feeding_rate`).
- **TimeseriesBlock**: `{ granularity: '5m', unit: 'mg/L', points: Array<[timestamp, value]> }`.
- **KPIBlock**: totales agregados (`consumo_diario`, `biomasa`, `alertas_activos`).
- **AlertBlock**: mensajes derivados de códigos HTTP o reglas de negocio.

Cada `NormalizedPayload` se envía como `result.Payload`. El router ya añade `metric`, `time_range` y `latency_ms`, por lo que los clientes reciben:

```json
{
  "type": "DATA",
  "stream": { "tenant": "...", "siteId": "mowi-001", "kind": "climate", "metric": "climate.oxygen" },
  "payload": {
    "metric": "climate.oxygen",
    "time_range": { "from": "2025-11-24T10:00:00Z", "to": "2025-11-24T10:05:00Z" },
    "latency_ms": 840,
    "data": {
      "context": { "provider": "innovex", ... },
      "timeseries": {
        "oxygen_ppm": {
          "granularity": "60s",
          "unit": "mg/L",
          "points": [[1732447200000, 7.4], [1732447260000, 7.5]]
        }
      },
      "snapshots": {
        "oxygen": { "value": 7.4, "unit": "mg/L", "depth": 12 }
      }
    }
  }
}
```

## 4. Mapeo proveedor → bloques

### ScaleAQ

| Endpoint                                | Bloque destino                                     | Métrica (`result.Metric`)                 |
| --------------------------------------- | -------------------------------------------------- | ----------------------------------------- |
| `/time-series/retrieve`                 | `timeseries.oxygen_ppm`, `timeseries.feeding_rate` | `climate.oxygen`, `feeding.delivery_rate` |
| `/time-series/retrieve/data-types`      | `assets.sensors`                                   | `ops.inventory.channels`                  |
| `/time-series/retrieve/units/aggregate` | `kpis.units`                                       | `feeding.units_kpi`                       |
| `/time-series/retrieve/silos/aggregate` | `kpis.silos`                                       | `feeding.silos_stock`                     |
| `/feeding-dashboard/units`              | `snapshots.units` + alerts                         | `feeding.units_snapshot`                  |
| `/feeding-dashboard/timeline`           | `timeseries.feeding_timeline`                      | `feeding.timeline`                        |
| `/analytics/kpis`                       | `kpis.analytics`                                   | `ops.analytics`                           |

### Innovex

| Endpoint                                | Bloque destino                                    | Métrica                      |
| --------------------------------------- | ------------------------------------------------- | ---------------------------- |
| `/api_dataweb/all_monitors`             | `assets.monitors`                                 | `climate.assets.monitors`    |
| `/api_dataweb/monitor_detail`           | `assets.sensors`                                  | `climate.assets.sensors`     |
| `/api_dataweb/monitor_sensor_last_data` | `snapshots` (`oxygen`, `temperature`, `salinity`) | `climate.snapshots`          |
| `/api_dataweb/get_last_data`            | `snapshots.sensor_specific`                       | `climate.sensor_snapshot`    |
| `/api_dataweb/get_data_range`           | `timeseries.*`                                    | `climate.timeseries.window`  |
| `/api_dataweb/monitor_sensor_time_data` | `timeseries.grouped`                              | `climate.timeseries.grouped` |
| Tabla de errores                        | `alerts`                                          | `ops.provider_alerts`        |

## 5. Publicación hacia WebSocket y Broker

1. **Dentro de la Strategy**
   ```go
   data := normalizer.BuildPayload(ctxInfo, innovexResp)
   return json.Marshal(data)
   ```
2. `Requester` emite `result.Payload` + `Metric` (ej. `climate.oxygen`).
3. `Router` crea `CanonicalEvent` y lo manda al WebSocket Hub.
4. **Broker publisher** escucha los mismos `CanonicalEvent` (usando `Router.SetEventCallback` o un nuevo hook) y los republica en MQTT topics:
   - Topic: `tenant/{tenantId}/site/{siteId}/{metric}`
   - Payload: el `payload.data` serializado.

Esto evita duplicar agregaciones: el broker reutiliza el mensaje ya normalizado.

## 6. Plan de implementación

1. **Crear clientes proveedores**

   - `internal/providers/scaleaq/client.go`
   - `internal/providers/innovex/client.go`
   - Manejar auth con `internal/adapters/auth_adapters.go`.

2. **Normalizador**

   - Nuevo paquete `internal/ingest/normalizer` con funciones `NormalizeScaleAQ`, `NormalizeInnovex`.
   - Definir structs compartidos en `internal/ingest/types.go` (conversiones directas a JSON).

3. **Strategies reales**

   - Implementar `ScaleAQCloudStrategy.Execute` para llamar cliente + normalizer.
   - Crear `InnovexStrategy` que use el mismo patrón.

4. **Mapeo de métricas**

   - Tabla en `configs/connections.yaml` con métricas a pedir por sitio.
   - Cada `Request` especifica `Metric` (p.ej. `climate.oxygen`) y `Source` (`cloud`/`processapi`).

5. **Broker publisher**

   - Opción A: nuevo `internal/broker/publisher` que se registra como listener del Router.
   - Opción B: usar `internal/connectors/adapters/mqttfeed` como base para publicar.

6. **Tests y verificación**

   - Unit tests del normalizer con fixtures reales (colocar en `testdata/providers/...`).
   - Integration test que simule ciclo completo y valide que WebSocket y MQTT reciben el mismo payload.

7. **Observabilidad**
   - Extender métricas Prometheus (`metrics` package) para contar eventos por proveedor/metric.
   - Loggear `sourceEndpoints` en caso de error para depurar rápidamente.

Con esta organización, cualquier nueva fuente (p.ej. ProcessAPI) sólo necesita implementar un collector y una función de normalización que emita el mismo `NormalizedPayload`, manteniendo el dashboard y los clientes sincronizados por WebSocket o broker.
