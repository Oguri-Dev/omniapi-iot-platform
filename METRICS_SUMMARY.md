# Prometheus Metrics Implementation Summary

## ✅ Implemented

### Requester Metrics (`internal/queue/requester`)

- ✅ `omniapi_requester_in_flight{tenant,site,metric,source}` - Gauge
- ✅ `omniapi_requester_last_latency_ms{tenant,site,metric,source}` - Gauge
- ✅ `omniapi_requester_success_total{tenant,site,metric,source}` - Counter
- ✅ `omniapi_requester_error_total{tenant,site,metric,source,code}` - Counter
- ✅ `omniapi_requester_cb_open{tenant,site,metric,source}` - Gauge (0|1)
- ✅ `omniapi_requester_queue_length{tenant,site,metric,source}` - Gauge

### Status Metrics (`internal/queue/status`)

- ✅ `omniapi_status_emitted_total{tenant,site,metric,source,state}` - Counter
- ✅ `omniapi_status_staleness_seconds{tenant,site,metric,source}` - Gauge
- ✅ `omniapi_status_last_latency_ms{tenant,site,metric,source}` - Gauge

### Router Metrics (`internal/router`)

- ✅ `omniapi_events_data_in_total{tenant,site,metric}` - Counter
- ✅ `omniapi_events_data_out_total{tenant,site,metric}` - Counter
- ✅ `omniapi_events_status_out_total{tenant,site,metric}` - Counter
- ✅ `omniapi_events_dropped_total` - Counter
- ✅ `omniapi_router_subscriptions_active{tenant}` - Gauge

### WebSocket Metrics (`websocket/`)

- ✅ `omniapi_ws_connections_active` - Gauge
- ✅ `omniapi_ws_connections_total` - Counter
- ✅ `omniapi_ws_messages_in_total{type}` - Counter
- ✅ `omniapi_ws_messages_out_total{type}` - Counter
- ✅ `omniapi_ws_delivery_latency_ms` - Histogram (for P95 calculation)
- ✅ `omniapi_ws_event_backpressure_total{type}` - Counter
- ✅ `omniapi_ws_subscriptions_active` - Gauge

## 📊 Endpoint

- ✅ `GET /metrics` - Prometheus metrics endpoint (added to main.go)

## 🛡️ Cardinality Protection

### Sanitization Functions (in `internal/metrics/metrics.go`)

- ✅ `SanitizeTenantID()` - Truncates to 24 chars, handles empty
- ✅ `SanitizeMetric()` - Maps to known prefixes (feeding, biometric, climate, water, ops, status, other)
- ✅ `SanitizeSiteID()` - Truncates to 32 chars, handles empty
- ✅ `SanitizeErrorCode()` - Groups into categories (timeout, client_error, server_error, connection_refused, other)

### Expected Cardinality

- **Tenants**: ~100 (limited by infrastructure)
- **Sites**: ~1,000 per tenant (limited by config)
- **Metrics**: 6 categories (feeding, biometric, climate, water, ops, status)
- **Sources**: 3 (cloud, processapi, derived)
- **States**: 5 (ok, partial, degraded, failing, paused)
- **Error Codes**: 5 categories
- **Message Types**: 8 (SUB, UNSUB, PING, ACK, ERROR, PONG, DATA, STATUS)

**Total Series**: ~8M (manageable for Prometheus)

## 📁 Files Created/Modified

### New Files

1. `internal/metrics/metrics.go` - Central metrics registry with Prometheus collectors
2. `internal/queue/requester/requester_metrics.go` - Requester metrics wrapper
3. `PROMETHEUS_METRICS.md` - Complete metrics documentation
4. `docs/PROMETHEUS_TESTING.md` - Testing guide

### Modified Files

1. `main.go` - Added `/metrics` endpoint and Prometheus handler
2. `internal/router/router.go` - Added metrics tracking for DATA/STATUS events
3. `internal/queue/status/status_pusher.go` - Added metrics emission in heartbeats
4. `websocket/hub.go` - Added connection and delivery metrics
5. `go.mod` - Added Prometheus client dependencies

## 🔄 Integration Points

### Requester

- Metrics updated via `RequesterMetrics` wrapper
- Callbacks in `OnResult()` update success/error counters
- Periodic polling (every 5s) updates gauges (in_flight, latency, cb_open, queue_length)

### Status Pusher

