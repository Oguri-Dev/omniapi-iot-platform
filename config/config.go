package config

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"omniapi/internal/domain"

	"go.mongodb.org/mongo-driver/bson/primitive"
	"gopkg.in/yaml.v3"
)

// Config contiene toda la configuración de la aplicación
type Config struct {
	// Configuración básica (env vars)
	Port        string        `yaml:"port"`
	Environment string        `yaml:"environment"`
	LogLevel    string        `yaml:"log_level"`
	MongoDB     MongoDBConfig `yaml:"mongodb"`

	// Configuración desde archivos YAML
	App         AppConfig                  `yaml:"app"`
	Tenants     []TenantConfig             `yaml:"tenants"`
	Connections []ConnectionInstanceConfig `yaml:"connections"`
	Mappings    map[string]MappingConfig   `yaml:"mappings"`
}

// MongoDBConfig configuración específica de MongoDB
type MongoDBConfig struct {
	URI      string `yaml:"uri"`
	Database string `yaml:"database"`
	Timeout  string `yaml:"timeout"`
}

// AppConfig configuración de la aplicación
type AppConfig struct {
	HTTP      HTTPConfig      `yaml:"http"`
	WebSocket WSConfig        `yaml:"websocket"`
	Auth      AuthConfig      `yaml:"auth"`
	Quotas    QuotasConfig    `yaml:"quotas"`
	Policies  PoliciesConfig  `yaml:"policies"`
	Requester RequesterConfig `yaml:"requester"`
	Status    StatusConfig    `yaml:"status"`
}

// HTTPConfig configuración del servidor HTTP
type HTTPConfig struct {
	Host         string        `yaml:"host"`
	Port         int           `yaml:"port"`
	ReadTimeout  time.Duration `yaml:"read_timeout"`
	WriteTimeout time.Duration `yaml:"write_timeout"`
	CORS         CORSConfig    `yaml:"cors"`
}

// WSConfig configuración de WebSockets
type WSConfig struct {
	ReadBufferSize  int           `yaml:"read_buffer_size"`
	WriteBufferSize int           `yaml:"write_buffer_size"`
	WriteWait       time.Duration `yaml:"write_wait"`
	PongWait        time.Duration `yaml:"pong_wait"`
	PingPeriod      time.Duration `yaml:"ping_period"`
	MaxMessageSize  int64         `yaml:"max_message_size"`
}

// CORSConfig configuración de CORS
type CORSConfig struct {
	AllowedOrigins   []string `yaml:"allowed_origins"`
	AllowedMethods   []string `yaml:"allowed_methods"`
	AllowedHeaders   []string `yaml:"allowed_headers"`
	AllowCredentials bool     `yaml:"allow_credentials"`
}

// AuthConfig configuración de autenticación
type AuthConfig struct {
	JWTSecret     SecretRef     `yaml:"jwt_secret"`
	TokenExpiry   time.Duration `yaml:"token_expiry"`
	RefreshExpiry time.Duration `yaml:"refresh_expiry"`
	Issuer        string        `yaml:"issuer"`
}

// QuotasConfig configuración de quotas por defecto
type QuotasConfig struct {
	Default []QuotaConfig `yaml:"default"`
}

// QuotaConfig configuración de una quota específica
type QuotaConfig struct {
	Resource string `yaml:"resource"`
	Limit    int64  `yaml:"limit"`
}

// PoliciesConfig configuración de políticas
type PoliciesConfig struct {
	RateLimit     RateLimitConfig     `yaml:"rate_limit"`
	DataRetention DataRetentionConfig `yaml:"data_retention"`
}

// RateLimitConfig configuración de rate limiting
type RateLimitConfig struct {
	Enabled           bool `yaml:"enabled"`
	RequestsPerMinute int  `yaml:"requests_per_minute"`
	BurstSize         int  `yaml:"burst_size"`
}

// DataRetentionConfig configuración de retención de datos
type DataRetentionConfig struct {
	Events  time.Duration `yaml:"events"`
	Logs    time.Duration `yaml:"logs"`
	Metrics time.Duration `yaml:"metrics"`
}

// RequesterConfig configuración del módulo requester
type RequesterConfig struct {
	TimeoutSeconds int                  `yaml:"timeout_seconds"`
	BackoffSeconds []int                `yaml:"backoff_seconds"`
	CircuitBreaker CircuitBreakerConfig `yaml:"circuit_breaker"`
}

