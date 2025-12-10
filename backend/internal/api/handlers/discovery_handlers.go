package handlers

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"omniapi/internal/database"
	"omniapi/internal/models"
	"omniapi/internal/services"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

// DiscoveryRequest request para ejecutar discovery
type DiscoveryRequest struct {
	Provider  string `json:"provider"`   // innovex, scaleaq
	ServiceID string `json:"service_id"` // ID del ExternalService
	SiteID    string `json:"site_id"`    // ID del Site
	MonitorID string `json:"monitor_id"` // Para Innovex: monitor_id específico
}

// DiscoveryEndpointResult resultado de un endpoint individual
type DiscoveryEndpointResult struct {
	Label        string      `json:"label"`
	Method       string      `json:"method"`
	Path         string      `json:"path"`
	FullURL      string      `json:"full_url"`
	Description  string      `json:"description"`
	StatusCode   int         `json:"status_code"`
	Availability string      `json:"availability"` // ready, partial, error
	LatencyMS    int64       `json:"latency_ms"`
	Data         interface{} `json:"data,omitempty"`
	Error        string      `json:"error,omitempty"`
	LastSync     string      `json:"last_sync"`
}

// DiscoveryGroupResult grupo de endpoints
type DiscoveryGroupResult struct {
	ID          string                    `json:"id"`
	Title       string                    `json:"title"`
	Description string                    `json:"description"`
	Endpoints   []DiscoveryEndpointResult `json:"endpoints"`
}

// DiscoveryResponse respuesta completa del discovery
type DiscoveryResponse struct {
	Provider    string                 `json:"provider"`
	SiteID      string                 `json:"site_id"`
	SiteName    string                 `json:"site_name"`
	TenantCode  string                 `json:"tenant_code,omitempty"`
	GeneratedAt string                 `json:"generated_at"`
	HeadersUsed map[string]string      `json:"headers_used"`
	Summary     map[string]interface{} `json:"summary"`
	Groups      []DiscoveryGroupResult `json:"groups"`
}

// RunDiscoveryHandler ejecuta discovery real contra las APIs
func RunDiscoveryHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method == "OPTIONS" {
		w.WriteHeader(http.StatusOK)
		return
	}

	if r.Method != "POST" {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var req DiscoveryRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		sendJSONError(w, "Invalid request body: "+err.Error(), http.StatusBadRequest)
		return
	}

	// Obtener el ExternalService
	service, err := getExternalService(req.ServiceID)
	if err != nil {
		sendJSONError(w, "Service not found: "+err.Error(), http.StatusNotFound)
		return
	}

	// Obtener el Site
	site, err := getSite(req.SiteID)
	if err != nil {
		sendJSONError(w, "Site not found: "+err.Error(), http.StatusNotFound)
		return
	}

	// Ejecutar discovery según provider
	var response DiscoveryResponse
	switch req.Provider {
	case "innovex":
		response, err = runInnovexDiscovery(service, site, req.MonitorID)
	case "scaleaq":
		response, err = runScaleAQDiscovery(service, site)
	default:
		sendJSONError(w, "Unknown provider: "+req.Provider, http.StatusBadRequest)
		return
	}

	if err != nil {
		sendJSONError(w, "Discovery failed: "+err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"success": true,
		"data":    response,
	})
}

