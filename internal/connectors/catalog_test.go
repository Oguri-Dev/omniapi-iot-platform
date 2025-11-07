package connectors

import (
	"context"
	"fmt"
	"testing"
	"time"

	"omniapi/internal/domain"

	"go.mongodb.org/mongo-driver/bson/primitive"
)

// MockConnector implementa la interfaz Connector para tests
type MockConnector struct {
	id           string
	connType     string
	capabilities []domain.Capability
	config       map[string]interface{}
	running      bool
	eventChan    chan<- CanonicalEvent
	health       HealthInfo
}

func NewMockConnector(id, connType string, capabilities []domain.Capability) *MockConnector {
	return &MockConnector{
		id:           id,
		connType:     connType,
		capabilities: capabilities,
		config:       make(map[string]interface{}),
		health: HealthInfo{
			Status:    HealthStatusHealthy,
			LastCheck: time.Now(),
		},
	}
}

func (m *MockConnector) Start(ctx context.Context) error {
	m.running = true
	return nil
}

func (m *MockConnector) Stop() error {
	m.running = false
	return nil
}

func (m *MockConnector) Capabilities() []domain.Capability {
	return m.capabilities
}

func (m *MockConnector) Subscribe(filters ...EventFilter) error {
	return nil
}

func (m *MockConnector) OnEvent(eventChan chan<- CanonicalEvent) {
	m.eventChan = eventChan
}

func (m *MockConnector) Health() HealthInfo {
	return m.health
}

func (m *MockConnector) ID() string {
	return m.id
}

func (m *MockConnector) Type() string {
	return m.connType
}

func (m *MockConnector) Config() map[string]interface{} {
	return m.config
}

// Factory para el mock connector
func mockFactory(config map[string]interface{}) (Connector, error) {
	id := "mock-1"
	if val, exists := config["__instance_id"]; exists {
		if str, ok := val.(string); ok {
			id = str
		}
	}

	return NewMockConnector(id, "mock", []domain.Capability{domain.CapabilityFeedingRead}), nil
}

func TestCatalog_Register(t *testing.T) {
	catalog := NewCatalog()

	registration := &ConnectorRegistration{
		Type:         "test-connector",
		Version:      "1.0.0",
		Factory:      mockFactory,
		Capabilities: []domain.Capability{domain.CapabilityFeedingRead},
		Description:  "Test connector",
	}

	// Test successful registration
	err := catalog.Register(registration)
	if err != nil {
		t.Errorf("Failed to register connector: %v", err)
	}

	// Test duplicate registration
	err = catalog.Register(registration)
	if err == nil {
		t.Error("Expected error when registering duplicate connector type")
	}

	// Test nil registration
	err = catalog.Register(nil)
	if err == nil {
		t.Error("Expected error when registering nil registration")
	}

	// Test empty type
	invalidReg := &ConnectorRegistration{
		Type:    "",
		Factory: mockFactory,
	}
	err = catalog.Register(invalidReg)
	if err == nil {
		t.Error("Expected error when registering connector with empty type")
	}
}

func TestCatalog_GetRegistration(t *testing.T) {
	catalog := NewCatalog()

	registration := &ConnectorRegistration{
		Type:         "test-connector",
		Version:      "1.0.0",
		Factory:      mockFactory,
		Capabilities: []domain.Capability{domain.CapabilityFeedingRead},
		Description:  "Test connector",
	}

	// Register connector
	err := catalog.Register(registration)
	if err != nil {
		t.Fatalf("Failed to register connector: %v", err)
	}

	// Test get existing registration
	reg, exists := catalog.GetRegistration("test-connector")
	if !exists {
		t.Error("Expected to find registered connector")
	}
	if reg.Type != "test-connector" {
		t.Errorf("Expected type 'test-connector', got '%s'", reg.Type)
	}

	// Test get non-existing registration
	_, exists = catalog.GetRegistration("non-existing")
	if exists {
		t.Error("Expected not to find non-existing connector")
	}
}

