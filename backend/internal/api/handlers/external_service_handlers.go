package handlers

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"

	"omniapi/internal/crypto"
	"omniapi/internal/database"
	"omniapi/internal/models"
	"omniapi/internal/services"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
)

// ============================================================
// GET /api/external-services - Listar servicios externos
// ============================================================

func GetExternalServicesHandler(w http.ResponseWriter, r *http.Request) {
	collection := database.GetCollection("external_services")

	// Construir filtro
	filter := bson.M{}

	// Filtro por site_id
	if siteID := r.URL.Query().Get("site_id"); siteID != "" {
		objID, err := primitive.ObjectIDFromHex(siteID)
		if err == nil {
			filter["site_id"] = objID
		}
	}

	// Filtro por tenant_id
	if tenantID := r.URL.Query().Get("tenant_id"); tenantID != "" {
		objID, err := primitive.ObjectIDFromHex(tenantID)
		if err == nil {
			filter["tenant_id"] = objID
		}
	}

	// Filtro por service_type
	if serviceType := r.URL.Query().Get("service_type"); serviceType != "" {
		filter["service_type"] = strings.ToLower(serviceType)
	}

	// Filtro por status
	if status := r.URL.Query().Get("status"); status != "" {
		filter["status"] = status
	}

	// Búsqueda por nombre o código
	if search := r.URL.Query().Get("search"); search != "" {
		filter["$or"] = []bson.M{
			{"name": bson.M{"$regex": search, "$options": "i"}},
			{"code": bson.M{"$regex": search, "$options": "i"}},
		}
	}

	// Ejecutar query
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	cursor, err := collection.Find(ctx, filter)
	if err != nil {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(map[string]interface{}{
			"success": false,
			"error":   fmt.Sprintf("Error buscando servicios: %v", err),
		})
		return
	}
	defer cursor.Close(ctx)

	// Parsear resultados
	var externalServices []models.ExternalService
	if err := cursor.All(ctx, &externalServices); err != nil {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(map[string]interface{}{
			"success": false,
			"error":   fmt.Sprintf("Error parseando servicios: %v", err),
		})
		return
	}

	// Respuesta exitosa
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(map[string]interface{}{
		"success": true,
		"data":    externalServices,
		"count":   len(externalServices),
	})
}

// ============================================================
// GET /api/external-services/get?id={id} - Obtener servicio por ID
// ============================================================

func GetExternalServiceHandler(w http.ResponseWriter, r *http.Request) {
	collection := database.GetCollection("external_services")

	// Obtener ID
	id := r.URL.Query().Get("id")
	if id == "" {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]interface{}{
			"success": false,
			"error":   "ID del servicio requerido",
		})
		return
	}

	objID, err := primitive.ObjectIDFromHex(id)
	if err != nil {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]interface{}{
			"success": false,
			"error":   "ID inválido",
		})
		return
	}

	// Buscar servicio
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	var service models.ExternalService
	err = collection.FindOne(ctx, bson.M{"_id": objID}).Decode(&service)
	if err == mongo.ErrNoDocuments {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusNotFound)
		json.NewEncoder(w).Encode(map[string]interface{}{
			"success": false,
			"error":   "Servicio no encontrado",
		})
		return
	} else if err != nil {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(map[string]interface{}{
			"success": false,
			"error":   fmt.Sprintf("Error buscando servicio: %v", err),
		})
		return
	}

	// Respuesta exitosa
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(map[string]interface{}{
		"success": true,
		"data":    service,
	})
}

// ============================================================
// POST /api/external-services/create - Crear servicio externo
// ============================================================

