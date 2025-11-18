# Testing Prometheus Metrics

## Quick Start

### 1. Start the Server

```powershell
go run main.go
```

### 2. Access Metrics Endpoint

```
http://localhost:8080/metrics
```

## Example Metrics Output

### Requester Metrics

```
# HELP omniapi_requester_in_flight Indica si hay un request en progreso (0=idle, 1=in_flight)
# TYPE omniapi_requester_in_flight gauge
omniapi_requester_in_flight{metric="feeding",site="greenhouse-1",source="cloud",tenant="655f1c2e8c4b2a1234567890"} 0

# HELP omniapi_requester_last_latency_ms Última latencia de request en milisegundos
# TYPE omniapi_requester_last_latency_ms gauge
omniapi_requester_last_latency_ms{metric="feeding",site="greenhouse-1",source="cloud",tenant="655f1c2e8c4b2a1234567890"} 123.5

# HELP omniapi_requester_success_total Total de requests exitosos
# TYPE omniapi_requester_success_total counter
omniapi_requester_success_total{metric="feeding",site="greenhouse-1",source="cloud",tenant="655f1c2e8c4b2a1234567890"} 45

# HELP omniapi_requester_error_total Total de requests fallidos
# TYPE omniapi_requester_error_total counter
omniapi_requester_error_total{code="timeout",metric="feeding",site="greenhouse-1",source="cloud",tenant="655f1c2e8c4b2a1234567890"} 2

# HELP omniapi_requester_cb_open Circuit breaker abierto (0=closed, 1=open)
# TYPE omniapi_requester_cb_open gauge
omniapi_requester_cb_open{metric="feeding",site="greenhouse-1",source="cloud",tenant="655f1c2e8c4b2a1234567890"} 0
```

### Status Metrics

```
# HELP omniapi_status_emitted_total Total de status heartbeats emitidos
# TYPE omniapi_status_emitted_total counter
omniapi_status_emitted_total{metric="feeding",site="greenhouse-1",source="cloud",state="ok",tenant="655f1c2e8c4b2a1234567890"} 120

# HELP omniapi_status_staleness_seconds Segundos desde el último request exitoso
# TYPE omniapi_status_staleness_seconds gauge
omniapi_status_staleness_seconds{metric="feeding",site="greenhouse-1",source="cloud",tenant="655f1c2e8c4b2a1234567890"} 15
```

### Router Metrics

```
# HELP omniapi_events_data_in_total Total de eventos DATA recibidos de requesters
# TYPE omniapi_events_data_in_total counter
omniapi_events_data_in_total{metric="feeding",site="greenhouse-1",tenant="655f1c2e8c4b2a1234567890"} 234

# HELP omniapi_events_data_out_total Total de eventos DATA enviados a clientes WebSocket
# TYPE omniapi_events_data_out_total counter
omniapi_events_data_out_total{metric="feeding",site="greenhouse-1",tenant="655f1c2e8c4b2a1234567890"} 468

# HELP omniapi_events_dropped_total Total de eventos descartados por buffers llenos
# TYPE omniapi_events_dropped_total counter
omniapi_events_dropped_total 3
```

### WebSocket Metrics

```
# HELP omniapi_ws_connections_active Número de conexiones WebSocket activas
# TYPE omniapi_ws_connections_active gauge
omniapi_ws_connections_active 2

# HELP omniapi_ws_connections_total Total de conexiones WebSocket establecidas
# TYPE omniapi_ws_connections_total counter
omniapi_ws_connections_total 5

# HELP omniapi_ws_delivery_latency_ms Latencia de entrega de eventos WebSocket en milisegundos
# TYPE omniapi_ws_delivery_latency_ms histogram
omniapi_ws_delivery_latency_ms_bucket{le="1"} 120
omniapi_ws_delivery_latency_ms_bucket{le="5"} 450
omniapi_ws_delivery_latency_ms_bucket{le="10"} 780
omniapi_ws_delivery_latency_ms_bucket{le="+Inf"} 1000
omniapi_ws_delivery_latency_ms_sum 4567.89
omniapi_ws_delivery_latency_ms_count 1000
```

## Testing with curl

