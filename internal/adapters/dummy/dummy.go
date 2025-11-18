package dummy

import (
	"context"
	"encoding/json"
	"fmt"
	"math/rand"
	"sync"
	"time"

	"omniapi/internal/connectors"
	"omniapi/internal/domain"

	"go.mongodb.org/mongo-driver/bson/primitive"
)

// DummyConnector implementa un conector de prueba que emite eventos sintéticos
type DummyConnector struct {
	mu         sync.RWMutex
	id         string
	config     map[string]interface{}
	eventChan  chan<- connectors.CanonicalEvent
	stopChan   chan struct{}
	running    bool
	sequence   uint64
	startTime  time.Time
	errorCount int
	filters    []connectors.EventFilter
	tenantID   primitive.ObjectID
	streamKey  domain.StreamKey
}

// NewDummyConnector crea una nueva instancia del conector dummy
func NewDummyConnector(config map[string]interface{}) (connectors.Connector, error) {
	// Extraer configuración
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

	// Configuración del stream
	farmID := "dummy-farm-001"
	siteID := "dummy-site-001"
	var cageID *string

	if val, exists := config["farm_id"]; exists {
		if str, ok := val.(string); ok {
			farmID = str
		}
	}

	if val, exists := config["site_id"]; exists {
		if str, ok := val.(string); ok {
			siteID = str
		}
	}

	if val, exists := config["cage_id"]; exists {
		if str, ok := val.(string); ok && str != "" {
			cageID = &str
		}
	}

	streamKey := domain.StreamKey{
		TenantID: tenantID,
		Kind:     domain.StreamKindFeeding, // Por defecto
		FarmID:   farmID,
		SiteID:   siteID,
		CageID:   cageID,
	}

	return &DummyConnector{
		id:        instanceID,
		config:    config,
		stopChan:  make(chan struct{}),
		tenantID:  tenantID,
		streamKey: streamKey,
	}, nil
}

// ID retorna el ID de la instancia
func (d *DummyConnector) ID() string {
	return d.id
}

// Type retorna el tipo de conector
func (d *DummyConnector) Type() string {
	return "dummy"
}

// Config retorna la configuración actual
func (d *DummyConnector) Config() map[string]interface{} {
	d.mu.RLock()
	defer d.mu.RUnlock()

	// Crear copia de la configuración
	config := make(map[string]interface{})
	for k, v := range d.config {
		config[k] = v
	}

	return config
}

// Capabilities retorna las capabilities soportadas
func (d *DummyConnector) Capabilities() []domain.Capability {
	return []domain.Capability{
		domain.CapabilityFeedingRead,
		domain.CapabilityBiometricRead,
		domain.CapabilityClimateRead,
	}
}

// Subscribe configura filtros para eventos
func (d *DummyConnector) Subscribe(filters ...connectors.EventFilter) error {
	d.mu.Lock()
	defer d.mu.Unlock()

	d.filters = filters
	return nil
}

// OnEvent configura el canal de eventos
func (d *DummyConnector) OnEvent(eventChan chan<- connectors.CanonicalEvent) {
	d.mu.Lock()
	defer d.mu.Unlock()

	d.eventChan = eventChan
}

// Start inicia el conector
func (d *DummyConnector) Start(ctx context.Context) error {
	d.mu.Lock()
	defer d.mu.Unlock()

	if d.running {
		return fmt.Errorf("connector is already running")
	}

	if d.eventChan == nil {
		return fmt.Errorf("event channel not configured")
	}

	d.running = true
	d.startTime = time.Now()
	d.stopChan = make(chan struct{})

	// Iniciar goroutine que emite eventos
	go d.eventLoop(ctx)

	return nil
}

// Stop detiene el conector
func (d *DummyConnector) Stop() error {
	d.mu.Lock()
	defer d.mu.Unlock()

	if !d.running {
		return nil
	}

	d.running = false
	close(d.stopChan)

	return nil
}

// Health retorna información de salud
func (d *DummyConnector) Health() connectors.HealthInfo {
	d.mu.RLock()
	defer d.mu.RUnlock()

	status := connectors.HealthStatusHealthy
	if !d.running {
		status = connectors.HealthStatusUnhealthy
	}

	var uptime time.Duration
	if !d.startTime.IsZero() {
		uptime = time.Since(d.startTime)
	}

	return connectors.HealthInfo{
		Status:     status,
		Message:    fmt.Sprintf("Dummy connector %s", status),
		LastCheck:  time.Now(),
		ErrorCount: d.errorCount,
		Uptime:     uptime,
		Metrics: map[string]interface{}{
			"events_sent": d.sequence,
			"running":     d.running,
		},
	}
}

// eventLoop es el bucle principal que emite eventos
func (d *DummyConnector) eventLoop(ctx context.Context) {
	ticker := time.NewTicker(1 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-d.stopChan:
			return
		case <-ticker.C:
			d.emitEvent()
		}
	}
}

