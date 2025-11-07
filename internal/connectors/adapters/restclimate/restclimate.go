package restclimate

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"sync"
	"time"

	"omniapi/internal/connectors"
	"omniapi/internal/domain"
	"omniapi/internal/schema"

	"go.mongodb.org/mongo-driver/bson/primitive"
)

// RESTClimateConnector implementa un conector que hace polling a un endpoint REST para datos climáticos
type RESTClimateConnector struct {
	mu          sync.RWMutex
	id          string
	config      map[string]interface{}
	eventChan   chan<- connectors.CanonicalEvent
	running     bool
	sequence    uint64
	startTime   time.Time
	errorCount  int
	lastLatency time.Duration
	lastPoll    time.Time
	filters     []connectors.EventFilter
	tenantID    primitive.ObjectID
	mappings    []domain.Mapping

	// REST configuration
	endpoint     string
	pollInterval time.Duration
	timeout      time.Duration
	headers      map[string]string

	// HTTP client
	client   *http.Client
	stopChan chan struct{}
}

// NewRESTClimateConnector crea una nueva instancia del conector REST Climate
func NewRESTClimateConnector(config map[string]interface{}) (connectors.Connector, error) {
	instanceID, ok := config["__instance_id"].(string)
	if !ok {
		return nil, fmt.Errorf("missing instance_id in config")
	}

	tenantIDStr, ok := config["__tenant_id"].(string)
	if !ok {
		return nil, fmt.Errorf("missing tenant_id in config")
	}

	tenantID, err := primitive.ObjectIDFromHex(tenantIDStr)
	if err != nil {
		return nil, fmt.Errorf("invalid tenant_id: %w", err)
	}

	// Extraer configuración REST
	endpoint, ok := config["endpoint"].(string)
	if !ok {
		return nil, fmt.Errorf("missing endpoint in config")
	}

	// Intervalo de polling
	pollInterval := 30 * time.Second
	if intervalStr, ok := config["poll_interval"].(string); ok {
		if parsed, err := time.ParseDuration(intervalStr); err == nil {
			pollInterval = parsed
		}
	} else if intervalSec, ok := config["poll_interval_seconds"].(int); ok {
		pollInterval = time.Duration(intervalSec) * time.Second
	}

	// Timeout de requests
	timeout := 10 * time.Second
	if timeoutStr, ok := config["timeout"].(string); ok {
		if parsed, err := time.ParseDuration(timeoutStr); err == nil {
			timeout = parsed
		}
	} else if timeoutSec, ok := config["timeout_seconds"].(int); ok {
		timeout = time.Duration(timeoutSec) * time.Second
	}

	// Headers HTTP
	headers := make(map[string]string)
	if headersData, ok := config["headers"].(map[string]interface{}); ok {
		for k, v := range headersData {
			if strVal, ok := v.(string); ok {
				headers[k] = strVal
			}
		}
	}

	// Headers por defecto
	if _, exists := headers["User-Agent"]; !exists {
		headers["User-Agent"] = fmt.Sprintf("OmniAPI-RESTClimate/%s", instanceID)
	}
	if _, exists := headers["Accept"]; !exists {
		headers["Accept"] = "application/json"
	}

	// Extraer mappings
	var mappings []domain.Mapping
	if mappingsData, exists := config["__mappings"]; exists {
		if mappingsSlice, ok := mappingsData.([]domain.Mapping); ok {
			mappings = mappingsSlice
		}
	}

	return &RESTClimateConnector{
		id:           instanceID,
		config:       config,
		tenantID:     tenantID,
		endpoint:     endpoint,
		pollInterval: pollInterval,
		timeout:      timeout,
		headers:      headers,
		mappings:     mappings,
		client: &http.Client{
			Timeout: timeout,
		},
		stopChan: make(chan struct{}),
	}, nil
}

// ID retorna el ID de la instancia
func (r *RESTClimateConnector) ID() string {
	return r.id
}

// Type retorna el tipo de conector
func (r *RESTClimateConnector) Type() string {
	return "restclimate"
}

// Config retorna la configuración actual
func (r *RESTClimateConnector) Config() map[string]interface{} {
	r.mu.RLock()
	defer r.mu.RUnlock()

	config := make(map[string]interface{})
	for k, v := range r.config {
		config[k] = v
	}

	return config
}

// Capabilities retorna las capabilities soportadas
func (r *RESTClimateConnector) Capabilities() []domain.Capability {
	return []domain.Capability{domain.CapabilityClimateRead}
}

