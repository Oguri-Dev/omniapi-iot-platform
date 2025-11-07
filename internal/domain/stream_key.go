package domain

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"strings"

	"go.mongodb.org/mongo-driver/bson/primitive"
)

// StreamKind representa el tipo de stream de datos
type StreamKind string

const (
	StreamKindFeeding   StreamKind = "feeding"
	StreamKindBiometric StreamKind = "biometric"
	StreamKindClimate   StreamKind = "climate"
	StreamKindOps       StreamKind = "ops"
)

// AllStreamKinds retorna todos los tipos de stream disponibles
func AllStreamKinds() []StreamKind {
	return []StreamKind{
		StreamKindFeeding,
		StreamKindBiometric,
		StreamKindClimate,
		StreamKindOps,
	}
}

// IsValid verifica si el StreamKind es válido
func (sk StreamKind) IsValid() bool {
	for _, valid := range AllStreamKinds() {
		if sk == valid {
			return true
		}
	}
	return false
}

// String implementa el método Stringer
func (sk StreamKind) String() string {
	return string(sk)
}

// StreamKey representa una clave única para identificar un stream de datos
type StreamKey struct {
	TenantID primitive.ObjectID `json:"tenant_id" bson:"tenant_id"`                 // ID del tenant
	Kind     StreamKind         `json:"kind" bson:"kind"`                           // Tipo de stream
	FarmID   string             `json:"farm_id" bson:"farm_id"`                     // ID de la granja
	SiteID   string             `json:"site_id" bson:"site_id"`                     // ID del sitio
	CageID   *string            `json:"cage_id,omitempty" bson:"cage_id,omitempty"` // ID de la jaula (opcional)
}

// NewStreamKey crea una nueva StreamKey
func NewStreamKey(tenantID primitive.ObjectID, kind StreamKind, farmID, siteID string, cageID *string) *StreamKey {
	return &StreamKey{
		TenantID: tenantID,
		Kind:     kind,
		FarmID:   farmID,
		SiteID:   siteID,
		CageID:   cageID,
	}
}

// Validate valida la StreamKey
func (sk *StreamKey) Validate() error {
	if sk.TenantID.IsZero() {
		return ErrStreamKeyTenantRequired
	}

	if !sk.Kind.IsValid() {
		return ErrStreamKeyKindRequired
	}

	if sk.FarmID == "" {
		return ErrStreamKeyFarmRequired
	}

	if sk.SiteID == "" {
		return ErrStreamKeySiteRequired
	}

	return nil
}

// String genera una representación en string de la StreamKey
func (sk *StreamKey) String() string {
	parts := []string{
		sk.TenantID.Hex(),
		string(sk.Kind),
		sk.FarmID,
		sk.SiteID,
	}

	if sk.CageID != nil && *sk.CageID != "" {
		parts = append(parts, *sk.CageID)
	}

	return strings.Join(parts, ":")
}

// Hash genera un hash SHA256 de la StreamKey
func (sk *StreamKey) Hash() string {
	keyString := sk.String()
	hash := sha256.Sum256([]byte(keyString))
	return hex.EncodeToString(hash[:])
}

// ParseStreamKey parsea una string a StreamKey
func ParseStreamKey(keyString string) (*StreamKey, error) {
	parts := strings.Split(keyString, ":")

	if len(parts) < 4 || len(parts) > 5 {
		return nil, ErrInvalidStreamKey
	}

	// Parsear TenantID
	tenantID, err := primitive.ObjectIDFromHex(parts[0])
	if err != nil {
		return nil, ErrInvalidStreamKey
	}

	kind := StreamKind(parts[1])
	if !kind.IsValid() {
		return nil, ErrInvalidStreamKey
	}

	streamKey := &StreamKey{
		TenantID: tenantID,
		Kind:     kind,
		FarmID:   parts[2],
		SiteID:   parts[3],
	}

	// CageID opcional
	if len(parts) == 5 && parts[4] != "" {
		streamKey.CageID = &parts[4]
	}

	return streamKey, streamKey.Validate()
}

// Equals verifica si dos StreamKeys son iguales
func (sk *StreamKey) Equals(other *StreamKey) bool {
	if other == nil {
		return false
	}

	if sk.TenantID != other.TenantID ||
		sk.Kind != other.Kind ||
		sk.FarmID != other.FarmID ||
		sk.SiteID != other.SiteID {
		return false
	}

	// Comparar CageID (ambos nil, ambos con valor, o diferentes)
	if sk.CageID == nil && other.CageID == nil {
		return true
	}

	if sk.CageID != nil && other.CageID != nil {
		return *sk.CageID == *other.CageID
	}

	return false
}

// GetResourceScope retorna el scope del recurso para esta StreamKey
func (sk *StreamKey) GetResourceScope() string {
	if sk.CageID != nil && *sk.CageID != "" {
		return fmt.Sprintf("cage:%s", *sk.CageID)
	}
	return fmt.Sprintf("site:%s", sk.SiteID)
}

// GetFarmScope retorna el scope de la granja
func (sk *StreamKey) GetFarmScope() string {
	return fmt.Sprintf("farm:%s", sk.FarmID)
}

// GetCapabilityRequired retorna la capability requerida para este stream
func (sk *StreamKey) GetCapabilityRequired() Capability {
	switch sk.Kind {
	case StreamKindFeeding:
		return CapabilityFeedingRead
	case StreamKindBiometric:
		return CapabilityBiometricRead
	case StreamKindClimate:
		return CapabilityClimateRead
	case StreamKindOps:
		return CapabilityOpsRead
	default:
		return ""
	}
}
