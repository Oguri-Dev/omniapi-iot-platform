package models

import (
	"time"

	"go.mongodb.org/mongo-driver/bson/primitive"
)

// User modelo para usuarios en MongoDB
type User struct {
	ID        primitive.ObjectID     `bson:"_id,omitempty" json:"id,omitempty"`
	Username  string                 `bson:"username" json:"username"`
	Email     string                 `bson:"email" json:"email"`
	Password  string                 `bson:"password" json:"-"` // No incluir en JSON
	FullName  string                 `bson:"full_name" json:"full_name"`
	Avatar    string                 `bson:"avatar,omitempty" json:"avatar,omitempty"`
	Status    string                 `bson:"status" json:"status"` // active, inactive, banned
	Role      string                 `bson:"role" json:"role"`     // admin, user, moderator
	CreatedAt time.Time              `bson:"created_at" json:"created_at"`
	UpdatedAt time.Time              `bson:"updated_at" json:"updated_at"`
	LastLogin *time.Time             `bson:"last_login,omitempty" json:"last_login,omitempty"`
	Metadata  map[string]interface{} `bson:"metadata,omitempty" json:"metadata,omitempty"`
}

// Message modelo para mensajes de chat en MongoDB
type Message struct {
	ID        primitive.ObjectID     `bson:"_id,omitempty" json:"id,omitempty"`
	Type      string                 `bson:"type" json:"type"` // chat, system, notification
	Content   string                 `bson:"content" json:"content"`
	FromUser  primitive.ObjectID     `bson:"from_user,omitempty" json:"from_user,omitempty"`
	ToUser    *primitive.ObjectID    `bson:"to_user,omitempty" json:"to_user,omitempty"` // null = broadcast
	Channel   string                 `bson:"channel" json:"channel"`                     // general, private, etc.
	CreatedAt time.Time              `bson:"created_at" json:"created_at"`
	ReadBy    []primitive.ObjectID   `bson:"read_by,omitempty" json:"read_by,omitempty"`
	Metadata  map[string]interface{} `bson:"metadata,omitempty" json:"metadata,omitempty"`
}

// Session modelo para sesiones de usuario
type Session struct {
	ID        primitive.ObjectID `bson:"_id,omitempty" json:"id,omitempty"`
	UserID    primitive.ObjectID `bson:"user_id" json:"user_id"`
	Token     string             `bson:"token" json:"token"`
	IPAddress string             `bson:"ip_address" json:"ip_address"`
	UserAgent string             `bson:"user_agent" json:"user_agent"`
	CreatedAt time.Time          `bson:"created_at" json:"created_at"`
	ExpiresAt time.Time          `bson:"expires_at" json:"expires_at"`
	IsActive  bool               `bson:"is_active" json:"is_active"`
}

// APILog modelo para logs de API
type APILog struct {
	ID           primitive.ObjectID  `bson:"_id,omitempty" json:"id,omitempty"`
	Method       string              `bson:"method" json:"method"`
	Path         string              `bson:"path" json:"path"`
	StatusCode   int                 `bson:"status_code" json:"status_code"`
	UserID       *primitive.ObjectID `bson:"user_id,omitempty" json:"user_id,omitempty"`
	IPAddress    string              `bson:"ip_address" json:"ip_address"`
	UserAgent    string              `bson:"user_agent" json:"user_agent"`
	Duration     time.Duration       `bson:"duration" json:"duration"`
	CreatedAt    time.Time           `bson:"created_at" json:"created_at"`
	RequestBody  interface{}         `bson:"request_body,omitempty" json:"request_body,omitempty"`
	ResponseBody interface{}         `bson:"response_body,omitempty" json:"response_body,omitempty"`
}

// WSConnection modelo para conexiones WebSocket activas
type WSConnection struct {
	ID          primitive.ObjectID `bson:"_id,omitempty" json:"id,omitempty"`
	UserID      primitive.ObjectID `bson:"user_id" json:"user_id"`
	SocketID    string             `bson:"socket_id" json:"socket_id"`
	IPAddress   string             `bson:"ip_address" json:"ip_address"`
	UserAgent   string             `bson:"user_agent" json:"user_agent"`
	Channel     string             `bson:"channel" json:"channel"`
	ConnectedAt time.Time          `bson:"connected_at" json:"connected_at"`
	LastPing    time.Time          `bson:"last_ping" json:"last_ping"`
	IsActive    bool               `bson:"is_active" json:"is_active"`
}

