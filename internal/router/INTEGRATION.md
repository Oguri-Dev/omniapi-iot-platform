# Router Integration with Requester and Status Modules

This document describes the integration between the `internal/router`, `internal/queue/requester`, and `internal/queue/status` modules.

## Overview

The router now supports two message types:

- **DATA**: Events from the requester module (query results)
- **STATUS**: Heartbeat events from the status module (stream health)

## Architecture

```
┌─────────────────┐
│   Requester     │────┐
│   (Queue)       │    │  OnRequesterResult()
└─────────────────┘    │
                       ▼
                  ┌────────────┐
                  │   Router   │────► Subscribers
                  └────────────┘
                       ▲
┌─────────────────┐    │  OnStatusHeartbeat()
│    Status       │────┘
│   (Heartbeat)   │
└─────────────────┘
```

## Message Types

### MessageType Enum

```go
type MessageType string

const (
    MessageTypeDATA   MessageType = "DATA"   // Eventos de datos (resultados de consultas)
    MessageTypeSTATUS MessageType = "STATUS" // Eventos de estado (heartbeats)
)
```

### DATA Events

DATA events originate from the `requester` module when queries complete:

```go
router.OnRequesterResult(result requester.Result)
```

**Characteristics:**

- `MessageType = DATA`
- `Kind` = metric name (e.g., "feeding.appetite", "climate.temperature")
- `Payload` contains query results or error information
- `Flags = EventFlagSynthetic` if the result contains an error
- Routed to all matching subscribers

### STATUS Events

STATUS events originate from the `status` module as periodic heartbeats:

```go
router.OnStatusHeartbeat(st status.Status)
```

**Characteristics:**

- `MessageType = STATUS`
- `Kind` = "status." + metric (e.g., "status.feeding.appetite")
- `Stream.Kind = "status"`
- `Payload` contains state, staleness, and health metrics
- `Flags = EventFlagSynthetic` (always)
- Routed **only** to subscribers with `IncludeStatus = true`

## Subscription Configuration

### IncludeStatus Field

Subscriptions can opt-in to receive STATUS events:

```go
subscription, _ := router.Subscribe(clientID, filter)
subscription.IncludeStatus = true  // Enable status events
```

**Behavior:**

- `IncludeStatus = false` (default): Receives only DATA events
- `IncludeStatus = true`: Receives both DATA and STATUS events

### Example

```go
// Client that wants both data and status
client1 := "monitoring-dashboard"
router.RegisterClient(client1, tenantID, capabilities, nil, nil)
filter := SubscriptionFilter{TenantID: &tenantID}
sub, _ := router.Subscribe(client1, filter)
sub.IncludeStatus = true  // Opt-in to status events

// Client that wants only data
client2 := "data-consumer"
router.RegisterClient(client2, tenantID, capabilities, nil, nil)
filter2 := SubscriptionFilter{TenantID: &tenantID}
router.Subscribe(client2, filter2)
// IncludeStatus defaults to false
```

## Routing Logic

### DATA Event Routing

1. `OnRequesterResult()` receives a `requester.Result`
2. Converts `TenantID` from string (hex) to `primitive.ObjectID`
3. Creates a `CanonicalEvent` with:
   - `MessageType = DATA`
   - `Kind = result.Metric`
   - `Payload` = JSON with data/error
4. Calls `RouteEvent()` → queues in `eventChan`
5. `processEvent()` calls `resolver.Resolve()` → finds all matching subscribers
6. Event sent to **all** matching subscribers (regardless of `IncludeStatus`)

### STATUS Event Routing

1. `OnStatusHeartbeat()` receives a `status.Status`
2. Converts `TenantID` from string (hex) to `primitive.ObjectID`
3. Creates a `CanonicalEvent` with:
   - `MessageType = STATUS`
   - `Kind = "status." + st.Metric`
   - `Stream.Kind = "status"`
   - `Payload` = JSON with state, staleness, etc.
4. Calls `RouteStatusEvent()` instead of `RouteEvent()`
5. `RouteStatusEvent()` calls `resolver.ResolveStatus()` → finds **only** subscribers with `IncludeStatus = true`
6. Event sent only to subscribers that opted in

## Router Methods

### OnRequesterResult

```go
func (r *Router) OnRequesterResult(result requester.Result)
```

Transforms `requester.Result` into a DATA event and routes it.

**Input:**

- `result.TenantID` (string, hex)
- `result.Metric` (e.g., "feeding.appetite")
- `result.Payload` (json.RawMessage) or `result.Err`

**Output:**

- `CanonicalEvent` with `MessageType = DATA`
- Increments `RouterStats.EventsDataOut`

### OnStatusHeartbeat

```go
func (r *Router) OnStatusHeartbeat(st status.Status)
```

Transforms `status.Status` into a STATUS event and routes it to opted-in subscribers.

**Input:**

- `st.TenantID` (string, hex)
- `st.Metric` (e.g., "feeding.appetite")
- `st.State` ("ok", "partial", "degraded", "failing", "paused")
- `st.StalenessSec` (time since last success)

**Output:**

- `CanonicalEvent` with `MessageType = STATUS`, `Kind = "status." + metric`
- Increments `RouterStats.EventsStatusOut`

### RouteStatusEvent

```go
func (r *Router) RouteStatusEvent(event *connectors.CanonicalEvent)
```

Routes a STATUS event only to subscribers with `IncludeStatus = true`.

## Resolver Extensions

### ResolveStatus

```go
func (r *Resolver) ResolveStatus(event *connectors.CanonicalEvent) (*RoutingDecision, error)
```

