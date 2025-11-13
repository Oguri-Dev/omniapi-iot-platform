package main

import (
	"context"
	"fmt"
	"log"
	"time"

	"omniapi/config"
	"omniapi/database"

	"go.mongodb.org/mongo-driver/bson"
)

func main() {
	// Cargar configuración
	cfg, err := config.Load()
	if err != nil {
		log.Fatalf("Error loading config: %v", err)
	}

	// Configuración de MongoDB
	mongoConfig := database.MongoConfig{
		URI:      cfg.MongoDB.URI,
		Database: cfg.MongoDB.Database,
	}

	// Conectar a MongoDB
	fmt.Println("Connecting to MongoDB...")
	if err := database.Connect(mongoConfig); err != nil {
		log.Fatalf("Error connecting to MongoDB: %v", err)
	}
	defer database.Disconnect()

	// Eliminar usuarios admin
	collection := database.GetCollection("users")
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	result, err := collection.DeleteMany(ctx, bson.M{"role": "admin"})
	if err != nil {
		log.Fatalf("Error deleting admin users: %v", err)
	}

	fmt.Printf("✅ Deleted %d admin user(s)\n", result.DeletedCount)
}
