package handlers

import (
	"encoding/json"
	"net/http"
	"strconv"
	"time"

	"omniapi/models"
	"omniapi/services"

	"go.mongodb.org/mongo-driver/bson"
)

// Servicios globales
var (
	userService    *services.UserService
	messageService *services.MessageService
)

// InitServices inicializa los servicios de MongoDB
func InitServices() {
	userService = services.NewUserService()
	messageService = services.NewMessageService()
}

// CreateUserHandler crea un nuevo usuario
func CreateUserHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Método no permitido", http.StatusMethodNotAllowed)
		return
	}

	var user models.User
	if err := json.NewDecoder(r.Body).Decode(&user); err != nil {
		respondWithError(w, "JSON inválido", http.StatusBadRequest)
		return
	}

	// Validaciones básicas
	if user.Username == "" || user.Email == "" {
		respondWithError(w, "Username y email son requeridos", http.StatusBadRequest)
		return
	}

	if err := userService.Create(&user); err != nil {
		respondWithError(w, err.Error(), http.StatusConflict)
		return
	}

	respondWithSuccess(w, "Usuario creado exitosamente", user)
}

// GetUserHandler obtiene un usuario por ID
func GetUserHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Método no permitido", http.StatusMethodNotAllowed)
		return
	}

	userID := r.URL.Query().Get("id")
	if userID == "" {
		respondWithError(w, "ID de usuario requerido", http.StatusBadRequest)
		return
	}

	user, err := userService.GetByID(userID)
	if err != nil {
		respondWithError(w, "Usuario no encontrado", http.StatusNotFound)
		return
	}

	respondWithSuccess(w, "Usuario encontrado", user)
}

// GetUsersHandler obtiene lista paginada de usuarios
func GetUsersHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Método no permitido", http.StatusMethodNotAllowed)
		return
	}

	// Parámetros de paginación
	page, _ := strconv.Atoi(r.URL.Query().Get("page"))
	if page < 1 {
		page = 1
	}

	perPage, _ := strconv.Atoi(r.URL.Query().Get("per_page"))
	if perPage < 1 || perPage > 100 {
		perPage = 20
	}

	// Filtros opcionales
	filter := bson.M{"status": bson.M{"$ne": "deleted"}}

	if status := r.URL.Query().Get("status"); status != "" {
		filter["status"] = status
	}

	if role := r.URL.Query().Get("role"); role != "" {
		filter["role"] = role
	}

	users, total, err := userService.List(page, perPage, filter)
	if err != nil {
		respondWithError(w, "Error obteniendo usuarios", http.StatusInternalServerError)
		return
	}

	// Crear información de paginación
	totalPages := int((total + int64(perPage) - 1) / int64(perPage))
	pagination := &models.PaginationInfo{
		Page:       page,
		PerPage:    perPage,
		Total:      total,
		TotalPages: totalPages,
		HasNext:    page < totalPages,
		HasPrev:    page > 1,
	}

	respondWithPagination(w, "Usuarios obtenidos", users, pagination)
}

// UpdateUserHandler actualiza un usuario
func UpdateUserHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPut {
		http.Error(w, "Método no permitido", http.StatusMethodNotAllowed)
		return
	}

	userID := r.URL.Query().Get("id")
	if userID == "" {
		respondWithError(w, "ID de usuario requerido", http.StatusBadRequest)
		return
	}

	var updates bson.M
	if err := json.NewDecoder(r.Body).Decode(&updates); err != nil {
		respondWithError(w, "JSON inválido", http.StatusBadRequest)
		return
	}

	// Remover campos que no se pueden actualizar directamente
	delete(updates, "_id")
	delete(updates, "created_at")
	delete(updates, "updated_at")

	if err := userService.Update(userID, updates); err != nil {
		respondWithError(w, err.Error(), http.StatusInternalServerError)
		return
	}

	respondWithSuccess(w, "Usuario actualizado exitosamente", nil)
}

