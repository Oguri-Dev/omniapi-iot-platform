package dummy

import (
	"context"
	"testing"
	"time"

	"omniapi/internal/connectors"
	"omniapi/internal/domain"

	"go.mongodb.org/mongo-driver/bson/primitive"
)

func TestDummyConnector_Basic(t *testing.T) {
	tenantID := primitive.NewObjectID()

	config := map[string]interface{}{
		"__instance_id": "test-dummy-1",
		"__tenant_id":   tenantID.Hex(),
		"farm_id":       "test-farm",
		"site_id":       "test-site",
		"cage_id":       "test-cage",
	}

	connector, err := NewDummyConnector(config)
	if err != nil {
		t.Fatalf("Failed to create dummy connector: %v", err)
	}

	// Test basic properties
	if connector.ID() != "test-dummy-1" {
		t.Errorf("Expected ID 'test-dummy-1', got '%s'", connector.ID())
	}

	if connector.Type() != "dummy" {
		t.Errorf("Expected type 'dummy', got '%s'", connector.Type())
	}

	// Test capabilities
	caps := connector.Capabilities()
	expectedCaps := []domain.Capability{
		domain.CapabilityFeedingRead,
		domain.CapabilityBiometricRead,
		domain.CapabilityClimateRead,
	}

	if len(caps) != len(expectedCaps) {
		t.Errorf("Expected %d capabilities, got %d", len(expectedCaps), len(caps))
	}

	// Test health (should be unhealthy when not running)
	health := connector.Health()
	if health.Status != connectors.HealthStatusUnhealthy {
		t.Errorf("Expected unhealthy status when not running, got %s", health.Status)
	}
}

func TestDummyConnector_StartStop(t *testing.T) {
	tenantID := primitive.NewObjectID()

	config := map[string]interface{}{
		"__instance_id": "test-dummy-2",
		"__tenant_id":   tenantID.Hex(),
	}

	connector, err := NewDummyConnector(config)
	if err != nil {
		t.Fatalf("Failed to create dummy connector: %v", err)
	}

	// Create event channel
	eventChan := make(chan connectors.CanonicalEvent, 100)
	connector.OnEvent(eventChan)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Start connector
	err = connector.Start(ctx)
	if err != nil {
		t.Fatalf("Failed to start connector: %v", err)
	}

	// Check health after starting
	health := connector.Health()
	if health.Status != connectors.HealthStatusHealthy {
		t.Errorf("Expected healthy status when running, got %s", health.Status)
	}

	// Wait for some events
	time.Sleep(2500 * time.Millisecond) // Wait for at least 2 events

	// Stop connector
	err = connector.Stop()
	if err != nil {
		t.Errorf("Failed to stop connector: %v", err)
	}

	// Check health after stopping
	health = connector.Health()
	if health.Status != connectors.HealthStatusUnhealthy {
		t.Errorf("Expected unhealthy status when stopped, got %s", health.Status)
	}

	// Check that we received events
	eventCount := len(eventChan)
	if eventCount < 2 {
		t.Errorf("Expected at least 2 events, got %d", eventCount)
	}

	// Validate event structure
	if eventCount > 0 {
		event := <-eventChan

		if event.Envelope.Version != "1.0" {
			t.Errorf("Expected envelope version '1.0', got '%s'", event.Envelope.Version)
		}

		if event.Envelope.Stream.TenantID != tenantID {
			t.Error("Event tenant ID doesn't match")
		}

		if event.SchemaVersion != "v1" {
			t.Errorf("Expected schema version 'v1', got '%s'", event.SchemaVersion)
		}

		if event.Envelope.Flags != connectors.EventFlagSynthetic {
			t.Errorf("Expected synthetic flag, got %d", event.Envelope.Flags)
		}
	}
}

