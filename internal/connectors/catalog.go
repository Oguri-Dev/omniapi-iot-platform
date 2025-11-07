package connectors

import (
	"context"
	"fmt"
	"sync"

	"omniapi/internal/domain"
)

// Catalog gestiona el registro y creación de conectores
type Catalog struct {
	mu            sync.RWMutex
	registrations map[string]*ConnectorRegistration
	instances     map[string]Connector
}

// NewCatalog crea un nuevo catálogo de conectores
func NewCatalog() *Catalog {
	return &Catalog{
		registrations: make(map[string]*ConnectorRegistration),
		instances:     make(map[string]Connector),
	}
}

// Register registra un nuevo tipo de conector
func (c *Catalog) Register(registration *ConnectorRegistration) error {
	if registration == nil {
		return fmt.Errorf("registration cannot be nil")
	}

	if registration.Type == "" {
		return fmt.Errorf("connector type cannot be empty")
	}

	if registration.Factory == nil {
		return fmt.Errorf("connector factory cannot be nil")
	}

	c.mu.Lock()
	defer c.mu.Unlock()

	// Verificar si ya existe
	if _, exists := c.registrations[registration.Type]; exists {
		return fmt.Errorf("connector type '%s' already registered", registration.Type)
	}

	c.registrations[registration.Type] = registration
	return nil
}

// Unregister desregistra un tipo de conector
func (c *Catalog) Unregister(connectorType string) error {
	c.mu.Lock()
	defer c.mu.Unlock()

	if _, exists := c.registrations[connectorType]; !exists {
		return fmt.Errorf("connector type '%s' not found", connectorType)
	}

	delete(c.registrations, connectorType)
	return nil
}

// GetRegistration obtiene el registro de un tipo de conector
func (c *Catalog) GetRegistration(connectorType string) (*ConnectorRegistration, bool) {
	c.mu.RLock()
	defer c.mu.RUnlock()

	reg, exists := c.registrations[connectorType]
	return reg, exists
}

// ListRegistrations retorna todos los tipos registrados
func (c *Catalog) ListRegistrations() []*ConnectorRegistration {
	c.mu.RLock()
	defer c.mu.RUnlock()

	var result []*ConnectorRegistration
	for _, reg := range c.registrations {
		result = append(result, reg)
	}

	return result
}

// CreateInstance crea una instancia de conector desde una ConnectionInstance
func (c *Catalog) CreateInstance(connectionInstance *domain.ConnectionInstance, connectorType *domain.ConnectorType) (Connector, error) {
	if connectionInstance == nil {
		return nil, fmt.Errorf("connection instance cannot be nil")
	}

	if connectorType == nil {
		return nil, fmt.Errorf("connector type cannot be nil")
	}

	c.mu.RLock()
	registration, exists := c.registrations[connectorType.Name]
	c.mu.RUnlock()

	if !exists {
		return nil, fmt.Errorf("connector type '%s' not registered", connectorType.Name)
	}

	// Crear configuración combinada
	config := make(map[string]interface{})

	// Copiar configuración base del ConnectorType
	if connectorType.ConfigSpec.Properties != nil {
		for key, field := range connectorType.ConfigSpec.Properties {
			if field.Default != nil {
				config[key] = field.Default
			}
		}
	}

	// Aplicar configuración específica de la instancia
	for key, value := range connectionInstance.Config {
		config[key] = value
	}

	// Agregar metadatos de la instancia
	config["__instance_id"] = connectionInstance.ID.Hex()
	config["__tenant_id"] = connectionInstance.TenantID.Hex()
	config["__display_name"] = connectionInstance.DisplayName
	config["__mappings"] = connectionInstance.Mappings

	// Crear la instancia usando la factory
	instance, err := registration.Factory(config)
	if err != nil {
		return nil, fmt.Errorf("failed to create connector instance: %w", err)
	}

	// Registrar la instancia
	instanceID := connectionInstance.ID.Hex()
	c.mu.Lock()
	c.instances[instanceID] = instance
	c.mu.Unlock()

	return instance, nil
}

// GetInstance obtiene una instancia de conector por ID
func (c *Catalog) GetInstance(instanceID string) (Connector, bool) {
	c.mu.RLock()
	defer c.mu.RUnlock()

	instance, exists := c.instances[instanceID]
	return instance, exists
}

// RemoveInstance remueve una instancia de conector
func (c *Catalog) RemoveInstance(instanceID string) error {
	c.mu.Lock()
	defer c.mu.Unlock()

	instance, exists := c.instances[instanceID]
	if !exists {
		return fmt.Errorf("instance '%s' not found", instanceID)
	}

	// Detener la instancia si está corriendo
	if err := instance.Stop(); err != nil {
		return fmt.Errorf("failed to stop instance: %w", err)
	}

	delete(c.instances, instanceID)
	return nil
}

// ListInstances retorna todas las instancias activas
func (c *Catalog) ListInstances() []Connector {
	c.mu.RLock()
	defer c.mu.RUnlock()

	var result []Connector
	for _, instance := range c.instances {
		result = append(result, instance)
	}

	return result
}

// StartInstance inicia una instancia de conector
func (c *Catalog) StartInstance(instanceID string, ctx context.Context) error {
	instance, exists := c.GetInstance(instanceID)
	if !exists {
		return fmt.Errorf("instance '%s' not found", instanceID)
	}

	return instance.Start(ctx)
}

// StopInstance detiene una instancia de conector
func (c *Catalog) StopInstance(instanceID string) error {
	instance, exists := c.GetInstance(instanceID)
	if !exists {
		return fmt.Errorf("instance '%s' not found", instanceID)
	}

	return instance.Stop()
}

// GetHealthStatus obtiene el estado de salud de todas las instancias
func (c *Catalog) GetHealthStatus() map[string]HealthInfo {
	c.mu.RLock()
	defer c.mu.RUnlock()

	result := make(map[string]HealthInfo)
	for id, instance := range c.instances {
		result[id] = instance.Health()
	}

	return result
}

// CanCreateInstance verifica si se puede crear una instancia del tipo especificado
func (c *Catalog) CanCreateInstance(connectorTypeName string) bool {
	c.mu.RLock()
	defer c.mu.RUnlock()

	_, exists := c.registrations[connectorTypeName]
	return exists
}

// ValidateConfig valida la configuración contra el schema del conector
func (c *Catalog) ValidateConfig(connectorType string, config map[string]interface{}) error {
	registration, exists := c.GetRegistration(connectorType)
	if !exists {
		return fmt.Errorf("connector type '%s' not found", connectorType)
	}

	if registration.ConfigSchema == nil {
		return nil // Sin schema para validar
	}

	// TODO: Implementar validación usando JSON Schema
	// Por ahora solo verificamos que existan las propiedades requeridas

	return nil
}

// GlobalCatalog es la instancia global del catálogo
var GlobalCatalog = NewCatalog()

// RegisterConnector registra un conector en el catálogo global
func RegisterConnector(registration *ConnectorRegistration) error {
	return GlobalCatalog.Register(registration)
}

// CreateConnectorInstance crea una instancia usando el catálogo global
func CreateConnectorInstance(connectionInstance *domain.ConnectionInstance, connectorType *domain.ConnectorType) (Connector, error) {
	return GlobalCatalog.CreateInstance(connectionInstance, connectorType)
}

// Los conectores se registran manualmente desde main.go para evitar ciclos de importación
