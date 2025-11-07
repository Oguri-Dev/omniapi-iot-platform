package integration

import (
	"context"
	"fmt"
	"testing"
	"time"

	"omniapi/adapters/dummy"
	"omniapi/internal/connectors"
	"omniapi/internal/domain"

	"go.mongodb.org/mongo-driver/bson/primitive"
)

func TestCatalog_Integration(t *testing.T) {
	// Crear un catálogo nuevo para el test
	catalog := connectors.NewCatalog()

	// Registrar el conector dummy
	err := catalog.Register(dummy.Registration)
	if err != nil {
		t.Fatalf("Failed to register dummy connector: %v", err)
	}

	// Verificar que se registró correctamente
	reg, exists := catalog.GetRegistration("dummy")
	if !exists {
		t.Fatal("Dummy connector not found in catalog")
	}

	if reg.Type != "dummy" {
		t.Errorf("Expected type 'dummy', got '%s'", reg.Type)
	}

	// Crear una ConnectionInstance
	tenantID := primitive.NewObjectID()
	typeID := primitive.NewObjectID()

	connectionInstance := domain.NewConnectionInstance(tenantID, typeID, "Test Dummy Connection", "admin")
	connectionInstance.Config = map[string]interface{}{
		"farm_id": "integration-farm",
		"site_id": "integration-site",
		"cage_id": "integration-cage",
	}

	// Crear un ConnectorType
	connectorType := domain.NewConnectorType("dummy", "Dummy Connector", "Integration test dummy", "1.0.0", "admin")
	connectorType.Capabilities = []domain.Capability{
		domain.CapabilityFeedingRead,
		domain.CapabilityBiometricRead,
		domain.CapabilityClimateRead,
	}

	// Crear instancia usando el catálogo
	instance, err := catalog.CreateInstance(connectionInstance, connectorType)
	if err != nil {
		t.Fatalf("Failed to create instance: %v", err)
	}

	if instance == nil {
		t.Fatal("Created instance is nil")
	}

	// Verificar propiedades de la instancia
	if instance.Type() != "dummy" {
		t.Errorf("Expected instance type 'dummy', got '%s'", instance.Type())
	}

	expectedID := connectionInstance.ID.Hex()
	if instance.ID() != expectedID {
		t.Errorf("Expected instance ID '%s', got '%s'", expectedID, instance.ID())
	}

	// Verificar capabilities
	caps := instance.Capabilities()
	if len(caps) != 3 {
		t.Errorf("Expected 3 capabilities, got %d", len(caps))
	}

	// Configurar canal de eventos
	eventChan := make(chan connectors.CanonicalEvent, 50)
	instance.OnEvent(eventChan)

	// Iniciar la instancia
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	err = catalog.StartInstance(expectedID, ctx)
	if err != nil {
		t.Fatalf("Failed to start instance: %v", err)
	}

	// Esperar algunos eventos
	time.Sleep(2500 * time.Millisecond)

	// Detener la instancia
	err = catalog.StopInstance(expectedID)
	if err != nil {
		t.Errorf("Failed to stop instance: %v", err)
	}

	// Verificar que recibimos eventos
	eventCount := len(eventChan)
	if eventCount < 2 {
		t.Errorf("Expected at least 2 events, got %d", eventCount)
	}

	// Validar un evento
	if eventCount > 0 {
		event := <-eventChan

		if event.Envelope.Stream.TenantID != tenantID {
			t.Error("Event tenant ID doesn't match")
		}

		if event.Envelope.Stream.FarmID != "integration-farm" {
			t.Errorf("Expected farm ID 'integration-farm', got '%s'", event.Envelope.Stream.FarmID)
		}

		if event.Envelope.Stream.SiteID != "integration-site" {
			t.Errorf("Expected site ID 'integration-site', got '%s'", event.Envelope.Stream.SiteID)
		}
	}

	// Verificar estado de salud
	healthMap := catalog.GetHealthStatus()
	if len(healthMap) != 1 {
		t.Errorf("Expected 1 health status, got %d", len(healthMap))
	}

	if health, exists := healthMap[expectedID]; exists {
		if health.Status != connectors.HealthStatusUnhealthy { // Debería estar unhealthy después de detener
			t.Errorf("Expected unhealthy status after stop, got %s", health.Status)
		}
	} else {
		t.Error("Health status not found for instance")
	}

	// Remover la instancia
	err = catalog.RemoveInstance(expectedID)
	if err != nil {
		t.Errorf("Failed to remove instance: %v", err)
	}

	// Verificar que se removió
	_, exists = catalog.GetInstance(expectedID)
	if exists {
		t.Error("Instance should have been removed from catalog")
	}
}