func CreateExternalServiceHandler(w http.ResponseWriter, r *http.Request) {
	var service models.ExternalService
	if err := json.NewDecoder(r.Body).Decode(&service); err != nil {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]interface{}{
			"success": false,
			"error":   fmt.Sprintf("Error parseando JSON: %v", err),
		})
		return
	}

	// Validaciones
	if service.Name == "" {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]interface{}{
			"success": false,
			"error":   "El nombre del servicio es requerido",
		})
		return
	}

	if service.ServiceType == "" {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]interface{}{
			"success": false,
			"error":   "El tipo de servicio es requerido",
		})
		return
	}

	if service.BaseURL == "" {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]interface{}{
			"success": false,
			"error":   "La URL base es requerida",
		})
		return
	}

	// Normalizar service_type
	service.ServiceType = strings.ToLower(service.ServiceType)

	// Normalizar code
	if service.Code == "" {
		service.Code = strings.ToLower(strings.ReplaceAll(service.Name, " ", "-"))
	} else {
		service.Code = strings.ToLower(strings.ReplaceAll(service.Code, " ", "-"))
	}

	// Verificar que el site existe
	if !service.SiteID.IsZero() {
		sitesCollection := database.GetCollection("sites")
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()

		var site models.Site
		err := sitesCollection.FindOne(ctx, bson.M{"_id": service.SiteID}).Decode(&site)
		if err == mongo.ErrNoDocuments {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusBadRequest)
			json.NewEncoder(w).Encode(map[string]interface{}{
				"success": false,
				"error":   "El centro de cultivo especificado no existe",
			})
			return
		} else if err != nil {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusInternalServerError)
			json.NewEncoder(w).Encode(map[string]interface{}{
				"success": false,
				"error":   fmt.Sprintf("Error verificando centro de cultivo: %v", err),
			})
			return
		}

		// Copiar datos del site para búsquedas rápidas
		service.SiteCode = site.Code
		service.TenantID = site.TenantID
		service.TenantCode = site.TenantCode
	}

	// Verificar unicidad del código
	collection := database.GetCollection("external_services")
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	count, _ := collection.CountDocuments(ctx, bson.M{"code": service.Code})
	if count > 0 {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusConflict)
		json.NewEncoder(w).Encode(map[string]interface{}{
			"success": false,
			"error":   fmt.Sprintf("Ya existe un servicio con el código '%s'", service.Code),
		})
		return
	}

	// Encriptar credenciales sensibles con AES (solo si están en texto plano)
	if service.Credentials != nil {
		cryptoService, err := crypto.GetService()
		if err != nil {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusInternalServerError)
			json.NewEncoder(w).Encode(map[string]interface{}{
				"success": false,
				"error":   fmt.Sprintf("Error inicializando servicio de encriptación: %v", err),
			})
			return
		}

		if _, err := encryptServiceCredentials(service.Credentials, cryptoService); err != nil {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusInternalServerError)
			json.NewEncoder(w).Encode(map[string]interface{}{
				"success": false,
				"error":   err.Error(),
			})
			return
		}
	}

	// Preparar documento
	service.ID = primitive.NewObjectID()
	service.CreatedAt = time.Now()
	service.UpdatedAt = time.Now()

	if service.Status == "" {
		service.Status = "inactive" // Por defecto inactivo hasta que se pruebe
	}

	// Insertar en base de datos
	ctx, cancel = context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	_, err := collection.InsertOne(ctx, service)
	if err != nil {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(map[string]interface{}{
			"success": false,
			"error":   fmt.Sprintf("Error creando servicio: %v", err),
		})
		return
	}

	// Respuesta exitosa
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(map[string]interface{}{
		"success": true,
		"message": "Servicio externo creado exitosamente",
		"data":    service,
	})
}

// ============================================================
// POST /api/external-services/update?id={id} - Actualizar servicio
// ============================================================

