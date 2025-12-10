package broker

import (
	"context"
	"fmt"
	"sync"
	"time"

	"omniapi/internal/database"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo/options"
)

// Manager gestiona múltiples publishers MQTT
type Manager struct {
	publishers map[string]*Publisher // Key: broker_id
	configs    map[string]*BrokerConfig
	mu         sync.RWMutex
	ctx        context.Context
	cancelFunc context.CancelFunc
}

// NewManager crea un nuevo manager de brokers
func NewManager() *Manager {
	return &Manager{
		publishers: make(map[string]*Publisher),
		configs:    make(map[string]*BrokerConfig),
	}
}

// Start inicia el manager y carga brokers desde MongoDB
func (m *Manager) Start(ctx context.Context) error {
	m.ctx, m.cancelFunc = context.WithCancel(ctx)

	// Cargar configuraciones desde MongoDB
	if err := m.loadConfigs(); err != nil {
		return fmt.Errorf("error loading broker configs: %w", err)
	}

	// Conectar a los brokers habilitados
	m.mu.RLock()
	for id, config := range m.configs {
		if config.Enabled {
			go func(brokerID string, cfg *BrokerConfig) {
				publisher := NewPublisher(cfg)
				if err := publisher.Connect(m.ctx); err != nil {
					fmt.Printf("⚠️ Error connecting to broker %s: %v\n", cfg.Name, err)
					return
				}
				m.mu.Lock()
				m.publishers[brokerID] = publisher
				m.mu.Unlock()
			}(id, config)
		}
	}
	m.mu.RUnlock()

	return nil
}

// Stop detiene el manager y desconecta todos los publishers
func (m *Manager) Stop() {
	if m.cancelFunc != nil {
		m.cancelFunc()
	}

	m.mu.Lock()
	defer m.mu.Unlock()

	for _, publisher := range m.publishers {
		publisher.Disconnect()
	}
	m.publishers = make(map[string]*Publisher)
	fmt.Println("🔌 Broker Manager stopped")
}

// GetPublisher obtiene un publisher por ID
func (m *Manager) GetPublisher(brokerID string) (*Publisher, bool) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	publisher, exists := m.publishers[brokerID]
	return publisher, exists
}

// Publish publica a un broker específico
func (m *Manager) Publish(brokerID, topic string, payload interface{}) PublishResult {
	publisher, exists := m.GetPublisher(brokerID)
	if !exists {
		return PublishResult{
			Topic:     topic,
			Error:     fmt.Sprintf("broker %s not found", brokerID),
			Timestamp: time.Now(),
		}
	}
	return publisher.Publish(topic, payload)
}

// PublishAsync publica de forma asíncrona a un broker específico
func (m *Manager) PublishAsync(brokerID, topic string, payload interface{}) {
	go m.Publish(brokerID, topic, payload)
}

// AddBroker agrega y guarda un nuevo broker
func (m *Manager) AddBroker(config *BrokerConfig) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	// Asignar ID si no tiene
	if config.ID == "" {
		config.ID = primitive.NewObjectID().Hex()
	}
	config.CreatedAt = time.Now()
	config.UpdatedAt = time.Now()

	// Guardar en MongoDB
	if err := m.saveBrokerConfig(config); err != nil {
		return err
	}

	m.configs[config.ID] = config

	// Si está habilitado, conectar
	if config.Enabled {
		publisher := NewPublisher(config)
		go func() {
			if err := publisher.Connect(m.ctx); err != nil {
				fmt.Printf("⚠️ Error connecting to broker %s: %v\n", config.Name, err)
				return
			}
			m.mu.Lock()
			m.publishers[config.ID] = publisher
			m.mu.Unlock()
		}()
	}

	return nil
}

