package main

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"omniapi/internal/adapters"
	"omniapi/internal/api/handlers"
	"omniapi/internal/config"
	"omniapi/internal/database"
	"omniapi/internal/queue/requester"
	"omniapi/internal/queue/status"
	"omniapi/internal/router"
	"omniapi/internal/services"
	"omniapi/internal/websocket"

	"github.com/joho/godotenv"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

func main() {
	fmt.Println("🚀 Starting OmniAPI Server...")

	// Cargar variables de entorno desde .env
	if err := godotenv.Load(); err != nil {
		log.Printf("⚠️  Warning: No .env file found or error loading it: %v", err)
	}

	// Cargar configuración extendida
	cfg, err := config.LoadConfig()
	if err != nil {
		log.Fatalf("❌ Error loading Configuration: %v", err)
	}

	// Registrar todos los adaptadores
	if err := adapters.RegisterAllAdapters(); err != nil {
		log.Fatalf("❌ Error registering adapters: %v", err)
	}

	// Mostrar resumen de configuración
	cfg.LogConfigSummary()

	// Conectar a MongoDB
	timeout, _ := time.ParseDuration(cfg.MongoDB.Timeout)
	mongoConfig := database.MongoConfig{
		URI:      cfg.MongoDB.URI,
		Database: cfg.MongoDB.Database,
		Timeout:  timeout,
	}

	fmt.Printf("🔌 Connecting to MongoDB: %s/%s\n", cfg.MongoDB.URI, cfg.MongoDB.Database)
	if err := database.Connect(mongoConfig); err != nil {
		log.Fatalf("❌ Error conectando a MongoDB: %v", err)
	}
	fmt.Println("✅ MongoDB connection established")

	// Inicializar servicios de MongoDB
	handlers.InitServices()

	// Verificar si existe un usuario administrador
	fmt.Println("\n🔐 Checking admin user...")
	adminExists, err := services.CheckAdminExists()
	if err != nil {
		log.Printf("⚠️  Warning: Could not check admin user: %v", err)
	} else if !adminExists {
		fmt.Println("⚠️  No admin user found. Please complete setup via /api/auth/setup")
	} else {
		fmt.Println("✅ Admin user exists")
	}

	// Crear contexto global con cancelación
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// ═══════════════════════════════════════════════════════════
	// FASE 1: Crear Router (núcleo del sistema)
	// ═══════════════════════════════════════════════════════════
	fmt.Println("\n📡 Initializing Router...")
	r := router.NewRouter()

	// Iniciar router
	if err := r.Start(ctx); err != nil {
		log.Fatalf("❌ Error starting router: %v", err)
	}
	fmt.Println("✅ Router started successfully")

	// ═══════════════════════════════════════════════════════════
	// FASE 2: Crear Requesters (uno por provider-site)
	// ═══════════════════════════════════════════════════════════
	fmt.Printf("\n🔄 Building Requesters from %d connections...\n", len(cfg.Connections))

	requesters := make(map[string]requester.Requester) // Key: provider:tenantId:siteId
	streamTracker := status.NewStreamTracker()

	for _, connCfg := range cfg.Connections {
		// Solo procesar conexiones activas
		if connCfg.Status != "active" {
			continue
		}

		// Obtener site_id del config
		siteID, ok := connCfg.Config["site_id"].(string)
		if !ok {
			log.Printf("⚠️  Connection %s missing site_id in config, skipping", connCfg.ID)
			continue
		}

		// Determinar estrategia según tipo de conector
		var strategy requester.Strategy
		switch connCfg.TypeID {
		case "scaleaq-cloud":
			// Obtener credenciales de config
			apiKey, _ := connCfg.Config["api_key"].(string)
			endpoint, _ := connCfg.Config["endpoint"].(string)
			strategy = requester.NewScaleAQCloudStrategy(endpoint, apiKey)
		case "process-api":
			endpoint, _ := connCfg.Config["endpoint"].(string)
			strategy = requester.NewProcessAPIStrategy(endpoint)
		default:
			// Usar NoOp para tipos no implementados o de prueba
			strategy = requester.NewNoOpStrategy()
		}

		// Configurar requester desde app.yaml
		reqConfig := requester.Config{
			RequestTimeout:       time.Duration(cfg.App.Requester.TimeoutSeconds) * time.Second,
			MaxConsecutiveErrors: cfg.App.Requester.CircuitBreaker.FailuresThreshold,
			CircuitPauseDuration: time.Duration(cfg.App.Requester.CircuitBreaker.PauseMinutes) * time.Minute,
			MaxQueueSize:         1000,
			CoalescingEnabled:    true,
		}

		// Configurar backoff steps
		if len(cfg.App.Requester.BackoffSeconds) >= 3 {
			reqConfig.BackoffInitial = time.Duration(cfg.App.Requester.BackoffSeconds[0]) * time.Second
			reqConfig.BackoffStep2 = time.Duration(cfg.App.Requester.BackoffSeconds[1]) * time.Second
			reqConfig.BackoffStep3 = time.Duration(cfg.App.Requester.BackoffSeconds[2]) * time.Second
		} else {
			// Defaults
			reqConfig.BackoffInitial = 60 * time.Second
			reqConfig.BackoffStep2 = 120 * time.Second
			reqConfig.BackoffStep3 = 300 * time.Second
		}

		// Crear requester
		req := requester.NewSequentialRequester(reqConfig, strategy)

		// Registrar callback para resultados → Router
		req.OnResult(func(result requester.Result) {
			r.OnRequesterResult(result)
		})

		// Iniciar requester
		if err := req.Start(ctx); err != nil {
			log.Printf("❌ Error starting requester for %s: %v", connCfg.ID, err)
			continue
		}

		// Registrar streams en tracker (por cada métrica soportada)
		// Asumimos que cada conector soporta ciertas métricas según su tipo
		metrics := []string{"feeding", "biometric", "climate"} // Métricas genéricas
		for _, metric := range metrics {
			streamKey := status.StreamKey{
				TenantID: connCfg.TenantID,
				SiteID:   siteID,
				CageID:   nil, // Puede ser más específico según el conector
				Metric:   metric,
				Source:   string(requester.SourceCloud),
			}
			streamTracker.RegisterStream(streamKey)
		}

		// Guardar referencia
		key := fmt.Sprintf("%s:%s:%s", connCfg.TypeID, connCfg.TenantID, siteID)
		requesters[key] = req

		fmt.Printf("  ✓ Requester '%s' [%s] started (timeout=%ds, backoff=%v, cb_threshold=%d)\n",
			connCfg.DisplayName,
			connCfg.TypeID,
			cfg.App.Requester.TimeoutSeconds,
			cfg.App.Requester.BackoffSeconds,
			cfg.App.Requester.CircuitBreaker.FailuresThreshold,
		)
	}

	fmt.Printf("✅ %d Requesters initialized\n", len(requesters))

	// ═══════════════════════════════════════════════════════════
	// FASE 3: Crear StatusPusher
	// ═══════════════════════════════════════════════════════════
	fmt.Printf("\n💓 Initializing StatusPusher (heartbeat=%ds)...\n", cfg.App.Status.HeartbeatSeconds)

	statusConfig := status.Config{
		HeartbeatInterval:      time.Duration(cfg.App.Status.HeartbeatSeconds) * time.Second,
		StaleThresholdOK:       30,  // 30 segundos
		StaleThresholdDegraded: 120, // 2 minutos
		MaxConsecutiveErrors:   5,
	}

	statusPusher := status.NewStatusPusher(statusConfig, streamTracker)

	// Registrar callback para heartbeats → Router
	statusPusher.OnEmit(func(st status.Status) {
		r.OnStatusHeartbeat(st)
	})

	// Iniciar status pusher
	if err := statusPusher.Start(ctx); err != nil {
		log.Fatalf("❌ Error starting status pusher: %v", err)
	}
	fmt.Printf("✅ StatusPusher started (interval=%ds)\n", cfg.App.Status.HeartbeatSeconds)

	// ═══════════════════════════════════════════════════════════
	// FASE 4: Crear WebSocket Hub conectado al Router
	// ═══════════════════════════════════════════════════════════
	fmt.Println("\n🔌 Initializing WebSocket Hub...")
	wsHub := websocket.NewHub(r)
	go wsHub.Run()
	fmt.Println("✅ WebSocket Hub started")

	// ═══════════════════════════════════════════════════════════
	// FASE 5: Iniciar actualización periódica de métricas
	// ═══════════════════════════════════════════════════════════
	fmt.Println("\n📊 Starting Prometheus metrics collector...")
	go func() {
		ticker := time.NewTicker(5 * time.Second) // Actualizar cada 5 segundos
		defer ticker.Stop()

		for {
			select {
			case <-ctx.Done():
				return
			case <-ticker.C:
				// Actualizar métricas de requesters
				for _, req := range requesters {
					m := req.GetMetrics()
					state := req.GetState()

					// Las métricas se actualizarán automáticamente
					// a través de los callbacks OnResult y los wrappers
					_ = m
					_ = state
				}
			}
		}
	}()
	fmt.Println("✅ Prometheus metrics collector started")

	// ═══════════════════════════════════════════════════════════
	// Configurar cierre graceful
	// ═══════════════════════════════════════════════════════════
	c := make(chan os.Signal, 1)
	signal.Notify(c, os.Interrupt, syscall.SIGTERM)
	go func() {
		<-c
		fmt.Println("\n� Shutting down server...")

		// Cancelar contexto para detener todos los componentes
		cancel()

		// Esperar un poco para que se completen las operaciones
		time.Sleep(1 * time.Second)

		// Cerrar MongoDB
		fmt.Println("🔄 Cerrando conexión MongoDB...")
		database.Disconnect()

		fmt.Println("✅ Server stopped gracefully")
		os.Exit(0)
	}()

	// Configurar rutas HTTP básicas
	http.HandleFunc("/", handlers.HomeHandler)
	http.HandleFunc("/api/health", handlers.HealthHandler)
	http.HandleFunc("/api/info", handlers.InfoHandler)
	http.HandleFunc("/api/time", handlers.TimeHandler)

	// Configurar rutas de autenticación
	http.HandleFunc("/api/auth/login", handlers.CORSMiddleware(handlers.LoginHandler))
	http.HandleFunc("/api/auth/register", handlers.CORSMiddleware(handlers.RegisterHandler))
	http.HandleFunc("/api/auth/setup/check", handlers.CORSMiddleware(handlers.CheckSetupHandler))
	http.HandleFunc("/api/auth/setup", handlers.CORSMiddleware(handlers.SetupHandler))

	// Configurar rutas de servicios externos
	http.HandleFunc("/api/services", handlers.CORSMiddleware(handlers.GetServicesHandler))
	http.HandleFunc("/api/services/get", handlers.CORSMiddleware(handlers.GetServiceHandler))
	http.HandleFunc("/api/services/create", handlers.CORSMiddleware(handlers.CreateServiceHandler))
	http.HandleFunc("/api/services/update", handlers.CORSMiddleware(handlers.UpdateServiceHandler))
	http.HandleFunc("/api/services/delete", handlers.CORSMiddleware(handlers.DeleteServiceHandler))
	http.HandleFunc("/api/services/test", handlers.CORSMiddleware(handlers.TestServiceConnectionHandler))

	// Configurar rutas de tenants (empresas salmoneras)
	http.HandleFunc("/api/tenants", handlers.CORSMiddleware(handlers.GetTenantsHandler))
	http.HandleFunc("/api/tenants/get", handlers.CORSMiddleware(handlers.GetTenantHandler))
	http.HandleFunc("/api/tenants/create", handlers.CORSMiddleware(handlers.CreateTenantHandler))
	http.HandleFunc("/api/tenants/update", handlers.CORSMiddleware(handlers.UpdateTenantHandler))
	http.HandleFunc("/api/tenants/delete", handlers.CORSMiddleware(handlers.DeleteTenantHandler))

	// Configurar rutas de sites (centros de cultivo)
	http.HandleFunc("/api/sites", handlers.CORSMiddleware(handlers.GetSitesHandler))
	http.HandleFunc("/api/sites/get", handlers.CORSMiddleware(handlers.GetSiteHandler))
	http.HandleFunc("/api/sites/create", handlers.CORSMiddleware(handlers.CreateSiteHandler))
	http.HandleFunc("/api/sites/update", handlers.CORSMiddleware(handlers.UpdateSiteHandler))
	http.HandleFunc("/api/sites/delete", handlers.CORSMiddleware(handlers.DeleteSiteHandler))

	// Configurar rutas de external services (servicios externos)
	http.HandleFunc("/api/external-services", handlers.CORSMiddleware(handlers.GetExternalServicesHandler))
	http.HandleFunc("/api/external-services/get", handlers.CORSMiddleware(handlers.GetExternalServiceHandler))
	http.HandleFunc("/api/external-services/create", handlers.CORSMiddleware(handlers.CreateExternalServiceHandler))
	http.HandleFunc("/api/external-services/update", handlers.CORSMiddleware(handlers.UpdateExternalServiceHandler))
	http.HandleFunc("/api/external-services/delete", handlers.CORSMiddleware(handlers.DeleteExternalServiceHandler))
	http.HandleFunc("/api/external-services/test", handlers.CORSMiddleware(handlers.TestExternalServiceConnectionHandler))

	// Configurar rutas de MongoDB API
	http.HandleFunc("/api/users", handlers.GetUsersHandler)
	http.HandleFunc("/api/users/create", handlers.CreateUserHandler)
	http.HandleFunc("/api/users/get", handlers.GetUserHandler)
	http.HandleFunc("/api/users/update", handlers.UpdateUserHandler)
	http.HandleFunc("/api/users/delete", handlers.DeleteUserHandler)
	http.HandleFunc("/api/messages", handlers.GetMessagesHandler)
	http.HandleFunc("/api/messages/create", handlers.CreateMessageHandler)
	http.HandleFunc("/api/database/stats", handlers.GetDatabaseStatsHandler)

	// Configurar rutas de Schema Validation
	http.HandleFunc("/api/schemas", handlers.ListSchemasHandler)
	http.HandleFunc("/api/schemas/get", handlers.GetSchemaHandler)
	http.HandleFunc("/api/schemas/validate", handlers.ValidateSchemaHandler)

	// Configurar rutas del builder/discovery
	http.HandleFunc("/api/discovery/runs", handlers.CORSMiddleware(handlers.DiscoveryRunsHandler))

	// Configurar rutas WebSocket
	http.HandleFunc("/ws", func(w http.ResponseWriter, r *http.Request) {
		websocket.WSHandler(wsHub, w, r)
	})
	http.HandleFunc("/ws/stats", func(w http.ResponseWriter, r *http.Request) {
		websocket.WSStatsHandler(wsHub, w, r)
	})
	http.HandleFunc("/ws/test", websocket.WSTestHandler)

	// Página de integración WebSocket
	http.HandleFunc("/websocket", handlers.WSTestPageHandler)

	// ═══════════════════════════════════════════════════════════
	// Endpoint de Métricas Prometheus
	// ═══════════════════════════════════════════════════════════
	http.Handle("/metrics", promhttp.Handler())

	// Información de inicio
	fmt.Println("\n🚀 OmniAPI Server Started Successfully")
	fmt.Printf("📍 Port: %s\n", cfg.Port)
	fmt.Printf("🌍 Environment: %s\n", cfg.Environment)
	fmt.Printf("📊 Log Level: %s\n", cfg.LogLevel)
	fmt.Printf("🗄️  MongoDB: %s (Database: %s)\n", cfg.MongoDB.URI, cfg.MongoDB.Database)
	fmt.Printf("👥 Tenants Loaded: %d\n", len(cfg.Tenants))
	fmt.Printf("🔗 Connections Loaded: %d\n", len(cfg.Connections))
	fmt.Printf("🗺️  Mappings Loaded: %d\n", len(cfg.Mappings))
	fmt.Println("═══════════════════════════════════════════════════")
	fmt.Printf("🌐 Main Page: http://localhost:%s\n", cfg.Port)
	fmt.Printf("🏥 API Health: http://localhost:%s/api/health\n", cfg.Port)
	fmt.Printf("ℹ️  API Info: http://localhost:%s/api/info\n", cfg.Port)
	fmt.Printf("🕐 API Time: http://localhost:%s/api/time\n", cfg.Port)
	fmt.Println("───────────── MongoDB API Endpoints ──────────────")
	fmt.Printf("👥 Users API: http://localhost:%s/api/users\n", cfg.Port)
	fmt.Printf("💬 Messages API: http://localhost:%s/api/messages\n", cfg.Port)
	fmt.Printf("📊 DB Stats: http://localhost:%s/api/database/stats\n", cfg.Port)
	fmt.Println("───────────── Schema Validation API ───────────────")
	fmt.Printf("📋 List Schemas: http://localhost:%s/api/schemas\n", cfg.Port)
	fmt.Printf("🔍 Get Schema: http://localhost:%s/api/schemas/get?kind=feeding&version=v1\n", cfg.Port)
	fmt.Printf("✅ Validate Data: http://localhost:%s/api/schemas/validate\n", cfg.Port)
	fmt.Println("───────────── WebSocket Endpoints ─────────────────")
	fmt.Printf("🔗 WebSocket: ws://localhost:%s/ws\n", cfg.Port)
	fmt.Printf("🧪 Test Client: http://localhost:%s/ws/test\n", cfg.Port)
	fmt.Printf("📊 WS Stats: http://localhost:%s/ws/stats\n", cfg.Port)
	fmt.Printf("📖 WS Integration: http://localhost:%s/websocket\n", cfg.Port)
	fmt.Println("───────────── Monitoring Endpoints ────────────────")
	fmt.Printf("📈 Prometheus Metrics: http://localhost:%s/metrics\n", cfg.Port)
	fmt.Println("═══════════════════════════════════════════════════")

	// Iniciar servidor
	serverPort := fmt.Sprintf(":%s", cfg.Port)
	fmt.Printf("\n🎯 Server listening on port %s\n", cfg.Port)
	fmt.Println("🔥 Press Ctrl+C to stop the server")
	log.Fatal(http.ListenAndServe(serverPort, nil))
}