// runInnovexDiscovery ejecuta discovery para Innovex Dataweb
func runInnovexDiscovery(service *models.ExternalService, site *models.Site, monitorID string) (DiscoveryResponse, error) {
	response := DiscoveryResponse{
		Provider:    "innovex",
		SiteID:      site.ID.Hex(),
		SiteName:    site.Name,
		TenantCode:  site.TenantCode,
		GeneratedAt: time.Now().Format(time.RFC3339),
		HeadersUsed: map[string]string{},
		Summary:     map[string]interface{}{},
		Groups:      []DiscoveryGroupResult{},
	}

	// Obtener token
	token, err := services.GetTokenManager().GetToken(service)
	if err != nil {
		return response, fmt.Errorf("error obteniendo token: %v", err)
	}

	response.HeadersUsed["Authorization"] = "Bearer " + token[:20] + "..." // Truncar por seguridad
	response.HeadersUsed["Content-Type"] = "application/json"

	// Si no hay monitorID, usar el del config o el site code
	if monitorID == "" {
		if mid, ok := service.Config["monitor_id"].(string); ok && mid != "" {
			monitorID = mid
		} else {
			monitorID = site.Code
		}
	}

	baseURL := strings.TrimSuffix(service.BaseURL, "/")

	// Grupo 1: Catálogo de Monitores
	monitorsGroup := DiscoveryGroupResult{
		ID:          "monitors",
		Title:       "Catálogo de Monitores",
		Description: "Recursos para obtener la lista de centros disponibles y sus sensores asociados.",
		Endpoints:   []DiscoveryEndpointResult{},
	}

	// Endpoint: All Monitors
	allMonitorsResult := callEndpoint(
		baseURL+"/api_dataweb/all_monitors/?active=all",
		"GET",
		token,
		"All Monitors",
		"/api_dataweb/all_monitors/?active=all",
		"Devuelve todos los monitores asociados al cliente, incluyendo lat/lon y monitor_key.",
	)
	monitorsGroup.Endpoints = append(monitorsGroup.Endpoints, allMonitorsResult)

	// Endpoint: Monitor Detail
	monitorDetailPath := fmt.Sprintf("/api_dataweb/monitor_detail/?monitor_id=%s", monitorID)
	monitorDetailResult := callEndpoint(
		baseURL+monitorDetailPath,
		"GET",
		token,
		"Monitor Detail",
		monitorDetailPath,
		"Lista loggers y sensores del monitor, incluyendo jaulas, módulos y profundidad.",
	)
	monitorsGroup.Endpoints = append(monitorsGroup.Endpoints, monitorDetailResult)

	response.Groups = append(response.Groups, monitorsGroup)

	// Grupo 2: Últimas Lecturas
	latestGroup := DiscoveryGroupResult{
		ID:          "latest",
		Title:       "Últimas Lecturas",
		Description: "Lecturas frescas de sensores por monitor y tipo de medición (oxygen, flow, weather, etc.).",
		Endpoints:   []DiscoveryEndpointResult{},
	}

	// Endpoint: Monitor Sensor Last Data (oxygen)
	lastDataPath := fmt.Sprintf("/api_dataweb/monitor_sensor_last_data/?id=%s&medition=oxygen", monitorID)
	lastDataResult := callEndpoint(
		baseURL+lastDataPath,
		"GET",
		token,
		"Monitor Sensor Last Data (oxygen)",
		lastDataPath,
		"Retorna la última medición de oxígeno por sensor con temperatura, saturación y salinidad.",
	)
	latestGroup.Endpoints = append(latestGroup.Endpoints, lastDataResult)

	// Endpoint: Monitor Sensor Last Data (flow)
	lastDataFlowPath := fmt.Sprintf("/api_dataweb/monitor_sensor_last_data/?id=%s&medition=flow", monitorID)
	lastDataFlowResult := callEndpoint(
		baseURL+lastDataFlowPath,
		"GET",
		token,
		"Monitor Sensor Last Data (flow)",
		lastDataFlowPath,
		"Retorna la última medición de flujo por sensor.",
	)
	latestGroup.Endpoints = append(latestGroup.Endpoints, lastDataFlowResult)

	response.Groups = append(response.Groups, latestGroup)

	// Grupo 3: Series Históricas (solo metadata, no llamamos para no sobrecargar)
	rangesGroup := DiscoveryGroupResult{
		ID:          "ranges",
		Title:       "Series Históricas",
		Description: "Consultas para rangos de tiempo limitados (hasta 30 días) por monitor/sensor. Requieren parámetros adicionales.",
		Endpoints: []DiscoveryEndpointResult{
			{
				Label:        "Get Data Range",
				Method:       "GET",
				Path:         "/api_dataweb/get_data_range/?monitor_id={monitor_id}&sensor_id={sensor_id}&unixtime_since=...&unixtime_until=...",
				FullURL:      baseURL + "/api_dataweb/get_data_range/",
				Description:  "Devuelve lecturas en un rango definido. Requiere sensor_id y rango de tiempo.",
				Availability: "ready",
				LastSync:     time.Now().Format(time.RFC3339),
			},
			{
				Label:        "Monitor Sensor Time Data",
				Method:       "GET",
				Path:         "/api_dataweb/monitor_sensor_time_data/?monitor_id={monitor_id}&medition=oxygen&unixtime_since=...&unixtime_until=...",
				FullURL:      baseURL + "/api_dataweb/monitor_sensor_time_data/",
				Description:  "Rango con agrupación por sensor para un tipo de medición completo.",
				Availability: "ready",
				LastSync:     time.Now().Format(time.RFC3339),
			},
		},
	}
	response.Groups = append(response.Groups, rangesGroup)

	// Calcular summary
	totalEndpoints := 0
	successEndpoints := 0
	for _, group := range response.Groups {
		for _, ep := range group.Endpoints {
			totalEndpoints++
			if ep.Availability == "ready" {
				successEndpoints++
			}
		}
	}

	response.Summary["total_endpoints"] = totalEndpoints
	response.Summary["available_endpoints"] = successEndpoints
	response.Summary["monitor_id"] = monitorID

	return response, nil
}

