package database

import (
	"context"
	"fmt"
	"log"
	"time"

	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

// MongoDB connection instance
var Client *mongo.Client
var Database *mongo.Database

// MongoDB configuration
type MongoConfig struct {
	URI      string
	Database string
	Timeout  time.Duration
}

// Connect establece la conexión con MongoDB
func Connect(config MongoConfig) error {
	// Configurar opciones de cliente
	clientOptions := options.Client().ApplyURI(config.URI)

	// Configurar timeouts
	clientOptions.SetConnectTimeout(config.Timeout)
	clientOptions.SetServerSelectionTimeout(config.Timeout)

	// Crear contexto con timeout
	ctx, cancel := context.WithTimeout(context.Background(), config.Timeout)
	defer cancel()

	// Conectar a MongoDB
	client, err := mongo.Connect(ctx, clientOptions)
	if err != nil {
		return fmt.Errorf("error conectando a MongoDB: %v", err)
	}

	// Verificar la conexión
	err = client.Ping(ctx, nil)
	if err != nil {
		return fmt.Errorf("error haciendo ping a MongoDB: %v", err)
	}

	// Asignar cliente y base de datos global
	Client = client
	Database = client.Database(config.Database)

	log.Printf("✅ Conectado exitosamente a MongoDB - Base de datos: %s", config.Database)
	return nil
}

// Disconnect cierra la conexión con MongoDB
func Disconnect() error {
	if Client == nil {
		return nil
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	err := Client.Disconnect(ctx)
	if err != nil {
		return fmt.Errorf("error desconectando de MongoDB: %v", err)
	}

	log.Println("🔌 Desconectado de MongoDB")
	return nil
}

// GetCollection obtiene una colección específica
func GetCollection(name string) *mongo.Collection {
	if Database == nil {
		log.Fatal("Base de datos no inicializada. Llama Connect() primero.")
	}
	return Database.Collection(name)
}

// HealthCheck verifica el estado de la conexión
func HealthCheck() error {
	if Client == nil {
		return fmt.Errorf("cliente MongoDB no inicializado")
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	err := Client.Ping(ctx, nil)
	if err != nil {
		return fmt.Errorf("MongoDB no responde: %v", err)
	}

	return nil
}

// GetDatabaseStats obtiene estadísticas de la base de datos
func GetDatabaseStats() (map[string]interface{}, error) {
	if Database == nil {
		return nil, fmt.Errorf("base de datos no inicializada")
	}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	// Obtener estadísticas de la base de datos
	var stats map[string]interface{}
	err := Database.RunCommand(ctx, map[string]interface{}{"dbStats": 1}).Decode(&stats)
	if err != nil {
		return nil, fmt.Errorf("error obteniendo estadísticas: %v", err)
	}

	return stats, nil
}

// ListCollections obtiene lista de colecciones
func ListCollections() ([]string, error) {
	if Database == nil {
		return nil, fmt.Errorf("base de datos no inicializada")
	}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	cursor, err := Database.ListCollectionNames(ctx, map[string]interface{}{})
	if err != nil {
		return nil, fmt.Errorf("error listando colecciones: %v", err)
	}

	return cursor, nil
}