// emitEvent emite un evento sintético
func (d *DummyConnector) emitEvent() {
	d.mu.Lock()
	d.sequence++
	seq := d.sequence
	eventChan := d.eventChan
	d.mu.Unlock()

	if eventChan == nil {
		return
	}

	// Rotar entre diferentes tipos de eventos
	eventTypes := []struct {
		kind       domain.StreamKind
		capability domain.Capability
		generator  func() interface{}
	}{
		{
			kind:       domain.StreamKindFeeding,
			capability: domain.CapabilityFeedingRead,
			generator:  d.generateFeedingData,
		},
		{
			kind:       domain.StreamKindBiometric,
			capability: domain.CapabilityBiometricRead,
			generator:  d.generateBiometricData,
		},
		{
			kind:       domain.StreamKindClimate,
			capability: domain.CapabilityClimateRead,
			generator:  d.generateClimateData,
		},
	}

	eventType := eventTypes[seq%uint64(len(eventTypes))]

	// Crear streamKey específica para este evento
	streamKey := d.streamKey
	streamKey.Kind = eventType.kind

	// Generar payload
	payload := eventType.generator()
	payloadBytes, err := json.Marshal(payload)
	if err != nil {
		d.mu.Lock()
		d.errorCount++
		d.mu.Unlock()
		return
	}

	// Crear envelope
	envelope := connectors.Envelope{
		Version:   "1.0",
		Timestamp: time.Now(),
		Stream:    streamKey,
		Source:    fmt.Sprintf("dummy-connector-%s", d.id),
		Sequence:  seq,
		Flags:     connectors.EventFlagSynthetic,
		TraceID:   fmt.Sprintf("trace-%d", seq),
	}

	// Crear evento canónico
	event := connectors.CanonicalEvent{
		Envelope:      envelope,
		Payload:       payloadBytes,
		Kind:          string(eventType.kind),
		SchemaVersion: "v1",
	}

	// Enviar evento (no bloqueante)
	select {
	case eventChan <- event:
		// Evento enviado exitosamente
	default:
		// Canal lleno, incrementar contador de errores
		d.mu.Lock()
		d.errorCount++
		d.mu.Unlock()
	}
}

// generateFeedingData genera datos sintéticos de alimentación
func (d *DummyConnector) generateFeedingData() interface{} {
	feedTypes := []string{"pellets", "liquid", "mixed"}

	return map[string]interface{}{
		"feed_amount": 2.5 + rand.Float64()*2.0, // 2.5-4.5 kg
		"feed_type":   feedTypes[rand.Intn(len(feedTypes))],
		"timestamp":   time.Now().Format(time.RFC3339),
		"cage_id":     d.getCageID(),
		"automatic":   rand.Float64() > 0.3, // 70% automático
	}
}

// generateBiometricData genera datos sintéticos biométricos
func (d *DummyConnector) generateBiometricData() interface{} {
	return map[string]interface{}{
		"weight":       1.2 + rand.Float64()*0.8,   // 1.2-2.0 kg
		"length":       25.0 + rand.Float64()*10.0, // 25-35 cm
		"health_score": 75 + rand.Intn(25),         // 75-100
		"timestamp":    time.Now().Format(time.RFC3339),
		"cage_id":      d.getCageID(),
		"sample_count": 1 + rand.Intn(5), // 1-5 muestras
	}
}

// generateClimateData genera datos sintéticos climáticos
func (d *DummyConnector) generateClimateData() interface{} {
	return map[string]interface{}{
		"temperature":  18.0 + rand.Float64()*8.0,  // 18-26°C
		"humidity":     60.0 + rand.Float64()*20.0, // 60-80%
		"oxygen_level": 7.0 + rand.Float64()*2.0,   // 7-9 mg/L
		"ph_level":     6.5 + rand.Float64()*1.5,   // 6.5-8.0
		"timestamp":    time.Now().Format(time.RFC3339),
		"sensor_id":    fmt.Sprintf("sensor-%s-001", d.streamKey.SiteID),
	}
}

// getCageID retorna el cage_id para usar en los eventos
func (d *DummyConnector) getCageID() string {
	if d.streamKey.CageID != nil {
		return *d.streamKey.CageID
	}
	return "dummy-cage-001"
}

// Función de factory para registrar el conector
func Factory(config map[string]interface{}) (connectors.Connector, error) {
	return NewDummyConnector(config)
}

// Registration contiene la información de registro del conector dummy
var Registration = &connectors.ConnectorRegistration{
	Type:        "dummy",
	Version:     "1.0.0",
	Factory:     Factory,
	Description: "Dummy connector for testing and development",
	Capabilities: []domain.Capability{
		domain.CapabilityFeedingRead,
		domain.CapabilityBiometricRead,
		domain.CapabilityClimateRead,
	},
	ConfigSchema: map[string]interface{}{
		"type": "object",
		"properties": map[string]interface{}{
			"farm_id": map[string]interface{}{
				"type":        "string",
				"description": "Farm identifier",
				"default":     "dummy-farm-001",
			},
			"site_id": map[string]interface{}{
				"type":        "string",
				"description": "Site identifier",
				"default":     "dummy-site-001",
			},
			"cage_id": map[string]interface{}{
				"type":        "string",
				"description": "Cage identifier (optional)",
			},
		},
	},
}