func UpdateExternalServiceHandler(w http.ResponseWriter, r *http.Request) {
	// Obtener ID
	id := r.URL.Query().Get("id")
	if id == "" {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]interface{}{
			"success": false,
			"error":   "ID del servicio requerido",
		})
		return
	}

	objID, err := primitive.ObjectIDFromHex(id)
	if err != nil {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]interface{}{
			"success": false,
			"error":   "ID inválido",
		})
		return
	}

	// Parsear datos
	var updates models.ExternalService
	if err := json.NewDecoder(r.Body).Decode(&updates); err != nil {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]interface{}{
			"success": false,
			"error":   fmt.Sprintf("Error parseando JSON: %v", err),
		})
		return
	}

	// Encriptar credenciales si se proporcionan nuevas
	if updates.Credentials != nil {
		cryptoService, err := crypto.GetService()
		if err != nil {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusInternalServerError)
			json.NewEncoder(w).Encode(map[string]interface{}{
				"success": false,
				"error":   fmt.Sprintf("Error inicializando servicio de encriptación: %v", err),
			})
			return
		}

		if _, err := encryptServiceCredentials(updates.Credentials, cryptoService); err != nil {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusInternalServerError)
			json.NewEncoder(w).Encode(map[string]interface{}{
				"success": false,
				"error":   err.Error(),
			})
			return
		}

		// Invalidar token en cache si se cambian credenciales
		tokenManager := services.GetTokenManager()
		tokenManager.InvalidateToken(objID)
	}

	// Construir update
	updateDoc := bson.M{
		"$set": bson.M{
			"updated_at": time.Now(),
		},
	}

	// Campos actualizables
	if updates.Name != "" {
		updateDoc["$set"].(bson.M)["name"] = updates.Name
	}
	if updates.BaseURL != "" {
		updateDoc["$set"].(bson.M)["base_url"] = updates.BaseURL
	}
	if updates.Credentials != nil {
		updateDoc["$set"].(bson.M)["credentials"] = updates.Credentials
	}
	if updates.Config != nil {
		updateDoc["$set"].(bson.M)["config"] = updates.Config
	}
	if updates.Status != "" {
		updateDoc["$set"].(bson.M)["status"] = updates.Status
	}
	if updates.Metadata != nil {
		updateDoc["$set"].(bson.M)["metadata"] = updates.Metadata
	}

	// Actualizar
	collection := database.GetCollection("external_services")
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	result, err := collection.UpdateOne(ctx, bson.M{"_id": objID}, updateDoc)
	if err != nil {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(map[string]interface{}{
			"success": false,
			"error":   fmt.Sprintf("Error actualizando servicio: %v", err),
		})
		return
	}

	if result.MatchedCount == 0 {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusNotFound)
		json.NewEncoder(w).Encode(map[string]interface{}{
			"success": false,
			"error":   "Servicio no encontrado",
		})
		return
	}

	// Respuesta exitosa
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(map[string]interface{}{
		"success": true,
		"message": "Servicio actualizado exitosamente",
	})
}

// ============================================================
// DELETE /api/external-services/delete?id={id} - Eliminar servicio
// ============================================================

func DeleteExternalServiceHandler(w http.ResponseWriter, r *http.Request) {
	// Obtener ID
	id := r.URL.Query().Get("id")
	if id == "" {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]interface{}{
			"success": false,
			"error":   "ID del servicio requerido",
		})
		return
	}

	objID, err := primitive.ObjectIDFromHex(id)
	if err != nil {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]interface{}{
			"success": false,
			"error":   "ID inválido",
		})
		return
	}

	// Invalidar token en cache
	tokenManager := services.GetTokenManager()
	tokenManager.InvalidateToken(objID)

	// Eliminar
	collection := database.GetCollection("external_services")
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	result, err := collection.DeleteOne(ctx, bson.M{"_id": objID})
	if err != nil {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(map[string]interface{}{
			"success": false,
			"error":   fmt.Sprintf("Error eliminando servicio: %v", err),
		})
		return
	}

	if result.DeletedCount == 0 {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusNotFound)
		json.NewEncoder(w).Encode(map[string]interface{}{
			"success": false,
			"error":   "Servicio no encontrado",
		})
		return
	}

	// Respuesta exitosa
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(map[string]interface{}{
		"success": true,
		"message": "Servicio eliminado exitosamente",
	})
}

// ============================================================
// POST /api/external-services/test?id={id} - Probar conexión
// ============================================================