// Subscribe configura filtros para eventos
func (r *RESTClimateConnector) Subscribe(filters ...connectors.EventFilter) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	r.filters = filters
	return nil
}

// OnEvent configura el canal de eventos
func (r *RESTClimateConnector) OnEvent(eventChan chan<- connectors.CanonicalEvent) {
	r.mu.Lock()
	defer r.mu.Unlock()

	r.eventChan = eventChan
}

// Start inicia el conector REST
func (r *RESTClimateConnector) Start(ctx context.Context) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	if r.running {
		return fmt.Errorf("connector is already running")
	}

	if r.eventChan == nil {
		return fmt.Errorf("event channel not configured")
	}

	r.running = true
	r.startTime = time.Now()
	r.stopChan = make(chan struct{})

	// Iniciar goroutine de polling
	go r.pollLoop(ctx)

	return nil
}

// Stop detiene el conector
func (r *RESTClimateConnector) Stop() error {
	r.mu.Lock()
	defer r.mu.Unlock()

	if !r.running {
		return nil
	}

	r.running = false
	close(r.stopChan)

	return nil
}

// Health retorna información de salud con última latencia y estado
func (r *RESTClimateConnector) Health() connectors.HealthInfo {
	r.mu.RLock()
	defer r.mu.RUnlock()

	status := connectors.HealthStatusUnhealthy
	message := "Not running"

	if r.running {
		timeSinceLastPoll := time.Since(r.lastPoll)

		if timeSinceLastPoll < r.pollInterval*2 {
			status = connectors.HealthStatusHealthy
			message = "Polling actively"
		} else if timeSinceLastPoll < r.pollInterval*5 {
			status = connectors.HealthStatusDegraded
			message = "Delayed polling"
		} else {
			status = connectors.HealthStatusUnhealthy
			message = "Polling stalled"
		}

		// Considerar latencia alta como degradación
		if r.lastLatency > r.timeout/2 {
			if status == connectors.HealthStatusHealthy {
				status = connectors.HealthStatusDegraded
				message = "High latency detected"
			}
		}
	}

	var uptime time.Duration
	if !r.startTime.IsZero() {
		uptime = time.Since(r.startTime)
	}

	return connectors.HealthInfo{
		Status:     status,
		Message:    message,
		LastCheck:  time.Now(),
		ErrorCount: r.errorCount,
		Uptime:     uptime,
		Metrics: map[string]interface{}{
			"requests_made":     r.sequence,
			"last_latency_ms":   r.lastLatency.Milliseconds(),
			"poll_interval_sec": r.pollInterval.Seconds(),
			"endpoint":          r.endpoint,
			"last_poll":         r.lastPoll.Format(time.RFC3339),
		},
	}
}

// pollLoop es el bucle principal de polling
func (r *RESTClimateConnector) pollLoop(ctx context.Context) {
	ticker := time.NewTicker(r.pollInterval)
	defer ticker.Stop()

	// Hacer un poll inicial inmediatamente
	r.performPoll()

	for {
		select {
		case <-ctx.Done():
			return
		case <-r.stopChan:
			return
		case <-ticker.C:
			r.performPoll()
		}
	}
}

