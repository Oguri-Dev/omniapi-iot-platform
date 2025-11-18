package domain

import (
	"time"

	"go.mongodb.org/mongo-driver/bson/primitive"
)

// ConnectorStatus representa el estado de un tipo de conector
type ConnectorStatus string

const (
	ConnectorStatusActive     ConnectorStatus = "active"
	ConnectorStatusDeprecated ConnectorStatus = "deprecated"
	ConnectorStatusDisabled   ConnectorStatus = "disabled"
)

// ConfigField representa un campo de configuración en el JSON Schema
type ConfigField struct {
	Type        string                 `json:"type" bson:"type"` // string, number, boolean, object, array
	Description string                 `json:"description,omitempty" bson:"description,omitempty"`
	Required    bool                   `json:"required" bson:"required"`
	Default     interface{}            `json:"default,omitempty" bson:"default,omitempty"`
	Enum        []interface{}          `json:"enum,omitempty" bson:"enum,omitempty"`
	Properties  map[string]ConfigField `json:"properties,omitempty" bson:"properties,omitempty"` // para type: object
	Items       *ConfigField           `json:"items,omitempty" bson:"items,omitempty"`           // para type: array
}

// ConfigSpec representa la especificación de configuración como JSON Schema
type ConfigSpec struct {
	Schema     string                 `json:"$schema" bson:"schema"`
	Type       string                 `json:"type" bson:"type"` // siempre "object"
	Properties map[string]ConfigField `json:"properties" bson:"properties"`
	Required   []string               `json:"required" bson:"required"`
}

// OutputSchema representa el esquema de salida de datos para una capability
type OutputSchema struct {
	Capability Capability             `json:"capability" bson:"capability"`
	Schema     map[string]interface{} `json:"schema" bson:"schema"` // JSON Schema completo
	Version    string                 `json:"version" bson:"version"`
	Examples   []interface{}          `json:"examples,omitempty" bson:"examples,omitempty"`
}

// ConnectorType representa un tipo de conector en el sistema
type ConnectorType struct {
	ID          primitive.ObjectID `json:"id" bson:"_id,omitempty"`
	Name        string             `json:"name" bson:"name"`                 // ej: "ModbusConnector"
	DisplayName string             `json:"display_name" bson:"display_name"` // ej: "Modbus RTU/TCP Connector"
	Description string             `json:"description" bson:"description"`
	Version     string             `json:"version" bson:"version"` // ej: "1.2.0"
	Vendor      string             `json:"vendor,omitempty" bson:"vendor,omitempty"`
	Status      ConnectorStatus    `json:"status" bson:"status"`

	// Capacidades soportadas
	Capabilities []Capability `json:"capabilities" bson:"capabilities"`

	// Especificación de configuración (JSON Schema)
	ConfigSpec ConfigSpec `json:"config_spec" bson:"config_spec"`

	// Esquemas de salida por capability
	OutputSchemas []OutputSchema `json:"output_schemas" bson:"output_schemas"`

	// Metadatos adicionales
	Tags     []string               `json:"tags,omitempty" bson:"tags,omitempty"`
	Metadata map[string]interface{} `json:"metadata,omitempty" bson:"metadata,omitempty"`

	// Información de versionado y lifecycle
	CreatedAt time.Time `json:"created_at" bson:"created_at"`
	UpdatedAt time.Time `json:"updated_at" bson:"updated_at"`
	CreatedBy string    `json:"created_by" bson:"created_by"`
	UpdatedBy string    `json:"updated_by" bson:"updated_by"`
}

// NewConnectorType crea un nuevo ConnectorType
func NewConnectorType(name, displayName, description, version, createdBy string) *ConnectorType {
	now := time.Now()
	return &ConnectorType{
		ID:            primitive.NewObjectID(),
		Name:          name,
		DisplayName:   displayName,
		Description:   description,
		Version:       version,
		Status:        ConnectorStatusActive,
		Capabilities:  []Capability{},
		OutputSchemas: []OutputSchema{},
		Tags:          []string{},
		Metadata:      make(map[string]interface{}),
		CreatedAt:     now,
		UpdatedAt:     now,
		CreatedBy:     createdBy,
		UpdatedBy:     createdBy,
	}
}

