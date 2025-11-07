package main

import (
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"omniapi/adapters"
	"omniapi/config"
	"omniapi/database"
	"omniapi/handlers"
	"omniapi/websocket"

	"github.com/joho/godotenv"
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
		log.Fatalf("❌ Error loading configuration: %v", err)
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

	// Configurar cierre graceful de MongoDB
	c := make(chan os.Signal, 1)
	signal.Notify(c, os.Interrupt, syscall.SIGTERM)
	go func() {
		<-c
		fmt.Println("\n🔄 Cerrando conexión MongoDB...")
		database.Disconnect()
		os.Exit(0)
	}()

	// Inicializar servicios de MongoDB
	handlers.InitServices()

	// Crear y iniciar WebSocket Hub
	wsHub := websocket.NewHub()
	go wsHub.Run()

	// Configurar rutas HTTP básicas
	http.HandleFunc("/", handlers.HomeHandler)
	http.HandleFunc("/api/health", handlers.HealthHandler)
	http.HandleFunc("/api/info", handlers.InfoHandler)
	http.HandleFunc("/api/time", handlers.TimeHandler)

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
	fmt.Println("═══════════════════════════════════════════════════")

	// Iniciar servidor
	serverPort := fmt.Sprintf(":%s", cfg.Port)
	fmt.Printf("\n🎯 Server listening on port %s\n", cfg.Port)
	fmt.Println("🔥 Press Ctrl+C to stop the server")
	log.Fatal(http.ListenAndServe(serverPort, nil))
}
