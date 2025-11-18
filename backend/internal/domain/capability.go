package domain

import (
	"encoding/json"
	"fmt"
	"strings"
)

// Capability representa las capacidades disponibles en el sistema
type Capability string

const (
	// CapabilityFeedingRead permite leer datos de alimentación
	CapabilityFeedingRead Capability = "feeding.read"

	// CapabilityBiometricRead permite leer datos biométricos
	CapabilityBiometricRead Capability = "biometric.read"

	// CapabilityClimateRead permite leer datos climáticos
	CapabilityClimateRead Capability = "climate.read"

	// CapabilityOpsRead permite leer datos operacionales
	CapabilityOpsRead Capability = "ops.read"
)

// AllCapabilities retorna todas las capacidades disponibles
func AllCapabilities() []Capability {
	return []Capability{
		CapabilityFeedingRead,
		CapabilityBiometricRead,
		CapabilityClimateRead,
		CapabilityOpsRead,
	}
}

// IsValid verifica si la capability es válida
func (c Capability) IsValid() bool {
	for _, valid := range AllCapabilities() {
		if c == valid {
			return true
		}
	}
	return false
}

// String implementa el método Stringer
func (c Capability) String() string {
	return string(c)
}

// MarshalJSON implementa json.Marshaler
func (c Capability) MarshalJSON() ([]byte, error) {
	return json.Marshal(string(c))
}

// UnmarshalJSON implementa json.Unmarshaler
func (c *Capability) UnmarshalJSON(data []byte) error {
	var s string
	if err := json.Unmarshal(data, &s); err != nil {
		return err
	}

	capability := Capability(s)
	if !capability.IsValid() {
		return fmt.Errorf("invalid capability: %s", s)
	}

	*c = capability
	return nil
}

// Category retorna la categoría de la capability (parte antes del punto)
func (c Capability) Category() string {
	parts := strings.Split(string(c), ".")
	if len(parts) > 0 {
		return parts[0]
	}
	return ""
}

// Action retorna la acción de la capability (parte después del punto)
func (c Capability) Action() string {
	parts := strings.Split(string(c), ".")
	if len(parts) > 1 {
		return parts[1]
	}
	return ""
}
