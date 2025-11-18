package handlers

import (
	"context"
	"encoding/json"
	"net/http"
	"strings"
	"time"

	"omniapi/internal/database"
	"omniapi/internal/models"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo/options"
)

// GetTenantsHandler obtiene la lista de tenants con paginación
func GetTenantsHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Método no permitido", http.StatusMethodNotAllowed)
		return
	}

	collection := database.GetCollection("tenants")
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	// Construir filtro
	filter := bson.M{}

	// Filtro por status (opcional)
	if status := r.URL.Query().Get("status"); status != "" {
		filter["status"] = status
	}

	// Búsqueda por nombre o código (opcional)
	if search := r.URL.Query().Get("search"); search != "" {
		filter["$or"] = []bson.M{
			{"name": bson.M{"$regex": search, "$options": "i"}},
			{"code": bson.M{"$regex": search, "$options": "i"}},
		}
	}

	// Opciones de ordenamiento
	opts := options.Find().SetSort(bson.D{{Key: "created_at", Value: -1}})

	// Obtener tenants
	cursor, err := collection.Find(ctx, filter, opts)
	if err != nil {
		response := models.APIResponse{
			Success:   false,
			Message:   "Error obteniendo tenants: " + err.Error(),
			Timestamp: time.Now().Unix(),
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(response)
		return
	}
	defer cursor.Close(ctx)

	var tenants []models.Tenant
	if err = cursor.All(ctx, &tenants); err != nil {
		response := models.APIResponse{
			Success:   false,
			Message:   "Error parseando tenants: " + err.Error(),
			Timestamp: time.Now().Unix(),
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(response)
		return
	}

	// Si no hay tenants, devolver array vacío en lugar de null
	if tenants == nil {
		tenants = []models.Tenant{}
	}

	response := models.APIResponse{
		Success:   true,
		Message:   "Tenants obtenidos exitosamente",
		Data:      tenants,
		Timestamp: time.Now().Unix(),
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(response)
}

// GetTenantHandler obtiene un tenant por ID
func GetTenantHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Método no permitido", http.StatusMethodNotAllowed)
		return
	}

	id := r.URL.Query().Get("id")
	if id == "" {
		response := models.APIResponse{
			Success:   false,
			Message:   "ID es requerido",
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

	collection := database.GetCollection("tenants")
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	var tenant models.Tenant
	err = collection.FindOne(ctx, bson.M{"_id": objectID}).Decode(&tenant)
	if err != nil {
		response := models.APIResponse{
			Success:   false,
			Message:   "Tenant no encontrado",
			Timestamp: time.Now().Unix(),
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusNotFound)
		json.NewEncoder(w).Encode(response)
		return
	}

	response := models.APIResponse{
		Success:   true,
		Message:   "Tenant obtenido exitosamente",
		Data:      tenant,
		Timestamp: time.Now().Unix(),
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(response)
}

// CreateTenantHandler crea un nuevo tenant
func CreateTenantHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Método no permitido", http.StatusMethodNotAllowed)
		return
	}

	var tenant models.Tenant
	if err := json.NewDecoder(r.Body).Decode(&tenant); err != nil {
		response := models.APIResponse{
			Success:   false,
			Message:   "Datos inválidos: " + err.Error(),
			Timestamp: time.Now().Unix(),
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(response)
		return
	}

	// Validaciones
	var errors []models.ValidationError

	if tenant.Name == "" {
		errors = append(errors, models.ValidationError{
			Field:   "name",
			Message: "El nombre es requerido",
		})
	}

	if tenant.Code == "" {
		errors = append(errors, models.ValidationError{
			Field:   "code",
			Message: "El código es requerido",
		})
	} else {
		// Normalizar código (lowercase, sin espacios)
		tenant.Code = strings.ToLower(strings.TrimSpace(tenant.Code))
		tenant.Code = strings.ReplaceAll(tenant.Code, " ", "-")
	}

	if len(errors) > 0 {
		response := models.APIResponse{
			Success:   false,
			Message:   "Errores de validación",
			Errors:    errors,
			Timestamp: time.Now().Unix(),
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(response)
		return
	}

	collection := database.GetCollection("tenants")
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	// Verificar que el código no exista
	count, err := collection.CountDocuments(ctx, bson.M{"code": tenant.Code})
	if err != nil {
		response := models.APIResponse{
			Success:   false,
			Message:   "Error verificando código: " + err.Error(),
			Timestamp: time.Now().Unix(),
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(response)
		return
	}

	if count > 0 {
		response := models.APIResponse{
			Success: false,
			Message: "Ya existe un tenant con ese código",
			Errors: []models.ValidationError{
				{Field: "code", Message: "El código ya está en uso"},
			},
			Timestamp: time.Now().Unix(),
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusConflict)
		json.NewEncoder(w).Encode(response)
		return
	}

	// Establecer valores por defecto
	tenant.ID = primitive.NewObjectID()
	tenant.CreatedAt = time.Now()
	tenant.UpdatedAt = time.Now()

	if tenant.Status == "" {
		tenant.Status = "active"
	}

	if tenant.Type == "" {
		tenant.Type = "salmon_company"
	}

	// Insertar en MongoDB
	_, err = collection.InsertOne(ctx, tenant)
	if err != nil {
		response := models.APIResponse{
			Success:   false,
			Message:   "Error creando tenant: " + err.Error(),
			Timestamp: time.Now().Unix(),
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(response)
		return
	}

	response := models.APIResponse{
		Success:   true,
		Message:   "Tenant creado exitosamente",
		Data:      tenant,
		Timestamp: time.Now().Unix(),
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(response)
}

// UpdateTenantHandler actualiza un tenant existente
func UpdateTenantHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPut {
		http.Error(w, "Método no permitido", http.StatusMethodNotAllowed)
		return
	}

	id := r.URL.Query().Get("id")
	if id == "" {
		response := models.APIResponse{
			Success:   false,
			Message:   "ID es requerido",
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

	var updates models.Tenant
	if err := json.NewDecoder(r.Body).Decode(&updates); err != nil {
		response := models.APIResponse{
			Success:   false,
			Message:   "Datos inválidos: " + err.Error(),
			Timestamp: time.Now().Unix(),
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(response)
		return
	}

	collection := database.GetCollection("tenants")
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	// Construir documento de actualización
	updateDoc := bson.M{
		"updated_at": time.Now(),
	}

	if updates.Name != "" {
		updateDoc["name"] = updates.Name
	}
	if updates.Status != "" {
		updateDoc["status"] = updates.Status
	}
	if updates.Type != "" {
		updateDoc["type"] = updates.Type
	}
	if updates.Contact != nil {
		updateDoc["contact"] = updates.Contact
	}
	if updates.Address != nil {
		updateDoc["address"] = updates.Address
	}
	if updates.Metadata != nil {
		updateDoc["metadata"] = updates.Metadata
	}

	// Actualizar en MongoDB
	result, err := collection.UpdateOne(
		ctx,
		bson.M{"_id": objectID},
		bson.M{"$set": updateDoc},
	)

	if err != nil {
		response := models.APIResponse{
			Success:   false,
			Message:   "Error actualizando tenant: " + err.Error(),
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
			Message:   "Tenant no encontrado",
			Timestamp: time.Now().Unix(),
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusNotFound)
		json.NewEncoder(w).Encode(response)
		return
	}

	// Obtener tenant actualizado
	var tenant models.Tenant
	err = collection.FindOne(ctx, bson.M{"_id": objectID}).Decode(&tenant)
	if err != nil {
		response := models.APIResponse{
			Success:   false,
			Message:   "Error obteniendo tenant actualizado",
			Timestamp: time.Now().Unix(),
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(response)
		return
	}

	response := models.APIResponse{
		Success:   true,
		Message:   "Tenant actualizado exitosamente",
		Data:      tenant,
		Timestamp: time.Now().Unix(),
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(response)
}

// DeleteTenantHandler elimina un tenant
func DeleteTenantHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodDelete {
		http.Error(w, "Método no permitido", http.StatusMethodNotAllowed)
		return
	}

	id := r.URL.Query().Get("id")
	if id == "" {
		response := models.APIResponse{
			Success:   false,
			Message:   "ID es requerido",
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

	collection := database.GetCollection("tenants")
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	// Eliminar de MongoDB
	result, err := collection.DeleteOne(ctx, bson.M{"_id": objectID})
	if err != nil {
		response := models.APIResponse{
			Success:   false,
			Message:   "Error eliminando tenant: " + err.Error(),
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
			Message:   "Tenant no encontrado",
			Timestamp: time.Now().Unix(),
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusNotFound)
		json.NewEncoder(w).Encode(response)
		return
	}

	response := models.APIResponse{
		Success:   true,
		Message:   "Tenant eliminado exitosamente",
		Timestamp: time.Now().Unix(),
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(response)
}
