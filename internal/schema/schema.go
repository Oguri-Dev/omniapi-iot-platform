package schema

import (
	"encoding/json"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
	"sync"

	"github.com/xeipuuv/gojsonschema"
)

// Schema representa un schema JSON cargado
type Schema struct {
	Kind     string                  `json:"-"`        // ej: "feeding", "biometric", "climate"
	Version  string                  `json:"-"`        // ej: "v1"
	ID       string                  `json:"$id"`      // URI del schema
	Title    string                  `json:"title"`    // Título descriptivo
	Metadata SchemaMetadata          `json:"metadata"` // Metadatos del schema
	Schema   map[string]interface{}  `json:"-"`        // Schema JSON completo
	Compiled gojsonschema.JSONLoader `json:"-"`        // Schema compilado para validación
}

// SchemaMetadata contiene metadatos del schema
type SchemaMetadata struct {
	Version            string `json:"version"`
	Capability         string `json:"capability"`
	BackwardCompatible bool   `json:"backward_compatible"`
	CompatibilityNotes string `json:"compatibility_notes"`
	CreatedAt          string `json:"created_at"`
	UpdatedAt          string `json:"updated_at"`
}

// SchemaManager gestiona la carga y validación de schemas
type SchemaManager struct {
	schemas   map[string]*Schema // key: "kind.version" ej: "feeding.v1"
	schemaDir string
	mu        sync.RWMutex
}

// ValidationError representa un error de validación
type ValidationError struct {
	Field   string      `json:"field"`
	Message string      `json:"message"`
	Value   interface{} `json:"value,omitempty"`
}

// ValidationResult contiene el resultado de una validación
type ValidationResult struct {
	Valid  bool              `json:"valid"`
	Errors []ValidationError `json:"errors,omitempty"`
}

// NewSchemaManager crea un nuevo gestor de schemas
func NewSchemaManager(schemaDir string) *SchemaManager {
	return &SchemaManager{
		schemas:   make(map[string]*Schema),
		schemaDir: schemaDir,
	}
}

// LoadSchemas carga todos los schemas desde el directorio
func (sm *SchemaManager) LoadSchemas() error {
	sm.mu.Lock()
	defer sm.mu.Unlock()

	// Limpiar schemas existentes
	sm.schemas = make(map[string]*Schema)

	return filepath.WalkDir(sm.schemaDir, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}

		// Solo procesar archivos .json
		if d.IsDir() || !strings.HasSuffix(path, ".json") {
			return nil
		}

		// Extraer kind y version del nombre del archivo
		// Formato esperado: "feeding.v1.json", "biometric.v1.json", etc.
		filename := filepath.Base(path)
		parts := strings.Split(strings.TrimSuffix(filename, ".json"), ".")
		if len(parts) != 2 {
			return fmt.Errorf("invalid schema filename format: %s (expected: kind.version.json)", filename)
		}

		kind := parts[0]
		version := parts[1]

		schema, err := sm.loadSchemaFile(path, kind, version)
		if err != nil {
			return fmt.Errorf("failed to load schema %s: %w", path, err)
		}

		key := fmt.Sprintf("%s.%s", kind, version)
		sm.schemas[key] = schema

		return nil
	})
}

// loadSchemaFile carga un schema individual desde archivo
func (sm *SchemaManager) loadSchemaFile(filePath, kind, version string) (*Schema, error) {
	data, err := os.ReadFile(filePath)
	if err != nil {
		return nil, fmt.Errorf("failed to read schema file: %w", err)
	}

	var schemaData map[string]interface{}
	if err := json.Unmarshal(data, &schemaData); err != nil {
		return nil, fmt.Errorf("failed to parse schema JSON: %w", err)
	}

	// Extraer metadatos
	var metadata SchemaMetadata
	if metaData, exists := schemaData["metadata"]; exists {
		metaBytes, _ := json.Marshal(metaData)
		json.Unmarshal(metaBytes, &metadata)
	}

	// Crear el loader para gojsonschema
	schemaLoader := gojsonschema.NewGoLoader(schemaData)

	schema := &Schema{
		Kind:     kind,
		Version:  version,
		Metadata: metadata,
		Schema:   schemaData,
		Compiled: schemaLoader,
	}

	// Extraer ID y título si existen
	if id, exists := schemaData["$id"]; exists {
		if idStr, ok := id.(string); ok {
			schema.ID = idStr
		}
	}

	if title, exists := schemaData["title"]; exists {
		if titleStr, ok := title.(string); ok {
			schema.Title = titleStr
		}
	}

	return schema, nil
}

