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

// GetSitesHandler obtiene la lista de centros de cultivo
func GetSitesHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// Obtener parámetros de consulta
	tenantID := r.URL.Query().Get("tenant_id")
	tenantCode := r.URL.Query().Get("tenant_code")
	status := r.URL.Query().Get("status")
	search := r.URL.Query().Get("search")

	// Construir filtro
	filter := bson.M{}

	if tenantID != "" {
		objID, err := primitive.ObjectIDFromHex(tenantID)
		if err == nil {
			filter["tenant_id"] = objID
		}
	}

	if tenantCode != "" {
		filter["tenant_code"] = tenantCode
	}

	if status != "" {
		filter["status"] = status
	}

	if search != "" {
		filter["$or"] = []bson.M{
			{"name": bson.M{"$regex": search, "$options": "i"}},
			{"code": bson.M{"$regex": search, "$options": "i"}},
			{"cepa": bson.M{"$regex": search, "$options": "i"}},
		}
	}

	// Obtener colección
	collection := database.GetCollection("sites")
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	// Opciones de ordenamiento
	opts := options.Find().SetSort(bson.D{{Key: "created_at", Value: -1}})

	// Buscar sites
	cursor, err := collection.Find(ctx, filter, opts)
	if err != nil {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(models.APIResponse{
			Success: false,
			Message: "Error buscando centros de cultivo: " + err.Error(),
		})
		return
	}
	defer cursor.Close(ctx)

	// Decodificar resultados
	var sites []models.Site
	if err = cursor.All(ctx, &sites); err != nil {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(models.APIResponse{
			Success: false,
			Message: "Error decodificando centros de cultivo: " + err.Error(),
		})
		return
	}

	// Si no hay resultados, devolver array vacío
	if sites == nil {
		sites = []models.Site{}
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(models.APIResponse{
		Success: true,
		Message: "Centros de cultivo obtenidos exitosamente",
		Data:    sites,
	})
}

// GetSiteHandler obtiene un centro de cultivo por ID
func GetSiteHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// Obtener ID de los parámetros
	id := r.URL.Query().Get("id")
	if id == "" {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(models.APIResponse{
			Success: false,
			Message: "ID es requerido",
		})
		return
	}

	// Convertir a ObjectID
	objID, err := primitive.ObjectIDFromHex(id)
	if err != nil {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(models.APIResponse{
			Success: false,
			Message: "ID inválido",
		})
		return
	}

	// Buscar site
	collection := database.GetCollection("sites")
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	var site models.Site
	err = collection.FindOne(ctx, bson.M{"_id": objID}).Decode(&site)
	if err != nil {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusNotFound)
		json.NewEncoder(w).Encode(models.APIResponse{
			Success: false,
			Message: "Centro de cultivo no encontrado",
		})
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(models.APIResponse{
		Success: true,
		Message: "Centro de cultivo obtenido exitosamente",
		Data:    site,
	})
}