// CircuitBreakerConfig configuración del circuit breaker
type CircuitBreakerConfig struct {
	FailuresThreshold int `yaml:"failures_threshold"`
	PauseMinutes      int `yaml:"pause_minutes"`
}

// StatusConfig configuración del módulo status
type StatusConfig struct {
	HeartbeatSeconds int `yaml:"heartbeat_seconds"`
}

// TenantConfig configuración de un tenant
type TenantConfig struct {
	ID          string                 `yaml:"id"`
	Name        string                 `yaml:"name"`
	DisplayName string                 `yaml:"display_name"`
	Status      string                 `yaml:"status"`
	Quotas      []QuotaConfig          `yaml:"quotas"`
	Scopes      []ScopeConfig          `yaml:"scopes"`
	Settings    map[string]interface{} `yaml:"settings"`
	CreatedBy   string                 `yaml:"created_by"`
}

// ScopeConfig configuración de un scope
type ScopeConfig struct {
	Resource    string   `yaml:"resource"`
	Permissions []string `yaml:"permissions"`
	FarmIDs     []string `yaml:"farm_ids"`
	SiteIDs     []string `yaml:"site_ids"`
	CageIDs     []string `yaml:"cage_ids"`
}

// ConnectionInstanceConfig configuración de una instancia de conexión
type ConnectionInstanceConfig struct {
	ID          string                 `yaml:"id"`
	TenantID    string                 `yaml:"tenant_id"`
	TypeID      string                 `yaml:"type_id"`
	DisplayName string                 `yaml:"display_name"`
	Description string                 `yaml:"description"`
	SecretsRef  SecretRef              `yaml:"secrets_ref"`
	Config      map[string]interface{} `yaml:"config"`
	Mappings    []string               `yaml:"mappings"` // Referencias a archivos de mapping
	Status      string                 `yaml:"status"`
	Tags        []string               `yaml:"tags"`
	CreatedBy   string                 `yaml:"created_by"`
}

// MappingConfig configuración de un mapping
type MappingConfig struct {
	Name        string              `yaml:"name"`
	Description string              `yaml:"description"`
	Capability  string              `yaml:"capability"`
	Rules       []MappingRuleConfig `yaml:"rules"`
	StrictMode  bool                `yaml:"strict_mode"`
	IgnoreExtra bool                `yaml:"ignore_extra"`
	Version     string              `yaml:"version"`
}

// MappingRuleConfig configuración de una regla de mapping
type MappingRuleConfig struct {
	SourceField  string           `yaml:"source_field"`
	TargetField  string           `yaml:"target_field"`
	Transform    *TransformConfig `yaml:"transform,omitempty"`
	DefaultValue interface{}      `yaml:"default_value,omitempty"`
	Required     bool             `yaml:"required"`
	Description  string           `yaml:"description,omitempty"`
}

// TransformConfig configuración de una transformación
type TransformConfig struct {
	Type       string                 `yaml:"type"`
	Parameters map[string]interface{} `yaml:"parameters,omitempty"`
}

// SecretRef representa una referencia a un secreto
type SecretRef struct {
	Provider string `yaml:"provider"` // "env", "file", "vault", etc.
	Key      string `yaml:"key"`      // La clave del secreto
}

// SecretProvider interfaz para proveedores de secretos
type SecretProvider interface {
	GetSecret(ref SecretRef) (string, error)
}

// EnvSecretProvider proveedor de secretos desde variables de entorno
type EnvSecretProvider struct{}

// GetSecret obtiene un secreto desde variables de entorno
func (p *EnvSecretProvider) GetSecret(ref SecretRef) (string, error) {
	if ref.Provider != "env" {
		return "", fmt.Errorf("unsupported provider: %s", ref.Provider)
	}

	value := os.Getenv(ref.Key)
	if value == "" {
		return "", fmt.Errorf("environment variable %s not found", ref.Key)
	}

	return value, nil
}

