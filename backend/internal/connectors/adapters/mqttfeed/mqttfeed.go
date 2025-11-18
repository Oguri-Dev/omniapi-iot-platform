package mqttfeed

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"strings"
	"sync"
	"time"

	"omniapi/internal/connectors"
	"omniapi/internal/domain"
	"omniapi/internal/schema"

	mqtt "github.com/eclipse/paho.mqtt.golang"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

// MQTTFeedConnector implementa un conector que se suscribe a topics MQTT para datos de feeding
type MQTTFeedConnector struct {
	mu          sync.RWMutex
	id          string
	config      map[string]interface{}
	eventChan   chan<- connectors.CanonicalEvent
	client      mqtt.Client
	running     bool
	sequence    uint64
	startTime   time.Time
	errorCount  int
	lastLatency time.Duration
	lastMessage time.Time
	filters     []connectors.EventFilter
	tenantID    primitive.ObjectID
	mappings    []domain.Mapping

	// MQTT configuration
	brokerURL    string
	username     string
	password     string
	topicPattern string
	qos          byte
}

// NewMQTTFeedConnector crea una nueva instancia del conector MQTT Feed
func NewMQTTFeedConnector(config map[string]interface{}) (connectors.Connector, error) {
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

	// Extraer configuración MQTT
	brokerURL, ok := config["broker_url"].(string)
	if !ok {
		return nil, fmt.Errorf("missing broker_url in config")
	}

	username, _ := config["username"].(string)
	password, _ := config["password"].(string)

	topicPattern, ok := config["topic_pattern"].(string)
	if !ok {
		topicPattern = "+/+/+/feeding" // Patrón por defecto
	}

	qos := byte(1)
	if qosInt, ok := config["qos"].(int); ok && qosInt >= 0 && qosInt <= 2 {
		qos = byte(qosInt)
	}

	// Extraer mappings
	var mappings []domain.Mapping
	if mappingsData, exists := config["__mappings"]; exists {
		if mappingsSlice, ok := mappingsData.([]domain.Mapping); ok {
			mappings = mappingsSlice
		}
	}

	return &MQTTFeedConnector{
		id:           instanceID,
		config:       config,
		tenantID:     tenantID,
		brokerURL:    brokerURL,
		username:     username,
		password:     password,
		topicPattern: topicPattern,
		qos:          qos,
		mappings:     mappings,
	}, nil
}

// ID retorna el ID de la instancia
func (m *MQTTFeedConnector) ID() string {
	return m.id
}

// Type retorna el tipo de conector
func (m *MQTTFeedConnector) Type() string {
	return "mqttfeed"
}

// Config retorna la configuración actual
func (m *MQTTFeedConnector) Config() map[string]interface{} {
	m.mu.RLock()
	defer m.mu.RUnlock()

	config := make(map[string]interface{})
	for k, v := range m.config {
		config[k] = v
	}

	return config
}

// Capabilities retorna las capabilities soportadas
func (m *MQTTFeedConnector) Capabilities() []domain.Capability {
	return []domain.Capability{domain.CapabilityFeedingRead}
}

// Subscribe configura filtros para eventos
func (m *MQTTFeedConnector) Subscribe(filters ...connectors.EventFilter) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.filters = filters
	return nil
}

// OnEvent configura el canal de eventos
func (m *MQTTFeedConnector) OnEvent(eventChan chan<- connectors.CanonicalEvent) {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.eventChan = eventChan
}

// Start inicia el conector MQTT
func (m *MQTTFeedConnector) Start(ctx context.Context) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	if m.running {
		return fmt.Errorf("connector is already running")
	}

	if m.eventChan == nil {
		return fmt.Errorf("event channel not configured")
	}

	// Configurar cliente MQTT
	opts := mqtt.NewClientOptions()
	opts.AddBroker(m.brokerURL)
	opts.SetClientID(fmt.Sprintf("mqttfeed-%s", m.id))

	if m.username != "" {
		opts.SetUsername(m.username)
		opts.SetPassword(m.password)
	}

	opts.SetDefaultPublishHandler(func(client mqtt.Client, msg mqtt.Message) {
		log.Printf("Unexpected message: %s", msg.Topic())
	})

	opts.OnConnect = func(client mqtt.Client) {
		log.Printf("MQTT Feed Connector %s connected to %s", m.id, m.brokerURL)

		// Suscribirse al topic pattern
		token := client.Subscribe(m.topicPattern, m.qos, m.messageHandler)
		if token.Wait() && token.Error() != nil {
			log.Printf("Error subscribing to %s: %v", m.topicPattern, token.Error())
		} else {
			log.Printf("Subscribed to topic pattern: %s", m.topicPattern)
		}
	}

	opts.OnConnectionLost = func(client mqtt.Client, err error) {
		log.Printf("MQTT connection lost: %v", err)
		m.mu.Lock()
		m.errorCount++
		m.mu.Unlock()
	}

	m.client = mqtt.NewClient(opts)

	// Conectar al broker
	token := m.client.Connect()
	if token.Wait() && token.Error() != nil {
		return fmt.Errorf("failed to connect to MQTT broker: %w", token.Error())
	}

	m.running = true
	m.startTime = time.Now()

	// Monitorear el contexto para desconexión graceful
	go func() {
		<-ctx.Done()
		m.Stop()
	}()

	return nil
}