### Get All Metrics

```powershell
curl http://localhost:8080/metrics
```

### Filter Specific Metrics

```powershell
# Requester metrics
curl http://localhost:8080/metrics | Select-String "omniapi_requester"

# Status metrics
curl http://localhost:8080/metrics | Select-String "omniapi_status"

# Router metrics
curl http://localhost:8080/metrics | Select-String "omniapi_events"

# WebSocket metrics
curl http://localhost:8080/metrics | Select-String "omniapi_ws"
```

## Setting Up Prometheus

### 1. Download Prometheus

```
https://prometheus.io/download/
```

### 2. Configure prometheus.yml

```yaml
global:
  scrape_interval: 15s

scrape_configs:
  - job_name: 'omniapi'
    static_configs:
      - targets: ['localhost:8080']
```

### 3. Start Prometheus

```powershell
prometheus.exe --config.file=prometheus.yml
```

### 4. Access Prometheus UI

```
http://localhost:9090
```

### 5. Example Queries

**Requester success rate:**

```promql
rate(omniapi_requester_success_total[5m])
```

**WebSocket P95 latency:**

```promql
histogram_quantile(0.95, rate(omniapi_ws_delivery_latency_ms_bucket[5m]))
```

**Active connections:**

```promql
omniapi_ws_connections_active
```

**Status staleness:**

```promql
omniapi_status_staleness_seconds > 60
```

## Setting Up Grafana

### 1. Download Grafana

```
https://grafana.com/grafana/download
```

### 2. Start Grafana

```powershell
grafana-server.exe
```

### 3. Access Grafana

```
http://localhost:3000
Default: admin/admin
```

### 4. Add Prometheus Data Source

1. Configuration → Data Sources
2. Add Prometheus
3. URL: `http://localhost:9090`
4. Save & Test

### 5. Import Dashboard

See `PROMETHEUS_METRICS.md` for recommended panels and queries.

## Verifying Metrics Work

### Test Sequence

1. **Start Server**

   ```powershell
   go run main.go
   ```

2. **Check Initial Metrics**

   ```powershell
   curl http://localhost:8080/metrics | Select-String "omniapi_ws_connections_active"
   ```

   Should show `0` initially.

3. **Connect WebSocket Client**
   Open: `http://localhost:8080/ws/test`

4. **Re-check Metrics**

   ```powershell
   curl http://localhost:8080/metrics | Select-String "omniapi_ws_connections_active"
   ```

   Should show `1` now.

5. **Subscribe to Events**
   In test client, send SUB message.

6. **Check Message Metrics**

   ```powershell
   curl http://localhost:8080/metrics | Select-String "omniapi_ws_messages_in_total"
   ```

   Should show SUB count incremented.

7. **Trigger Requester** (if configured)
   Status heartbeats should emit automatically every 10 seconds.

8. **Check Status Metrics**
   ```powershell
   curl http://localhost:8080/metrics | Select-String "omniapi_status_emitted_total"
   ```
   Should increment over time.

## Troubleshooting

### No Metrics Appearing

**Check endpoint:**

```powershell
curl http://localhost:8080/metrics
```

If you get 404, verify `/metrics` route is registered in `main.go`.

### Metrics Not Updating

**Check if components are running:**

```powershell
curl http://localhost:8080/api/health
```

**Verify requesters started:**
Check logs for "✅ X Requesters initialized"

**Verify status pusher started:**
Check logs for "✅ StatusPusher started"

### High Cardinality Warning

If Prometheus shows cardinality warnings:

1. Check unique label combinations:

   ```promql
   count({__name__=~"omniapi_.*"}) by (__name__)
   ```

2. Verify sanitization is working:

   ```powershell
   curl http://localhost:8080/metrics | Select-String "tenant=" | Select-Object -First 10
   ```

3. Tenants should be truncated to 24 chars
4. Sites should be truncated to 32 chars
5. Metrics should be grouped (feeding, biometric, climate, etc.)

## See Also

- `PROMETHEUS_METRICS.md` - Complete metrics documentation
- `WIRING.md` - Component architecture
- `PROTOCOL.md` - WebSocket protocol details