// LoadConfig carga la configuración completa desde variables de entorno y archivos YAML
func LoadConfig() (*Config, error) {
	// Configuración base desde variables de entorno
	config := &Config{
		Port:        getEnv("PORT", "3000"),
		Environment: getEnv("ENVIRONMENT", "development"),
		LogLevel:    getEnv("LOG_LEVEL", "info"),
		MongoDB: MongoDBConfig{
			URI:      getEnv("MONGODB_URI", "mongodb://localhost:27017"),
			Database: getEnv("MONGODB_DATABASE", "omniapi"),
			Timeout:  getEnv("MONGODB_TIMEOUT", "10s"),
		},
		Mappings: make(map[string]MappingConfig),
	}

	// Cargar configuración de la aplicación
	if err := loadAppConfig(config); err != nil {
		return nil, fmt.Errorf("failed to load app config: %w", err)
	}

	// Cargar tenants
	if err := loadTenantsConfig(config); err != nil {
		return nil, fmt.Errorf("failed to load tenants config: %w", err)
	}

	// Cargar conexiones
	if err := loadConnectionsConfig(config); err != nil {
		return nil, fmt.Errorf("failed to load connections config: %w", err)
	}

	// Cargar mappings
	if err := loadMappingsConfig(config); err != nil {
		return nil, fmt.Errorf("failed to load mappings config: %w", err)
	}

	// Resolver referencias de secretos
	if err := resolveSecrets(config); err != nil {
		return nil, fmt.Errorf("failed to resolve secrets: %w", err)
	}

	return config, nil
}

// loadAppConfig carga la configuración de la aplicación desde configs/app.yaml
func loadAppConfig(config *Config) error {
	appConfigFile := "configs/app.yaml"

	if !fileExists(appConfigFile) {
		// Usar configuración por defecto si el archivo no existe
		config.App = getDefaultAppConfig()
		return nil
	}

	data, err := os.ReadFile(appConfigFile)
	if err != nil {
		return fmt.Errorf("failed to read app config file: %w", err)
	}

	var appConfig AppConfig
	if err := yaml.Unmarshal(data, &appConfig); err != nil {
		return fmt.Errorf("failed to parse app config: %w", err)
	}

	config.App = appConfig
	return nil
}

// loadTenantsConfig carga la configuración de tenants desde configs/tenants.yaml
func loadTenantsConfig(config *Config) error {
	tenantsConfigFile := "configs/tenants.yaml"

	if !fileExists(tenantsConfigFile) {
		config.Tenants = []TenantConfig{}
		return nil
	}

	data, err := os.ReadFile(tenantsConfigFile)
	if err != nil {
		return fmt.Errorf("failed to read tenants config file: %w", err)
	}

	var tenantsConfig struct {
		Tenants []TenantConfig `yaml:"tenants"`
	}

	if err := yaml.Unmarshal(data, &tenantsConfig); err != nil {
		return fmt.Errorf("failed to parse tenants config: %w", err)
	}

	config.Tenants = tenantsConfig.Tenants
	return nil
}

// loadConnectionsConfig carga la configuración de conexiones desde configs/connections.yaml
func loadConnectionsConfig(config *Config) error {
	connectionsConfigFile := "configs/connections.yaml"

	if !fileExists(connectionsConfigFile) {
		config.Connections = []ConnectionInstanceConfig{}
		return nil
	}

	data, err := os.ReadFile(connectionsConfigFile)
	if err != nil {
		return fmt.Errorf("failed to read connections config file: %w", err)
	}

	var connectionsConfig struct {
		Connections []ConnectionInstanceConfig `yaml:"connections"`
	}

	if err := yaml.Unmarshal(data, &connectionsConfig); err != nil {
		return fmt.Errorf("failed to parse connections config: %w", err)
	}

	config.Connections = connectionsConfig.Connections
	return nil
}

// loadMappingsConfig carga los mappings desde configs/mappings/*.yaml
func loadMappingsConfig(config *Config) error {
	mappingsDir := "configs/mappings"

	if !dirExists(mappingsDir) {
		return nil
	}

	files, err := filepath.Glob(filepath.Join(mappingsDir, "*.yaml"))
	if err != nil {
		return fmt.Errorf("failed to glob mapping files: %w", err)
	}

	for _, file := range files {
		mappingName := strings.TrimSuffix(filepath.Base(file), ".yaml")

		data, err := os.ReadFile(file)
		if err != nil {
			return fmt.Errorf("failed to read mapping file %s: %w", file, err)
		}

		var mappingConfig MappingConfig
		if err := yaml.Unmarshal(data, &mappingConfig); err != nil {
			return fmt.Errorf("failed to parse mapping file %s: %w", file, err)
		}

		config.Mappings[mappingName] = mappingConfig
	}

	return nil
}