// CreateSiteHandler crea un nuevo centro de cultivo
func CreateSiteHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var site models.Site
	if err := json.NewDecoder(r.Body).Decode(&site); err != nil {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(models.APIResponse{
			Success: false,
			Message: "Error decodificando JSON: " + err.Error(),
		})
		return
	}

	// Validaciones
	var validationErrors []models.ValidationError

	if site.Name == "" {
		validationErrors = append(validationErrors, models.ValidationError{
			Field:   "name",
			Message: "El nombre es requerido",
		})
	}

	if site.Code == "" {
		validationErrors = append(validationErrors, models.ValidationError{
			Field:   "code",
			Message: "El código es requerido",
		})
	}

	// Normalizar código (minúsculas, guiones)
	site.Code = strings.ToLower(strings.ReplaceAll(strings.TrimSpace(site.Code), " ", "-"))

	if site.TenantID.IsZero() {
		validationErrors = append(validationErrors, models.ValidationError{
			Field:   "tenant_id",
			Message: "El tenant_id es requerido",
		})
	}

	if site.NumeroJaulas <= 0 {
		validationErrors = append(validationErrors, models.ValidationError{
			Field:   "numero_jaulas",
			Message: "El número de jaulas debe ser mayor a 0",
		})
	}

	// Validar coordenadas si se proporcionan
	if site.Location != nil {
		if site.Location.Latitude < -90 || site.Location.Latitude > 90 {
			validationErrors = append(validationErrors, models.ValidationError{
				Field:   "location.latitude",
				Message: "La latitud debe estar entre -90 y 90",
			})
		}
		if site.Location.Longitude < -180 || site.Location.Longitude > 180 {
			validationErrors = append(validationErrors, models.ValidationError{
				Field:   "location.longitude",
				Message: "La longitud debe estar entre -180 y 180",
			})
		}
	}

	if len(validationErrors) > 0 {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(models.APIResponse{
			Success: false,
			Message: "Errores de validación",
			Data:    validationErrors,
		})
		return
	}

	// Verificar que el código sea único
	collection := database.GetCollection("sites")
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	var existingSite models.Site
	err := collection.FindOne(ctx, bson.M{"code": site.Code}).Decode(&existingSite)
	if err == nil {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusConflict)
		json.NewEncoder(w).Encode(models.APIResponse{
			Success: false,
			Message: "Ya existe un centro de cultivo con ese código",
			Data: []models.ValidationError{
				{Field: "code", Message: "El código ya está en uso"},
			},
		})
		return
	}

	// Verificar que el tenant existe
	tenantsCollection := database.GetCollection("tenants")
	var tenant models.Tenant
	err = tenantsCollection.FindOne(ctx, bson.M{"_id": site.TenantID}).Decode(&tenant)
	if err != nil {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(models.APIResponse{
			Success: false,
			Message: "El tenant especificado no existe",
			Data: []models.ValidationError{
				{Field: "tenant_id", Message: "Tenant no encontrado"},
			},
		})
		return
	}

	// Guardar tenant_code para facilitar búsquedas
	site.TenantCode = tenant.Code

	// Calcular porcentaje de mortalidad
	if site.CantidadInicialPeces > 0 {
		muertos := site.CantidadInicialPeces - site.CantidadActualPeces
		site.PorcentajeMortalidad = (float64(muertos) / float64(site.CantidadInicialPeces)) * 100
	}

	// Establecer valores por defecto
	if site.Status == "" {
		site.Status = "active"
	}

	// Establecer timestamps
	now := time.Now()
	site.CreatedAt = now
	site.UpdatedAt = now

	// Insertar en la base de datos
	result, err := collection.InsertOne(ctx, site)
	if err != nil {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(models.APIResponse{
			Success: false,
			Message: "Error creando centro de cultivo: " + err.Error(),
		})
		return
	}

	site.ID = result.InsertedID.(primitive.ObjectID)

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(models.APIResponse{
		Success: true,
		Message: "Centro de cultivo creado exitosamente",
		Data:    site,
	})
}