func TestDummyConnector_EventGeneration(t *testing.T) {
	tenantID := primitive.NewObjectID()

	config := map[string]interface{}{
		"__instance_id": "test-dummy-3",
		"__tenant_id":   tenantID.Hex(),
		"farm_id":       "test-farm-123",
		"site_id":       "test-site-456",
	}

	connector, err := NewDummyConnector(config)
	if err != nil {
		t.Fatalf("Failed to create dummy connector: %v", err)
	}

	eventChan := make(chan connectors.CanonicalEvent, 10)
	connector.OnEvent(eventChan)

	ctx, cancel := context.WithTimeout(context.Background(), 4*time.Second)
	defer cancel()

	err = connector.Start(ctx)
	if err != nil {
		t.Fatalf("Failed to start connector: %v", err)
	}
	defer connector.Stop()

	// Collect events for analysis
	var events []connectors.CanonicalEvent
	timeout := time.After(3500 * time.Millisecond)

collectLoop:
	for {
		select {
		case event := <-eventChan:
			events = append(events, event)
		case <-timeout:
			break collectLoop
		}
	}

	if len(events) < 3 {
		t.Errorf("Expected at least 3 events, got %d", len(events))
	}

	// Check event types rotation
	eventKinds := make(map[string]bool)
	for _, event := range events {
		eventKinds[event.Kind] = true

		// Validate common fields
		if event.Envelope.Stream.FarmID != "test-farm-123" {
			t.Errorf("Expected farm ID 'test-farm-123', got '%s'", event.Envelope.Stream.FarmID)
		}

		if event.Envelope.Stream.SiteID != "test-site-456" {
			t.Errorf("Expected site ID 'test-site-456', got '%s'", event.Envelope.Stream.SiteID)
		}

		// Validate payload is valid JSON
		if len(event.Payload) == 0 {
			t.Error("Event payload is empty")
		}
	}

	// Should have multiple event types
	expectedKinds := []string{"feeding", "biometric", "climate"}
	for _, kind := range expectedKinds {
		if !eventKinds[kind] {
			t.Errorf("Expected to see events of kind '%s'", kind)
		}
	}
}

func TestDummyConnector_Factory(t *testing.T) {
	tenantID := primitive.NewObjectID()

	config := map[string]interface{}{
		"__instance_id": "factory-test",
		"__tenant_id":   tenantID.Hex(),
	}

	connector, err := Factory(config)
	if err != nil {
		t.Fatalf("Factory failed: %v", err)
	}

	if connector == nil {
		t.Fatal("Factory returned nil connector")
	}

	if connector.Type() != "dummy" {
		t.Errorf("Expected type 'dummy', got '%s'", connector.Type())
	}
}

func TestDummyConnector_Registration(t *testing.T) {
	if Registration == nil {
		t.Fatal("Registration is nil")
	}

	if Registration.Type != "dummy" {
		t.Errorf("Expected type 'dummy', got '%s'", Registration.Type)
	}

	if Registration.Factory == nil {
		t.Error("Factory is nil in registration")
	}

	if len(Registration.Capabilities) == 0 {
		t.Error("Registration has no capabilities")
	}

	expectedCaps := []domain.Capability{
		domain.CapabilityFeedingRead,
		domain.CapabilityBiometricRead,
		domain.CapabilityClimateRead,
	}

	if len(Registration.Capabilities) != len(expectedCaps) {
		t.Errorf("Expected %d capabilities, got %d", len(expectedCaps), len(Registration.Capabilities))
	}
}

func TestDummyConnector_Subscribe(t *testing.T) {
	tenantID := primitive.NewObjectID()

	config := map[string]interface{}{
		"__instance_id": "subscribe-test",
		"__tenant_id":   tenantID.Hex(),
	}

	connector, err := NewDummyConnector(config)
	if err != nil {
		t.Fatalf("Failed to create dummy connector: %v", err)
	}

	// Test subscription
	filters := []connectors.EventFilter{
		{
			Capabilities: []domain.Capability{domain.CapabilityFeedingRead},
		},
	}

	err = connector.Subscribe(filters...)
	if err != nil {
		t.Errorf("Failed to subscribe: %v", err)
	}
}
