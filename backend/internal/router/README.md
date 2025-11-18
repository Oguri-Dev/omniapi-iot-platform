# Router - Event Routing Engine

El módulo `router` proporciona un sistema completo de enrutamiento de eventos canónicos a clientes WebSocket con soporte para filtrado, throttling, y políticas multi-conector.

## Características Principales

### 1. Índice de Suscripciones

El router mantiene un índice eficiente de suscripciones que permite búsqueda rápida por:

- **TenantID**: Eventos específicos de un tenant
- **Kind**: Tipo de evento (feeding, biometric, climate, ops)
- **FarmID**: Eventos de una granja específica
- **SiteID**: Eventos de un sitio específico
- **CageID**: Eventos de una jaula específica
- **Source**: Filtrar por fuente/conector específico

### 2. Resolución de Eventos

El resolver determina qué clientes WebSocket deben recibir cada evento basándose en:

- Coincidencia de filtros de suscripción
- Permisos del tenant (TenantID)
- Capabilities requeridas (read permissions)
- Scopes de acceso (farm/site/cage level)

### 3. Throttling y Coalescing

Control de tasa de eventos por cliente con soporte para:

- **ThrottleMs**: Tiempo mínimo entre eventos (ms)
- **MaxRate**: Máximo eventos por segundo
- **BurstSize**: Control de ráfagas con token bucket
- **Coalescing**: Combinar eventos similares
- **Keep-Latest**: Política de backpressure (mantener solo último evento por stream)
- **Buffer por stream**: Buffering independiente por cada stream key

### 4. Políticas Multi-Conector

Gestión de múltiples conectores por tipo de evento:

- **Priority**: Usar conector de mayor prioridad disponible
- **Fallback**: Conmutación automática a backup si falla primario
- **Merge**: Combinar eventos de múltiples conectores
- **Round-Robin**: Balanceo de carga entre conectores

## Arquitectura

```
┌─────────────────┐
│   CanonicalEvent│
└────────┬────────┘
         │
         ▼
    ┌────────┐
    │ Router │
    └────┬───┘
         │
         ├──► Resolver ────► SubscriptionIndex
         │                      │
         │                      ├─ byClient
         │                      ├─ byTenant
         │                      ├─ byKind
         │                      ├─ bySite
         │                      ├─ byCage
         │                      └─ byFarm
         │
         └──► Throttler ───► ClientThrottleState
                               │
                               ├─ Token Bucket
                               ├─ Rate Limiting
                               └─ Stream Buffers
```

## Uso Básico

### Inicializar el Router

```go
import "omniapi/internal/router"

// Crear router
r := router.NewRouter()

// Iniciar procesamiento
ctx := context.Background()
r.Start(ctx)

// Configurar callback para enviar eventos
r.SetEventCallback(func(clientID string, event *connectors.CanonicalEvent) error {
    // Enviar evento al cliente WebSocket
    return sendToWebSocket(clientID, event)
})
```

### Registrar Cliente

```go
import (
    "omniapi/internal/domain"
    "go.mongodb.org/mongo-driver/bson/primitive"
)

clientID := "ws-client-123"
tenantID := primitive.NewObjectID()

// Definir permisos y scopes
permissions := []domain.Capability{
    domain.CapabilityFeedingRead,
    domain.CapabilityClimateRead,
}

scopes := []domain.Scope{
    {
        TenantID: tenantID,
        Resource: "farm:farm-001",
        Permissions: permissions,
        FarmIDs: []string{"farm-001"},
        SiteIDs: []string{"site-001", "site-002"},
    },
}

// Configurar throttle (opcional)
throttleConfig := &router.ThrottleConfig{
    ThrottleMs:        100,    // Min 100ms entre eventos
    MaxRate:           10.0,   // Max 10 eventos/segundo
    BurstSize:         5,      // Permite ráfagas de 5
    CoalescingEnabled: true,   // Activar coalescing
    KeepLatest:        true,   // Keep-latest policy
    BufferSize:        100,    // Buffer de 100 eventos por stream
}

// Registrar cliente
err := r.RegisterClient(clientID, tenantID, permissions, scopes, throttleConfig)
```

### Crear Suscripciones

```go
// Suscripción a todos los eventos de feeding de un sitio
feedingKind := domain.StreamKindFeeding
siteID := "site-001"

filter := router.SubscriptionFilter{
    TenantID: &tenantID,
    Kind:     &feedingKind,
    SiteID:   &siteID,
}

sub, err := r.Subscribe(clientID, filter)

// Suscripción a eventos de una jaula específica
cageID := "cage-001"
filter2 := router.SubscriptionFilter{
    TenantID: &tenantID,
    Kind:     &feedingKind,
    SiteID:   &siteID,
    CageID:   &cageID,
}

sub2, err := r.Subscribe(clientID, filter2)
```

### Enrutar Eventos

```go
// Recibir evento de conector
event := &connectors.CanonicalEvent{
    Envelope: connectors.Envelope{
        Version: "1.0",
        Timestamp: time.Now(),
        Stream: domain.StreamKey{
            TenantID: tenantID,
            Kind:     domain.StreamKindFeeding,
            FarmID:   "farm-001",
            SiteID:   "site-001",
            CageID:   &cageID,
        },
        Source:   "mqtt-connector-1",
        Sequence: 1,
    },
    Kind:          "feeding",
    SchemaVersion: "v1",
    Payload:       payloadJSON,
}

// Enrutar evento (será procesado asíncronamente)
err := r.RouteEvent(event)
```

### Configurar Políticas Multi-Conector

