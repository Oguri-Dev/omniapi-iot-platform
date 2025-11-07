package domain

import (
	"time"

	"go.mongodb.org/mongo-driver/bson/primitive"
)

// ConnectionStatus representa el estado de una conexión
type ConnectionStatus string

const (
	ConnectionStatusActive      ConnectionStatus = "active"
	ConnectionStatusInactive    ConnectionStatus = "inactive"
	ConnectionStatusError       ConnectionStatus = "error"
	ConnectionStatusConfiguring ConnectionStatus = "configuring"
	ConnectionStatusTesting     ConnectionStatus = "testing"
)

// MappingRule representa una regla de mapeo de datos
type MappingRule struct {
	SourceField  string      `json:"source_field" bson:"source_field"` // campo del proveedor
	TargetField  string      `json:"target_field" bson:"target_field"` // campo canónico
	Transform    *Transform  `json:"transform,omitempty" bson:"transform,omitempty"`
	DefaultValue interface{} `json:"default_value,omitempty" bson:"default_value,omitempty"`
	Required     bool        `json:"required" bson:"required"`
	Description  string      `json:"description,omitempty" bson:"description,omitempty"`
}

// Transform representa transformaciones de datos
type Transform struct {
	Type       TransformType          `json:"type" bson:"type"`
	Parameters map[string]interface{} `json:"parameters,omitempty" bson:"parameters,omitempty"`
}

// TransformType representa el tipo de transformación
type TransformType string

const (
	TransformTypeRename     TransformType = "rename"     // renombrar campo
	TransformTypeUnit       TransformType = "unit"       // conversión de unidades
	TransformTypeEnum       TransformType = "enum"       // mapeo de enums
	TransformTypeScale      TransformType = "scale"      // escalado numérico
	TransformTypeTimestamp  TransformType = "timestamp"  // conversión de timestamp
	TransformTypeCalculated TransformType = "calculated" // campo calculado
)

// Mapping representa el conjunto completo de reglas de mapeo
type Mapping struct {
	ID          primitive.ObjectID `json:"id" bson:"_id,omitempty"`
	Name        string             `json:"name" bson:"name"`
	Description string             `json:"description,omitempty" bson:"description,omitempty"`
	Capability  Capability         `json:"capability" bson:"capability"`
	Rules       []MappingRule      `json:"rules" bson:"rules"`

	// Configuración de validación
	StrictMode  bool `json:"strict_mode" bson:"strict_mode"`   // fallar si faltan campos requeridos
	IgnoreExtra bool `json:"ignore_extra" bson:"ignore_extra"` // ignorar campos no mapeados

	// Metadata
	CreatedAt time.Time `json:"created_at" bson:"created_at"`
	UpdatedAt time.Time `json:"updated_at" bson:"updated_at"`
	Version   string    `json:"version" bson:"version"`
}

// ConnectionInstance representa una instancia de conexión configurada
type ConnectionInstance struct {
	ID          primitive.ObjectID `json:"id" bson:"_id,omitempty"`
	TenantID    primitive.ObjectID `json:"tenant_id" bson:"tenant_id"`
	TypeID      primitive.ObjectID `json:"type_id" bson:"type_id"` // referencia a ConnectorType
	DisplayName string             `json:"display_name" bson:"display_name"`
	Description string             `json:"description,omitempty" bson:"description,omitempty"`

	// Configuración
	SecretsRef string                 `json:"secrets_ref" bson:"secrets_ref"` // referencia a secrets store
	Config     map[string]interface{} `json:"config" bson:"config"`           // configuración específica

	// Mapeos por capability
	Mappings []Mapping `json:"mappings" bson:"mappings"`

	// Estado y monitoreo
	Status         ConnectionStatus `json:"status" bson:"status"`
	LastConnection *time.Time       `json:"last_connection,omitempty" bson:"last_connection,omitempty"`
	LastError      string           `json:"last_error,omitempty" bson:"last_error,omitempty"`
	ErrorCount     int              `json:"error_count" bson:"error_count"`

	// Configuración de retry y timeouts
	RetryCount    int           `json:"retry_count" bson:"retry_count"`
	RetryInterval time.Duration `json:"retry_interval" bson:"retry_interval"`
	Timeout       time.Duration `json:"timeout" bson:"timeout"`

	// Metadata
	Tags      []string               `json:"tags,omitempty" bson:"tags,omitempty"`
	Metadata  map[string]interface{} `json:"metadata,omitempty" bson:"metadata,omitempty"`
	CreatedAt time.Time              `json:"created_at" bson:"created_at"`
	UpdatedAt time.Time              `json:"updated_at" bson:"updated_at"`
	CreatedBy string                 `json:"created_by" bson:"created_by"`
	UpdatedBy string                 `json:"updated_by" bson:"updated_by"`
}

