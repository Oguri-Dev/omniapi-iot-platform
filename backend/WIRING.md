# OmniAPI Component Wiring Documentation

## Overview

This document describes how the main components of OmniAPI (Router, Requester, StatusPusher, and WebSocket Hub) are wired together in `main.go`.

## Architecture Flow

```
┌─────────────────────────────────────────────────────────────┐
│                       Main Application                       │
└─────────────────────────────────────────────────────────────┘
                              │
                              ├─── Configuration Loading (YAML)
                              │
                    ┌─────────▼──────────┐
                    │   Router (Core)     │
                    │  - Event routing    │
                    │  - Subscriptions    │
                    └─────────┬──────────┘
                              │
           ┌──────────────────┼──────────────────┐
           │                  │                  │
    ┌──────▼───────┐   ┌─────▼─────┐   ┌───────▼────────┐
    │  Requester   │   │  Status    │   │  WebSocket     │
    │  (per site)  │   │  Pusher    │   │  Hub           │
    │              │   │            │   │                │
    │ - Polls APIs │   │ - Health   │   │ - Clients      │
    │ - Circuit    │   │ - KPIs     │   │ - Events out   │
    │   breaker    │   │ - Heartbeat│   │                │
    └──────┬───────┘   └─────┬─────┘   └────────────────┘
           │                 │
           │   OnResult      │   OnEmit
           └────────►────────┴───────►  Router callbacks
```

## Configuration (configs/app.yaml)

### Requester Configuration

```yaml
requester:
  timeout_seconds: 30 # Request timeout
  backoff_seconds: [60, 120, 300] # Backoff steps (1m, 2m, 5m)
  circuit_breaker:
    failures_threshold: 5 # Open circuit after 5 failures
    pause_minutes: 5 # Pause for 5 minutes when open
```

### Status Configuration

```yaml
status:
  heartbeat_seconds: 10 # Emit status heartbeats every 10 seconds
```

## Startup Sequence

### Phase 1: Router Initialization

```go
r := router.NewRouter()
r.Start(ctx)
```

- Creates the core routing engine
- Starts event loop with context cancellation support

### Phase 2: Requester Initialization

```go
for _, connCfg := range cfg.Connections {
    // Only active connections
    if connCfg.Status != "active" { continue }

    // Create strategy (ScaleAQ, ProcessAPI, NoOp)
    strategy := getStrategy(connCfg)

    // Configure from app.yaml
    reqConfig := requester.Config{
        RequestTimeout:       30s,
        MaxConsecutiveErrors: 5,
        CircuitPauseDuration: 5m,
        BackoffInitial:       60s,
        BackoffStep2:         120s,
        BackoffStep3:         300s,
    }

    // Create and start requester
    req := requester.NewSequentialRequester(reqConfig, strategy)
    req.OnResult(r.OnRequesterResult)  // Wire to router
    req.Start(ctx)

    // Register streams for tracking
    streamTracker.RegisterStream(...)
}
```

**Strategy Selection:**

- `scaleaq-cloud` → ScaleAQCloudStrategy
- `process-api` → ProcessAPIStrategy
- Other types → NoOpStrategy (for testing/demo)

**Stream Metrics:**
Each requester is registered for 3 metrics: `feeding`, `biometric`, `climate`

### Phase 3: StatusPusher Initialization

```go
statusConfig := status.Config{
    HeartbeatInterval:      10s,
    StaleThresholdOK:       30,   // 30 seconds
    StaleThresholdDegraded: 120,  // 2 minutes
    MaxConsecutiveErrors:   5,
}

statusPusher := status.NewStatusPusher(statusConfig, streamTracker)
statusPusher.OnEmit(r.OnStatusHeartbeat)  // Wire to router
statusPusher.Start(ctx)
```

The StatusPusher:

- Reads KPIs from streamTracker (populated by requesters)
- Emits status heartbeats every 10 seconds
- Calculates staleness and stream state
- Sends status events to router via callback

### Phase 4: WebSocket Hub Initialization

```go
wsHub := websocket.NewHub(r)  // Pass router reference
go wsHub.Run()
```

The WebSocket Hub:

- Receives router reference for event subscription
- Manages client connections
- Handles SUB/UNSUB/PING messages
- Routes DATA and STATUS events to subscribed clients

## Callback Wiring

### Requester → Router

```go
req.OnResult(func(result requester.Result) {
    r.OnRequesterResult(result)
})
```

- Requester calls this when API request completes
- Router converts Result to CanonicalEvent
- Router routes event to subscribed WebSocket clients

### StatusPusher → Router

```go
statusPusher.OnEmit(func(st status.Status) {
    r.OnStatusHeartbeat(st)
})
```

- StatusPusher emits every 10 seconds
- Router converts Status to STATUS event
- Router routes to clients with `includeStatus: true`

### Router → WebSocket Hub

The Hub registers an event callback internally:

```go
// In websocket/hub.go
func (h *Hub) onRouterEvent(event *CanonicalEvent, msgType MessageType)
```

- Router calls this for each matching subscription
- Hub applies backpressure (keep-latest for STATUS, drop for DATA)
- Hub sends to client via WebSocket

