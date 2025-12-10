package polling

import (
	"context"
	"fmt"
	"sync"
	"time"

	"omniapi/internal/broker"
	"omniapi/internal/database"
	"omniapi/internal/models"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

// Engine gestiona todos los workers de polling
type Engine struct {
	workers map[string]*Worker        // Key: configID:instanceID
	configs map[string]*PollingConfig // Key: configID
	mu      sync.RWMutex

	ctx       context.Context
	cancel    context.CancelFunc
	startedAt time.Time

	// Callback global para resultados
	onResult func(PollingResult)

	// Broker manager para publicar resultados
	brokerManager *broker.Manager
}

var (
	engineInstance *Engine
	engineOnce     sync.Once
)

// GetEngine retorna la instancia singleton del Engine
func GetEngine() *Engine {
	engineOnce.Do(func() {
		engineInstance = &Engine{
			workers: make(map[string]*Worker),
			configs: make(map[string]*PollingConfig),
		}
	})
	return engineInstance
}

// Start inicia el engine
func (e *Engine) Start(ctx context.Context) error {
	e.mu.Lock()
	defer e.mu.Unlock()

	if e.ctx != nil {
		return fmt.Errorf("engine already started")
	}

	e.ctx, e.cancel = context.WithCancel(ctx)
	e.startedAt = time.Now()

	fmt.Println("🔄 Polling Engine started")

	// Iniciar el broker manager
	e.brokerManager = broker.NewManager()
	if err := e.brokerManager.Start(ctx); err != nil {
		fmt.Printf("⚠️  Warning: could not start broker manager: %v\n", err)
	}

	// Cargar configuraciones activas desde MongoDB
	if err := e.loadActiveConfigs(); err != nil {
		fmt.Printf("⚠️  Warning: could not load active polling configs: %v\n", err)
	}

	return nil
}

// Stop detiene el engine y todos los workers
func (e *Engine) Stop() {
	e.mu.Lock()
	defer e.mu.Unlock()

	if e.cancel != nil {
		e.cancel()
	}

	// Detener el broker manager
	if e.brokerManager != nil {
		e.brokerManager.Stop()
	}

	// Detener todos los workers
	for _, worker := range e.workers {
		worker.Stop()
	}

	e.workers = make(map[string]*Worker)
	e.configs = make(map[string]*PollingConfig)

	fmt.Println("🛑 Polling Engine stopped")
}

// OnResult registra callback global para resultados
func (e *Engine) OnResult(callback func(PollingResult)) {
	e.onResult = callback
}

// GetBrokerManager retorna el manager de brokers
func (e *Engine) GetBrokerManager() *broker.Manager {
	return e.brokerManager
}

// StartPolling inicia polling para una configuración
func (e *Engine) StartPolling(req StartPollingRequest) (*PollingConfig, error) {
	e.mu.Lock()
	defer e.mu.Unlock()

	// Validar request
	if req.Provider == "" {
		return nil, fmt.Errorf("provider is required")
	}
	if req.SiteID == "" {
		return nil, fmt.Errorf("site_id is required")
	}
	if req.ServiceID == "" {
		return nil, fmt.Errorf("service_id is required")
	}
	if len(req.Endpoints) == 0 {
		return nil, fmt.Errorf("at least one endpoint is required")
	}

	// Obtener el ExternalService para autenticación
	service, err := e.getExternalService(req.ServiceID)
	if err != nil {
		return nil, fmt.Errorf("error getting external service: %w", err)
	}

	// Crear configuración
	config := &PollingConfig{
		ID:         primitive.NewObjectID(),
		Provider:   req.Provider,
		SiteID:     req.SiteID,
		SiteCode:   req.SiteCode,
		SiteName:   req.SiteName,
		TenantID:   req.TenantID,
		TenantCode: req.TenantCode,
		ServiceID:  req.ServiceID,
		Endpoints:  req.Endpoints,
		IntervalMS: req.IntervalMS,
		AutoStart:  true, // Siempre autostart cuando se crea manualmente
		Status:     "active",
		Output:     req.Output, // Configuración de salida MQTT
		CreatedAt:  time.Now(),
		UpdatedAt:  time.Now(),
	}

	if config.IntervalMS <= 0 {
		config.IntervalMS = 2000 // Default 2 segundos
	}

	// Generar instance IDs si no vienen
	for i := range config.Endpoints {
		if config.Endpoints[i].InstanceID == "" {
			config.Endpoints[i].InstanceID = fmt.Sprintf("%s-%s-%d",
				config.Provider,
				config.Endpoints[i].EndpointID,
				time.Now().UnixNano())
		}
		if !config.Endpoints[i].Enabled {
			config.Endpoints[i].Enabled = true
		}
	}

	// Guardar en MongoDB
	if err := e.saveConfig(config); err != nil {
		return nil, fmt.Errorf("error saving config: %w", err)
	}

	// Crear y arrancar workers
	configID := config.ID.Hex()
	e.configs[configID] = config

	for _, endpoint := range config.Endpoints {
		if !endpoint.Enabled {
			continue
		}

		workerKey := fmt.Sprintf("%s:%s", configID, endpoint.InstanceID)
		worker := NewWorker(config, endpoint, service)

		// Asignar broker manager para publicación MQTT
		worker.SetBrokerManager(e.brokerManager)

		// Registrar callback
		if e.onResult != nil {
			worker.OnResult(e.onResult)
		}

		if err := worker.Start(e.ctx); err != nil {
			fmt.Printf("⚠️  Error starting worker %s: %v\n", workerKey, err)
			continue
		}

		e.workers[workerKey] = worker
		fmt.Printf("✅ Worker started: %s (%s)\n", endpoint.Label, endpoint.InstanceID)
	}

	fmt.Printf("🚀 Polling started for %s/%s with %d endpoints\n",
		config.Provider, config.SiteID, len(config.Endpoints))

	return config, nil
}

// StopPolling detiene polling según los criterios del request
func (e *Engine) StopPolling(req StopPollingRequest) (int, error) {
	e.mu.Lock()
	defer e.mu.Unlock()

	stopped := 0

	for workerKey, worker := range e.workers {
		shouldStop := false

		// Verificar criterios
		if req.ConfigID != "" {
			configID := e.extractConfigID(workerKey)
			if configID == req.ConfigID {
				shouldStop = true
			}
		}

		if req.InstanceID != "" && worker.instance.InstanceID == req.InstanceID {
			shouldStop = true
		}

		if req.SiteID != "" && worker.config.SiteID == req.SiteID {
			if req.Provider == "" || worker.config.Provider == req.Provider {
				shouldStop = true
			}
		}

		if shouldStop {
			worker.Stop()
			delete(e.workers, workerKey)
			stopped++
			fmt.Printf("🛑 Worker stopped: %s\n", workerKey)
		}
	}

	// Actualizar estado en MongoDB si se detuvo por configID
	if req.ConfigID != "" {
		e.updateConfigStatus(req.ConfigID, "stopped")
		delete(e.configs, req.ConfigID)
	}

	return stopped, nil
}

// GetStatus retorna el estado del engine
func (e *Engine) GetStatus() EngineStatus {
	e.mu.RLock()
	defer e.mu.RUnlock()

	status := EngineStatus{
		Status:        "stopped",
		ActiveWorkers: len(e.workers),
		TotalConfigs:  len(e.configs),
		Workers:       make(map[string]WorkerStatus),
		StartedAt:     e.startedAt,
	}

	if e.ctx != nil {
		status.Status = "running"
	}

	for key, worker := range e.workers {
		status.Workers[key] = worker.GetStatus()
	}

	return status
}

// GetConfigStatus retorna el estado de una configuración específica
func (e *Engine) GetConfigStatus(configID string) (*PollingConfig, []WorkerStatus, error) {
	e.mu.RLock()
	defer e.mu.RUnlock()

	config, exists := e.configs[configID]
	if !exists {
		return nil, nil, fmt.Errorf("config not found")
	}

	var workerStatuses []WorkerStatus
	for workerKey, worker := range e.workers {
		if e.extractConfigID(workerKey) == configID {
			workerStatuses = append(workerStatuses, worker.GetStatus())
		}
	}

	return config, workerStatuses, nil
}

// ListConfigs retorna todas las configuraciones activas
func (e *Engine) ListConfigs() []*PollingConfig {
	e.mu.RLock()
	defer e.mu.RUnlock()

	configs := make([]*PollingConfig, 0, len(e.configs))
	for _, config := range e.configs {
		configs = append(configs, config)
	}
	return configs
}

// ═══════════════════════════════════════════════════════════
// Helpers
// ═══════════════════════════════════════════════════════════

func (e *Engine) extractConfigID(workerKey string) string {
	// workerKey format: configID:instanceID
	parts := splitFirst(workerKey, ":")
	if len(parts) > 0 {
		return parts[0]
	}
	return ""
}

func splitFirst(s string, sep string) []string {
	idx := 0
	for i, c := range s {
		if string(c) == sep {
			idx = i
			break
		}
	}
	if idx > 0 {
		return []string{s[:idx], s[idx+1:]}
	}
	return []string{s}
}

func (e *Engine) getExternalService(serviceID string) (*models.ExternalService, error) {
	collection := database.GetCollection("external_services")
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	objID, err := primitive.ObjectIDFromHex(serviceID)
	if err != nil {
		return nil, fmt.Errorf("invalid service ID: %w", err)
	}

	var service models.ExternalService
	err = collection.FindOne(ctx, bson.M{"_id": objID}).Decode(&service)
	if err != nil {
		if err == mongo.ErrNoDocuments {
			return nil, fmt.Errorf("service not found")
		}
		return nil, err
	}

	return &service, nil
}

func (e *Engine) saveConfig(config *PollingConfig) error {
	collection := database.GetCollection("polling_configs")
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	// Usar upsert para evitar duplicados - clave única: provider + site_id
	filter := bson.M{
		"provider": config.Provider,
		"site_id":  config.SiteID,
	}

	update := bson.M{
		"$set": bson.M{
			"site_code":   config.SiteCode,
			"site_name":   config.SiteName,
			"tenant_id":   config.TenantID,
			"tenant_code": config.TenantCode,
			"service_id":  config.ServiceID,
			"endpoints":   config.Endpoints,
			"interval_ms": config.IntervalMS,
			"auto_start":  config.AutoStart,
			"status":      config.Status,
			"updated_at":  time.Now(),
		},
		"$setOnInsert": bson.M{
			"_id":        config.ID,
			"created_at": config.CreatedAt,
		},
	}

	opts := options.Update().SetUpsert(true)
	result, err := collection.UpdateOne(ctx, filter, update, opts)
	if err != nil {
		return err
	}

	// Si se actualizó un documento existente, obtener su ID
	if result.UpsertedCount == 0 && result.MatchedCount > 0 {
		var existingConfig PollingConfig
		if err := collection.FindOne(ctx, filter).Decode(&existingConfig); err == nil {
			config.ID = existingConfig.ID
		}
	}

	return nil
}

func (e *Engine) updateConfigStatus(configID string, status string) error {
	collection := database.GetCollection("polling_configs")
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	objID, err := primitive.ObjectIDFromHex(configID)
	if err != nil {
		return err
	}

	_, err = collection.UpdateOne(ctx,
		bson.M{"_id": objID},
		bson.M{"$set": bson.M{"status": status, "updated_at": time.Now()}},
	)
	return err
}

// cleanupDuplicateConfigs elimina configs duplicadas manteniendo solo la más reciente
func (e *Engine) cleanupDuplicateConfigs() error {
	collection := database.GetCollection("polling_configs")
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	// Encontrar todos los documentos agrupados por provider+site_id
	pipeline := mongo.Pipeline{
		{{Key: "$group", Value: bson.M{
			"_id": bson.M{
				"provider": "$provider",
				"site_id":  "$site_id",
			},
			"count": bson.M{"$sum": 1},
			"docs": bson.M{"$push": bson.M{
				"_id":        "$_id",
				"updated_at": "$updated_at",
			}},
		}}},
		{{Key: "$match", Value: bson.M{
			"count": bson.M{"$gt": 1},
		}}},
	}

	cursor, err := collection.Aggregate(ctx, pipeline)
	if err != nil {
		return err
	}
	defer cursor.Close(ctx)

	var results []struct {
		ID struct {
			Provider string `bson:"provider"`
			SiteID   string `bson:"site_id"`
		} `bson:"_id"`
		Count int `bson:"count"`
		Docs  []struct {
			ID        primitive.ObjectID `bson:"_id"`
			UpdatedAt time.Time          `bson:"updated_at"`
		} `bson:"docs"`
	}

	if err := cursor.All(ctx, &results); err != nil {
		return err
	}

	totalDeleted := 0
	for _, result := range results {
		// Ordenar por updated_at DESC y mantener solo el más reciente
		docs := result.Docs
		if len(docs) <= 1 {
			continue
		}

		// Encontrar el más reciente
		var newestID primitive.ObjectID
		var newestTime time.Time
		for _, doc := range docs {
			if doc.UpdatedAt.After(newestTime) {
				newestTime = doc.UpdatedAt
				newestID = doc.ID
			}
		}

		// Eliminar todos excepto el más reciente
		var idsToDelete []primitive.ObjectID
		for _, doc := range docs {
			if doc.ID != newestID {
				idsToDelete = append(idsToDelete, doc.ID)
			}
		}

		if len(idsToDelete) > 0 {
			deleteResult, err := collection.DeleteMany(ctx, bson.M{
				"_id": bson.M{"$in": idsToDelete},
			})
			if err != nil {
				fmt.Printf("⚠️  Error deleting duplicates for %s/%s: %v\n",
					result.ID.Provider, result.ID.SiteID, err)
				continue
			}
			totalDeleted += int(deleteResult.DeletedCount)
			fmt.Printf("🧹 Cleaned %d duplicate configs for %s/%s\n",
				deleteResult.DeletedCount, result.ID.Provider, result.ID.SiteID)
		}
	}

	if totalDeleted > 0 {
		fmt.Printf("🧹 Total duplicate configs cleaned: %d\n", totalDeleted)
	}

	return nil
}

func (e *Engine) loadActiveConfigs() error {
	// Limpiar duplicados antes de cargar
	if err := e.cleanupDuplicateConfigs(); err != nil {
		fmt.Printf("⚠️  Error cleaning duplicate configs: %v\n", err)
	}

	collection := database.GetCollection("polling_configs")
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	// Buscar configs activas O con auto_start habilitado
	cursor, err := collection.Find(ctx, bson.M{
		"$or": []bson.M{
			{"status": "active"},
			{"auto_start": true, "status": bson.M{"$ne": "stopped"}},
		},
	})
	if err != nil {
		return err
	}
	defer cursor.Close(ctx)

	var configs []PollingConfig
	if err := cursor.All(ctx, &configs); err != nil {
		return err
	}

	if len(configs) == 0 {
		fmt.Println("📋 No active polling configs found to restore")
		return nil
	}

	fmt.Printf("📋 Found %d polling configs to restore\n", len(configs))

	for _, config := range configs {
		fmt.Printf("\n🔄 Restoring config: %s/%s (%s)\n", config.Provider, config.SiteCode, config.SiteName)

		// Obtener el servicio externo
		service, err := e.getExternalService(config.ServiceID)
		if err != nil {
			fmt.Printf("   ❌ Could not get external service: %v\n", err)
			continue
		}
		fmt.Printf("   ✓ External service found: %s (%s)\n", service.Name, service.ServiceType)

		// Verificar autenticación antes de iniciar workers
		fmt.Printf("   🔐 Validating authentication for %s...\n", config.Provider)

		configID := config.ID.Hex()
		configCopy := config
		e.configs[configID] = &configCopy

		workersStarted := 0
		for _, endpoint := range config.Endpoints {
			if !endpoint.Enabled {
				continue
			}

			workerKey := fmt.Sprintf("%s:%s", configID, endpoint.InstanceID)
			worker := NewWorker(&configCopy, endpoint, service)

			// Asignar broker manager para publicación MQTT
			worker.SetBrokerManager(e.brokerManager)

			if e.onResult != nil {
				worker.OnResult(e.onResult)
			}

			if err := worker.Start(e.ctx); err != nil {
				fmt.Printf("   ⚠️  Error starting worker %s: %v\n", endpoint.Label, err)
				continue
			}

			e.workers[workerKey] = worker
			workersStarted++
		}

		// Actualizar status a active si se iniciaron workers
		if workersStarted > 0 {
			e.updateConfigStatus(configID, "active")
			fmt.Printf("   ✅ Started %d workers for %s/%s\n", workersStarted, config.Provider, config.SiteCode)
		} else {
			fmt.Printf("   ⚠️  No workers started for %s/%s\n", config.Provider, config.SiteCode)
		}
	}

	fmt.Printf("\n✅ Polling Engine restored %d configs with %d total workers\n", len(e.configs), len(e.workers))
	return nil
}