// NewConnectionInstance crea una nueva instancia de conexión
func NewConnectionInstance(tenantID, typeID primitive.ObjectID, displayName, createdBy string) *ConnectionInstance {
	now := time.Now()
	return &ConnectionInstance{
		ID:            primitive.NewObjectID(),
		TenantID:      tenantID,
		TypeID:        typeID,
		DisplayName:   displayName,
		Config:        make(map[string]interface{}),
		Mappings:      []Mapping{},
		Status:        ConnectionStatusConfiguring,
		RetryCount:    3,
		RetryInterval: 30 * time.Second,
		Timeout:       60 * time.Second,
		Tags:          []string{},
		Metadata:      make(map[string]interface{}),
		CreatedAt:     now,
		UpdatedAt:     now,
		CreatedBy:     createdBy,
		UpdatedBy:     createdBy,
	}
}

// Validate valida la ConnectionInstance
func (ci *ConnectionInstance) Validate() error {
	if ci.TenantID.IsZero() {
		return ErrInvalidConnectionInstance
	}

	if ci.TypeID.IsZero() {
		return ErrInvalidConnectionInstance
	}

	if ci.DisplayName == "" {
		return ErrInvalidConnectionInstance
	}

	if !ci.Status.IsValid() {
		return ErrInvalidConnectionInstance
	}

	// Validar mappings
	for _, mapping := range ci.Mappings {
		if err := mapping.Validate(); err != nil {
			return err
		}
	}

	return nil
}

// IsValid verifica si el status de la conexión es válido
func (cs ConnectionStatus) IsValid() bool {
	switch cs {
	case ConnectionStatusActive, ConnectionStatusInactive, ConnectionStatusError,
		ConnectionStatusConfiguring, ConnectionStatusTesting:
		return true
	}
	return false
}

// IsActive verifica si la conexión está activa
func (ci *ConnectionInstance) IsActive() bool {
	return ci.Status == ConnectionStatusActive
}

// GetMappingByCapability obtiene el mapping para una capability específica
func (ci *ConnectionInstance) GetMappingByCapability(capability Capability) (*Mapping, bool) {
	for i, mapping := range ci.Mappings {
		if mapping.Capability == capability {
			return &ci.Mappings[i], true
		}
	}
	return nil, false
}

// AddMapping agrega un nuevo mapping
func (ci *ConnectionInstance) AddMapping(mapping Mapping) error {
	if err := mapping.Validate(); err != nil {
		return err
	}

	// Verificar que no exista ya un mapping para esta capability
	if _, exists := ci.GetMappingByCapability(mapping.Capability); exists {
		return ErrInvalidMapping
	}

	mapping.ID = primitive.NewObjectID()
	mapping.CreatedAt = time.Now()
	mapping.UpdatedAt = time.Now()

	ci.Mappings = append(ci.Mappings, mapping)
	ci.UpdatedAt = time.Now()

	return nil
}