func TestExternalServiceConnectionHandler(w http.ResponseWriter, r *http.Request) {
	// Obtener ID
	id := r.URL.Query().Get("id")
	if id == "" {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]interface{}{
			"success": false,
			"error":   "ID del servicio requerido",
		})
		return
	}

	objID, err := primitive.ObjectIDFromHex(id)
	if err != nil {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]interface{}{
			"success": false,
			"error":   "ID inválido",
		})
		return
	}

	// Buscar servicio
	collection := database.GetCollection("external_services")
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	var service models.ExternalService
	err = collection.FindOne(ctx, bson.M{"_id": objID}).Decode(&service)
	if err != nil {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusNotFound)
		json.NewEncoder(w).Encode(map[string]interface{}{
			"success": false,
			"error":   "Servicio no encontrado",
		})
		return
	}

	if err := ensureServiceCredentialsEncrypted(collection, &service); err != nil {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(map[string]interface{}{
			"success": false,
			"error":   fmt.Sprintf("Error asegurando credenciales encriptadas: %v", err),
		})
		return
	}

	// Probar conexión (NO guarda token en cache)
	tokenManager := services.GetTokenManager()
	tokenResp, err := tokenManager.TestConnection(&service)
	if err != nil {
		// Actualizar last_error
		collection.UpdateOne(ctx, bson.M{"_id": objID}, bson.M{
			"$set": bson.M{
				"last_error": err.Error(),
				"status":     "error",
				"updated_at": time.Now(),
			},
		})

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]interface{}{
			"success": false,
			"error":   fmt.Sprintf("Error en autenticación: %v", err),
		})
		return
	}

	// Actualizar información de autenticación exitosa
	collection.UpdateOne(ctx, bson.M{"_id": objID}, bson.M{
		"$set": bson.M{
			"last_auth":  time.Now(),
			"last_error": "",
			"status":     "active",
			"updated_at": time.Now(),
		},
	})

	// Respuesta exitosa (NO incluye el token completo por seguridad)
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(map[string]interface{}{
		"success": true,
		"message": "Conexión exitosa",
		"data": map[string]interface{}{
			"token_type":   tokenResp.TokenType,
			"expires_in":   tokenResp.ExpiresIn,
			"expires_at":   tokenResp.ExpiresAt,
			"token_length": len(tokenResp.AccessToken), // Solo mostrar longitud
		},
	})
}

// encryptServiceCredentials asegura que las credenciales sensibles se almacenen cifradas
func encryptServiceCredentials(credentials *models.ServiceCredentials, cryptoService crypto.CryptoService) (bool, error) {
	if credentials == nil {
		return false, nil
	}

	changed := false

	if credentials.Password != "" && !cryptoService.IsEncrypted(credentials.Password) {
		encryptedPassword, err := cryptoService.Encrypt(credentials.Password)
		if err != nil {
			return false, fmt.Errorf("error encriptando password: %w", err)
		}
		credentials.Password = encryptedPassword
		changed = true
	}

	if credentials.ClientSecret != "" && !cryptoService.IsEncrypted(credentials.ClientSecret) {
		encryptedSecret, err := cryptoService.Encrypt(credentials.ClientSecret)
		if err != nil {
			return false, fmt.Errorf("error encriptando client secret: %w", err)
		}
		credentials.ClientSecret = encryptedSecret
		changed = true
	}

	if credentials.APIKey != "" && !cryptoService.IsEncrypted(credentials.APIKey) {
		encryptedAPIKey, err := cryptoService.Encrypt(credentials.APIKey)
		if err != nil {
			return false, fmt.Errorf("error encriptando API Key: %w", err)
		}
		credentials.APIKey = encryptedAPIKey
		changed = true
	}

	return changed, nil
}

func ensureServiceCredentialsEncrypted(collection *mongo.Collection, service *models.ExternalService) error {
	if service.Credentials == nil {
		return nil
	}

	cryptoService, err := crypto.GetService()
	if err != nil {
		return fmt.Errorf("error inicializando servicio de encriptación: %w", err)
	}

	changed, err := encryptServiceCredentials(service.Credentials, cryptoService)
	if err != nil {
		return err
	}

	if !changed {
		return nil
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	_, err = collection.UpdateOne(ctx, bson.M{"_id": service.ID}, bson.M{
		"$set": bson.M{
			"credentials": service.Credentials,
			"updated_at":  time.Now(),
		},
	})
	return err
}
