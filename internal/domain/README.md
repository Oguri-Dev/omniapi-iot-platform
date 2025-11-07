# Domain Package - OmniAPI

Este paquete contiene los modelos de dominio y la lógica de negocio para OmniAPI, un sistema multi-tenant para gestión de datos IoT/agrícolas.

## Arquitectura del Dominio

### Modelos Principales

#### 1. Tenant

- **Propósito**: Representa una organización o cliente en el sistema
- **Características**:
  - Multi-tenancy con aislamiento de datos
  - Sistema de quotas por recurso
  - Scopes y permisos granulares
  - Estados del tenant (active, inactive, suspended)

#### 2. Capability (Enum)

- **Valores disponibles**:
  - `feeding.read`: Lectura de datos de alimentación
  - `biometric.read`: Lectura de datos biométricos
  - `climate.read`: Lectura de datos climáticos
  - `ops.read`: Lectura de datos operacionales

#### 3. StreamKey

- **Propósito**: Identificador único para streams de datos
- **Estructura**: `{TenantID, Kind, FarmID, SiteID, CageID?}`
- **Funcionalidades**:
  - Serialización/deserialización a string
  - Validación de estructura
  - Hash SHA256 para indexación
  - Comparación de igualdad

#### 4. ConnectorType

- **Propósito**: Define tipos de conectores disponibles
- **Características**:
  - Capabilities soportadas
  - ConfigSpec (JSON Schema para configuración)
  - OutputSchemas por capability
  - Versionado y lifecycle management

#### 5. ConnectionInstance

- **Propósito**: Instancia configurada de un conector
- **Características**:
  - Configuración específica por tenant
  - Mappings de datos por capability
  - Estado y monitoreo de conexión
  - Sistema de retry y timeouts

#### 6. Mapping

- **Propósito**: Reglas de mapeo de datos proveedor → canónico
- **Tipos de transformación**:
  - `rename`: Renombrado de campos
  - `unit`: Conversión de unidades
  - `enum`: Mapeo de enumeraciones
  - `scale`: Escalado numérico
  - `timestamp`: Conversión de timestamps
  - `calculated`: Campos calculados

### Control de Acceso

#### Funciones de Validación

- `CanAccess(scope, capability, streamKey)`: Verifica permisos de acceso
- `ValidateConnectionForTenant()`: Valida ownership de conexiones
- `ValidateCapabilityAccess()`: Validación completa de acceso a capabilities

#### Sistema de Scopes

- **Global**: `*` - Acceso completo
- **Por Farm**: Restricción por FarmIDs específicos
- **Por Site**: Restricción por SiteIDs específicos
- **Por Cage**: Restricción por CageIDs específicos

## Patrones de Diseño

### Domain-Driven Design (DDD)

- Separación clara entre modelos de dominio y persistencia
- Validación de negocio en los modelos
- Agregados bien definidos con invariantes

### Error Handling

- Errores específicos del dominio con códigos identificadores
- Validación exhaustiva en todos los modelos
- Error wrapping para contexto adicional

### Factory Methods

- `NewTenant()`, `NewConnectionInstance()`, `NewMapping()`
- Inicialización consistente con valores por defecto
- Timestamps automáticos

### Value Objects

- `StreamKey`: Inmutable con validación
- `Capability`: Enum type-safe
- `ConfigSpec`: JSON Schema validation

## Casos de Uso

### 1. Configuración de Tenant

```go
tenant := NewTenant("acme-farms", "ACME Farms Inc", "admin")
tenant.Scopes = []Scope{
    {
        TenantID: tenant.ID,
        Resource: "*",
        Permissions: []Capability{CapabilityFeedingRead, CapabilityClimateRead},
        FarmIDs: []string{"farm-001", "farm-002"},
    },
}
```

### 2. Creación de Conexión

```go
connType := NewConnectorType("ModbusRTU", "Modbus RTU Connector", "v1.0", "admin")
connType.Capabilities = []Capability{CapabilityClimateRead}

connection := NewConnectionInstance(tenantID, connType.ID, "Greenhouse Sensors", "user")
connection.Config = map[string]interface{}{
    "port": "/dev/ttyUSB0",
    "baudrate": 9600,
}
```

### 3. Configuración de Mapping

```go
mapping := NewMapping("Temperature Data", CapabilityClimateRead)
mapping.AddRule(MappingRule{
    SourceField: "temp_celsius",
    TargetField: "temperature",
    Transform: &Transform{
        Type: TransformTypeUnit,
        Parameters: map[string]interface{}{
            "from": "celsius",
            "to": "kelvin",
        },
    },
    Required: true,
})
```

### 4. Validación de Acceso

```go
streamKey := NewStreamKey(tenantID, StreamKindClimate, "farm-001", "greenhouse-1", nil)
if CanAccess(userScope, CapabilityClimateRead, *streamKey) {
    // Procesar acceso a datos
}
```

## Testing

El paquete incluye tests unitarios exhaustivos que validan:

- Validación de todos los modelos
- Serialización/deserialización de StreamKeys
- Control de acceso y permisos
- Factory methods y inicialización

Para ejecutar los tests:

```bash
go test ./internal/domain -v
```

## Estructura de Archivos

```
internal/domain/
├── capability.go          # Enum de capabilities
├── tenant.go             # Modelo de tenant y scopes
├── stream_key.go         # Identificadores de streams
├── connector_type.go     # Tipos de conectores
├── connection_instance.go # Instancias de conexión y mappings
├── access_control.go     # Funciones de control de acceso
├── errors.go            # Errores específicos del dominio
└── domain_test.go       # Tests unitarios
```

## Próximos Pasos

1. **Servicios de Dominio**: Implementar servicios para operaciones complejas
2. **Event Sourcing**: Agregar eventos de dominio para auditoria
3. **Especificaciones**: Implementar pattern Specification para queries complejas
4. **Agregados**: Definir boundaries de agregados más claros
5. **Policies**: Implementar policies de negocio como objetos de primera clase