// Stop detiene el conector
func (m *MQTTFeedConnector) Stop() error {
	m.mu.Lock()
	defer m.mu.Unlock()

	if !m.running {
		return nil
	}

	if m.client != nil && m.client.IsConnected() {
		m.client.Disconnect(250)
	}

	m.running = false
	return nil
}

// Health retorna información de salud
func (m *MQTTFeedConnector) Health() connectors.HealthInfo {
	m.mu.RLock()
	defer m.mu.RUnlock()

	status := connectors.HealthStatusUnhealthy
	message := "Not running"

	if m.running && m.client != nil && m.client.IsConnected() {
		timeSinceLastMessage := time.Since(m.lastMessage)

		if timeSinceLastMessage < 5*time.Minute {
			status = connectors.HealthStatusHealthy
			message = "Connected and receiving messages"
		} else if timeSinceLastMessage < 15*time.Minute {
			status = connectors.HealthStatusDegraded
			message = "Connected but no recent messages"
		} else {
			status = connectors.HealthStatusUnhealthy
			message = "Connected but stale messages"
		}
	}

	var uptime time.Duration
	if !m.startTime.IsZero() {
		uptime = time.Since(m.startTime)
	}

	return connectors.HealthInfo{
		Status:     status,
		Message:    message,
		LastCheck:  time.Now(),
		ErrorCount: m.errorCount,
		Uptime:     uptime,
		Metrics: map[string]interface{}{
			"messages_processed": m.sequence,
			"last_latency_ms":    m.lastLatency.Milliseconds(),
			"connected":          m.client != nil && m.client.IsConnected(),
			"topic_pattern":      m.topicPattern,
		},
	}
}

// messageHandler maneja los mensajes MQTT recibidos
func (m *MQTTFeedConnector) messageHandler(client mqtt.Client, msg mqtt.Message) {
	startTime := time.Now()

	m.mu.Lock()
	m.sequence++
	seq := m.sequence
	eventChan := m.eventChan
	m.lastMessage = startTime
	m.mu.Unlock()

	if eventChan == nil {
		return
	}

	// Parsear el topic para extraer farm, site, cage
	topicParts := strings.Split(msg.Topic(), "/")
	if len(topicParts) < 4 {
		log.Printf("Invalid topic format: %s", msg.Topic())
		m.mu.Lock()
		m.errorCount++
		m.mu.Unlock()
		return
	}

	farmID := topicParts[0]
	siteID := topicParts[1]
	cageID := topicParts[2]
	// topicParts[3] should be "feeding"

	// Parsear el payload JSON
	var rawPayload map[string]interface{}
	if err := json.Unmarshal(msg.Payload(), &rawPayload); err != nil {
		log.Printf("Error parsing JSON payload: %v", err)
		m.mu.Lock()
		m.errorCount++
		m.mu.Unlock()
		return
	}

	// Aplicar mapping proveedor → canónico
	canonicalPayload, err := m.applyMapping(rawPayload)
	if err != nil {
		log.Printf("Error applying mapping: %v", err)
		m.mu.Lock()
		m.errorCount++
		m.mu.Unlock()
		return
	}

	// Validar contra schema feeding.v1
	_, err = schema.Validate("feeding", "v1", canonicalPayload)
	if err != nil {
		log.Printf("Schema validation failed: %v", err)
		m.mu.Lock()
		m.errorCount++
		m.mu.Unlock()
		return
	}

	// Convertir a JSON
	payloadBytes, err := json.Marshal(canonicalPayload)
	if err != nil {
		log.Printf("Error marshaling canonical payload: %v", err)
		m.mu.Lock()
		m.errorCount++
		m.mu.Unlock()
		return
	}

	// Crear StreamKey
	streamKey := domain.StreamKey{
		TenantID: m.tenantID,
		Kind:     domain.StreamKindFeeding,
		FarmID:   farmID,
		SiteID:   siteID,
		CageID:   &cageID,
	}

	// Crear envelope
	envelope := connectors.Envelope{
		Version:   "1.0",
		Timestamp: startTime,
		Stream:    streamKey,
		Source:    fmt.Sprintf("mqtt-feed-%s", m.id),
		Sequence:  seq,
		Flags:     connectors.EventFlagNone,
		TraceID:   fmt.Sprintf("mqtt-%d", seq),
	}

	// Crear evento canónico
	event := connectors.CanonicalEvent{
		Envelope:      envelope,
		Payload:       payloadBytes,
		Kind:          "feeding",
		SchemaVersion: "v1",
	}

	// Calcular latencia
	latency := time.Since(startTime)
	m.mu.Lock()
	m.lastLatency = latency
	m.mu.Unlock()

	// Enviar evento (no bloqueante)
	select {
	case eventChan <- event:
		// Evento enviado exitosamente
	default:
		// Canal lleno, incrementar contador de errores
		m.mu.Lock()
		m.errorCount++
		m.mu.Unlock()
	}
}