```go
multiConfig := &router.MultiConnectorConfig{
    TenantID: tenantID,
    Kind:     domain.StreamKindFeeding,
    Policy:   router.PolicyFallback,
    Connectors: []router.ConnectorConfig{
        {
            ID:       "mqtt-primary",
            Type:     "mqtt",
            Priority: 100,    // Mayor prioridad
            Enabled:  true,
            Timeout:  5000,
            MaxRetries: 3,
        },
        {
            ID:       "rest-backup",
            Type:     "rest",
            Priority: 50,     // Menor prioridad (fallback)
            Enabled:  true,
            Timeout:  10000,
            MaxRetries: 2,
        },
    },
}

err := r.SetMultiConnectorPolicy(multiConfig)

// Seleccionar conector apropiado
connector, err := r.SelectConnector(tenantID.Hex(), domain.StreamKindFeeding)
```

## Filtros de Suscripción

### Niveles de Especificidad

Los filtros permiten suscripciones a diferentes niveles de granularidad:

1. **Tenant Level**: Todos los eventos del tenant

```go
filter := router.SubscriptionFilter{
    TenantID: &tenantID,
}
```

2. **Kind Level**: Todos los eventos de un tipo

```go
filter := router.SubscriptionFilter{
    TenantID: &tenantID,
    Kind:     &domain.StreamKindFeeding,
}
```

3. **Farm Level**: Eventos de una granja

```go
filter := router.SubscriptionFilter{
    TenantID: &tenantID,
    Kind:     &domain.StreamKindFeeding,
    FarmID:   &farmID,
}
```

4. **Site Level**: Eventos de un sitio

```go
filter := router.SubscriptionFilter{
    TenantID: &tenantID,
    Kind:     &domain.StreamKindFeeding,
    SiteID:   &siteID,
}
```

5. **Cage Level**: Eventos de una jaula específica (máxima especificidad)

```go
filter := router.SubscriptionFilter{
    TenantID: &tenantID,
    Kind:     &domain.StreamKindFeeding,
    SiteID:   &siteID,
    CageID:   &cageID,
}
```

### Filtros por Fuente

```go
filter := router.SubscriptionFilter{
    TenantID: &tenantID,
    Sources:  []string{"mqtt-connector-1", "rest-connector-2"},
}
```

## Throttling

### Configuraciones Comunes

**Alta frecuencia (tiempo real)**

```go
config := router.ThrottleConfig{
    ThrottleMs: 10,      // 10ms mínimo
    MaxRate:    100.0,   // 100 eventos/seg
    BurstSize:  20,      // Ráfagas de 20
}
```

**Frecuencia media**

```go
config := router.ThrottleConfig{
    ThrottleMs: 100,     // 100ms mínimo
    MaxRate:    10.0,    // 10 eventos/seg
    BurstSize:  5,       // Ráfagas de 5
}
```

**Baja frecuencia**

```go
config := router.ThrottleConfig{
    ThrottleMs: 1000,    // 1 segundo mínimo
    MaxRate:    1.0,     // 1 evento/seg
    BurstSize:  2,       // Ráfagas de 2
}
```

### Keep-Latest Policy

Cuando `KeepLatest: true`, el router mantiene solo el evento más reciente por stream en caso de backpressure:

```go
config := router.ThrottleConfig{
    KeepLatest: true,
    BufferSize: 100,
}
```

Esto es útil para datos de estado (temperatura, nivel de oxígeno) donde solo importa el valor más reciente.

## Estadísticas

### Estadísticas Globales

```go
stats := r.GetStats()
fmt.Printf("Eventos enrutados: %d\n", stats.EventsRouted)
fmt.Printf("Eventos descartados: %d\n", stats.EventsDropped)
fmt.Printf("Clientes activos: %d\n", stats.ActiveClients)
fmt.Printf("Suscripciones activas: %d\n", stats.ActiveSubscriptions)
fmt.Printf("Tiempo promedio de routing: %.2fms\n", stats.AvgRoutingTimeMs)
```

### Estadísticas por Cliente

```go
clientStats, err := r.GetClientStats(clientID)
fmt.Printf("Eventos enviados: %d\n", clientStats["events_sent"])
fmt.Printf("Eventos descartados: %d\n", clientStats["events_dropped"])
fmt.Printf("Throttled: %d\n", clientStats["throttled"])
```

## Testing

El módulo incluye tests comprehensivos:

```bash
# Ejecutar todos los tests
go test ./internal/router/...

# Tests de filtros
go test ./internal/router/ -run TestSubscriptionFilter

# Tests de throttle
go test ./internal/router/ -run TestThrottler

# Tests con verbose
go test -v ./internal/router/...

# Coverage
go test -cover ./internal/router/...
```

## Performance

### Consideraciones

- **Índices múltiples**: Búsqueda O(1) por tenant/kind/site/cage
- **Buffering por stream**: Aislamiento entre diferentes streams
- **Token bucket**: Rate limiting eficiente sin locks pesados
- **Coalescing**: Reduce carga de red manteniendo eventos recientes
- **EWMA**: Estadísticas de tiempo de routing con memoria

### Límites Recomendados

- **Clientes activos**: 1,000 - 10,000 por instancia
- **Suscripciones por cliente**: 1 - 100
- **Buffer size**: 100 - 1,000 eventos por stream
- **MaxRate**: 1 - 100 eventos/segundo por cliente

## Integración con WebSocket Hub

Ver el siguiente prompt para la integración completa con el Hub WebSocket.

## Tipos de Datos Principales

- `Router`: Coordinador principal
- `Resolver`: Resolución de eventos a clientes
- `Throttler`: Control de tasa y buffering
- `SubscriptionIndex`: Índice de suscripciones
- `ClientState`: Estado del cliente
- `SubscriptionFilter`: Criterios de filtrado
- `ThrottleConfig`: Configuración de throttle
- `MultiConnectorConfig`: Políticas multi-conector