// runScaleAQDiscovery ejecuta discovery para ScaleAQ
func runScaleAQDiscovery(service *models.ExternalService, site *models.Site) (DiscoveryResponse, error) {
	response := DiscoveryResponse{
		Provider:    "scaleaq",
		SiteID:      site.ID.Hex(),
		SiteName:    site.Name,
		TenantCode:  site.TenantCode,
		GeneratedAt: time.Now().Format(time.RFC3339),
		HeadersUsed: map[string]string{},
		Summary:     map[string]interface{}{},
		Groups:      []DiscoveryGroupResult{},
	}

	// Obtener token
	token, err := services.GetTokenManager().GetToken(service)
	if err != nil {
		return response, fmt.Errorf("error obteniendo token: %v", err)
	}

	response.HeadersUsed["Authorization"] = "Bearer " + token[:20] + "..."
	response.HeadersUsed["Scale-Version"] = "2025-01-01"
	response.HeadersUsed["Accept"] = "application/json"

	baseURL := strings.TrimSuffix(service.BaseURL, "/")
	siteID := site.Code
	if sid, ok := service.Config["scaleaq_site_id"].(string); ok && sid != "" {
		siteID = sid
	}

	// Grupo 1: Meta
	metaGroup := DiscoveryGroupResult{
		ID:          "meta",
		Title:       "Metadata",
		Description: "Información de compañía y centros.",
		Endpoints:   []DiscoveryEndpointResult{},
	}

	// Company Info
	companyResult := callScaleAQEndpoint(
		baseURL+"/meta/company?include=all",
		"GET",
		token,
		"Company Info",
		"/meta/company?include=all",
		"Ficha de compañía completa con lista de sitios.",
	)
	metaGroup.Endpoints = append(metaGroup.Endpoints, companyResult)

	// Site Info
	sitePath := fmt.Sprintf("/meta/sites/%s?include=all", siteID)
	siteResult := callScaleAQEndpoint(
		baseURL+sitePath,
		"GET",
		token,
		"Site Info",
		sitePath,
		"Información detallada del centro seleccionado.",
	)
	metaGroup.Endpoints = append(metaGroup.Endpoints, siteResult)

	response.Groups = append(response.Groups, metaGroup)

	// Grupo 2: Time Series
	timeseriesGroup := DiscoveryGroupResult{
		ID:          "timeseries",
		Title:       "Time Series",
		Description: "Consulta de series temporales.",
		Endpoints: []DiscoveryEndpointResult{
			{
				Label:        "Time Series Retrieve",
				Method:       "POST",
				Path:         "/time_series/retrieve",
				FullURL:      baseURL + "/time_series/retrieve",
				Description:  "Consulta cruda de series basadas en canales y timeRange.",
				Availability: "ready",
				LastSync:     time.Now().Format(time.RFC3339),
			},
			{
				Label:        "Available Data Types",
				Method:       "POST",
				Path:         "/time_series/retrieve/data_types",
				FullURL:      baseURL + "/time_series/retrieve/data_types",
				Description:  "Catálogo de canales y unidades disponibles para el sitio.",
				Availability: "ready",
				LastSync:     time.Now().Format(time.RFC3339),
			},
		},
	}
	response.Groups = append(response.Groups, timeseriesGroup)

	// Grupo 3: Feeding
	feedingGroup := DiscoveryGroupResult{
		ID:          "feeding",
		Title:       "Feeding",
		Description: "Datos de alimentación por unidad y timeline.",
		Endpoints: []DiscoveryEndpointResult{
			{
				Label:        "Feeding Units",
				Method:       "GET",
				Path:         fmt.Sprintf("/feeding-dashboard/units/%s/details", siteID),
				FullURL:      baseURL + fmt.Sprintf("/feeding-dashboard/units/%s/details", siteID),
				Description:  "Resumen operacional por unidad (consumo, mortalidad, alertas).",
				Availability: "ready",
				LastSync:     time.Now().Format(time.RFC3339),
			},
			{
				Label:        "Feeding Timeline",
				Method:       "GET",
				Path:         fmt.Sprintf("/feeding-dashboard/timeline/%s/{siteId}", siteID),
				FullURL:      baseURL + fmt.Sprintf("/feeding-dashboard/timeline/%s/", siteID),
				Description:  "Serie consolidada de alimento por bloques de 10 minutos.",
				Availability: "ready",
				LastSync:     time.Now().Format(time.RFC3339),
			},
		},
	}
	response.Groups = append(response.Groups, feedingGroup)

	// Calcular summary
	totalEndpoints := 0
	successEndpoints := 0
	for _, group := range response.Groups {
		for _, ep := range group.Endpoints {
			totalEndpoints++
			if ep.Availability == "ready" {
				successEndpoints++
			}
		}
	}

	response.Summary["total_endpoints"] = totalEndpoints
	response.Summary["available_endpoints"] = successEndpoints
	response.Summary["site_id"] = siteID

	return response, nil
}

