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
