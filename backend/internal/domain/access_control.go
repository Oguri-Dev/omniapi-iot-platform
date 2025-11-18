package domain

import "go.mongodb.org/mongo-driver/bson/primitive"

// CanAccess verifica si un scope tiene acceso a una capability específica en un stream
func CanAccess(scope Scope, capability Capability, streamKey StreamKey) bool {
	// Verificar si el scope incluye la capability
	if !scope.HasCapability(capability) {
		return false
	}

	// Verificar acceso por TenantID (siempre debe coincidir)
	if scope.TenantID != streamKey.TenantID {
		return false
	}

	// Verificar restricciones específicas del scope
	return scope.CanAccessStream(streamKey)
}

// ValidateConnectionForTenant verifica que una ConnectionInstance pertenezca al tenant
func ValidateConnectionForTenant(connection *ConnectionInstance, tenantID primitive.ObjectID) error {
	if connection.TenantID != tenantID {
		return ErrUnauthorized
	}
	return nil
}

// ValidateStreamKeyForConnection verifica que un StreamKey sea válido para una conexión
func ValidateStreamKeyForConnection(streamKey StreamKey, connection *ConnectionInstance) error {
	// El StreamKey debe pertenecer al mismo tenant que la conexión
	if streamKey.TenantID != connection.TenantID {
		return ErrUnauthorized
	}

	// Validar el StreamKey
	if err := streamKey.Validate(); err != nil {
		return err
	}

	return nil
}

// GetCapabilitiesForConnection obtiene las capabilities disponibles en una conexión
func GetCapabilitiesForConnection(connection *ConnectionInstance, connectorType *ConnectorType) []Capability {
	if connection == nil || connectorType == nil {
		return []Capability{}
	}

	// Las capabilities de la conexión son las que están configuradas en mappings
	// y que también están soportadas por el ConnectorType
	var capabilities []Capability
	capabilityMap := make(map[Capability]bool)

	// Capabilities soportadas por el ConnectorType
	supportedCaps := make(map[Capability]bool)
	for _, cap := range connectorType.Capabilities {
		supportedCaps[cap] = true
	}

	// Capabilities con mappings configurados
	for _, mapping := range connection.Mappings {
		if supportedCaps[mapping.Capability] && !capabilityMap[mapping.Capability] {
			capabilities = append(capabilities, mapping.Capability)
			capabilityMap[mapping.Capability] = true
		}
	}

	return capabilities
}

// ValidateCapabilityAccess valida acceso a una capability específica
func ValidateCapabilityAccess(scope Scope, capability Capability, streamKey StreamKey, connection *ConnectionInstance) error {
	// Verificar acceso básico
	if !CanAccess(scope, capability, streamKey) {
		return ErrUnauthorized
	}

	// Verificar que la conexión pertenezca al tenant
	if err := ValidateConnectionForTenant(connection, scope.TenantID); err != nil {
		return err
	}

	// Verificar que la conexión esté activa
	if !connection.IsActive() {
		return ErrConnectionNotActive
	}

	// Verificar que la conexión tenga mapping para esta capability
	if _, exists := connection.GetMappingByCapability(capability); !exists {
		return ErrCapabilityNotConfigured
	}

	return nil
}

// GetFilteredConnections filtra conexiones basado en el scope
func GetFilteredConnections(connections []ConnectionInstance, scope Scope) []ConnectionInstance {
	var filtered []ConnectionInstance

	for _, conn := range connections {
		// Solo incluir conexiones del mismo tenant
		if conn.TenantID == scope.TenantID {
			// TODO: Aquí se podrían agregar filtros adicionales basados en
			// otras restricciones del scope como FarmIDs, SiteIDs, etc.
			filtered = append(filtered, conn)
		}
	}

	return filtered
}

// ValidateMappingCompatibility verifica que un mapping sea compatible con un ConnectorType
func ValidateMappingCompatibility(mapping *Mapping, connectorType *ConnectorType) error {
	if mapping == nil || connectorType == nil {
		return ErrInvalidMapping
	}

	// Verificar que el ConnectorType soporte la capability del mapping
	capabilitySupported := false
	for _, cap := range connectorType.Capabilities {
		if cap == mapping.Capability {
			capabilitySupported = true
			break
		}
	}

	if !capabilitySupported {
		return ErrCapabilityNotSupported
	}

	// Verificar que exista un schema de salida para esta capability
	schemaExists := false
	for _, schema := range connectorType.OutputSchemas {
		if schema.Capability == mapping.Capability {
			schemaExists = true
			break
		}
	}

	if !schemaExists {
		return ErrCapabilityNotSupported
	}

	return nil
}
