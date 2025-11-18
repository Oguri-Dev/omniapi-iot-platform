# Connectors Package - OmniAPI

Este paquete proporciona la infraestructura para conectores que integran diferentes fuentes de datos IoT y transforman los datos a formato canónico.

## Arquitectura

### Interfaz Connector

Todos los conectores deben implementar la interfaz `Connector`:

```go
type Connector interface {
    Start(ctx context.Context) error
    Stop() error
    Capabilities() []domain.Capability
    Subscribe(filters ...EventFilter) error
    OnEvent(eventChan chan<- CanonicalEvent)
    Health() HealthInfo
    ID() string
    Type() string
    Config() map[string]interface{}
}
```

### Tipos Principales

#### 1. **CanonicalEvent**

Formato estándar para todos los eventos:

```go
type CanonicalEvent struct {
    Envelope      Envelope        `json:"envelope"`
    Payload       json.RawMessage `json:"payload"`
    Kind          string          `json:"kind"`
    SchemaVersion string          `json:"schema_version"`
}
```

#### 2. **Envelope**

Metadatos del evento:

```go
type Envelope struct {
    Version      string            `json:"version"`
    Timestamp    time.Time         `json:"timestamp"`
    Stream       domain.StreamKey  `json:"stream"`
    Source       string            `json:"source"`
    Sequence     uint64            `json:"sequence"`
    Flags        EventFlags        `json:"flags"`
    TraceID      string            `json:"trace_id,omitempty"`
    CorrelationID string           `json:"correlation_id,omitempty"`
}
```

### Catálogo de Conectores

El `Catalog` gestiona el registro y ciclo de vida de conectores:

#### **Registro de Tipos**

```go
catalog := NewCatalog()

registration := &ConnectorRegistration{
    Type:         "dummy",
    Version:      "1.0.0",
    Factory:      DummyFactory,
    Capabilities: []domain.Capability{domain.CapabilityFeedingRead},
    ConfigSchema: map[string]interface{}{...},
    Description:  "Dummy connector for testing",
}

err := catalog.Register(registration)
```

#### **Creación de Instancias**

```go
// Desde ConnectionInstance del dominio
instance, err := catalog.CreateInstance(connectionInstance, connectorType)

// Iniciar la instancia
ctx := context.Background()
err = catalog.StartInstance(instanceID, ctx)

// Configurar canal de eventos
eventChan := make(chan CanonicalEvent, 100)
instance.OnEvent(eventChan)
```

### Event Filters

Sistema de filtros para suscripciones selectivas:

```go
filters := []EventFilter{
    {
        StreamKey:    &streamKey,
        Capabilities: []domain.Capability{domain.CapabilityFeedingRead},
        Sources:      []string{"sensor-001"},
        Tags:         map[string]string{"location": "greenhouse-1"},
    },
}

err := connector.Subscribe(filters...)
```

### Event Flags

Flags especiales para eventos:

- `EventFlagNone`: Sin flags especiales
- `EventFlagRetry`: Evento en retry
- `EventFlagDuplicate`: Posible duplicado
- `EventFlagLate`: Evento tardío
- `EventFlagSynthetic`: Evento sintético/calculado

## Adaptadores Incluidos

### Dummy Connector

Conector de prueba que emite eventos sintéticos cada segundo:

#### **Configuración**

```go
config := map[string]interface{}{
    "__instance_id": "dummy-001",
    "__tenant_id":   tenantID.Hex(),
    "farm_id":       "farm-123",
    "site_id":       "site-456",
    "cage_id":       "cage-789", // opcional
}
```

#### **Eventos Generados**

- **Feeding**: Datos de alimentación con cantidad, tipo, timestamp
- **Biometric**: Datos biométricos con peso, longitud, health score
- **Climate**: Datos climáticos con temperatura, humedad, oxígeno, pH

#### **Payloads Ejemplo**

**Feeding Event:**

```json
{
  "feed_amount": 3.2,
  "feed_type": "pellets",
  "timestamp": "2025-11-07T12:00:00Z",
  "cage_id": "cage-789",
  "automatic": true
}
```

**Biometric Event:**

```json
{
  "weight": 1.8,
  "length": 28.5,
  "health_score": 87,
  "timestamp": "2025-11-07T12:00:01Z",
  "cage_id": "cage-789",
  "sample_count": 3
}
```