// callEndpoint hace una llamada HTTP y retorna el resultado formateado
func callEndpoint(url, method, token, label, path, description string) DiscoveryEndpointResult {
	result := DiscoveryEndpointResult{
		Label:       label,
		Method:      method,
		Path:        path,
		FullURL:     url,
		Description: description,
		LastSync:    time.Now().Format(time.RFC3339),
	}

	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()

	req, err := http.NewRequestWithContext(ctx, method, url, nil)
	if err != nil {
		result.Availability = "error"
		result.Error = err.Error()
		return result
	}

	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("Content-Type", "application/json")

	start := time.Now()
	client := &http.Client{Timeout: 15 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		result.Availability = "error"
		result.Error = err.Error()
		result.LatencyMS = time.Since(start).Milliseconds()
		return result
	}
	defer resp.Body.Close()

	result.LatencyMS = time.Since(start).Milliseconds()
	result.StatusCode = resp.StatusCode

	body, _ := io.ReadAll(resp.Body)

	var data interface{}
	if err := json.Unmarshal(body, &data); err != nil {
		data = string(body)
	}
	result.Data = data

	if resp.StatusCode >= 200 && resp.StatusCode < 300 {
		result.Availability = "ready"
	} else if resp.StatusCode == 401 || resp.StatusCode == 403 {
		result.Availability = "error"
		result.Error = fmt.Sprintf("HTTP %d: Auth error", resp.StatusCode)
	} else {
		result.Availability = "partial"
		result.Error = fmt.Sprintf("HTTP %d", resp.StatusCode)
	}

	return result
}

// callScaleAQEndpoint hace una llamada HTTP con headers de ScaleAQ
func callScaleAQEndpoint(url, method, token, label, path, description string) DiscoveryEndpointResult {
	result := DiscoveryEndpointResult{
		Label:       label,
		Method:      method,
		Path:        path,
		FullURL:     url,
		Description: description,
		LastSync:    time.Now().Format(time.RFC3339),
	}

	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()

	req, err := http.NewRequestWithContext(ctx, method, url, nil)
	if err != nil {
		result.Availability = "error"
		result.Error = err.Error()
		return result
	}

	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("Scale-Version", "2025-01-01")
	req.Header.Set("Accept", "application/json")

	start := time.Now()
	client := &http.Client{Timeout: 15 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		result.Availability = "error"
		result.Error = err.Error()
		result.LatencyMS = time.Since(start).Milliseconds()
		return result
	}
	defer resp.Body.Close()

	result.LatencyMS = time.Since(start).Milliseconds()
	result.StatusCode = resp.StatusCode

	body, _ := io.ReadAll(resp.Body)

	var data interface{}
	if err := json.Unmarshal(body, &data); err != nil {
		data = string(body)
	}
	result.Data = data

	if resp.StatusCode >= 200 && resp.StatusCode < 300 {
		result.Availability = "ready"
	} else if resp.StatusCode == 401 || resp.StatusCode == 403 {
		result.Availability = "error"
		result.Error = fmt.Sprintf("HTTP %d: Auth error", resp.StatusCode)
	} else {
		result.Availability = "partial"
		result.Error = fmt.Sprintf("HTTP %d", resp.StatusCode)
	}

	return result
}

// getExternalService obtiene un ExternalService por ID
func getExternalService(id string) (*models.ExternalService, error) {
	objectID, err := primitive.ObjectIDFromHex(id)
	if err != nil {
		return nil, fmt.Errorf("invalid service ID: %v", err)
	}

	collection := database.GetCollection("external_services")
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	var service models.ExternalService
	err = collection.FindOne(ctx, bson.M{"_id": objectID}).Decode(&service)
	if err != nil {
		return nil, err
	}

	return &service, nil
}

// getSite obtiene un Site por ID
func getSite(id string) (*models.Site, error) {
	objectID, err := primitive.ObjectIDFromHex(id)
	if err != nil {
		return nil, fmt.Errorf("invalid site ID: %v", err)
	}

	collection := database.GetCollection("sites")
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	var site models.Site
	err = collection.FindOne(ctx, bson.M{"_id": objectID}).Decode(&site)
	if err != nil {
		return nil, err
	}

	return &site, nil
}

func sendJSONError(w http.ResponseWriter, message string, status int) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(map[string]interface{}{
		"success": false,
		"message": message,
	})
}
