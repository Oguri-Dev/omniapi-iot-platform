package main

import (
	"context"
	"fmt"
	"log"
	"time"

	"omniapi/database"
	"omniapi/models"

	"github.com/joho/godotenv"
	"golang.org/x/crypto/bcrypt"
)

func main() {
	fmt.Println("🔧 Creando usuario de prueba...")

	// Cargar .env
	if err := godotenv.Load(); err != nil {
		log.Printf("⚠️  No se encontró archivo .env")
	}

	// Conectar a MongoDB
	mongoConfig := database.MongoConfig{
		URI:      "mongodb://localhost:27017",
		Database: "omniapi",
		Timeout:  10 * time.Second,
	}

	if err := database.Connect(mongoConfig); err != nil {
		log.Fatalf("❌ Error conectando a MongoDB: %v", err)
	}
	defer database.Disconnect()

	// Hash de contraseña
	hashedPassword, err := bcrypt.GenerateFromPassword([]byte("admin123"), bcrypt.DefaultCost)
	if err != nil {
		log.Fatalf("❌ Error generando hash: %v", err)
	}

	// Crear usuario admin
	user := models.User{
		Username:  "admin",
		Email:     "admin@omniapi.com",
		Password:  string(hashedPassword),
		FullName:  "Administrador",
		Status:    "active",
		Role:      "admin",
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}

	collection := database.GetCollection("users")
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	result, err := collection.InsertOne(ctx, user)
	if err != nil {
		log.Fatalf("❌ Error creando usuario: %v", err)
	}

	fmt.Printf("✅ Usuario creado exitosamente\n")
	fmt.Printf("   ID: %v\n", result.InsertedID)
	fmt.Printf("   Username: %s\n", user.Username)
	fmt.Printf("   Password: admin123\n")
	fmt.Printf("   Email: %s\n", user.Email)
	fmt.Printf("   Role: %s\n", user.Role)
}