// applyMapping aplica las reglas de mapping a los datos del proveedor
func (m *MQTTFeedConnector) applyMapping(rawData map[string]interface{}) (map[string]interface{}, error) {
	if len(m.mappings) == 0 {
		// Sin mappings configurados, asumir que los datos ya están en formato canónico
		return rawData, nil
	}

	// Buscar mapping para feeding
	var feedingMapping *domain.Mapping
	for i := range m.mappings {
		if m.mappings[i].Capability == domain.CapabilityFeedingRead {
			feedingMapping = &m.mappings[i]
			break
		}
	}

	if feedingMapping == nil {
		return nil, fmt.Errorf("no feeding mapping configured")
	}

	result := make(map[string]interface{})

	// Aplicar reglas de mapping
	for _, rule := range feedingMapping.Rules {
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
				transformedValue, err := m.applyTransform(value, rule.Transform)
				if err != nil {
					return nil, fmt.Errorf("error applying transform to field %s: %w", rule.TargetField, err)
				}
				value = transformedValue
			}

			result[rule.TargetField] = value
		}
	}

	return result, nil
}

// applyTransform aplica una transformación específica a un valor
func (m *MQTTFeedConnector) applyTransform(value interface{}, transform *domain.Transform) (interface{}, error) {
	switch transform.Type {
	case domain.TransformTypeRename:
		// Rename no cambia el valor, solo el campo
		return value, nil

	case domain.TransformTypeUnit:
		// Conversión de unidades (implementación básica)
		if numVal, ok := value.(float64); ok {
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

	default:
		return value, nil
	}
}

// Factory para el conector MQTT Feed
func Factory(config map[string]interface{}) (connectors.Connector, error) {
	return NewMQTTFeedConnector(config)
}

// Registration contiene la información de registro del conector MQTT Feed
var Registration = &connectors.ConnectorRegistration{
	Type:        "mqttfeed",
	Version:     "1.0.0",
	Factory:     Factory,
	Description: "MQTT Feed connector for receiving feeding data",
	Capabilities: []domain.Capability{
		domain.CapabilityFeedingRead,
	},
	ConfigSchema: map[string]interface{}{
		"type":     "object",
		"required": []string{"broker_url"},
		"properties": map[string]interface{}{
			"broker_url": map[string]interface{}{
				"type":        "string",
				"description": "MQTT broker URL (tcp://host:port)",
			},
			"username": map[string]interface{}{
				"type":        "string",
				"description": "MQTT username (optional)",
			},
			"password": map[string]interface{}{
				"type":        "string",
				"description": "MQTT password (optional)",
			},
			"topic_pattern": map[string]interface{}{
				"type":        "string",
				"description": "MQTT topic pattern for subscription",
				"default":     "+/+/+/feeding",
			},
			"qos": map[string]interface{}{
				"type":        "integer",
				"description": "MQTT QoS level (0-2)",
				"default":     1,
				"minimum":     0,
				"maximum":     2,
			},
		},
	},
}