// performPoll hace una request HTTP y procesa la respuesta
func (r *RESTClimateConnector) performPoll() {
	startTime := time.Now()

	r.mu.Lock()
	r.sequence++
	seq := r.sequence
	eventChan := r.eventChan
	r.lastPoll = startTime
	r.mu.Unlock()

	if eventChan == nil {
		return
	}

	// Crear request HTTP
	req, err := http.NewRequest("GET", r.endpoint, nil)
	if err != nil {
		r.recordError(fmt.Errorf("error creating request: %w", err))
		return
	}

	// Agregar headers
	for k, v := range r.headers {
		req.Header.Set(k, v)
	}

	// Hacer la request
	resp, err := r.client.Do(req)
	if err != nil {
		r.recordError(fmt.Errorf("error making request: %w", err))
		return
	}
	defer resp.Body.Close()

	// Verificar status code
	if resp.StatusCode != http.StatusOK {
		r.recordError(fmt.Errorf("HTTP error: %d %s", resp.StatusCode, resp.Status))
		return
	}

	// Leer respuesta
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		r.recordError(fmt.Errorf("error reading response: %w", err))
		return
	}

	// Parsear JSON
	var rawData map[string]interface{}
	if err := json.Unmarshal(body, &rawData); err != nil {
		r.recordError(fmt.Errorf("error parsing JSON: %w", err))
		return
	}

	// Aplicar mapping proveedor → canónico
	canonicalPayload, err := r.applyMapping(rawData)
	if err != nil {
		r.recordError(fmt.Errorf("error applying mapping: %w", err))
		return
	}

	// Validar contra schema climate.v1
	_, err = schema.Validate("climate", "v1", canonicalPayload)
	if err != nil {
		r.recordError(fmt.Errorf("schema validation failed: %w", err))
		return
	}

	// Convertir a JSON
	payloadBytes, err := json.Marshal(canonicalPayload)
	if err != nil {
		r.recordError(fmt.Errorf("error marshaling canonical payload: %w", err))
		return
	}

	// Extraer información de ubicación desde la configuración o payload
	farmID := "rest-farm-001"
	siteID := "rest-site-001"
	var cageID *string

	if farm, ok := r.config["farm_id"].(string); ok {
		farmID = farm
	}
	if site, ok := r.config["site_id"].(string); ok {
		siteID = site
	}
	if cage, ok := r.config["cage_id"].(string); ok && cage != "" {
		cageID = &cage
	}

	// Crear StreamKey
	streamKey := domain.StreamKey{
		TenantID: r.tenantID,
		Kind:     domain.StreamKindClimate,
		FarmID:   farmID,
		SiteID:   siteID,
		CageID:   cageID,
	}

	// Crear envelope
	envelope := connectors.Envelope{
		Version:   "1.0",
		Timestamp: startTime,
		Stream:    streamKey,
		Source:    fmt.Sprintf("rest-climate-%s", r.id),
		Sequence:  seq,
		Flags:     connectors.EventFlagNone,
		TraceID:   fmt.Sprintf("rest-%d", seq),
	}

	// Crear evento canónico
	event := connectors.CanonicalEvent{
		Envelope:      envelope,
		Payload:       payloadBytes,
		Kind:          "climate",
		SchemaVersion: "v1",
	}

	// Calcular latencia
	latency := time.Since(startTime)
	r.mu.Lock()
	r.lastLatency = latency
	r.mu.Unlock()

	// Enviar evento (no bloqueante)
	select {
	case eventChan <- event:
		// Evento enviado exitosamente
	default:
		// Canal lleno, incrementar contador de errores
		r.mu.Lock()
		r.errorCount++
		r.mu.Unlock()
	}
}

// recordError incrementa el contador de errores y registra el error
func (r *RESTClimateConnector) recordError(err error) {
	r.mu.Lock()
	r.errorCount++
	r.mu.Unlock()

	// Log del error (en producción usaríamos un logger estructurado)
	fmt.Printf("RESTClimate Connector %s error: %v\n", r.id, err)
}

// applyMapping aplica las reglas de mapping a los datos del proveedor
func (r *RESTClimateConnector) applyMapping(rawData map[string]interface{}) (map[string]interface{}, error) {
	if len(r.mappings) == 0 {
		// Sin mappings configurados, asumir que los datos ya están en formato canónico
		return rawData, nil
	}

	// Buscar mapping para climate
	var climateMapping *domain.Mapping
	for i := range r.mappings {
		if r.mappings[i].Capability == domain.CapabilityClimateRead {
			climateMapping = &r.mappings[i]
			break
		}
	}

	if climateMapping == nil {
		return nil, fmt.Errorf("no climate mapping configured")
	}

	result := make(map[string]interface{})

	// Aplicar reglas de mapping
	for _, rule := range climateMapping.Rules {
		var value interface{}
		var exists bool

		// Obtener valor del campo fuente
		if rule.SourceField != "" {
			value, exists = rawData[rule.SourceField]
		}

		// Usar valor por defecto si no existe
		if !exists && rule.DefaultValue != nil {
			value = rule.DefaultValue
			exists = true
		}

		// Campo requerido pero no encontrado
		if rule.Required && !exists {
			return nil, fmt.Errorf("required field %s not found", rule.SourceField)
		}

		if exists {
			// Aplicar transformación si está configurada
			if rule.Transform != nil {
				transformedValue, err := r.applyTransform(value, rule.Transform)
				if err != nil {
					return nil, fmt.Errorf("error applying transform to field %s: %w", rule.TargetField, err)
				}
				value = transformedValue
			}

			result[rule.TargetField] = value
		}
	}

	// Agregar timestamp si no existe
	if _, exists := result["timestamp"]; !exists {
		result["timestamp"] = time.Now().Format(time.RFC3339)
	}

	return result, nil
}

