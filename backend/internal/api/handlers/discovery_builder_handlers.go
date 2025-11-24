package handlers

import (
	"context"
	"encoding/json"
	"net/http"
	"strconv"
	"strings"
	"time"

	"omniapi/internal/database"
	"omniapi/internal/models"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo/options"
)

type discoveryRunRequest struct {
	Provider         string                            `json:"provider"`
	Site             models.BuilderSiteInfo            `json:"site"`
	Endpoints        []models.BuilderEndpointSelection `json:"endpoints"`
	DiscoverySummary map[string]interface{}            `json:"discovery_summary,omitempty"`
	Notes            string                            `json:"notes,omitempty"`
	CreatedBy        string                            `json:"created_by,omitempty"`
}

// DiscoveryRunsHandler maneja operaciones de listado y creación de corridas del builder
func DiscoveryRunsHandler(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		handleGetDiscoveryRuns(w, r)
	case http.MethodPost:
		handleCreateDiscoveryRun(w, r)
	default:
		http.Error(w, "Método no permitido", http.StatusMethodNotAllowed)
	}
}

func handleGetDiscoveryRuns(w http.ResponseWriter, r *http.Request) {
	collection := database.GetCollection("discovery_runs")

	filter := bson.M{}
	if provider := r.URL.Query().Get("provider"); provider != "" {
		filter["provider"] = provider
	}
	if siteID := r.URL.Query().Get("site_id"); siteID != "" {
		filter["site.id"] = siteID
	}
	if tenantCode := r.URL.Query().Get("tenant_code"); tenantCode != "" {
		filter["site.tenant_code"] = tenantCode
	}

	limit := int64(20)
	if limitParam := r.URL.Query().Get("limit"); limitParam != "" {
		if parsed, err := strconv.Atoi(limitParam); err == nil {
			if parsed > 0 && parsed <= 100 {
				limit = int64(parsed)
			}
		}
	}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	opts := options.Find().SetSort(bson.D{{Key: "created_at", Value: -1}}).SetLimit(limit)
	cursor, err := collection.Find(ctx, filter, opts)
	if err != nil {
		respondJSON(w, http.StatusInternalServerError, models.APIResponse{
			Success:   false,
			Message:   "Error obteniendo ejecuciones: " + err.Error(),
			Timestamp: time.Now().Unix(),
		})
		return
	}
	defer cursor.Close(ctx)

	var runs []models.DiscoveryBuilderRun
	if err := cursor.All(ctx, &runs); err != nil {
		respondJSON(w, http.StatusInternalServerError, models.APIResponse{
			Success:   false,
			Message:   "Error leyendo ejecuciones: " + err.Error(),
			Timestamp: time.Now().Unix(),
		})
		return
	}

	if runs == nil {
		runs = []models.DiscoveryBuilderRun{}
	}

	respondJSON(w, http.StatusOK, models.APIResponse{
		Success:   true,
		Message:   "Ejecuciones obtenidas",
		Data:      runs,
		Timestamp: time.Now().Unix(),
	})
}

func handleCreateDiscoveryRun(w http.ResponseWriter, r *http.Request) {
	var req discoveryRunRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respondJSON(w, http.StatusBadRequest, models.APIResponse{
			Success:   false,
			Message:   "JSON inválido: " + err.Error(),
			Timestamp: time.Now().Unix(),
		})
		return
	}

	req.Provider = strings.TrimSpace(req.Provider)
	req.Site.ID = strings.TrimSpace(req.Site.ID)
	req.Site.Code = strings.TrimSpace(req.Site.Code)
	req.Site.Name = strings.TrimSpace(req.Site.Name)
	req.Site.TenantID = strings.TrimSpace(req.Site.TenantID)
	req.Site.TenantCode = strings.TrimSpace(req.Site.TenantCode)
	req.Notes = strings.TrimSpace(req.Notes)
	req.CreatedBy = strings.TrimSpace(req.CreatedBy)

	if req.Provider == "" {
		respondJSON(w, http.StatusBadRequest, models.APIResponse{
			Success:   false,
			Message:   "El proveedor es requerido",
			Timestamp: time.Now().Unix(),
		})
		return
	}

	if len(req.Endpoints) == 0 {
		respondJSON(w, http.StatusBadRequest, models.APIResponse{
			Success:   false,
			Message:   "Debes seleccionar al menos un endpoint",
			Timestamp: time.Now().Unix(),
		})
		return
	}

	if req.Site.ID == "" && req.Site.Code == "" && req.Site.Name == "" {
		respondJSON(w, http.StatusBadRequest, models.APIResponse{
			Success:   false,
			Message:   "Información del sitio requerida",
			Timestamp: time.Now().Unix(),
		})
		return
	}

	for i := range req.Endpoints {
		req.Endpoints[i].EndpointID = strings.TrimSpace(req.Endpoints[i].EndpointID)
		req.Endpoints[i].Label = strings.TrimSpace(req.Endpoints[i].Label)
		req.Endpoints[i].Method = strings.ToUpper(strings.TrimSpace(req.Endpoints[i].Method))
		req.Endpoints[i].Path = strings.TrimSpace(req.Endpoints[i].Path)
		req.Endpoints[i].TargetBlock = strings.TrimSpace(req.Endpoints[i].TargetBlock)
	}

	run := models.DiscoveryBuilderRun{
		Provider:         req.Provider,
		Site:             req.Site,
		Endpoints:        req.Endpoints,
		DiscoverySummary: req.DiscoverySummary,
		Notes:            req.Notes,
		CreatedBy:        req.CreatedBy,
		CreatedAt:        time.Now(),
		UpdatedAt:        time.Now(),
	}

	collection := database.GetCollection("discovery_runs")
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	result, err := collection.InsertOne(ctx, run)
	if err != nil {
		respondJSON(w, http.StatusInternalServerError, models.APIResponse{
			Success:   false,
			Message:   "Error guardando la ejecución: " + err.Error(),
			Timestamp: time.Now().Unix(),
		})
		return
	}

	if insertedID, ok := result.InsertedID.(primitive.ObjectID); ok {
		run.ID = insertedID
	}

	respondJSON(w, http.StatusCreated, models.APIResponse{
		Success:   true,
		Message:   "Builder guardado correctamente",
		Data:      run,
		Timestamp: time.Now().Unix(),
	})
}

func respondJSON(w http.ResponseWriter, status int, payload models.APIResponse) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(payload)
}