// UpdateSiteHandler actualiza un centro de cultivo existente
func UpdateSiteHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPut {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// Obtener ID de los parámetros
	id := r.URL.Query().Get("id")
	if id == "" {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(models.APIResponse{
			Success: false,
			Message: "ID es requerido",
		})
		return
	}

	objID, err := primitive.ObjectIDFromHex(id)
	if err != nil {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(models.APIResponse{
			Success: false,
			Message: "ID inválido",
		})
		return
	}

	var updateData models.Site
	if err := json.NewDecoder(r.Body).Decode(&updateData); err != nil {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(models.APIResponse{
			Success: false,
			Message: "Error decodificando JSON: " + err.Error(),
		})
		return
	}

	// Validaciones
	var validationErrors []models.ValidationError

	if updateData.Name == "" {
		validationErrors = append(validationErrors, models.ValidationError{
			Field:   "name",
			Message: "El nombre es requerido",
		})
	}

	if updateData.NumeroJaulas <= 0 {
		validationErrors = append(validationErrors, models.ValidationError{
			Field:   "numero_jaulas",
			Message: "El número de jaulas debe ser mayor a 0",
		})
	}

	// Validar coordenadas si se proporcionan
	if updateData.Location != nil {
		if updateData.Location.Latitude < -90 || updateData.Location.Latitude > 90 {
			validationErrors = append(validationErrors, models.ValidationError{
				Field:   "location.latitude",
				Message: "La latitud debe estar entre -90 y 90",
			})
		}
		if updateData.Location.Longitude < -180 || updateData.Location.Longitude > 180 {
			validationErrors = append(validationErrors, models.ValidationError{
				Field:   "location.longitude",
				Message: "La longitud debe estar entre -180 y 180",
			})
		}
	}

	if len(validationErrors) > 0 {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(models.APIResponse{
			Success: false,
			Message: "Errores de validación",
			Data:    validationErrors,
		})
		return
	}

	// Recalcular porcentaje de mortalidad
	if updateData.CantidadInicialPeces > 0 {
		muertos := updateData.CantidadInicialPeces - updateData.CantidadActualPeces
		updateData.PorcentajeMortalidad = (float64(muertos) / float64(updateData.CantidadInicialPeces)) * 100
	}

	// Preparar actualización (no permitir cambiar code, tenant_id, created_at)
	update := bson.M{
		"$set": bson.M{
			"name":                   updateData.Name,
			"location":               updateData.Location,
			"fecha_apertura":         updateData.FechaApertura,
			"numero_jaulas":          updateData.NumeroJaulas,
			"cepa":                   updateData.Cepa,
			"tipo_alimentacion":      updateData.TipoAlimentacion,
			"biomasa_promedio":       updateData.BiomasaPromedio,
			"cantidad_inicial_peces": updateData.CantidadInicialPeces,
			"cantidad_actual_peces":  updateData.CantidadActualPeces,
			"porcentaje_mortalidad":  updateData.PorcentajeMortalidad,
			"status":                 updateData.Status,
			"metadata":               updateData.Metadata,
			"updated_at":             time.Now(),
		},
	}

	collection := database.GetCollection("sites")
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	// Actualizar
	opts := options.FindOneAndUpdate().SetReturnDocument(options.After)
	var updatedSite models.Site
	err = collection.FindOneAndUpdate(ctx, bson.M{"_id": objID}, update, opts).Decode(&updatedSite)
	if err != nil {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusNotFound)
		json.NewEncoder(w).Encode(models.APIResponse{
			Success: false,
			Message: "Centro de cultivo no encontrado",
		})
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(models.APIResponse{
		Success: true,
		Message: "Centro de cultivo actualizado exitosamente",
		Data:    updatedSite,
	})
}

// DeleteSiteHandler elimina un centro de cultivo
func DeleteSiteHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodDelete {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// Obtener ID de los parámetros
	id := r.URL.Query().Get("id")
	if id == "" {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(models.APIResponse{
			Success: false,
			Message: "ID es requerido",
		})
		return
	}

	objID, err := primitive.ObjectIDFromHex(id)
	if err != nil {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(models.APIResponse{
			Success: false,
			Message: "ID inválido",
		})
		return
	}

	collection := database.GetCollection("sites")
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	result, err := collection.DeleteOne(ctx, bson.M{"_id": objID})
	if err != nil {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(models.APIResponse{
			Success: false,
			Message: "Error eliminando centro de cultivo: " + err.Error(),
		})
		return
	}

	if result.DeletedCount == 0 {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusNotFound)
		json.NewEncoder(w).Encode(models.APIResponse{
			Success: false,
			Message: "Centro de cultivo no encontrado",
		})
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(models.APIResponse{
		Success: true,
		Message: "Centro de cultivo eliminado exitosamente",
	})
}
