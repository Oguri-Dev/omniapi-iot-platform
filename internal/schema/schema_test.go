package schema

import (
	"os"
	"path/filepath"
	"testing"
)

func TestSchemaManager_LoadSchemas(t *testing.T) {
	// Verificar que existe el directorio de schemas
	schemasDir := "../../configs/schemas"
	if _, err := os.Stat(schemasDir); os.IsNotExist(err) {
		t.Skip("Schemas directory not found, skipping test")
	}

	// Usar el directorio real de schemas para el test
	manager := NewSchemaManager("../../configs/schemas")

	err := manager.LoadSchemas()
	if err != nil {
		t.Fatalf("Failed to load schemas: %v", err)
	}

	// Verificar que se cargaron los schemas esperados
	schemas := manager.ListSchemas()
	expectedSchemas := []string{"feeding.v1", "biometric.v1", "climate.v1"}

	for _, expected := range expectedSchemas {
		if _, exists := schemas[expected]; !exists {
			t.Errorf("Expected schema %s not found", expected)
		}
	}
}

func TestSchemaManager_GetSchema(t *testing.T) {
	manager := NewSchemaManager("../../configs/schemas")
	err := manager.LoadSchemas()
	if err != nil {
		t.Fatalf("Failed to load schemas: %v", err)
	}

	// Test obtener schema válido
	schema, err := manager.GetSchema("feeding", "v1")
	if err != nil {
		t.Errorf("Failed to get feeding.v1 schema: %v", err)
	}

	if schema.Kind != "feeding" || schema.Version != "v1" {
		t.Errorf("Schema kind/version mismatch: got %s.%s, want feeding.v1", schema.Kind, schema.Version)
	}

	// Test schema no existente
	_, err = manager.GetSchema("nonexistent", "v1")
	if err == nil {
		t.Error("Expected error for non-existent schema")
	}
}

func TestValidate_FeedingSchema(t *testing.T) {
	manager := NewSchemaManager("../../configs/schemas")
	err := manager.LoadSchemas()
	if err != nil {
		t.Fatalf("Failed to load schemas: %v", err)
	}

	// Test payload válido
	validPayload := map[string]interface{}{
		"timestamp": "2025-11-07T12:00:00Z",
		"device_id": "feeder_001",
		"feed_type": "pellets",
		"quantity":  150.5,
		"status":    "completed",
		"cage_id":   "cage_01",
	}

	result, err := manager.Validate("feeding", "v1", validPayload)
	if err != nil {
		t.Fatalf("Validation failed: %v", err)
	}

	if !result.Valid {
		t.Errorf("Valid payload should pass validation. Errors: %v", result.Errors)
	}

	// Test payload inválido - falta campo requerido
	invalidPayload := map[string]interface{}{
		"device_id": "feeder_001",
		"feed_type": "pellets",
		// Falta timestamp, quantity, status (campos requeridos)
	}

	result, err = manager.Validate("feeding", "v1", invalidPayload)
	if err != nil {
		t.Fatalf("Validation failed: %v", err)
	}

	if result.Valid {
		t.Error("Invalid payload should fail validation")
	}

	if len(result.Errors) == 0 {
		t.Error("Expected validation errors for invalid payload")
	}

	// Test payload inválido - tipo incorrecto
	invalidTypePayload := map[string]interface{}{
		"timestamp": "2025-11-07T12:00:00Z",
		"device_id": "feeder_001",
		"feed_type": "invalid_type", // No está en el enum
		"quantity":  "not_a_number", // Debería ser number
		"status":    "completed",
	}

	result, err = manager.Validate("feeding", "v1", invalidTypePayload)
	if err != nil {
		t.Fatalf("Validation failed: %v", err)
	}

	if result.Valid {
		t.Error("Invalid payload with wrong types should fail validation")
	}
}