// applyTransform aplica una transformación específica a un valor
func (r *RESTClimateConnector) applyTransform(value interface{}, transform *domain.Transform) (interface{}, error) {
	switch transform.Type {
	case domain.TransformTypeRename:
		// Rename no cambia el valor, solo el campo
		return value, nil

	case domain.TransformTypeUnit:
		// Conversión de unidades (ejemplo: Celsius a Fahrenheit, etc.)
		if numVal, ok := value.(float64); ok {
			if fromUnit, exists := transform.Parameters["from"]; exists {
				if toUnit, exists := transform.Parameters["to"]; exists {
					return r.convertUnits(numVal, fromUnit.(string), toUnit.(string))
				}
			}

			// Factor de conversión simple
			if factor, exists := transform.Parameters["factor"]; exists {
				if factorFloat, ok := factor.(float64); ok {
					return numVal * factorFloat, nil
				}
			}
		}
		return value, nil

	case domain.TransformTypeEnum:
		// Mapeo de enumeraciones
		if enumMap, exists := transform.Parameters["mapping"]; exists {
			if mapping, ok := enumMap.(map[string]interface{}); ok {
				if strVal, ok := value.(string); ok {
					if mappedValue, exists := mapping[strVal]; exists {
						return mappedValue, nil
					}
				}
			}
		}
		return value, nil

	case domain.TransformTypeScale:
		// Escalado numérico
		if numVal, ok := value.(float64); ok {
			if scale, exists := transform.Parameters["scale"]; exists {
				if scaleFloat, ok := scale.(float64); ok {
					return numVal * scaleFloat, nil
				}
			}
		}
		return value, nil

	case domain.TransformTypeTimestamp:
		// Conversión de timestamps
		if strVal, ok := value.(string); ok {
			// Intentar parsear diferentes formatos
			formats := []string{
				time.RFC3339,
				time.RFC822,
				"2006-01-02 15:04:05",
				"2006/01/02 15:04:05",
				"01/02/2006 15:04:05",
			}

			for _, format := range formats {
				if t, err := time.Parse(format, strVal); err == nil {
					return t.Format(time.RFC3339), nil
				}
			}
		}
		return value, nil

	default:
		return value, nil
	}
}

// convertUnits convierte entre diferentes unidades
func (r *RESTClimateConnector) convertUnits(value float64, from, to string) (float64, error) {
	// Conversiones de temperatura
	if from == "fahrenheit" && to == "celsius" {
		return (value - 32) * 5 / 9, nil
	}
	if from == "celsius" && to == "fahrenheit" {
		return value*9/5 + 32, nil
	}
	if from == "kelvin" && to == "celsius" {
		return value - 273.15, nil
	}
	if from == "celsius" && to == "kelvin" {
		return value + 273.15, nil
	}

	// Sin conversión conocida, retornar valor original
	return value, nil
}

// Factory para el conector REST Climate
func Factory(config map[string]interface{}) (connectors.Connector, error) {
	return NewRESTClimateConnector(config)
}

// Registration contiene la información de registro del conector REST Climate
var Registration = &connectors.ConnectorRegistration{
	Type:        "restclimate",
	Version:     "1.0.0",
	Factory:     Factory,
	Description: "REST Climate connector for polling climate data from HTTP endpoints",
	Capabilities: []domain.Capability{
		domain.CapabilityClimateRead,
	},
	ConfigSchema: map[string]interface{}{
		"type":     "object",
		"required": []string{"endpoint"},
		"properties": map[string]interface{}{
			"endpoint": map[string]interface{}{
				"type":        "string",
				"description": "HTTP endpoint URL to poll for climate data",
			},
			"poll_interval": map[string]interface{}{
				"type":        "string",
				"description": "Polling interval (e.g., '30s', '1m', '5m')",
				"default":     "30s",
			},
			"poll_interval_seconds": map[string]interface{}{
				"type":        "integer",
				"description": "Polling interval in seconds",
				"default":     30,
			},
			"timeout": map[string]interface{}{
				"type":        "string",
				"description": "HTTP request timeout (e.g., '10s', '30s')",
				"default":     "10s",
			},
			"timeout_seconds": map[string]interface{}{
				"type":        "integer",
				"description": "HTTP request timeout in seconds",
				"default":     10,
			},
			"headers": map[string]interface{}{
				"type":        "object",
				"description": "Additional HTTP headers to send",
			},
			"farm_id": map[string]interface{}{
				"type":        "string",
				"description": "Farm identifier for events",
				"default":     "rest-farm-001",
			},
			"site_id": map[string]interface{}{
				"type":        "string",
				"description": "Site identifier for events",
				"default":     "rest-site-001",
			},
			"cage_id": map[string]interface{}{
				"type":        "string",
				"description": "Cage identifier for events (optional)",
			},
		},
	},
}