// resolveSecrets resuelve todas las referencias de secretos en la configuración
func resolveSecrets(config *Config) error {
	secretProvider := &EnvSecretProvider{}

	// Resolver secreto de JWT
	if config.App.Auth.JWTSecret.Provider != "" {
		secret, err := secretProvider.GetSecret(config.App.Auth.JWTSecret)
		if err != nil {
			return fmt.Errorf("failed to resolve JWT secret: %w", err)
		}
		config.App.Auth.JWTSecret.Key = secret
	}

	// Resolver secretos en conexiones
	for i := range config.Connections {
		conn := &config.Connections[i]
		if conn.SecretsRef.Provider != "" {
			secret, err := secretProvider.GetSecret(conn.SecretsRef)
			if err != nil {
				return fmt.Errorf("failed to resolve secret for connection %s: %w", conn.ID, err)
			}
			// Agregar el secreto a la configuración
			if conn.Config == nil {
				conn.Config = make(map[string]interface{})
			}
			conn.Config["__resolved_secret"] = secret
		}
	}

	return nil
}

// getDefaultAppConfig retorna una configuración por defecto de la aplicación
func getDefaultAppConfig() AppConfig {
	return AppConfig{
		HTTP: HTTPConfig{
			Host:         "0.0.0.0",
			Port:         3000,
			ReadTimeout:  30 * time.Second,
			WriteTimeout: 30 * time.Second,
			CORS: CORSConfig{
				AllowedOrigins:   []string{"*"},
				AllowedMethods:   []string{"GET", "POST", "PUT", "DELETE", "OPTIONS"},
				AllowedHeaders:   []string{"*"},
				AllowCredentials: true,
			},
		},
		WebSocket: WSConfig{
			ReadBufferSize:  1024,
			WriteBufferSize: 1024,
			WriteWait:       10 * time.Second,
			PongWait:        60 * time.Second,
			PingPeriod:      54 * time.Second,
			MaxMessageSize:  512,
		},
		Auth: AuthConfig{
			JWTSecret:     SecretRef{Provider: "env", Key: "JWT_SECRET"},
			TokenExpiry:   24 * time.Hour,
			RefreshExpiry: 7 * 24 * time.Hour,
			Issuer:        "omniapi",
		},
		Quotas: QuotasConfig{
			Default: []QuotaConfig{
				{Resource: "api_calls_per_hour", Limit: 1000},
				{Resource: "storage_gb", Limit: 10},
				{Resource: "connections", Limit: 5},
				{Resource: "streams", Limit: 50},
			},
		},
		Policies: PoliciesConfig{
			RateLimit: RateLimitConfig{
				Enabled:           true,
				RequestsPerMinute: 100,
				BurstSize:         10,
			},
			DataRetention: DataRetentionConfig{
				Events:  30 * 24 * time.Hour, // 30 días
				Logs:    7 * 24 * time.Hour,  // 7 días
				Metrics: 90 * 24 * time.Hour, // 90 días
			},
		},
	}
}

// fileExists verifica si un archivo existe
func fileExists(filename string) bool {
	_, err := os.Stat(filename)
	return !os.IsNotExist(err)
}

// dirExists verifica si un directorio existe
func dirExists(dirname string) bool {
	info, err := os.Stat(dirname)
	return err == nil && info.IsDir()
}