// GetSchema obtiene un schema por kind y version
func (sm *SchemaManager) GetSchema(kind, version string) (*Schema, error) {
	sm.mu.RLock()
	defer sm.mu.RUnlock()

	key := fmt.Sprintf("%s.%s", kind, version)
	schema, exists := sm.schemas[key]
	if !exists {
		return nil, fmt.Errorf("schema not found: %s.%s", kind, version)
	}

	return schema, nil
}

// Validate valida un payload contra un schema específico
func (sm *SchemaManager) Validate(kind, version string, payload interface{}) (*ValidationResult, error) {
	schema, err := sm.GetSchema(kind, version)
	if err != nil {
		return nil, err
	}

	// Crear el loader del documento a validar
	documentLoader := gojsonschema.NewGoLoader(payload)

	// Realizar la validación
	result, err := gojsonschema.Validate(schema.Compiled, documentLoader)
	if err != nil {
		return nil, fmt.Errorf("validation failed: %w", err)
	}

	validationResult := &ValidationResult{
		Valid:  result.Valid(),
		Errors: []ValidationError{},
	}

	// Convertir errores de gojsonschema a nuestro formato
	if !result.Valid() {
		for _, err := range result.Errors() {
			validationError := ValidationError{
				Field:   err.Field(),
				Message: err.Description(),
			}

			// Intentar obtener el valor si está disponible
			if err.Value() != nil {
				validationError.Value = err.Value()
			}

			validationResult.Errors = append(validationResult.Errors, validationError)
		}
	}

	return validationResult, nil
}

// ListSchemas retorna todos los schemas cargados
func (sm *SchemaManager) ListSchemas() map[string]*Schema {
	sm.mu.RLock()
	defer sm.mu.RUnlock()

	// Crear una copia para evitar modificaciones concurrentes
	result := make(map[string]*Schema)
	for key, schema := range sm.schemas {
		result[key] = schema
	}

	return result
}

// GetSchemasForKind obtiene todas las versiones de un kind específico
func (sm *SchemaManager) GetSchemasForKind(kind string) []*Schema {
	sm.mu.RLock()
	defer sm.mu.RUnlock()

	var schemas []*Schema
	prefix := kind + "."

	for key, schema := range sm.schemas {
		if strings.HasPrefix(key, prefix) {
			schemas = append(schemas, schema)
		}
	}

	return schemas
}

// IsBackwardCompatible verifica si un schema es backward compatible
func (s *Schema) IsBackwardCompatible() bool {
	return s.Metadata.BackwardCompatible
}

// GetCapability retorna la capability asociada al schema
func (s *Schema) GetCapability() string {
	return s.Metadata.Capability
}

// String retorna una representación string del schema
func (s *Schema) String() string {
	return fmt.Sprintf("%s.%s (%s)", s.Kind, s.Version, s.Title)
}

// Validate es una función de conveniencia para validar un payload
func Validate(kind, version string, payload interface{}) (*ValidationResult, error) {
	// Usar el directorio por defecto
	defaultDir := "configs/schemas"
	manager := NewSchemaManager(defaultDir)

	if err := manager.LoadSchemas(); err != nil {
		return nil, fmt.Errorf("failed to load schemas: %w", err)
	}

	return manager.Validate(kind, version, payload)
}