// UpdateMapping actualiza un mapping existente
func (ci *ConnectionInstance) UpdateMapping(capability Capability, updatedMapping Mapping) error {
	if err := updatedMapping.Validate(); err != nil {
		return err
	}

	for i, mapping := range ci.Mappings {
		if mapping.Capability == capability {
			updatedMapping.ID = mapping.ID
			updatedMapping.CreatedAt = mapping.CreatedAt
			updatedMapping.UpdatedAt = time.Now()
			ci.Mappings[i] = updatedMapping
			ci.UpdatedAt = time.Now()
			return nil
		}
	}

	return ErrInvalidMapping
}

// RemoveMapping remueve un mapping
func (ci *ConnectionInstance) RemoveMapping(capability Capability) {
	newMappings := make([]Mapping, 0, len(ci.Mappings))
	for _, mapping := range ci.Mappings {
		if mapping.Capability != capability {
			newMappings = append(newMappings, mapping)
		}
	}
	ci.Mappings = newMappings
	ci.UpdatedAt = time.Now()
}

// SetStatus actualiza el status de la conexión
func (ci *ConnectionInstance) SetStatus(status ConnectionStatus, errorMsg string) {
	ci.Status = status
	if status == ConnectionStatusError {
		ci.LastError = errorMsg
		ci.ErrorCount++
	} else if status == ConnectionStatusActive {
		now := time.Now()
		ci.LastConnection = &now
		ci.LastError = ""
	}
	ci.UpdatedAt = time.Now()
}

// NewMapping crea un nuevo Mapping
func NewMapping(name string, capability Capability) *Mapping {
	now := time.Now()
	return &Mapping{
		ID:          primitive.NewObjectID(),
		Name:        name,
		Capability:  capability,
		Rules:       []MappingRule{},
		StrictMode:  false,
		IgnoreExtra: true,
		CreatedAt:   now,
		UpdatedAt:   now,
		Version:     "1.0.0",
	}
}

// Validate valida el Mapping
func (m *Mapping) Validate() error {
	if m.Name == "" {
		return ErrInvalidMapping
	}

	if !m.Capability.IsValid() {
		return ErrInvalidCapability
	}

	// Validar reglas
	targetFields := make(map[string]bool)
	for _, rule := range m.Rules {
		if err := rule.Validate(); err != nil {
			return err
		}

		// Verificar campos target únicos
		if targetFields[rule.TargetField] {
			return ErrInvalidMappingRule
		}
		targetFields[rule.TargetField] = true
	}

	return nil
}

// Validate valida la MappingRule
func (mr *MappingRule) Validate() error {
	if mr.SourceField == "" && mr.DefaultValue == nil {
		return ErrInvalidMappingRule
	}

	if mr.TargetField == "" {
		return ErrInvalidMappingRule
	}

	// Validar transform si existe
	if mr.Transform != nil {
		if err := mr.Transform.Validate(); err != nil {
			return err
		}
	}

	return nil
}

// Validate valida el Transform
func (t *Transform) Validate() error {
	switch t.Type {
	case TransformTypeRename, TransformTypeUnit, TransformTypeEnum,
		TransformTypeScale, TransformTypeTimestamp, TransformTypeCalculated:
		return nil
	default:
		return ErrInvalidMappingRule
	}
}

// AddRule agrega una nueva regla al mapping
func (m *Mapping) AddRule(rule MappingRule) error {
	if err := rule.Validate(); err != nil {
		return err
	}

	// Verificar que el campo target no exista ya
	for _, existingRule := range m.Rules {
		if existingRule.TargetField == rule.TargetField {
			return ErrInvalidMappingRule
		}
	}

	m.Rules = append(m.Rules, rule)
	m.UpdatedAt = time.Now()

	return nil
}

// RemoveRule remueve una regla del mapping
func (m *Mapping) RemoveRule(targetField string) {
	newRules := make([]MappingRule, 0, len(m.Rules))
	for _, rule := range m.Rules {
		if rule.TargetField != targetField {
			newRules = append(newRules, rule)
		}
	}
	m.Rules = newRules
	m.UpdatedAt = time.Now()
}