// Setting modelo para configuraciones del sistema
type Setting struct {
	ID          primitive.ObjectID `bson:"_id,omitempty" json:"id,omitempty"`
	Key         string             `bson:"key" json:"key"`
	Value       interface{}        `bson:"value" json:"value"`
	Description string             `bson:"description,omitempty" json:"description,omitempty"`
	Category    string             `bson:"category" json:"category"`
	IsPublic    bool               `bson:"is_public" json:"is_public"`
	CreatedAt   time.Time          `bson:"created_at" json:"created_at"`
	UpdatedAt   time.Time          `bson:"updated_at" json:"updated_at"`
}

// Tenant modelo para empresas salmoneras (clientes)
type Tenant struct {
	ID        primitive.ObjectID     `bson:"_id,omitempty" json:"id,omitempty"`
	Code      string                 `bson:"code" json:"code"` // Identificador único (ej: mowi-chile)
	Name      string                 `bson:"name" json:"name"` // Nombre de la empresa
	Type      string                 `bson:"type" json:"type"` // Tipo: salmon_company, aquaculture, etc.
	Contact   *TenantContact         `bson:"contact,omitempty" json:"contact,omitempty"`
	Address   *TenantAddress         `bson:"address,omitempty" json:"address,omitempty"`
	Status    string                 `bson:"status" json:"status"` // active, inactive, suspended
	Metadata  map[string]interface{} `bson:"metadata,omitempty" json:"metadata,omitempty"`
	CreatedAt time.Time              `bson:"created_at" json:"created_at"`
	UpdatedAt time.Time              `bson:"updated_at" json:"updated_at"`
	CreatedBy string                 `bson:"created_by,omitempty" json:"created_by,omitempty"`
}

// TenantContact información de contacto del tenant
type TenantContact struct {
	Email       string `bson:"email,omitempty" json:"email,omitempty"`
	Phone       string `bson:"phone,omitempty" json:"phone,omitempty"`
	ContactName string `bson:"contact_name,omitempty" json:"contact_name,omitempty"`
}

// TenantAddress dirección del tenant
type TenantAddress struct {
	Country string `bson:"country,omitempty" json:"country,omitempty"`
	Region  string `bson:"region,omitempty" json:"region,omitempty"`
	City    string `bson:"city,omitempty" json:"city,omitempty"`
	Street  string `bson:"street,omitempty" json:"street,omitempty"`
	ZipCode string `bson:"zip_code,omitempty" json:"zip_code,omitempty"`
}

// Site modelo para centros de cultivo
type Site struct {
	ID                   primitive.ObjectID     `bson:"_id,omitempty" json:"id,omitempty"`
	TenantID             primitive.ObjectID     `bson:"tenant_id" json:"tenant_id"`     // Referencia al tenant
	TenantCode           string                 `bson:"tenant_code" json:"tenant_code"` // Código del tenant (para búsquedas)
	Code                 string                 `bson:"code" json:"code"`               // Código único del centro (ej: mowi-reloncavi-1)
	Name                 string                 `bson:"name" json:"name"`               // Nombre del centro (ej: "Centro Reloncaví")
	Location             *SiteLocation          `bson:"location,omitempty" json:"location,omitempty"`
	FechaApertura        time.Time              `bson:"fecha_apertura" json:"fecha_apertura"`       // Fecha de inicio operaciones
	NumeroJaulas         int                    `bson:"numero_jaulas" json:"numero_jaulas"`         // Cantidad de jaulas
	Cepa                 string                 `bson:"cepa" json:"cepa"`                           // Especie: Atlantic Salmon, Coho, etc.
	TipoAlimentacion     string                 `bson:"tipo_alimentacion" json:"tipo_alimentacion"` // monorracion, ciclico
	BiomasaPromedio      float64                `bson:"biomasa_promedio" json:"biomasa_promedio"`   // En toneladas
	CantidadInicialPeces int                    `bson:"cantidad_inicial_peces" json:"cantidad_inicial_peces"`
	CantidadActualPeces  int                    `bson:"cantidad_actual_peces" json:"cantidad_actual_peces"`
	PorcentajeMortalidad float64                `bson:"porcentaje_mortalidad" json:"porcentaje_mortalidad"` // Calculado: (inicial - actual) / inicial * 100
	Status               string                 `bson:"status" json:"status"`                               // active, inactive, maintenance
	Metadata             map[string]interface{} `bson:"metadata,omitempty" json:"metadata,omitempty"`
	CreatedAt            time.Time              `bson:"created_at" json:"created_at"`
	UpdatedAt            time.Time              `bson:"updated_at" json:"updated_at"`
	CreatedBy            string                 `bson:"created_by,omitempty" json:"created_by,omitempty"`
}

