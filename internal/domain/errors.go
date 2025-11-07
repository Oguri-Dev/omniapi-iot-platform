package domain

import "errors"

// Errores de dominio
var (
	// Errores de Tenant
	ErrTenantNameRequired        = errors.New("tenant name is required")
	ErrTenantDisplayNameRequired = errors.New("tenant display name is required")
	ErrInvalidTenantStatus       = errors.New("invalid tenant status")
	ErrTenantNotFound            = errors.New("tenant not found")

	// Errores de Quota
	ErrInvalidQuota  = errors.New("invalid quota configuration")
	ErrQuotaNotFound = errors.New("quota not found")
	ErrQuotaExceeded = errors.New("quota exceeded")

	// Errores de Scope
	ErrInvalidScope      = errors.New("invalid scope configuration")
	ErrAccessDenied      = errors.New("access denied")
	ErrInvalidCapability = errors.New("invalid capability")

	// Errores de StreamKey
	ErrInvalidStreamKey        = errors.New("invalid stream key")
	ErrStreamKeyTenantRequired = errors.New("stream key tenant ID is required")
	ErrStreamKeyKindRequired   = errors.New("stream key kind is required")
	ErrStreamKeyFarmRequired   = errors.New("stream key farm ID is required")
	ErrStreamKeySiteRequired   = errors.New("stream key site ID is required")

	// Errores de ConnectorType
	ErrConnectorTypeNotFound = errors.New("connector type not found")
	ErrInvalidConnectorType  = errors.New("invalid connector type")
	ErrInvalidConfigSpec     = errors.New("invalid config specification")
	ErrInvalidOutputSchema   = errors.New("invalid output schema")

	// Errores de ConnectionInstance
	ErrConnectionInstanceNotFound = errors.New("connection instance not found")
	ErrInvalidConnectionInstance  = errors.New("invalid connection instance")
	ErrInvalidSecretsRef          = errors.New("invalid secrets reference")
	ErrInvalidConfig              = errors.New("invalid configuration")
	ErrInvalidMapping             = errors.New("invalid mapping configuration")

	// Errores de Mapping
	ErrInvalidMappingRule        = errors.New("invalid mapping rule")
	ErrUnsupportedUnitConversion = errors.New("unsupported unit conversion")
	ErrInvalidEnumValue          = errors.New("invalid enum value")

	// Errores de Access Control
	ErrUnauthorized            = errors.New("unauthorized access")
	ErrConnectionNotActive     = errors.New("connection is not active")
	ErrCapabilityNotConfigured = errors.New("capability not configured for connection")
	ErrCapabilityNotSupported  = errors.New("capability not supported by connector type")
)
