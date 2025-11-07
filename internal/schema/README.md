# Schema Package - OmniAPI

Este paquete proporciona validación de JSON Schema para datos IoT/agrícolas en el sistema OmniAPI. Permite cargar y validar datos contra esquemas predefinidos con soporte para compatibilidad backward.

## Características

### ✅ **Carga Automática de Schemas**

- Carga desde directorio `configs/schemas`
- Formato: `{kind}.{version}.json`
- Validación automática de estructura

### ✅ **Validación de Datos**

- Función `Validate(kind, version, payload)`
- Errores detallados con campo y mensaje
- Soporte para payloads válidos/inválidos

### ✅ **Compatibilidad Backward**

- Campo `metadata.backward_compatible` en schemas
- Notas de compatibilidad documentadas
- Versionado semántico

### ✅ **Schemas Disponibles**

- **feeding.v1**: Datos de alimentación
- **biometric.v1**: Datos biométricos
- **climate.v1**: Datos climáticos/ambientales

## Uso Básico

### Validación Simple

```go
import "omniapi/internal/schema"

// Payload de ejemplo
payload := map[string]interface{}{
    "timestamp":  "2025-11-07T12:00:00Z",
    "device_id":  "feeder_001",
    "feed_type":  "pellets",
    "quantity":   150.5,
    "status":     "completed",
}

// Validar contra schema
result, err := schema.Validate("feeding", "v1", payload)
if err != nil {
    log.Fatal(err)
}

if !result.Valid {
    for _, error := range result.Errors {
        fmt.Printf("Error en %s: %s\n", error.Field, error.Message)
    }
}
```

### Gestor de Schemas Avanzado

```go
// Crear gestor con directorio personalizado
manager := schema.NewSchemaManager("custom/schemas")
err := manager.LoadSchemas()
if err != nil {
    log.Fatal(err)
}

// Obtener schema específico
feedingSchema, err := manager.GetSchema("feeding", "v1")
if err != nil {
    log.Fatal(err)
}

// Verificar compatibilidad
if feedingSchema.IsBackwardCompatible() {
    fmt.Println("Schema es backward compatible")
}

// Validar datos
result, err := manager.Validate("feeding", "v1", payload)
```

## Schemas Implementados

### 1. **Feeding Schema (feeding.v1.json)**

**Propósito**: Validación de datos de sistemas de alimentación

**Campos Requeridos**:

- `timestamp`: ISO 8601 timestamp
- `device_id`: ID único del dispositivo
- `feed_type`: Tipo de alimento (enum)
- `quantity`: Cantidad en gramos
- `status`: Estado de la operación

**Campos Opcionales**:

- `duration`, `feed_rate`, `hopper_level`
- `temperature`, `ph_level`, `cage_id`, `batch_id`

**Ejemplo**:

```json
{
  "timestamp": "2025-11-07T12:00:00Z",
  "device_id": "feeder_001",
  "feed_type": "pellets",
  "quantity": 150.5,
  "status": "completed",
  "cage_id": "cage_01",
  "hopper_level": 75.0
}
```

### 2. **Biometric Schema (biometric.v1.json)**

**Propósito**: Validación de datos biométricos de organismos acuáticos

**Campos Requeridos**:

- `timestamp`: ISO 8601 timestamp
- `device_id`: ID del sensor biométrico
- `organism_id`: ID único del organismo
- `species`: Especie del organismo
- `weight`: Peso en gramos

**Campos Opcionales**:

- `length`, `heart_rate`, `activity_level`
- `stress_indicators`, `health_score`, `body_condition`
- `age_estimate`, `growth_rate`, `mortality_risk`

**Ejemplo**:

```json
{
  "timestamp": "2025-11-07T12:00:00Z",
  "device_id": "bio_sensor_001",
  "organism_id": "fish_12345",
  "species": "Salmo salar",
  "weight": 245.8,
  "length": 25.4,
  "health_score": 87.5,
  "activity_level": "medium"
}
```

### 3. **Climate Schema (climate.v1.json)**

**Propósito**: Validación de datos ambientales/climáticos

**Campos Requeridos**:

- `timestamp`: ISO 8601 timestamp
- `device_id`: ID del sensor climático
- `temperature`: Temperatura en Celsius

**Campos Opcionales**:

- `humidity`, `water_temperature`, `dissolved_oxygen`
- `ph_level`, `salinity`, `ammonia`, `nitrite`, `nitrate`
- `turbidity`, `light_intensity`, `uv_index`
- `pressure`, `wind_speed`, `wind_direction`
- `location` (objeto con lat/lng)

**Ejemplo**:

```json
{
  "timestamp": "2025-11-07T12:00:00Z",
  "device_id": "climate_001",
  "temperature": 22.5,
  "humidity": 65.0,
  "water_temperature": 18.5,
  "dissolved_oxygen": 7.8,
  "ph_level": 7.2,
  "location": {
    "latitude": 40.7128,
    "longitude": -74.006,
    "altitude": 10.5
  }
}
```

## Validación de Errores

### Tipos de Errores Detectados

1. **Campos Requeridos Faltantes**

```json
{
  "valid": false,
  "errors": [
    {
      "field": "(root)",
      "message": "timestamp is required"
    }
  ]
}
```

2. **Tipos de Datos Incorrectos**

```json
{
  "valid": false,
  "errors": [
    {
      "field": "quantity",
      "message": "Invalid type. Expected: number, given: string",
      "value": "not_a_number"
    }
  ]
}
```

3. **Valores Fuera de Rango**

```json
{
  "valid": false,
  "errors": [
    {
      "field": "temperature",
      "message": "Must be greater than or equal to -50",
      "value": -100
    }
  ]
}
```

4. **Enum Inválido**

```json
{
  "valid": false,
  "errors": [
    {
      "field": "feed_type",
      "message": "feed_type does not match any of the allowed values",
      "value": "invalid_type"
    }
  ]
}
```

## Compatibilidad Backward

Todos los schemas incluyen metadatos de compatibilidad:

```json
{
  "metadata": {
    "version": "1.0.0",
    "capability": "feeding.read",
    "backward_compatible": true,
    "compatibility_notes": "Initial version - no breaking changes",
    "created_at": "2025-11-07T00:00:00Z",
    "updated_at": "2025-11-07T00:00:00Z"
  }
}
```

### Funciones de Compatibilidad

- `schema.IsBackwardCompatible()`: Verifica compatibilidad
- `schema.GetCapability()`: Obtiene capability asociada
- `schema.Metadata`: Acceso completo a metadatos

## Testing

### Tests Implementados ✅

1. **Carga de Schemas**: Verifica carga correcta desde directorio
2. **Obtención de Schemas**: Test de retrieval por kind/version
3. **Validación Feeding**: Payloads válidos/inválidos para feeding
4. **Validación Biometric**: Payloads válidos/inválidos para biometric
5. **Validación Climate**: Payloads válidos/inválidos para climate
6. **Compatibilidad Backward**: Verificación de metadatos
7. **Función de Conveniencia**: Test de `Validate()` global

### Ejecutar Tests

```bash
go test ./internal/schema -v
```

### Resultados de Tests

```
=== RUN   TestSchemaManager_LoadSchemas
--- PASS: TestSchemaManager_LoadSchemas (0.02s)
=== RUN   TestSchemaManager_GetSchema
--- PASS: TestSchemaManager_GetSchema (0.00s)
=== RUN   TestValidate_FeedingSchema
--- PASS: TestValidate_FeedingSchema (0.00s)
=== RUN   TestValidate_BiometricSchema
--- PASS: TestValidate_BiometricSchema (0.00s)
=== RUN   TestValidate_ClimateSchema
--- PASS: TestValidate_ClimateSchema (0.00s)
=== RUN   TestSchema_BackwardCompatibility
--- PASS: TestSchema_BackwardCompatibility (0.00s)
=== RUN   TestConvenienceValidateFunction
--- PASS: TestConvenienceValidateFunction (0.00s)
PASS
ok      omniapi/internal/schema 0.228s
```

## Estructura de Archivos

```
configs/
└── schemas/
    ├── feeding.v1.json    # Schema de alimentación
    ├── biometric.v1.json  # Schema biométrico
    └── climate.v1.json    # Schema climático

internal/schema/
├── schema.go      # Implementación principal
└── schema_test.go # Tests unitarios
```

## Integración con Domain Package

El paquete schema se integra perfectamente con el domain package:

```go
import (
    "omniapi/internal/domain"
    "omniapi/internal/schema"
)

// Validar datos antes de procesamiento
func ProcessFeedingData(streamKey domain.StreamKey, payload interface{}) error {
    // Validar schema
    result, err := schema.Validate("feeding", "v1", payload)
    if err != nil {
        return err
    }

    if !result.Valid {
        return fmt.Errorf("invalid payload: %v", result.Errors)
    }

    // Verificar permisos
    if !domain.CanAccess(userScope, domain.CapabilityFeedingRead, streamKey) {
        return domain.ErrUnauthorized
    }

    // Procesar datos validados...
    return nil
}
```

## Próximas Mejoras

1. **Versionado Automático**: Detección de cambios breaking
2. **Schema Registry**: Base de datos centralizada de schemas
3. **Transformaciones**: Auto-conversión entre versiones
4. **Métricas**: Estadísticas de validación y errores
5. **UI Schema Editor**: Interfaz web para editar schemas
6. **OpenAPI Integration**: Generación automática de specs
