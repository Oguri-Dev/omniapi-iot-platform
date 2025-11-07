package domain

import (
	"time"

	"go.mongodb.org/mongo-driver/bson/primitive"
)

// TenantStatus representa el estado de un tenant
type TenantStatus string

const (
	TenantStatusActive    TenantStatus = "active"
	TenantStatusInactive  TenantStatus = "inactive"
	TenantStatusSuspended TenantStatus = "suspended"
)

// Quota representa una cuota de recursos para un tenant
type Quota struct {
	Resource string `json:"resource" bson:"resource"` // ej: "api_calls", "storage_gb", "connections"
	Limit    int64  `json:"limit" bson:"limit"`       // límite máximo
	Used     int64  `json:"used" bson:"used"`         // cantidad utilizada actualmente
}

// IsExceeded verifica si la cuota ha sido excedida
func (q *Quota) IsExceeded() bool {
	return q.Used >= q.Limit
}

// Remaining retorna la cantidad restante disponible
func (q *Quota) Remaining() int64 {
	remaining := q.Limit - q.Used
	if remaining < 0 {
		return 0
	}
	return remaining
}

// PercentageUsed retorna el porcentaje utilizado de la cuota
func (q *Quota) PercentageUsed() float64 {
	if q.Limit == 0 {
		return 0
	}
	return float64(q.Used) / float64(q.Limit) * 100
}

// Scope representa un ámbito de acceso para un tenant
type Scope struct {
	TenantID    primitive.ObjectID `json:"tenant_id" bson:"tenant_id"`                   // ID del tenant
	Resource    string             `json:"resource" bson:"resource"`                     // ej: "farm:123", "site:456", "*"
	Permissions []Capability       `json:"permissions" bson:"permissions"`               // capacidades permitidas en este recurso
	FarmIDs     []string           `json:"farm_ids,omitempty" bson:"farm_ids,omitempty"` // IDs de granjas específicas
	SiteIDs     []string           `json:"site_ids,omitempty" bson:"site_ids,omitempty"` // IDs de sitios específicos
	CageIDs     []string           `json:"cage_ids,omitempty" bson:"cage_ids,omitempty"` // IDs de jaulas específicas
}

// HasCapability verifica si el scope tiene una capability específica
func (s *Scope) HasCapability(capability Capability) bool {
	for _, perm := range s.Permissions {
		if perm == capability {
			return true
		}
	}
	return false
}

// CanAccessStream verifica si el scope puede acceder a un stream específico
func (s *Scope) CanAccessStream(streamKey StreamKey) bool {
	// El tenant debe coincidir
	if s.TenantID != streamKey.TenantID {
		return false
	}

	// Si es acceso global, permitir
	if s.Resource == "*" {
		return true
	}

	// Verificar acceso por Farm
	if len(s.FarmIDs) > 0 {
		farmAllowed := false
		for _, farmID := range s.FarmIDs {
			if farmID == streamKey.FarmID {
				farmAllowed = true
				break
			}
		}
		if !farmAllowed {
			return false
		}
	}

	// Verificar acceso por Site
	if len(s.SiteIDs) > 0 {
		siteAllowed := false
		for _, siteID := range s.SiteIDs {
			if siteID == streamKey.SiteID {
				siteAllowed = true
				break
			}
		}
		if !siteAllowed {
			return false
		}
	}

	// Verificar acceso por Cage (si se especifica en el streamKey)
	if streamKey.CageID != nil && *streamKey.CageID != "" {
		if len(s.CageIDs) > 0 {
			cageAllowed := false
			for _, cageID := range s.CageIDs {
				if cageID == *streamKey.CageID {
					cageAllowed = true
					break
				}
			}
			if !cageAllowed {
				return false
			}
		}
	}

	return true
}

