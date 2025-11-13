package handlers

import (
	"context"
	"crypto/rand"
	"encoding/base64"
	"encoding/json"
	"net/http"
	"time"

	"omniapi/database"
	"omniapi/models"
	"omniapi/services"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"golang.org/x/crypto/bcrypt"
)

// LoginRequest estructura para el login
type LoginRequest struct {
	Username string `json:"username"`
	Password string `json:"password"`
}

// LoginResponse estructura para la respuesta del login
type LoginResponse struct {
	Success bool   `json:"success"`
	Message string `json:"message"`
	Data    struct {
		Token string      `json:"token"`
		User  models.User `json:"user"`
	} `json:"data"`
	Timestamp int64 `json:"timestamp"`
}

// LoginHandler maneja el inicio de sesión
func LoginHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Método no permitido", http.StatusMethodNotAllowed)
		return
	}

	var req LoginRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		response := LoginResponse{
			Success:   false,
			Message:   "Datos inválidos",
			Timestamp: time.Now().Unix(),
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(response)
		return
	}

	// Buscar usuario en MongoDB
	collection := database.GetCollection("users")
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	var user models.User
	err := collection.FindOne(ctx, bson.M{"username": req.Username}).Decode(&user)
	if err != nil {
		response := LoginResponse{
			Success:   false,
			Message:   "Usuario o contraseña incorrectos",
			Timestamp: time.Now().Unix(),
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusUnauthorized)
		json.NewEncoder(w).Encode(response)
		return
	}

	// Verificar contraseña
	if err := bcrypt.CompareHashAndPassword([]byte(user.Password), []byte(req.Password)); err != nil {
		response := LoginResponse{
			Success:   false,
			Message:   "Usuario o contraseña incorrectos",
			Timestamp: time.Now().Unix(),
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusUnauthorized)
		json.NewEncoder(w).Encode(response)
		return
	}

	// Generar token
	token, err := generateToken()
	if err != nil {
		response := LoginResponse{
			Success:   false,
			Message:   "Error generando token",
			Timestamp: time.Now().Unix(),
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(response)
		return
	}

	// Crear sesión
	session := models.Session{
		UserID:    user.ID,
		Token:     token,
		IPAddress: r.RemoteAddr,
		UserAgent: r.UserAgent(),
		CreatedAt: time.Now(),
		ExpiresAt: time.Now().Add(24 * time.Hour),
		IsActive:  true,
	}

	sessionsCollection := database.GetCollection("sessions")
	_, err = sessionsCollection.InsertOne(ctx, session)
	if err != nil {
		response := LoginResponse{
			Success:   false,
			Message:   "Error creando sesión",
			Timestamp: time.Now().Unix(),
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(response)
		return
	}

	// Actualizar último login
	now := time.Now()
	user.LastLogin = &now
	collection.UpdateOne(ctx, bson.M{"_id": user.ID}, bson.M{"$set": bson.M{"last_login": now}})

	// Respuesta exitosa
	response := LoginResponse{
		Success:   true,
		Message:   "Login exitoso",
		Timestamp: time.Now().Unix(),
	}
	response.Data.Token = token
	response.Data.User = user

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(response)
}

// RegisterHandler maneja el registro de usuarios
func RegisterHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Método no permitido", http.StatusMethodNotAllowed)
		return
	}

	var user models.User
	if err := json.NewDecoder(r.Body).Decode(&user); err != nil {
		response := models.APIResponse{
			Success:   false,
			Message:   "Datos inválidos",
			Timestamp: time.Now().Unix(),
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(response)
		return
	}

	// Hash de la contraseña
	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(user.Password), bcrypt.DefaultCost)
	if err != nil {
		response := models.APIResponse{
			Success:   false,
			Message:   "Error procesando contraseña",
			Timestamp: time.Now().Unix(),
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(response)
		return
	}

	user.Password = string(hashedPassword)
	user.CreatedAt = time.Now()
	user.UpdatedAt = time.Now()
	user.Status = "active"
	if user.Role == "" {
		user.Role = "user"
	}

	// Insertar en MongoDB
	collection := database.GetCollection("users")
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	result, err := collection.InsertOne(ctx, user)
	if err != nil {
		response := models.APIResponse{
			Success:   false,
			Message:   "Error creando usuario: " + err.Error(),
			Timestamp: time.Now().Unix(),
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(response)
		return
	}

	user.ID = result.InsertedID.(primitive.ObjectID)

	response := models.APIResponse{
		Success:   true,
		Message:   "Usuario creado exitosamente",
		Data:      user,
		Timestamp: time.Now().Unix(),
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(response)
}

// generateToken genera un token aleatorio
func generateToken() (string, error) {
	b := make([]byte, 32)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	return base64.URLEncoding.EncodeToString(b), nil
}

// SetupRequest estructura para el setup inicial
type SetupRequest struct {
	Username string `json:"username"`
	Email    string `json:"email"`
	Password string `json:"password"`
	FullName string `json:"fullName"`
}

// CheckSetupHandler verifica si el sistema necesita configuración inicial
func CheckSetupHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Método no permitido", http.StatusMethodNotAllowed)
		return
	}

	adminExists, err := services.CheckAdminExists()
	if err != nil {
		response := models.APIResponse{
			Success:   false,
			Message:   "Error verificando sistema: " + err.Error(),
			Timestamp: time.Now().Unix(),
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(response)
		return
	}

	response := models.APIResponse{
		Success: true,
		Message: "Setup status",
		Data: map[string]bool{
			"needsSetup": !adminExists,
		},
		Timestamp: time.Now().Unix(),
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(response)
}

// SetupHandler maneja la configuración inicial del sistema (crear primer admin)
func SetupHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Método no permitido", http.StatusMethodNotAllowed)
		return
	}

	var req SetupRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		response := models.APIResponse{
			Success:   false,
			Message:   "Datos inválidos",
			Timestamp: time.Now().Unix(),
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(response)
		return
	}

	// Validaciones
	if req.Username == "" || req.Email == "" || req.Password == "" {
		response := models.APIResponse{
			Success:   false,
			Message:   "Todos los campos son obligatorios",
			Timestamp: time.Now().Unix(),
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(response)
		return
	}

	if len(req.Password) < 6 {
		response := models.APIResponse{
			Success:   false,
			Message:   "La contraseña debe tener al menos 6 caracteres",
			Timestamp: time.Now().Unix(),
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(response)
		return
	}

	// Crear primer admin
	admin, err := services.CreateFirstAdmin(req.Username, req.Email, req.Password, req.FullName)
	if err != nil {
		response := models.APIResponse{
			Success:   false,
			Message:   "Error creando administrador: " + err.Error(),
			Timestamp: time.Now().Unix(),
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(response)
		return
	}

	// No devolver la contraseña hasheada
	admin.Password = ""

	response := models.APIResponse{
		Success:   true,
		Message:   "Administrador creado exitosamente",
		Data:      admin,
		Timestamp: time.Now().Unix(),
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(response)
}