func TestCatalog_MultipleInstances(t *testing.T) {
	catalog := connectors.NewCatalog()

	// Registrar el conector dummy
	err := catalog.Register(dummy.Registration)
	if err != nil {
		t.Fatalf("Failed to register dummy connector: %v", err)
	}

	// Crear múltiples instancias
	var instances []connectors.Connector
	var instanceIDs []string

	for i := 0; i < 3; i++ {
		tenantID := primitive.NewObjectID()
		typeID := primitive.NewObjectID()

		connectionInstance := domain.NewConnectionInstance(tenantID, typeID,
			fmt.Sprintf("Test Connection %d", i), "admin")

		connectorType := domain.NewConnectorType("dummy", "Dummy Connector", "Test", "1.0.0", "admin")

		instance, err := catalog.CreateInstance(connectionInstance, connectorType)
		if err != nil {
			t.Fatalf("Failed to create instance %d: %v", i, err)
		}

		instances = append(instances, instance)
		instanceIDs = append(instanceIDs, connectionInstance.ID.Hex())
	}

	// Verificar que todas las instancias están en el catálogo
	catalogInstances := catalog.ListInstances()
	if len(catalogInstances) != 3 {
		t.Errorf("Expected 3 instances in catalog, got %d", len(catalogInstances))
	}

	// Verificar que se pueden obtener individualmente
	for i, id := range instanceIDs {
		instance, exists := catalog.GetInstance(id)
		if !exists {
			t.Errorf("Instance %d not found in catalog", i)
			continue
		}

		if instance.ID() != id {
			t.Errorf("Instance %d ID mismatch: expected '%s', got '%s'", i, id, instance.ID())
		}
	}

	// Limpiar - remover todas las instancias
	for i, id := range instanceIDs {
		err := catalog.RemoveInstance(id)
		if err != nil {
			t.Errorf("Failed to remove instance %d: %v", i, err)
		}
	}

	// Verificar que el catálogo está vacío
	catalogInstances = catalog.ListInstances()
	if len(catalogInstances) != 0 {
		t.Errorf("Expected empty catalog after cleanup, got %d instances", len(catalogInstances))
	}
}

func TestCatalog_CanCreateInstance(t *testing.T) {
	catalog := connectors.NewCatalog()

	// Inicialmente no debería poder crear instancias
	if catalog.CanCreateInstance("dummy") {
		t.Error("Should not be able to create dummy instance before registration")
	}

	// Registrar el conector
	err := catalog.Register(dummy.Registration)
	if err != nil {
		t.Fatalf("Failed to register dummy connector: %v", err)
	}

	// Ahora debería poder crear instancias
	if !catalog.CanCreateInstance("dummy") {
		t.Error("Should be able to create dummy instance after registration")
	}

	// Tipos no registrados no deberían poder crearse
	if catalog.CanCreateInstance("non-existent") {
		t.Error("Should not be able to create non-existent connector type")
	}
}

func TestGlobalCatalog(t *testing.T) {
	// Verificar que el catálogo global existe
	if connectors.GlobalCatalog == nil {
		t.Fatal("GlobalCatalog is nil")
	}

	// Test de las funciones de conveniencia
	err := connectors.RegisterConnector(dummy.Registration)
	if err != nil {
		// Puede fallar si ya está registrado, eso está bien
		t.Logf("Registration returned error (may be already registered): %v", err)
	}

	// Crear datos de test
	tenantID := primitive.NewObjectID()
	typeID := primitive.NewObjectID()

	connectionInstance := domain.NewConnectionInstance(tenantID, typeID, "Global Test", "admin")
	connectorType := domain.NewConnectorType("dummy", "Dummy Connector", "Global test", "1.0.0", "admin")

	// Test función de conveniencia para crear instancias
	instance, err := connectors.CreateConnectorInstance(connectionInstance, connectorType)
	if err != nil {
		t.Fatalf("Failed to create instance using global catalog: %v", err)
	}

	if instance == nil {
		t.Fatal("Global catalog returned nil instance")
	}

	// Limpiar
	err = connectors.GlobalCatalog.RemoveInstance(connectionInstance.ID.Hex())
	if err != nil {
		t.Logf("Failed to cleanup global instance: %v", err)
	}
}