Similar to `Resolve()` but filters subscriptions by `IncludeStatus = true`.

**Behavior:**

1. Calls `r.index.FindMatchingStatus(event.Envelope.Stream)` → only status-enabled subscriptions
2. Verifies permissions (same as DATA events)
3. Returns `RoutingDecision` with filtered client list

## SubscriptionIndex Extensions

### FindMatchingStatus

```go
func (si *SubscriptionIndex) FindMatchingStatus(streamKey domain.StreamKey) []*Subscription
```

Finds subscriptions matching the stream **and** with `IncludeStatus = true`.

**Behavior:**

- Searches all indices (byCage, bySite, byFarm, byKind, byTenant)
- Filters by `sub.IncludeStatus == true`
- Returns deduplicated list

## Statistics

### RouterStats Extensions

```go
type RouterStats struct {
    EventsDataOut   int64   `json:"events_data_out"`   // Eventos DATA emitidos
    EventsStatusOut int64   `json:"events_status_out"` // Eventos STATUS emitidos
    RouteP95Ms      float64 `json:"route_p95_ms"`      // Percentil 95 de tiempo de routing
    // ... existing fields
}
```

**Metrics:**

- `EventsDataOut`: Count of DATA events sent from requester results
- `EventsStatusOut`: Count of STATUS events sent from status heartbeats
- `RouteP95Ms`: 95th percentile of routing time in milliseconds

### RecordRoutingTime

```go
func (rs *RouterStats) RecordRoutingTime(durationMs float64)
```

Records routing time for P95 calculation. Maintains a rolling buffer of 1000 samples.

## Configuration

### IntegrationConfig

```go
type IntegrationConfig struct {
    // Status module
    StatusHeartbeatInterval time.Duration  // Default: 10s
    EnableStatusHeartbeats  bool           // Default: true

    // Requester module
    RequesterTimeout             time.Duration  // Default: 30s
    CircuitBreakerMaxErrors      int            // Default: 5
    CircuitBreakerPauseDuration  time.Duration  // Default: 5min
    BackoffInitial               time.Duration  // Default: 1min
    BackoffMultiplier            float64        // Default: 2.0
    BackoffMaxDuration           time.Duration  // Default: 5min
    EnableRequester              bool           // Default: true
}
```

**Usage:**

```go
config := DefaultIntegrationConfig()
config.StatusHeartbeatInterval = 5 * time.Second
config.RequesterTimeout = 60 * time.Second
```

## Integration Tests

The `integration_test.go` file contains 7 test cases:

1. **TestRouter_RequesterResult_Success**: Verifies DATA event creation and routing
2. **TestRouter_RequesterResult_Error**: Verifies error handling with synthetic flag
3. **TestRouter_StatusHeartbeat_Distribution**: Validates IncludeStatus filtering
4. **TestRouter_StatusFiltering**: Multiple clients with/without status subscription
5. **TestRouter_StatusHeartbeat_States**: All 5 states (ok/partial/degraded/failing/paused)
6. **TestRouter_Metrics_P95**: P95 calculation with 100 events
7. **TestRouter_DataVsStatus_Segregation**: Separate counters for DATA vs STATUS

**Run tests:**

```bash
go test -v ./internal/router/ -run "TestRouter_"
```

## Example Integration

```go
package main

import (
    "omniapi/internal/router"
    "omniapi/internal/queue/requester"
    "omniapi/internal/queue/status"
)

func main() {
    // Create router
    r := router.NewRouter()
    r.Start(ctx)

    // Register client
    clientID := "dashboard"
    tenantID := primitive.NewObjectID()
    r.RegisterClient(clientID, tenantID, capabilities, nil, nil)

    // Subscribe with status enabled
    filter := router.SubscriptionFilter{TenantID: &tenantID}
    sub, _ := r.Subscribe(clientID, filter)
    sub.IncludeStatus = true  // Receive both DATA and STATUS

    // Handle requester results
    go func() {
        for result := range requesterResultChan {
            r.OnRequesterResult(result)
        }
    }()

    // Handle status heartbeats
    go func() {
        for st := range statusHeartbeatChan {
            r.OnStatusHeartbeat(st)
        }
    }()

    // Set event callback to send to clients
    r.SetEventCallback(func(clientID string, event *connectors.CanonicalEvent) error {
        return websocket.Send(clientID, event)
    })
}
```

## Status States

The `status.Status` events can have the following states:

- **ok**: Healthy, recent success
- **partial**: Some queries successful, some failed
- **degraded**: Many failures, still operational
- **failing**: Mostly failures, critical
- **paused**: Circuit breaker activated, no queries being sent

## TenantID Conversion

Both `requester.Result` and `status.Status` use `TenantID` as a string (hex representation). The router converts this to `primitive.ObjectID`:

```go
tenantOID, err := primitive.ObjectIDFromHex(result.TenantID)
if err != nil {
    tenantOID = primitive.NilObjectID
}
```

**Important**: Always use `tenantID.Hex()` when creating `requester.Result` or `status.Status` objects.

## Coverage

Integration adds 7 new tests to the router module:

- **Total tests**: 25 (18 existing + 7 integration)
- **Coverage**: 62.9% (up from 49.4%)

## Next Steps

The router is now ready to integrate with:

1. **WebSocket handlers**: Send events to WebSocket clients
2. **Requester module**: Hook up actual query results
3. **Status module**: Hook up actual heartbeat emitter

The integration with WebSocket will be handled in a separate step.