func TestValidate_BiometricSchema(t *testing.T) {
	manager := NewSchemaManager("../../configs/schemas")
	err := manager.LoadSchemas()
	if err != nil {
		t.Fatalf("Failed to load schemas: %v", err)
	}

	// Test payload válido
	validPayload := map[string]interface{}{
		"timestamp":    "2025-11-07T12:00:00Z",
		"device_id":    "bio_sensor_001",
		"organism_id":  "fish_12345",
		"species":      "Salmo salar",
		"weight":       245.8,
		"length":       25.4,
		"health_score": 87.5,
	}

	result, err := manager.Validate("biometric", "v1", validPayload)
	if err != nil {
		t.Fatalf("Validation failed: %v", err)
	}

	if !result.Valid {
		t.Errorf("Valid payload should pass validation. Errors: %v", result.Errors)
	}

	// Test payload inválido - valores fuera de rango
	invalidPayload := map[string]interface{}{
		"timestamp":    "2025-11-07T12:00:00Z",
		"device_id":    "bio_sensor_001",
		"organism_id":  "fish_12345",
		"species":      "Salmo salar",
		"weight":       -10.0, // Negativo, debería ser >= 0
		"health_score": 150.0, // Mayor a 100, debería ser <= 100
	}

	result, err = manager.Validate("biometric", "v1", invalidPayload)
	if err != nil {
		t.Fatalf("Validation failed: %v", err)
	}

	if result.Valid {
		t.Error("Invalid payload with out-of-range values should fail validation")
	}
}

func TestValidate_ClimateSchema(t *testing.T) {
	manager := NewSchemaManager("../../configs/schemas")
	err := manager.LoadSchemas()
	if err != nil {
		t.Fatalf("Failed to load schemas: %v", err)
	}

	// Test payload válido
	validPayload := map[string]interface{}{
		"timestamp":         "2025-11-07T12:00:00Z",
		"device_id":         "climate_001",
		"temperature":       22.5,
		"humidity":          65.0,
		"water_temperature": 18.5,
		"dissolved_oxygen":  7.8,
		"ph_level":          7.2,
		"location": map[string]interface{}{
			"latitude":  40.7128,
			"longitude": -74.0060,
			"altitude":  10.5,
		},
	}

	result, err := manager.Validate("climate", "v1", validPayload)
	if err != nil {
		t.Fatalf("Validation failed: %v", err)
	}

	if !result.Valid {
		t.Errorf("Valid payload should pass validation. Errors: %v", result.Errors)
	}

	// Test payload inválido - temperatura fuera de rango
	invalidPayload := map[string]interface{}{
		"timestamp":   "2025-11-07T12:00:00Z",
		"device_id":   "climate_001",
		"temperature": 150.0, // Fuera del rango -50 a 100
		"ph_level":    15.0,  // Fuera del rango 0 a 14
	}

	result, err = manager.Validate("climate", "v1", invalidPayload)
	if err != nil {
		t.Fatalf("Validation failed: %v", err)
	}

	if result.Valid {
		t.Error("Invalid payload with out-of-range values should fail validation")
	}
}

func TestSchema_BackwardCompatibility(t *testing.T) {
	manager := NewSchemaManager("../../configs/schemas")
	err := manager.LoadSchemas()
	if err != nil {
		t.Fatalf("Failed to load schemas: %v", err)
	}

	schema, err := manager.GetSchema("feeding", "v1")
	if err != nil {
		t.Fatalf("Failed to get schema: %v", err)
	}

	// Verificar que el schema marca backward compatibility
	if !schema.IsBackwardCompatible() {
		t.Error("Schema should be marked as backward compatible")
	}

	// Verificar capability
	if schema.GetCapability() != "feeding.read" {
		t.Errorf("Expected capability feeding.read, got %s", schema.GetCapability())
	}
}

func TestConvenienceValidateFunction(t *testing.T) {
	// Cambiar al directorio de trabajo correcto
	originalWd, _ := os.Getwd()
	defer os.Chdir(originalWd)

	// Cambiar al directorio raíz del proyecto
	projectRoot := filepath.Join(originalWd, "../..")
	os.Chdir(projectRoot)

	validPayload := map[string]interface{}{
		"timestamp": "2025-11-07T12:00:00Z",
		"device_id": "feeder_001",
		"feed_type": "pellets",
		"quantity":  150.5,
		"status":    "completed",
	}

	result, err := Validate("feeding", "v1", validPayload)
	if err != nil {
		t.Fatalf("Convenience function failed: %v", err)
	}

	if !result.Valid {
		t.Errorf("Valid payload should pass validation. Errors: %v", result.Errors)
	}
}