// Validate valida el ConnectorType
func (ct *ConnectorType) Validate() error {
	if ct.Name == "" {
		return ErrInvalidConnectorType
	}

	if ct.DisplayName == "" {
		return ErrInvalidConnectorType
	}

	if ct.Version == "" {
		return ErrInvalidConnectorType
	}

	if !ct.Status.IsValid() {
		return ErrInvalidConnectorType
	}

	// Validar capabilities
	for _, cap := range ct.Capabilities {
		if !cap.IsValid() {
			return ErrInvalidCapability
		}
	}

	// Validar que todas las capabilities en OutputSchemas estén en Capabilities
	capMap := make(map[Capability]bool)
	for _, cap := range ct.Capabilities {
		capMap[cap] = true
	}

	for _, schema := range ct.OutputSchemas {
		if !capMap[schema.Capability] {
			return ErrInvalidOutputSchema
		}
	}

	return nil
}

// IsValid verifica si el status del connector es válido
func (cs ConnectorStatus) IsValid() bool {
	switch cs {
	case ConnectorStatusActive, ConnectorStatusDeprecated, ConnectorStatusDisabled:
		return true
	}
	return false
}

// IsActive verifica si el ConnectorType está activo
func (ct *ConnectorType) IsActive() bool {
	return ct.Status == ConnectorStatusActive
}

// HasCapability verifica si el ConnectorType soporta una capability específica
func (ct *ConnectorType) HasCapability(capability Capability) bool {
	for _, cap := range ct.Capabilities {
		if cap == capability {
			return true
		}
	}
	return false
}

// GetOutputSchema obtiene el esquema de salida para una capability específica
func (ct *ConnectorType) GetOutputSchema(capability Capability) (*OutputSchema, bool) {
	for _, schema := range ct.OutputSchemas {
		if schema.Capability == capability {
			return &schema, true
		}
	}
	return nil, false
}

// AddCapability agrega una nueva capability al ConnectorType
func (ct *ConnectorType) AddCapability(capability Capability, outputSchema *OutputSchema) error {
	if !capability.IsValid() {
		return ErrInvalidCapability
	}

	// Verificar que no exista ya
	if ct.HasCapability(capability) {
		return nil // Ya existe, no error
	}

	ct.Capabilities = append(ct.Capabilities, capability)

	if outputSchema != nil {
		outputSchema.Capability = capability
		ct.OutputSchemas = append(ct.OutputSchemas, *outputSchema)
	}

	ct.UpdatedAt = time.Now()
	return nil
}

// RemoveCapability remueve una capability del ConnectorType
func (ct *ConnectorType) RemoveCapability(capability Capability) {
	// Remover de Capabilities
	newCaps := make([]Capability, 0, len(ct.Capabilities))
	for _, cap := range ct.Capabilities {
		if cap != capability {
			newCaps = append(newCaps, cap)
		}
	}
	ct.Capabilities = newCaps

	// Remover de OutputSchemas
	newSchemas := make([]OutputSchema, 0, len(ct.OutputSchemas))
	for _, schema := range ct.OutputSchemas {
		if schema.Capability != capability {
			newSchemas = append(newSchemas, schema)
		}
	}
	ct.OutputSchemas = newSchemas

	ct.UpdatedAt = time.Now()
}

// ValidateConfig valida una configuración contra el ConfigSpec
func (ct *ConnectorType) ValidateConfig(config map[string]interface{}) error {
	// Verificar campos requeridos
	for _, required := range ct.ConfigSpec.Required {
		if _, exists := config[required]; !exists {
			return ErrInvalidConfig
		}
	}

	// Validar tipos de campos (validación básica)
	for fieldName, fieldSpec := range ct.ConfigSpec.Properties {
		if value, exists := config[fieldName]; exists {
			if !ct.validateFieldType(value, fieldSpec) {
				return ErrInvalidConfig
			}
		}
	}

	return nil
}

// validateFieldType valida el tipo de un campo específico
func (ct *ConnectorType) validateFieldType(value interface{}, fieldSpec ConfigField) bool {
	switch fieldSpec.Type {
	case "string":
		_, ok := value.(string)
		return ok
	case "number":
		switch value.(type) {
		case int, int32, int64, float32, float64:
			return true
		}
		return false
	case "boolean":
		_, ok := value.(bool)
		return ok
	case "object":
		valueMap, ok := value.(map[string]interface{})
		if !ok {
			return false
		}
		// Validación recursiva de propiedades del objeto
		for propName, propSpec := range fieldSpec.Properties {
			if propValue, exists := valueMap[propName]; exists {
				if !ct.validateFieldType(propValue, propSpec) {
					return false
				}
			}
		}
		return true
	case "array":
		valueArray, ok := value.([]interface{})
		if !ok {
			return false
		}
		// Validar elementos del array si se especifica el tipo de items
		if fieldSpec.Items != nil {
			for _, item := range valueArray {
				if !ct.validateFieldType(item, *fieldSpec.Items) {
					return false
				}
			}
		}
		return true
	}
	return false
}