func TestCatalog_CreateInstance(t *testing.T) {
	catalog := NewCatalog()

	// Register connector type
	registration := &ConnectorRegistration{
		Type:         "test-connector",
		Version:      "1.0.0",
		Factory:      mockFactory,
		Capabilities: []domain.Capability{domain.CapabilityFeedingRead},
		Description:  "Test connector",
	}

	err := catalog.Register(registration)
	if err != nil {
		t.Fatalf("Failed to register connector: %v", err)
	}

	// Create test data
	tenantID := primitive.NewObjectID()
	typeID := primitive.NewObjectID()

	connectionInstance := domain.NewConnectionInstance(tenantID, typeID, "Test Connection", "admin")
	connectionInstance.Config = map[string]interface{}{
		"test_param": "test_value",
	}

	connectorType := domain.NewConnectorType("test-connector", "Test Connector", "Test Description", "1.0.0", "admin")

	// Test successful instance creation
	instance, err := catalog.CreateInstance(connectionInstance, connectorType)
	if err != nil {
		t.Errorf("Failed to create instance: %v", err)
	}
	if instance == nil {
		t.Error("Expected non-nil instance")
	}

	// Verify instance is registered
	instanceID := connectionInstance.ID.Hex()
	retrievedInstance, exists := catalog.GetInstance(instanceID)
	if !exists {
		t.Error("Expected to find created instance in catalog")
	}
	if retrievedInstance.ID() != instanceID {
		t.Errorf("Expected instance ID '%s', got '%s'", instanceID, retrievedInstance.ID())
	}

	// Test creating instance for non-registered type
	unknownType := domain.NewConnectorType("unknown-type", "Unknown", "Unknown", "1.0.0", "admin")
	_, err = catalog.CreateInstance(connectionInstance, unknownType)
	if err == nil {
		t.Error("Expected error when creating instance for unknown connector type")
	}
}

func TestCatalog_ListRegistrations(t *testing.T) {
	catalog := NewCatalog()

	// Initially should be empty
	registrations := catalog.ListRegistrations()
	if len(registrations) != 0 {
		t.Errorf("Expected 0 registrations initially, got %d", len(registrations))
	}

	// Register some connectors
	for i := 0; i < 3; i++ {
		registration := &ConnectorRegistration{
			Type:         fmt.Sprintf("test-connector-%d", i),
			Version:      "1.0.0",
			Factory:      mockFactory,
			Capabilities: []domain.Capability{domain.CapabilityFeedingRead},
		}

		err := catalog.Register(registration)
		if err != nil {
			t.Fatalf("Failed to register connector %d: %v", i, err)
		}
	}

	// Check list
	registrations = catalog.ListRegistrations()
	if len(registrations) != 3 {
		t.Errorf("Expected 3 registrations, got %d", len(registrations))
	}
}

func TestCatalog_RemoveInstance(t *testing.T) {
	catalog := NewCatalog()

	// Register connector type
	registration := &ConnectorRegistration{
		Type:         "test-connector",
		Version:      "1.0.0",
		Factory:      mockFactory,
		Capabilities: []domain.Capability{domain.CapabilityFeedingRead},
	}

	err := catalog.Register(registration)
	if err != nil {
		t.Fatalf("Failed to register connector: %v", err)
	}

	// Create instance
	tenantID := primitive.NewObjectID()
	typeID := primitive.NewObjectID()

	connectionInstance := domain.NewConnectionInstance(tenantID, typeID, "Test Connection", "admin")
	connectorType := domain.NewConnectorType("test-connector", "Test Connector", "Test", "1.0.0", "admin")

	_, err = catalog.CreateInstance(connectionInstance, connectorType)
	if err != nil {
		t.Fatalf("Failed to create instance: %v", err)
	}

	instanceID := connectionInstance.ID.Hex()

	// Verify instance exists
	_, exists := catalog.GetInstance(instanceID)
	if !exists {
		t.Error("Expected to find instance before removal")
	}

	// Remove instance
	err = catalog.RemoveInstance(instanceID)
	if err != nil {
		t.Errorf("Failed to remove instance: %v", err)
	}

	// Verify instance is gone
	_, exists = catalog.GetInstance(instanceID)
	if exists {
		t.Error("Expected instance to be removed")
	}

	// Test removing non-existing instance
	err = catalog.RemoveInstance("non-existing")
	if err == nil {
		t.Error("Expected error when removing non-existing instance")
	}
}
