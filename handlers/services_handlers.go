package handlers

import (
	"context"
	"encoding/json"
	"net/http"
	"time"

	"omniapi/database"
	"omniapi/models"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

// ExternalService modelo para servicios externos
type ExternalService struct {
	ID        primitive.ObjectID     `bson:"_id,omitempty" json:"id,omitempty"`
	Name      string                 `bson:"name" json:"name"`
	Type      string                 `bson:"type" json:"type"` // mqtt, rest, websocket, graphql
	URL       string                 `bson:"url" json:"url"`
	AuthType  string                 `bson:"auth_type,omitempty" json:"auth_type,omitempty"` // none, basic, token, oauth
	Username  string                 `bson:"username,omitempty" json:"username,omitempty"`
	Password  string                 `bson:"password,omitempty" json:"password,omitempty"`
	Token     string                 `bson:"token,omitempty" json:"token,omitempty"`
	Status    string                 `bson:"status" json:"status"` // active, inactive, error
	CreatedAt time.Time              `bson:"created_at" json:"created_at"`
	UpdatedAt time.Time              `bson:"updated_at" json:"updated_at"`
	Metadata  map[string]interface{} `bson:"metadata,omitempty" json:"metadata,omitempty"`
}

// GetServicesHandler obtiene todos los servicios
func GetServicesHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Método no permitido", http.StatusMethodNotAllowed)
		return
	}

	collection := database.GetCollection("external_services")
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	cursor, err := collection.Find(ctx, bson.M{})
	if err != nil {
		response := models.APIResponse{
			Success:   false,
			Message:   "Error obteniendo servicios",
			Timestamp: time.Now().Unix(),
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(response)
		return
	}
	defer cursor.Close(ctx)

	var services []ExternalService
	if err := cursor.All(ctx, &services); err != nil {
		response := models.APIResponse{
			Success:   false,
			Message:   "Error procesando servicios",
			Timestamp: time.Now().Unix(),
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(response)
		return
	}

	if services == nil {
		services = []ExternalService{}
	}

	response := models.APIResponse{
		Success:   true,
		Message:   "Servicios obtenidos exitosamente",
		Data:      services,
		Timestamp: time.Now().Unix(),
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

// GetServiceHandler obtiene un servicio por ID
func GetServiceHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Método no permitido", http.StatusMethodNotAllowed)
		return
	}

	// Obtener ID del query param o path
	id := r.URL.Query().Get("id")
	if id == "" {
		response := models.APIResponse{
			Success:   false,
			Message:   "ID requerido",
			Timestamp: time.Now().Unix(),
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(response)
		return
	}

	objectID, err := primitive.ObjectIDFromHex(id)
	if err != nil {
		response := models.APIResponse{
			Success:   false,
			Message:   "ID inválido",
			Timestamp: time.Now().Unix(),
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(response)
		return
	}

	collection := database.GetCollection("external_services")
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	var service ExternalService
	err = collection.FindOne(ctx, bson.M{"_id": objectID}).Decode(&service)
	if err != nil {
		response := models.APIResponse{
			Success:   false,
			Message:   "Servicio no encontrado",
			Timestamp: time.Now().Unix(),
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusNotFound)
		json.NewEncoder(w).Encode(response)
		return
	}

	response := models.APIResponse{
		Success:   true,
		Message:   "Servicio obtenido exitosamente",
		Data:      service,
		Timestamp: time.Now().Unix(),
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

// CreateServiceHandler crea un nuevo servicio
func CreateServiceHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Método no permitido", http.StatusMethodNotAllowed)
		return
	}

	var service ExternalService
	if err := json.NewDecoder(r.Body).Decode(&service); err != nil {
		response := models.APIResponse{
			Success:   false,
			Message:   "Datos inválidos",
			Timestamp: time.Now().Unix(),
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(response)
		return
	}

	service.CreatedAt = time.Now()
	service.UpdatedAt = time.Now()
	service.Status = "active"

	collection := database.GetCollection("external_services")
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	result, err := collection.InsertOne(ctx, service)
	if err != nil {
		response := models.APIResponse{
			Success:   false,
			Message:   "Error creando servicio: " + err.Error(),
			Timestamp: time.Now().Unix(),
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(response)
		return
	}

	service.ID = result.InsertedID.(primitive.ObjectID)

	response := models.APIResponse{
		Success:   true,
		Message:   "Servicio creado exitosamente",
		Data:      service,
		Timestamp: time.Now().Unix(),
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(response)
}

// UpdateServiceHandler actualiza un servicio
func UpdateServiceHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPut {
		http.Error(w, "Método no permitido", http.StatusMethodNotAllowed)
		return
	}

	id := r.URL.Query().Get("id")
	if id == "" {
		response := models.APIResponse{
			Success:   false,
			Message:   "ID requerido",
			Timestamp: time.Now().Unix(),
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(response)
		return
	}

	objectID, err := primitive.ObjectIDFromHex(id)
	if err != nil {
		response := models.APIResponse{
			Success:   false,
			Message:   "ID inválido",
			Timestamp: time.Now().Unix(),
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(response)
		return
	}

	var service ExternalService
	if err := json.NewDecoder(r.Body).Decode(&service); err != nil {
		response := models.APIResponse{
			Success:   false,
			Message:   "Datos inválidos",
			Timestamp: time.Now().Unix(),
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(response)
		return
	}

	service.UpdatedAt = time.Now()

	collection := database.GetCollection("external_services")
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	update := bson.M{
		"$set": bson.M{
			"name":       service.Name,
			"type":       service.Type,
			"url":        service.URL,
			"auth_type":  service.AuthType,
			"username":   service.Username,
			"password":   service.Password,
			"token":      service.Token,
			"metadata":   service.Metadata,
			"updated_at": service.UpdatedAt,
		},
	}

	result, err := collection.UpdateOne(ctx, bson.M{"_id": objectID}, update)
	if err != nil {
		response := models.APIResponse{
			Success:   false,
			Message:   "Error actualizando servicio",
			Timestamp: time.Now().Unix(),
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(response)
		return
	}

	if result.MatchedCount == 0 {
		response := models.APIResponse{
			Success:   false,
			Message:   "Servicio no encontrado",
			Timestamp: time.Now().Unix(),
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusNotFound)
		json.NewEncoder(w).Encode(response)
		return
	}

	service.ID = objectID

	response := models.APIResponse{
		Success:   true,
		Message:   "Servicio actualizado exitosamente",
		Data:      service,
		Timestamp: time.Now().Unix(),
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

// DeleteServiceHandler elimina un servicio
func DeleteServiceHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodDelete {
		http.Error(w, "Método no permitido", http.StatusMethodNotAllowed)
		return
	}

	id := r.URL.Query().Get("id")
	if id == "" {
		response := models.APIResponse{
			Success:   false,
			Message:   "ID requerido",
			Timestamp: time.Now().Unix(),
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(response)
		return
	}

	objectID, err := primitive.ObjectIDFromHex(id)
	if err != nil {
		response := models.APIResponse{
			Success:   false,
			Message:   "ID inválido",
			Timestamp: time.Now().Unix(),
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(response)
		return
	}

	collection := database.GetCollection("external_services")
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	result, err := collection.DeleteOne(ctx, bson.M{"_id": objectID})
	if err != nil {
		response := models.APIResponse{
			Success:   false,
			Message:   "Error eliminando servicio",
			Timestamp: time.Now().Unix(),
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(response)
		return
	}

	if result.DeletedCount == 0 {
		response := models.APIResponse{
			Success:   false,
			Message:   "Servicio no encontrado",
			Timestamp: time.Now().Unix(),
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusNotFound)
		json.NewEncoder(w).Encode(response)
		return
	}

	response := models.APIResponse{
		Success:   true,
		Message:   "Servicio eliminado exitosamente",
		Timestamp: time.Now().Unix(),
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

// TestServiceConnectionHandler prueba la conexión a un servicio
func TestServiceConnectionHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Método no permitido", http.StatusMethodNotAllowed)
		return
	}

	// Por ahora, solo retornamos un mensaje de éxito
	// En una implementación real, aquí se haría la prueba de conexión real
	response := models.APIResponse{
		Success:   true,
		Message:   "Conexión probada exitosamente (simulado)",
		Timestamp: time.Now().Unix(),
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}