// DeleteUserHandler elimina (soft delete) un usuario
func DeleteUserHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodDelete {
		http.Error(w, "Método no permitido", http.StatusMethodNotAllowed)
		return
	}

	userID := r.URL.Query().Get("id")
	if userID == "" {
		respondWithError(w, "ID de usuario requerido", http.StatusBadRequest)
		return
	}

	if err := userService.Delete(userID); err != nil {
		respondWithError(w, err.Error(), http.StatusInternalServerError)
		return
	}

	respondWithSuccess(w, "Usuario eliminado exitosamente", nil)
}

// CreateMessageHandler crea un nuevo mensaje
func CreateMessageHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Método no permitido", http.StatusMethodNotAllowed)
		return
	}

	var message models.Message
	if err := json.NewDecoder(r.Body).Decode(&message); err != nil {
		respondWithError(w, "JSON inválido", http.StatusBadRequest)
		return
	}

	// Validaciones básicas
	if message.Content == "" {
		respondWithError(w, "Contenido del mensaje requerido", http.StatusBadRequest)
		return
	}

	if message.Type == "" {
		message.Type = "chat"
	}

	if err := messageService.Create(&message); err != nil {
		respondWithError(w, err.Error(), http.StatusInternalServerError)
		return
	}

	respondWithSuccess(w, "Mensaje creado exitosamente", message)
}

// GetMessagesHandler obtiene mensajes con paginación
func GetMessagesHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Método no permitido", http.StatusMethodNotAllowed)
		return
	}

	// Parámetros de consulta
	channel := r.URL.Query().Get("channel")
	if channel == "" {
		channel = "general"
	}

	page, _ := strconv.Atoi(r.URL.Query().Get("page"))
	if page < 1 {
		page = 1
	}

	perPage, _ := strconv.Atoi(r.URL.Query().Get("per_page"))
	if perPage < 1 || perPage > 100 {
		perPage = 50
	}

	messages, total, err := messageService.GetByChannel(channel, page, perPage)
	if err != nil {
		respondWithError(w, "Error obteniendo mensajes", http.StatusInternalServerError)
		return
	}

	// Crear información de paginación
	totalPages := int((total + int64(perPage) - 1) / int64(perPage))
	pagination := &models.PaginationInfo{
		Page:       page,
		PerPage:    perPage,
		Total:      total,
		TotalPages: totalPages,
		HasNext:    page < totalPages,
		HasPrev:    page > 1,
	}

	respondWithPagination(w, "Mensajes obtenidos", messages, pagination)
}

// GetDatabaseStatsHandler obtiene estadísticas de la base de datos
func GetDatabaseStatsHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Método no permitido", http.StatusMethodNotAllowed)
		return
	}

	// Obtener estadísticas de mensajes
	messageStats, err := messageService.GetMessageStats()
	if err != nil {
		respondWithError(w, "Error obteniendo estadísticas de mensajes", http.StatusInternalServerError)
		return
	}

	// Obtener usuarios activos
	activeUsers, err := userService.GetActiveUsers()
	if err != nil {
		respondWithError(w, "Error obteniendo usuarios activos", http.StatusInternalServerError)
		return
	}

	stats := map[string]interface{}{
		"database":     "omniapi",
		"active_users": len(activeUsers),
		"messages":     messageStats,
		"timestamp":    time.Now().Unix(),
	}

	respondWithSuccess(w, "Estadísticas de la base de datos", stats)
}

// Funciones auxiliares para respuestas JSON

func respondWithError(w http.ResponseWriter, message string, statusCode int) {
	response := models.APIResponse{
		Success:   false,
		Message:   message,
		Timestamp: time.Now().Unix(),
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(statusCode)
	json.NewEncoder(w).Encode(response)
}

func respondWithSuccess(w http.ResponseWriter, message string, data interface{}) {
	response := models.APIResponse{
		Success:   true,
		Message:   message,
		Data:      data,
		Timestamp: time.Now().Unix(),
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(response)
}

func respondWithPagination(w http.ResponseWriter, message string, data interface{}, pagination *models.PaginationInfo) {
	response := models.APIResponse{
		Success:    true,
		Message:    message,
		Data:       data,
		Pagination: pagination,
		Timestamp:  time.Now().Unix(),
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(response)
}