// ToDomainTenant convierte un TenantConfig a domain.Tenant
func (tc *TenantConfig) ToDomainTenant() (*domain.Tenant, error) {
	// Convertir ID desde string si es ObjectID hex, sino generar uno nuevo
	var tenantID primitive.ObjectID
	if tc.ID != "" {
		if oid, err := primitive.ObjectIDFromHex(tc.ID); err == nil {
			tenantID = oid
		} else {
			tenantID = primitive.NewObjectID()
		}
	} else {
		tenantID = primitive.NewObjectID()
	}

	tenant := &domain.Tenant{
		ID:          tenantID,
		Name:        tc.Name,
		DisplayName: tc.DisplayName,
		Status:      domain.TenantStatus(tc.Status),
		Quotas:      []domain.Quota{},
		Scopes:      []domain.Scope{},
		Settings:    tc.Settings,
		CreatedAt:   time.Now(),
		UpdatedAt:   time.Now(),
		CreatedBy:   tc.CreatedBy,
		UpdatedBy:   tc.CreatedBy,
	}

	// Convertir quotas
	for _, quotaConfig := range tc.Quotas {
		quota := domain.Quota{
			Resource: quotaConfig.Resource,
			Limit:    quotaConfig.Limit,
			Used:     0,
		}
		tenant.Quotas = append(tenant.Quotas, quota)
	}

	// Convertir scopes
	for _, scopeConfig := range tc.Scopes {
		var capabilities []domain.Capability
		for _, perm := range scopeConfig.Permissions {
			capabilities = append(capabilities, domain.Capability(perm))
		}

		scope := domain.Scope{
			TenantID:    tenantID,
			Resource:    scopeConfig.Resource,
			Permissions: capabilities,
			FarmIDs:     scopeConfig.FarmIDs,
			SiteIDs:     scopeConfig.SiteIDs,
			CageIDs:     scopeConfig.CageIDs,
		}
		tenant.Scopes = append(tenant.Scopes, scope)
	}

	return tenant, nil
}

// ToDomainConnectionInstance convierte un ConnectionInstanceConfig a domain.ConnectionInstance
func (cic *ConnectionInstanceConfig) ToDomainConnectionInstance() (*domain.ConnectionInstance, error) {
	// Convertir IDs
	instanceID, err := primitive.ObjectIDFromHex(cic.ID)
	if err != nil {
		instanceID = primitive.NewObjectID()
	}

	tenantID, err := primitive.ObjectIDFromHex(cic.TenantID)
	if err != nil {
		return nil, fmt.Errorf("invalid tenant_id: %s", cic.TenantID)
	}

	typeID, err := primitive.ObjectIDFromHex(cic.TypeID)
	if err != nil {
		return nil, fmt.Errorf("invalid type_id: %s", cic.TypeID)
	}

	conn := &domain.ConnectionInstance{
		ID:          instanceID,
		TenantID:    tenantID,
		TypeID:      typeID,
		DisplayName: cic.DisplayName,
		Description: cic.Description,
		SecretsRef:  fmt.Sprintf("%s://%s", cic.SecretsRef.Provider, cic.SecretsRef.Key),
		Config:      cic.Config,
		Mappings:    []domain.Mapping{},
		Status:      domain.ConnectionStatus(cic.Status),
		Tags:        cic.Tags,
		Metadata:    make(map[string]interface{}),
		CreatedAt:   time.Now(),
		UpdatedAt:   time.Now(),
		CreatedBy:   cic.CreatedBy,
		UpdatedBy:   cic.CreatedBy,
	}

	return conn, nil
}

// LogConfigSummary loguea un resumen de la configuración cargada
func (c *Config) LogConfigSummary() {
	fmt.Printf("📋 Configuration Summary:\n")
	fmt.Printf("   Environment: %s\n", c.Environment)
	fmt.Printf("   HTTP Port: %d\n", c.App.HTTP.Port)
	fmt.Printf("   MongoDB: %s/%s\n", c.MongoDB.URI, c.MongoDB.Database)
	fmt.Printf("   Tenants loaded: %d\n", len(c.Tenants))
	fmt.Printf("   Connections loaded: %d\n", len(c.Connections))
	fmt.Printf("   Mappings loaded: %d\n", len(c.Mappings))
	fmt.Printf("   Rate limit: %d req/min\n", c.App.Policies.RateLimit.RequestsPerMinute)

	if len(c.Tenants) > 0 {
		fmt.Printf("   Tenant names: ")
		for i, tenant := range c.Tenants {
			if i > 0 {
				fmt.Printf(", ")
			}
			fmt.Printf("%s", tenant.Name)
		}
		fmt.Printf("\n")
	}

	if len(c.Connections) > 0 {
		fmt.Printf("   Connection types: ")
		typeCount := make(map[string]int)
		for _, conn := range c.Connections {
			typeCount[conn.TypeID]++
		}
		first := true
		for typeID, count := range typeCount {
			if !first {
				fmt.Printf(", ")
			}
			fmt.Printf("%s(%d)", typeID, count)
			first = false
		}
		fmt.Printf("\n")
	}
}

// getEnv obtiene una variable de entorno con un valor por defecto
func getEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}