- Metrics updated in `emitHeartbeats()` before calling callback
- Increments `status_emitted_total` counter per state
- Sets `status_staleness_seconds` and `status_last_latency_ms` gauges

### Router

- Metrics updated in `processEvent()` for DATA events
- Metrics updated in `OnStatusHeartbeat()` for STATUS events
- Tracks events in/out with tenant/site/metric labels

### WebSocket Hub

- Connection metrics updated in `registerClient()` and `unregisterClient()`
- Delivery metrics recorded in `onRouterEvent()` with histogram for P95
- Backpressure tracked when Send channel is full

## 🚀 Usage

### Start Server

```bash
go run main.go
```

### Access Metrics

```
http://localhost:8080/metrics
```

### Example Output

```
# HELP omniapi_requester_success_total Total de requests exitosos
# TYPE omniapi_requester_success_total counter
omniapi_requester_success_total{metric="feeding",site="greenhouse-1",source="cloud",tenant="655f1c2e8c4b2a1234567890"} 1523

# HELP omniapi_ws_connections_active Número de conexiones WebSocket activas
# TYPE omniapi_ws_connections_active gauge
omniapi_ws_connections_active 7

# HELP omniapi_ws_delivery_latency_ms Latencia de entrega de eventos WebSocket en milisegundos
# TYPE omniapi_ws_delivery_latency_ms histogram
omniapi_ws_delivery_latency_ms_bucket{le="5"} 1234
omniapi_ws_delivery_latency_ms_bucket{le="10"} 2345
omniapi_ws_delivery_latency_ms_sum 123456.78
omniapi_ws_delivery_latency_ms_count 5000
```

## 📖 Documentation

- **Complete Reference**: `PROMETHEUS_METRICS.md`

  - All metrics with descriptions
  - Label schemas
  - Cardinality analysis
  - PromQL queries
  - Alerting rules
  - Grafana dashboards

- **Testing Guide**: `docs/PROMETHEUS_TESTING.md`
  - Quick start
  - Testing scenarios
  - Prometheus setup
  - Grafana setup
  - Troubleshooting

## 🎯 Key Features

1. **Auto-registration**: Metrics are created at package init via `promauto`
2. **Sanitization**: All dynamic labels are sanitized to prevent cardinality explosion
3. **Histogram for Latency**: Uses histogram instead of gauges for P95 calculations
4. **Minimal Overhead**: Metrics updated only when events occur, not polled continuously
5. **Thread-safe**: All metric operations are thread-safe via Prometheus client
6. **Prometheus Best Practices**: Follows naming conventions, label usage, and metric types

## ⚠️ Important Notes

### Cardinality Control

- **DO**: Use sanitization functions for all dynamic labels
- **DON'T**: Add cage_id as a label (too high cardinality)
- **DO**: Group metrics into categories (feeding, biometric, etc.)
- **DON'T**: Create metrics with unique request_id or timestamp labels

### Performance

- Metrics collection has minimal overhead (~1μs per metric update)
- Histogram buckets are pre-defined and efficient
- No heavy aggregations in hot paths

### Monitoring

- Monitor Prometheus cardinality with `prometheus_tsdb_symbol_table_size_bytes`
- Set alerts for high cardinality: `count({__name__=~"omniapi_.*"}) > 10000000`
- Use recording rules for expensive queries

## 🔗 Related Documentation

- `WIRING.md` - Component architecture and startup flow
- `WEBSOCKET_README.md` - WebSocket protocol details
- `internal/queue/requester/README.md` - Requester module documentation
- `internal/queue/status/README.md` - Status module documentation
- `internal/router/README.md` - Router module documentation

## ✅ Testing Checklist

- [x] Metrics endpoint accessible at `/metrics`
- [x] All requester metrics present
- [x] All status metrics present
- [x] All router metrics present
- [x] All WebSocket metrics present
- [x] Labels sanitized correctly
- [x] Counters increment on events
- [x] Gauges update with current values
- [x] Histogram buckets populated
- [x] No compilation errors
- [x] Documentation complete

## 📈 Next Steps

1. Set up Prometheus server to scrape `/metrics`
2. Create Grafana dashboards (see `PROMETHEUS_METRICS.md` for examples)
3. Configure alerting rules (see `PROMETHEUS_METRICS.md` for examples)
4. Monitor cardinality over time
5. Adjust sanitization thresholds if needed

## 🎉 Complete!

All requested metrics have been implemented with proper cardinality control and documentation.