// SiteLocation coordenadas GPS del centro de cultivo
type SiteLocation struct {
	Latitude  float64 `bson:"latitude" json:"latitude"`   // Latitud (ej: -41.4693)
	Longitude float64 `bson:"longitude" json:"longitude"` // Longitud (ej: -72.9318)
	Region    string  `bson:"region,omitempty" json:"region,omitempty"`
	Commune   string  `bson:"commune,omitempty" json:"commune,omitempty"`       // Comuna
	WaterBody string  `bson:"water_body,omitempty" json:"water_body,omitempty"` // Cuerpo de agua (ej: "Seno Reloncaví")
}

// ExternalService servicio externo (ScaleAQ, Innovex, etc.)
type ExternalService struct {
	ID          primitive.ObjectID     `bson:"_id,omitempty" json:"id,omitempty"`
	SiteID      primitive.ObjectID     `bson:"site_id" json:"site_id"`                   // Referencia al centro de cultivo
	SiteCode    string                 `bson:"site_code" json:"site_code"`               // Código del site (para búsquedas)
	TenantID    primitive.ObjectID     `bson:"tenant_id" json:"tenant_id"`               // Referencia al tenant
	TenantCode  string                 `bson:"tenant_code" json:"tenant_code"`           // Código del tenant
	Code        string                 `bson:"code" json:"code"`                         // Código único (ej: "mowi-reloncavi-scaleaq")
	Name        string                 `bson:"name" json:"name"`                         // Nombre descriptivo
	ServiceType string                 `bson:"service_type" json:"service_type"`         // scaleaq, innovex, custom
	BaseURL     string                 `bson:"base_url" json:"base_url"`                 // URL base del servicio
	Credentials *ServiceCredentials    `bson:"credentials" json:"credentials,omitempty"` // Credenciales (encriptadas en DB)
	Config      map[string]interface{} `bson:"config,omitempty" json:"config,omitempty"` // Configuración específica
	Status      string                 `bson:"status" json:"status"`                     // active, inactive, error
	LastAuth    time.Time              `bson:"last_auth,omitempty" json:"last_auth,omitempty"`
	LastError   string                 `bson:"last_error,omitempty" json:"last_error,omitempty"`
	Metadata    map[string]interface{} `bson:"metadata,omitempty" json:"metadata,omitempty"`
	CreatedAt   time.Time              `bson:"created_at" json:"created_at"`
	UpdatedAt   time.Time              `bson:"updated_at" json:"updated_at"`
	CreatedBy   string                 `bson:"created_by,omitempty" json:"created_by,omitempty"`
}

// ServiceCredentials credenciales para servicios externos
// NOTA: Se almacenan encriptadas en MongoDB, tokens solo en memoria
type ServiceCredentials struct {
	// ScaleAQ (Bearer Token)
	Username string `bson:"username,omitempty" json:"username,omitempty"`
	Password string `bson:"password,omitempty" json:"password,omitempty"` // Encriptado

	// Innovex (OAuth2)
	ClientID     string `bson:"client_id,omitempty" json:"client_id,omitempty"`
	ClientSecret string `bson:"client_secret,omitempty" json:"client_secret,omitempty"` // Encriptado

	// API Key (genérico)
	APIKey string `bson:"api_key,omitempty" json:"api_key,omitempty"` // Encriptado

	// Headers personalizados
	CustomHeaders map[string]string `bson:"custom_headers,omitempty" json:"custom_headers,omitempty"`
}

// ValidationError estructura para errores de validación
type ValidationError struct {
	Field   string `json:"field"`
	Message string `json:"message"`
}

// APIResponse estructura mejorada para respuestas con MongoDB
type APIResponse struct {
	Success    bool              `json:"success"`
	Message    string            `json:"message"`
	Data       interface{}       `json:"data,omitempty"`
	Errors     []ValidationError `json:"errors,omitempty"`
	Pagination *PaginationInfo   `json:"pagination,omitempty"`
	Timestamp  int64             `json:"timestamp"`
}

// PaginationInfo información de paginación
type PaginationInfo struct {
	Page       int   `json:"page"`
	PerPage    int   `json:"per_page"`
	Total      int64 `json:"total"`
	TotalPages int   `json:"total_pages"`
	HasNext    bool  `json:"has_next"`
	HasPrev    bool  `json:"has_prev"`
}

// CreateIndexes crea índices necesarios en MongoDB
func CreateIndexes() error {
	// TODO: Implementar creación de índices
	// Ejemplos:
	// - Username único
	// - Email único
	// - Índices de fecha para consultas rápidas
	// - Índices compuestos para queries frecuentes
	return nil
}
