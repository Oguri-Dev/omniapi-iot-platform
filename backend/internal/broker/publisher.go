package broker

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"sync"
	"time"

	mqtt "github.com/eclipse/paho.mqtt.golang"
)

// BrokerConfig configuración de conexión al broker MQTT
type BrokerConfig struct {
	ID           string    `bson:"_id,omitempty" json:"id,omitempty"`
	Name         string    `bson:"name" json:"name"`             // Nombre descriptivo
	BrokerURL    string    `bson:"broker_url" json:"broker_url"` // tcp://host:port o ssl://host:port
	ClientID     string    `bson:"client_id" json:"client_id"`   // ID único del cliente
	Username     string    `bson:"username,omitempty" json:"username,omitempty"`
	Password     string    `bson:"password,omitempty" json:"password,omitempty"`
	QoS          byte      `bson:"qos" json:"qos"`           // 0, 1, 2
	Retained     bool      `bson:"retained" json:"retained"` // Si los mensajes deben ser retained
	CleanSession bool      `bson:"clean_session" json:"clean_session"`
	KeepAlive    int       `bson:"keep_alive" json:"keep_alive"` // Segundos
	Enabled      bool      `bson:"enabled" json:"enabled"`
	CreatedAt    time.Time `bson:"created_at" json:"created_at"`
	UpdatedAt    time.Time `bson:"updated_at" json:"updated_at"`
}

// TopicTemplate template para generar topics dinámicamente
type TopicTemplate struct {
	ID          string    `bson:"_id,omitempty" json:"id,omitempty"`
	Name        string    `bson:"name" json:"name"`
	Pattern     string    `bson:"pattern" json:"pattern"` // ej: "omniapi/{tenant}/{site}/{provider}/{endpoint}"
	Description string    `bson:"description" json:"description"`
	CreatedAt   time.Time `bson:"created_at" json:"created_at"`
}

// PublishMessage mensaje a publicar
type PublishMessage struct {
	Topic     string      `json:"topic"`
	Payload   interface{} `json:"payload"`
	QoS       byte        `json:"qos"`
	Retained  bool        `json:"retained"`
	Timestamp time.Time   `json:"timestamp"`
}

// PublishResult resultado de publicación
type PublishResult struct {
	Success   bool      `json:"success"`
	Topic     string    `json:"topic"`
	MessageID uint16    `json:"message_id,omitempty"`
	Error     string    `json:"error,omitempty"`
	Timestamp time.Time `json:"timestamp"`
}

// Publisher cliente MQTT para publicar mensajes
type Publisher struct {
	config     *BrokerConfig
	client     mqtt.Client
	mu         sync.RWMutex
	connected  bool
	onPublish  func(PublishResult)
	stats      PublisherStats
	ctx        context.Context
	cancelFunc context.CancelFunc
}

// PublisherStats estadísticas del publisher
type PublisherStats struct {
	TotalPublished int64     `json:"total_published"`
	TotalSuccess   int64     `json:"total_success"`
	TotalErrors    int64     `json:"total_errors"`
	LastPublishAt  time.Time `json:"last_publish_at,omitempty"`
	LastErrorAt    time.Time `json:"last_error_at,omitempty"`
	LastError      string    `json:"last_error,omitempty"`
	ConnectedSince time.Time `json:"connected_since,omitempty"`
	ReconnectCount int64     `json:"reconnect_count"`
}

// NewPublisher crea un nuevo publisher MQTT
func NewPublisher(config *BrokerConfig) *Publisher {
	return &Publisher{
		config: config,
		stats:  PublisherStats{},
	}
}

// Connect conecta al broker MQTT
func (p *Publisher) Connect(ctx context.Context) error {
	p.mu.Lock()
	defer p.mu.Unlock()

	if p.connected {
		return nil
	}

	p.ctx, p.cancelFunc = context.WithCancel(ctx)

	opts := mqtt.NewClientOptions()
	opts.AddBroker(p.config.BrokerURL)
	opts.SetClientID(p.config.ClientID)

	if p.config.Username != "" {
		opts.SetUsername(p.config.Username)
		opts.SetPassword(p.config.Password)
	}

	opts.SetCleanSession(p.config.CleanSession)
	opts.SetKeepAlive(time.Duration(p.config.KeepAlive) * time.Second)
	opts.SetAutoReconnect(true)
	opts.SetMaxReconnectInterval(30 * time.Second)

	// Callbacks de conexión
	opts.SetOnConnectHandler(func(client mqtt.Client) {
		p.mu.Lock()
		p.connected = true
		p.stats.ConnectedSince = time.Now()
		p.mu.Unlock()
		fmt.Printf("📡 MQTT Publisher connected to %s\n", p.config.BrokerURL)
	})

	opts.SetConnectionLostHandler(func(client mqtt.Client, err error) {
		p.mu.Lock()
		p.connected = false
		p.stats.LastError = err.Error()
		p.stats.LastErrorAt = time.Now()
		p.mu.Unlock()
		fmt.Printf("⚠️ MQTT Publisher disconnected: %v\n", err)
	})

	opts.SetReconnectingHandler(func(client mqtt.Client, opts *mqtt.ClientOptions) {
		p.mu.Lock()
		p.stats.ReconnectCount++
		p.mu.Unlock()
		fmt.Printf("🔄 MQTT Publisher reconnecting...\n")
	})

	p.client = mqtt.NewClient(opts)

	token := p.client.Connect()
	if token.WaitTimeout(10 * time.Second) {
		if token.Error() != nil {
			return fmt.Errorf("error connecting to broker: %w", token.Error())
		}
	} else {
		return fmt.Errorf("timeout connecting to broker")
	}

	p.connected = true
	p.stats.ConnectedSince = time.Now()
	fmt.Printf("✅ MQTT Publisher connected to %s as %s\n", p.config.BrokerURL, p.config.ClientID)

	return nil
}