// Tenant representa una organización o cliente en el sistema
type Tenant struct {
	ID          primitive.ObjectID `json:"id" bson:"_id,omitempty"`
	Name        string             `json:"name" bson:"name"`
	DisplayName string             `json:"display_name" bson:"display_name"`
	Status      TenantStatus       `json:"status" bson:"status"`

	// Quotas y límites
	Quotas []Quota `json:"quotas" bson:"quotas"`

	// Permisos y alcances
	Scopes []Scope `json:"scopes" bson:"scopes"`

	// Configuración adicional
	Settings map[string]interface{} `json:"settings,omitempty" bson:"settings,omitempty"`

	// Metadata
	CreatedAt time.Time `json:"created_at" bson:"created_at"`
	UpdatedAt time.Time `json:"updated_at" bson:"updated_at"`
	CreatedBy string    `json:"created_by" bson:"created_by"`
	UpdatedBy string    `json:"updated_by" bson:"updated_by"`
}

// NewTenant crea un nuevo tenant con valores por defecto
func NewTenant(name, displayName, createdBy string) *Tenant {
	now := time.Now()
	return &Tenant{
		ID:          primitive.NewObjectID(),
		Name:        name,
		DisplayName: displayName,
		Status:      TenantStatusActive,
		Quotas:      defaultQuotas(),
		Scopes:      []Scope{},
		Settings:    make(map[string]interface{}),
		CreatedAt:   now,
		UpdatedAt:   now,
		CreatedBy:   createdBy,
		UpdatedBy:   createdBy,
	}
}

// defaultQuotas retorna las quotas por defecto para un tenant
func defaultQuotas() []Quota {
	return []Quota{
		{Resource: "api_calls_per_hour", Limit: 1000, Used: 0},
		{Resource: "storage_gb", Limit: 10, Used: 0},
		{Resource: "connections", Limit: 5, Used: 0},
		{Resource: "streams", Limit: 50, Used: 0},
	}
}

// Validate valida los datos del tenant
func (t *Tenant) Validate() error {
	if t.Name == "" {
		return ErrTenantNameRequired
	}

	if t.DisplayName == "" {
		return ErrTenantDisplayNameRequired
	}

	if !t.Status.IsValid() {
		return ErrInvalidTenantStatus
	}

	// Validar quotas
	for _, quota := range t.Quotas {
		if quota.Resource == "" {
			return ErrInvalidQuota
		}
		if quota.Limit < 0 {
			return ErrInvalidQuota
		}
	}

	// Validar scopes
	for _, scope := range t.Scopes {
		if scope.Resource == "" {
			return ErrInvalidScope
		}
		for _, perm := range scope.Permissions {
			if !perm.IsValid() {
				return ErrInvalidCapability
			}
		}
	}

	return nil
}

// IsActive verifica si el tenant está activo
func (t *Tenant) IsActive() bool {
	return t.Status == TenantStatusActive
}

// GetQuota obtiene una quota específica por resource
func (t *Tenant) GetQuota(resource string) (*Quota, bool) {
	for i := range t.Quotas {
		if t.Quotas[i].Resource == resource {
			return &t.Quotas[i], true
		}
	}
	return nil, false
}

// UpdateQuotaUsage actualiza el uso de una quota específica
func (t *Tenant) UpdateQuotaUsage(resource string, used int64) error {
	for i := range t.Quotas {
		if t.Quotas[i].Resource == resource {
			t.Quotas[i].Used = used
			t.UpdatedAt = time.Now()
			return nil
		}
	}
	return ErrQuotaNotFound
}

// HasScope verifica si el tenant tiene acceso a un recurso específico
func (t *Tenant) HasScope(resource string) bool {
	for _, scope := range t.Scopes {
		if scope.Resource == "*" || scope.Resource == resource {
			return true
		}
	}
	return false
}

// GetCapabilitiesForResource obtiene las capacidades disponibles para un recurso
func (t *Tenant) GetCapabilitiesForResource(resource string) []Capability {
	var capabilities []Capability

	for _, scope := range t.Scopes {
		if scope.Resource == "*" || scope.Resource == resource {
			capabilities = append(capabilities, scope.Permissions...)
		}
	}

	return capabilities
}

// IsValid verifica si el status del tenant es válido
func (ts TenantStatus) IsValid() bool {
	switch ts {
	case TenantStatusActive, TenantStatusInactive, TenantStatusSuspended:
		return true
	}
	return false
}