// UpdateBroker actualiza la configuración de un broker
func (m *Manager) UpdateBroker(config *BrokerConfig) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	config.UpdatedAt = time.Now()

	// Desconectar el publisher existente
	if publisher, exists := m.publishers[config.ID]; exists {
		publisher.Disconnect()
		delete(m.publishers, config.ID)
	}

	// Guardar en MongoDB
	if err := m.saveBrokerConfig(config); err != nil {
		return err
	}

	m.configs[config.ID] = config

	// Si está habilitado, reconectar
	if config.Enabled {
		publisher := NewPublisher(config)
		go func() {
			if err := publisher.Connect(m.ctx); err != nil {
				fmt.Printf("⚠️ Error connecting to broker %s: %v\n", config.Name, err)
				return
			}
			m.mu.Lock()
			m.publishers[config.ID] = publisher
			m.mu.Unlock()
		}()
	}

	return nil
}

// RemoveBroker elimina un broker
func (m *Manager) RemoveBroker(brokerID string) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	// Desconectar si está conectado
	if publisher, exists := m.publishers[brokerID]; exists {
		publisher.Disconnect()
		delete(m.publishers, brokerID)
	}

	// Eliminar de MongoDB
	if err := m.deleteBrokerConfig(brokerID); err != nil {
		return err
	}

	delete(m.configs, brokerID)
	return nil
}

// GetBrokerConfigs retorna todas las configuraciones
func (m *Manager) GetBrokerConfigs() []BrokerConfig {
	m.mu.RLock()
	defer m.mu.RUnlock()

	configs := make([]BrokerConfig, 0, len(m.configs))
	for _, config := range m.configs {
		configs = append(configs, *config)
	}
	return configs
}

// GetBrokerConfig retorna una configuración específica
func (m *Manager) GetBrokerConfig(brokerID string) (*BrokerConfig, bool) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	config, exists := m.configs[brokerID]
	return config, exists
}

// GetStatus retorna el estado de todos los publishers
func (m *Manager) GetStatus() map[string]interface{} {
	m.mu.RLock()
	defer m.mu.RUnlock()

	status := map[string]interface{}{
		"total_brokers":   len(m.configs),
		"connected_count": 0,
		"brokers":         []map[string]interface{}{},
	}

	brokers := []map[string]interface{}{}
	connectedCount := 0

	for id, config := range m.configs {
		brokerStatus := map[string]interface{}{
			"id":        id,
			"name":      config.Name,
			"url":       config.BrokerURL,
			"enabled":   config.Enabled,
			"connected": false,
		}

		if publisher, exists := m.publishers[id]; exists {
			brokerStatus["connected"] = publisher.IsConnected()
			brokerStatus["stats"] = publisher.GetStats()
			if publisher.IsConnected() {
				connectedCount++
			}
		}

		brokers = append(brokers, brokerStatus)
	}

	status["connected_count"] = connectedCount
	status["brokers"] = brokers

	return status
}

// loadConfigs carga configuraciones desde MongoDB
func (m *Manager) loadConfigs() error {
	collection := database.GetCollection("broker_configs")
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	cursor, err := collection.Find(ctx, bson.M{})
	if err != nil {
		return err
	}
	defer cursor.Close(ctx)

	for cursor.Next(ctx) {
		var config BrokerConfig
		if err := cursor.Decode(&config); err != nil {
			continue
		}
		m.configs[config.ID] = &config
	}

	fmt.Printf("📡 Loaded %d broker configurations\n", len(m.configs))
	return nil
}

// saveBrokerConfig guarda/actualiza una configuración en MongoDB
func (m *Manager) saveBrokerConfig(config *BrokerConfig) error {
	collection := database.GetCollection("broker_configs")
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	filter := bson.M{"_id": config.ID}
	update := bson.M{"$set": config}
	opts := options.Update().SetUpsert(true)

	_, err := collection.UpdateOne(ctx, filter, update, opts)
	return err
}

// deleteBrokerConfig elimina una configuración de MongoDB
func (m *Manager) deleteBrokerConfig(brokerID string) error {
	collection := database.GetCollection("broker_configs")
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	_, err := collection.DeleteOne(ctx, bson.M{"_id": brokerID})
	return err
}

// TestConnection prueba la conexión a un broker sin guardarlo
func (m *Manager) TestConnection(config *BrokerConfig) error {
	publisher := NewPublisher(config)

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	if err := publisher.Connect(ctx); err != nil {
		return err
	}

	publisher.Disconnect()
	return nil
}