**Climate Event:**

```json
{
  "temperature": 22.4,
  "humidity": 68.2,
  "oxygen_level": 8.1,
  "ph_level": 7.2,
  "timestamp": "2025-11-07T12:00:02Z",
  "sensor_id": "sensor-site-456-001"
}
```

## Uso Completo

### 1. Registro del Conector

```go
import "omniapi/adapters/dummy"

// Registrar en el catálogo global
err := connectors.RegisterConnector(dummy.Registration)
if err != nil {
    log.Fatal("Failed to register dummy connector:", err)
}
```

### 2. Creación de Instancia

```go
// Crear ConnectionInstance y ConnectorType del dominio
connectionInstance := domain.NewConnectionInstance(tenantID, typeID, "My Dummy", "admin")
connectorType := domain.NewConnectorType("dummy", "Dummy Connector", "Test connector", "1.0.0", "admin")

// Crear instancia del conector
instance, err := connectors.CreateConnectorInstance(connectionInstance, connectorType)
if err != nil {
    log.Fatal("Failed to create instance:", err)
}
```

### 3. Configuración de Eventos

```go
// Canal para recibir eventos
eventChan := make(chan connectors.CanonicalEvent, 100)
instance.OnEvent(eventChan)

// Configurar filtros (opcional)
filters := []connectors.EventFilter{
    {Capabilities: []domain.Capability{domain.CapabilityFeedingRead}},
}
instance.Subscribe(filters...)
```

### 4. Ciclo de Vida

```go
// Iniciar conector
ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
defer cancel()

err = instance.Start(ctx)
if err != nil {
    log.Fatal("Failed to start connector:", err)
}

// Procesar eventos
go func() {
    for event := range eventChan {
        log.Printf("Received %s event from %s", event.Kind, event.Envelope.Source)
        // Procesar evento...
    }
}()

// Monitorear salud
health := instance.Health()
log.Printf("Connector health: %s", health.Status)

// Detener cuando termine
defer instance.Stop()
```

## Testing

El paquete incluye tests exhaustivos:

```bash
# Tests unitarios del catálogo
go test ./internal/connectors -v

# Tests del conector dummy
go test ./adapters/dummy -v

# Tests de integración
go test ./internal/connectors/integration -v
```

### Criterios de Test

- ✅ **Catálogo permite registrar tipos**: Tests de registro y desregistro
- ✅ **Crear instancias desde ConnectionInstance**: Integración completa dominio→conector
- ✅ **Dummy emite eventos cada 1s**: Verificación de timing y contenido
- ✅ **Payloads canónicos válidos**: Validación contra schemas JSON
- ✅ **Gestión de ciclo de vida**: Start/Stop/Health monitoring

## Extensión

### Crear Nuevo Conector

1. **Implementar interfaz Connector:**

```go
type MyConnector struct {
    // campos internos
}

func (c *MyConnector) Start(ctx context.Context) error { ... }
func (c *MyConnector) Stop() error { ... }
func (c *MyConnector) Capabilities() []domain.Capability { ... }
// ... resto de métodos
```

2. **Crear Factory:**

```go
func MyConnectorFactory(config map[string]interface{}) (connectors.Connector, error) {
    return NewMyConnector(config)
}
```

3. **Registrar:**

```go
registration := &connectors.ConnectorRegistration{
    Type:         "my-connector",
    Version:      "1.0.0",
    Factory:      MyConnectorFactory,
    Capabilities: []domain.Capability{...},
    ConfigSchema: map[string]interface{}{...},
}

err := connectors.RegisterConnector(registration)
```

### Consideraciones

- **Thread Safety**: Todos los conectores deben ser thread-safe
- **Context Handling**: Respetar context cancellation en Start()
- **Error Handling**: Reportar errores en Health() y logs
- **Resource Cleanup**: Limpiar recursos en Stop()
- **Schema Compliance**: Payloads deben cumplir schemas JSON

## Próximos Pasos

1. **Conectores Reales**: Modbus, MQTT, HTTP APIs
2. **Persistencia**: Almacenar eventos en MongoDB
3. **Métricas**: Prometheus/Grafana integration
4. **Circuit Breaker**: Manejo de fallos en conectores
5. **Config Hot Reload**: Reconfiguración sin restart
