package handlers

import (
	"encoding/json"
	"net/http"

	"omniapi/internal/schema"
)

// ValidationRequest representa una solicitud de validación
type ValidationRequest struct {
	Kind    string      `json:"kind"`
	Version string      `json:"version"`
	Payload interface{} `json:"payload"`
}

// ValidationResponse representa la respuesta de validación
type ValidationResponse struct {
	Success bool                     `json:"success"`
	Message string                   `json:"message"`
	Data    *schema.ValidationResult `json:"data,omitempty"`
}

// ValidateSchemaHandler valida un payload contra un schema específico
func ValidateSchemaHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var req ValidationRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		response := ValidationResponse{
			Success: false,
			Message: "Invalid JSON payload",
		}
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(response)
		return
	}

	// Validar campos requeridos
	if req.Kind == "" {
		response := ValidationResponse{
			Success: false,
			Message: "Kind is required",
		}
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(response)
		return
	}

	if req.Version == "" {
		response := ValidationResponse{
			Success: false,
			Message: "Version is required",
		}
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(response)
		return
	}

	// Realizar validación
	result, err := schema.Validate(req.Kind, req.Version, req.Payload)
	if err != nil {
		response := ValidationResponse{
			Success: false,
			Message: err.Error(),
		}
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(response)
		return
	}

	// Enviar respuesta
	var message string
	if result.Valid {
		message = "Payload validation successful"
	} else {
		message = "Payload validation failed"
	}

	response := ValidationResponse{
		Success: result.Valid,
		Message: message,
		Data:    result,
	}

	if result.Valid {
		w.WriteHeader(http.StatusOK)
	} else {
		w.WriteHeader(http.StatusUnprocessableEntity)
	}

	json.NewEncoder(w).Encode(response)
}

// ListSchemasHandler lista todos los schemas disponibles
func ListSchemasHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// Crear manager y cargar schemas
	manager := schema.NewSchemaManager("configs/schemas")
	if err := manager.LoadSchemas(); err != nil {
		response := map[string]interface{}{
			"success": false,
			"message": "Failed to load schemas",
			"error":   err.Error(),
		}
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(response)
		return
	}

	// Obtener lista de schemas
	schemas := manager.ListSchemas()

	// Crear respuesta simplificada
	schemaList := make([]map[string]interface{}, 0, len(schemas))
	for key, schema := range schemas {
		schemaInfo := map[string]interface{}{
			"key":                 key,
			"kind":                schema.Kind,
			"version":             schema.Version,
			"title":               schema.Title,
			"capability":          schema.GetCapability(),
			"backward_compatible": schema.IsBackwardCompatible(),
			"metadata":            schema.Metadata,
		}
		schemaList = append(schemaList, schemaInfo)
	}

	response := map[string]interface{}{
		"success": true,
		"message": "Schemas retrieved successfully",
		"data": map[string]interface{}{
			"schemas": schemaList,
			"count":   len(schemaList),
		},
	}

	json.NewEncoder(w).Encode(response)
}

// GetSchemaHandler obtiene un schema específico
func GetSchemaHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// Obtener parámetros de query
	kind := r.URL.Query().Get("kind")
	version := r.URL.Query().Get("version")

	if kind == "" || version == "" {
		response := map[string]interface{}{
			"success": false,
			"message": "Kind and version parameters are required",
		}
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(response)
		return
	}

	// Crear manager y cargar schemas
	manager := schema.NewSchemaManager("configs/schemas")
	if err := manager.LoadSchemas(); err != nil {
		response := map[string]interface{}{
			"success": false,
			"message": "Failed to load schemas",
			"error":   err.Error(),
		}
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(response)
		return
	}

	// Obtener schema específico
	schemaObj, err := manager.GetSchema(kind, version)
	if err != nil {
		response := map[string]interface{}{
			"success": false,
			"message": "Schema not found",
			"error":   err.Error(),
		}
		w.WriteHeader(http.StatusNotFound)
		json.NewEncoder(w).Encode(response)
		return
	}

	response := map[string]interface{}{
		"success": true,
		"message": "Schema retrieved successfully",
		"data": map[string]interface{}{
			"kind":                schemaObj.Kind,
			"version":             schemaObj.Version,
			"title":               schemaObj.Title,
			"id":                  schemaObj.ID,
			"capability":          schemaObj.GetCapability(),
			"backward_compatible": schemaObj.IsBackwardCompatible(),
			"metadata":            schemaObj.Metadata,
			"schema":              schemaObj.Schema,
		},
	}

	json.NewEncoder(w).Encode(response)
}