## Graceful Shutdown

```go
c := make(chan os.Signal, 1)
signal.Notify(c, os.Interrupt, syscall.SIGTERM)
go func() {
    <-c
    cancel()  // Cancel context → stops all components
    time.Sleep(1 * time.Second)  // Grace period
    database.Disconnect()
    os.Exit(0)
}()
```

**Shutdown order:**

1. Signal received (Ctrl+C or SIGTERM)
2. Context cancelled → all goroutines stop
3. Grace period (1 second)
4. MongoDB disconnection
5. Clean exit

## Data Flow Example

### Requester Result → WebSocket Client

```
1. Requester polls ScaleAQ API
2. Requester.OnResult(result) called
3. Router.OnRequesterResult(result)
4. Router converts to CanonicalEvent (DATA)
5. Router finds matching subscriptions
6. Router calls Hub.onRouterEvent(event, DATA)
7. Hub applies backpressure
8. Hub sends to client: {"type":"DATA", "event":{...}}
```

### Status Heartbeat → WebSocket Client

```
1. StatusPusher timer fires (10s interval)
2. StatusPusher reads KPIs from StreamTracker
3. StatusPusher.OnEmit(status) called
4. Router.OnStatusHeartbeat(status)
5. Router converts to CanonicalEvent (STATUS)
6. Router finds subscriptions with includeStatus=true
7. Router calls Hub.onRouterEvent(event, STATUS)
8. Hub keeps latest STATUS (discards old)
9. Hub sends to client: {"type":"STATUS", "event":{...}}
```

## Configuration per Connection

From `configs/connections.yaml`:

```yaml
connections:
  - id: '655f1c2e8c4b2a1234567800'
    tenant_id: '655f1c2e8c4b2a1234567890'
    type_id: 'dummy-connector-type'
    display_name: 'Demo Dummy Connector'
    status: 'active' # Must be 'active' to initialize

    config:
      site_id: 'greenhouse-1' # Required for stream tracking
      # Strategy-specific settings...
```

**Required fields:**

- `status: 'active'` - Only active connections are initialized
- `config.site_id` - Used for stream key identification
- `type_id` - Determines which strategy to use

## Metrics and Monitoring

### Router Metrics

- `EventsDataIn` - Incoming DATA events from requesters
- `EventsDataOut` - Outgoing DATA events to clients
- `EventsStatusOut` - Outgoing STATUS events to clients
- `EventsDropped` - Events dropped due to full buffer

### Requester Metrics (per instance)

- `TotalProcessed` - Total requests processed
- `TotalSuccess` - Successful requests
- `TotalErrors` - Failed requests
- `ConsecErrors` - Consecutive errors (circuit breaker trigger)
- `AvgLatencyMS` - Average request latency
- `CircuitOpen` - Circuit breaker state

### StatusPusher Metrics

- Heartbeat count per stream
- Staleness calculations
- Stream state transitions (ok → degraded → failing)

### WebSocket Metrics

- `ws_events_data_out_total` - Total DATA events sent
- `ws_events_status_out_total` - Total STATUS events sent
- `ws_delivery_p95_ms` - 95th percentile delivery latency

## Testing

To test the wiring:

1. **Start server:**

   ```bash
   go run main.go
   ```

2. **Check startup logs:**

   ```
   📡 Initializing Router...
   ✅ Router started successfully

   🔄 Building Requesters from 8 connections...
     ✓ Requester 'Demo Dummy Connector' [dummy-connector-type] started
     ...
   ✅ 3 Requesters initialized

   💓 Initializing StatusPusher (heartbeat=10s)...
   ✅ StatusPusher started (interval=10s)

   🔌 Initializing WebSocket Hub...
   ✅ WebSocket Hub started
   ```

3. **Open test client:**

   ```
   http://localhost:8080/ws/test
   ```

4. **Subscribe to events:**

   ```json
   {
     "type": "SUB",
     "streams": [{ "kind": "feeding", "siteId": "greenhouse-1" }],
     "includeStatus": true
   }
   ```

5. **Verify DATA and STATUS events arrive**

## Common Issues

### No requesters initialized

**Cause:** All connections have `status: 'inactive'`  
**Fix:** Set at least one connection to `status: 'active'`

### Missing site_id error

**Cause:** Connection config missing `site_id` field  
**Fix:** Add `site_id: 'your-site'` to connection config

### Circuit breaker constantly open

**Cause:** API endpoint unreachable or credentials invalid  
**Fix:** Check endpoint URL and API keys in connection config

### No STATUS events received

**Cause:** WebSocket subscription missing `includeStatus: true`  
**Fix:** Add `"includeStatus": true` to SUB message

## Future Enhancements

- [ ] Dynamic requester creation via API
- [ ] Per-metric configuration (different intervals)
- [ ] Adaptive backoff based on API rate limits
- [ ] Status aggregation (farm-level, site-level)
- [ ] Metrics export (Prometheus)
- [ ] Distributed tracing integration
- [ ] Health check HTTP endpoints
- [ ] Admin UI for requester management