// Disconnect desconecta del broker
func (p *Publisher) Disconnect() {
	p.mu.Lock()
	defer p.mu.Unlock()

	if p.cancelFunc != nil {
		p.cancelFunc()
	}

	if p.client != nil && p.client.IsConnected() {
		p.client.Disconnect(1000)
		fmt.Printf("🔌 MQTT Publisher disconnected from %s\n", p.config.BrokerURL)
	}
	p.connected = false
}

// Publish publica un mensaje al broker
func (p *Publisher) Publish(topic string, payload interface{}) PublishResult {
	result := PublishResult{
		Topic:     topic,
		Timestamp: time.Now(),
	}

	p.mu.RLock()
	if !p.connected || p.client == nil {
		p.mu.RUnlock()
		result.Error = "not connected to broker"
		p.recordError(result.Error)
		return result
	}
	p.mu.RUnlock()

	// Serializar payload
	var payloadBytes []byte
	var err error

	switch v := payload.(type) {
	case []byte:
		payloadBytes = v
	case string:
		payloadBytes = []byte(v)
	default:
		payloadBytes, err = json.Marshal(payload)
		if err != nil {
			result.Error = fmt.Sprintf("error serializing payload: %v", err)
			p.recordError(result.Error)
			return result
		}
	}

	// Publicar
	token := p.client.Publish(topic, p.config.QoS, p.config.Retained, payloadBytes)
	if token.WaitTimeout(5 * time.Second) {
		if token.Error() != nil {
			result.Error = token.Error().Error()
			p.recordError(result.Error)
			return result
		}
	} else {
		result.Error = "timeout publishing message"
		p.recordError(result.Error)
		return result
	}

	result.Success = true
	p.recordSuccess()

	// Callback si está registrado
	if p.onPublish != nil {
		p.onPublish(result)
	}

	return result
}

// PublishAsync publica de forma asíncrona
func (p *Publisher) PublishAsync(topic string, payload interface{}) {
	go func() {
		p.Publish(topic, payload)
	}()
}

// OnPublish registra callback para resultados de publicación
func (p *Publisher) OnPublish(callback func(PublishResult)) {
	p.onPublish = callback
}

// IsConnected retorna si está conectado
func (p *Publisher) IsConnected() bool {
	p.mu.RLock()
	defer p.mu.RUnlock()
	return p.connected
}

// GetStats retorna estadísticas
func (p *Publisher) GetStats() PublisherStats {
	p.mu.RLock()
	defer p.mu.RUnlock()
	return p.stats
}

// GetConfig retorna la configuración
func (p *Publisher) GetConfig() *BrokerConfig {
	return p.config
}

func (p *Publisher) recordSuccess() {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.stats.TotalPublished++
	p.stats.TotalSuccess++
	p.stats.LastPublishAt = time.Now()
}

func (p *Publisher) recordError(errMsg string) {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.stats.TotalPublished++
	p.stats.TotalErrors++
	p.stats.LastError = errMsg
	p.stats.LastErrorAt = time.Now()
}

// BuildTopic genera un topic basado en un template y variables
// Template: "omniapi/{tenant}/{site}/{provider}/{endpoint}"
// Variables: map[string]string{"tenant": "acme", "site": "farm1", ...}
func BuildTopic(template string, vars map[string]string) string {
	result := template
	for key, value := range vars {
		placeholder := "{" + key + "}"
		// Sanitizar el valor (reemplazar espacios y caracteres especiales)
		sanitized := strings.ReplaceAll(value, " ", "_")
		sanitized = strings.ReplaceAll(sanitized, "/", "_")
		sanitized = strings.ToLower(sanitized)
		result = strings.ReplaceAll(result, placeholder, sanitized)
	}
	return result
}

// GetTopicPattern obtiene el pattern de un template por su ID
// Si el input ya es un pattern (contiene "{"), lo retorna tal cual
func GetTopicPattern(templateIDOrPattern string) string {
	// Si ya es un pattern, retornarlo
	if strings.Contains(templateIDOrPattern, "{") {
		return templateIDOrPattern
	}

	// Buscar en los templates predefinidos
	for _, t := range DefaultTopicTemplates {
		if t.Name == templateIDOrPattern {
			return t.Pattern
		}
	}

	// Si no se encuentra, usar el standard por defecto
	return "omniapi/{tenant}/{site}/{provider}/{endpoint}"
}

// DefaultTopicTemplates templates predefinidos
var DefaultTopicTemplates = []TopicTemplate{
	{
		Name:        "standard",
		Pattern:     "omniapi/{tenant}/{site}/{provider}/{endpoint}",
		Description: "Topic estándar: omniapi/tenant/site/provider/endpoint",
	},
	{
		Name:        "by-provider",
		Pattern:     "omniapi/{provider}/{tenant}/{site}/{endpoint}",
		Description: "Agrupado por proveedor: omniapi/provider/tenant/site/endpoint",
	},
	{
		Name:        "by-data-type",
		Pattern:     "omniapi/{target_block}/{tenant}/{site}/{provider}",
		Description: "Agrupado por tipo de dato: omniapi/snapshots|timeseries|kpis/...",
	},
	{
		Name:        "flat",
		Pattern:     "omniapi/data/{instance_id}",
		Description: "Topic plano por instancia",
	},
}
