package services

import (
	"context"
	"fmt"
	"time"

	"omniapi/internal/database"
	"omniapi/internal/models"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo/options"
	"golang.org/x/crypto/bcrypt"
)

// UserService servicio para operaciones de usuarios
type UserService struct {
	collection string
}

// NewUserService crea una nueva instancia del servicio de usuarios
func NewUserService() *UserService {
	return &UserService{
		collection: "users",
	}
}

// Create crea un nuevo usuario
func (us *UserService) Create(user *models.User) error {
	collection := database.GetCollection(us.collection)
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	// Establecer timestamps
	user.ID = primitive.NewObjectID()
	user.CreatedAt = time.Now()
	user.UpdatedAt = time.Now()

	if user.Status == "" {
		user.Status = "active"
	}
	if user.Role == "" {
		user.Role = "user"
	}

	// Verificar si el usuario ya existe
	exists, err := us.ExistsByUsername(user.Username)
	if err != nil {
		return err
	}
	if exists {
		return fmt.Errorf("el usuario '%s' ya existe", user.Username)
	}

	// Verificar si el email ya existe
	existsEmail, err := us.ExistsByEmail(user.Email)
	if err != nil {
		return err
	}
	if existsEmail {
		return fmt.Errorf("el email '%s' ya está registrado", user.Email)
	}

	_, err = collection.InsertOne(ctx, user)
	return err
}

// GetByID obtiene un usuario por ID
func (us *UserService) GetByID(id string) (*models.User, error) {
	collection := database.GetCollection(us.collection)
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	objectID, err := primitive.ObjectIDFromHex(id)
	if err != nil {
		return nil, fmt.Errorf("ID inválido: %v", err)
	}

	var user models.User
	err = collection.FindOne(ctx, bson.M{"_id": objectID}).Decode(&user)
	if err != nil {
		return nil, err
	}

	return &user, nil
}

// GetByUsername obtiene un usuario por nombre de usuario
func (us *UserService) GetByUsername(username string) (*models.User, error) {
	collection := database.GetCollection(us.collection)
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	var user models.User
	err := collection.FindOne(ctx, bson.M{"username": username}).Decode(&user)
	if err != nil {
		return nil, err
	}

	return &user, nil
}

// GetByEmail obtiene un usuario por email
func (us *UserService) GetByEmail(email string) (*models.User, error) {
	collection := database.GetCollection(us.collection)
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	var user models.User
	err := collection.FindOne(ctx, bson.M{"email": email}).Decode(&user)
	if err != nil {
		return nil, err
	}

	return &user, nil
}

// Update actualiza un usuario
func (us *UserService) Update(id string, updates bson.M) error {
	collection := database.GetCollection(us.collection)
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	objectID, err := primitive.ObjectIDFromHex(id)
	if err != nil {
		return fmt.Errorf("ID inválido: %v", err)
	}

	// Agregar timestamp de actualización
	updates["updated_at"] = time.Now()

	_, err = collection.UpdateOne(
		ctx,
		bson.M{"_id": objectID},
		bson.M{"$set": updates},
	)

	return err
}

// Delete elimina un usuario (soft delete cambiando status)
func (us *UserService) Delete(id string) error {
	return us.Update(id, bson.M{"status": "deleted"})
}

// List obtiene lista paginada de usuarios
func (us *UserService) List(page, perPage int, filter bson.M) ([]models.User, int64, error) {
	collection := database.GetCollection(us.collection)
	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()

	// Calcular skip
	skip := (page - 1) * perPage

	// Opciones de consulta
	opts := options.Find()
	opts.SetSkip(int64(skip))
	opts.SetLimit(int64(perPage))
	opts.SetSort(bson.D{{Key: "created_at", Value: -1}})

	// Contar total
	total, err := collection.CountDocuments(ctx, filter)
	if err != nil {
		return nil, 0, err
	}

	// Buscar usuarios
	cursor, err := collection.Find(ctx, filter, opts)
	if err != nil {
		return nil, 0, err
	}
	defer cursor.Close(ctx)

	var users []models.User
	if err = cursor.All(ctx, &users); err != nil {
		return nil, 0, err
	}

	return users, total, nil
}

// ExistsByUsername verifica si existe un usuario con ese username
func (us *UserService) ExistsByUsername(username string) (bool, error) {
	collection := database.GetCollection(us.collection)
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	count, err := collection.CountDocuments(ctx, bson.M{"username": username})
	if err != nil {
		return false, err
	}

	return count > 0, nil
}

// ExistsByEmail verifica si existe un usuario con ese email
func (us *UserService) ExistsByEmail(email string) (bool, error) {
	collection := database.GetCollection(us.collection)
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	count, err := collection.CountDocuments(ctx, bson.M{"email": email})
	if err != nil {
		return false, err
	}

	return count > 0, nil
}

// UpdateLastLogin actualiza el timestamp de último login
func (us *UserService) UpdateLastLogin(id string) error {
	now := time.Now()
	return us.Update(id, bson.M{"last_login": now})
}

// GetActiveUsers obtiene usuarios activos
func (us *UserService) GetActiveUsers() ([]models.User, error) {
	users, _, err := us.List(1, 1000, bson.M{"status": "active"})
	return users, err
}

// SearchUsers busca usuarios por texto
func (us *UserService) SearchUsers(query string, page, perPage int) ([]models.User, int64, error) {
	filter := bson.M{
		"$or": []bson.M{
			{"username": bson.M{"$regex": query, "$options": "i"}},
			{"full_name": bson.M{"$regex": query, "$options": "i"}},
			{"email": bson.M{"$regex": query, "$options": "i"}},
		},
		"status": bson.M{"$ne": "deleted"},
	}

	return us.List(page, perPage, filter)
}

// CheckAdminExists verifica si existe al menos un usuario administrador
// Retorna true si existe, false si no existe
func CheckAdminExists() (bool, error) {
	collection := database.GetCollection("users")
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	// Buscar si existe algún usuario admin
	count, err := collection.CountDocuments(ctx, bson.M{"role": "admin", "status": "active"})
	if err != nil {
		return false, fmt.Errorf("error verificando usuarios admin: %v", err)
	}

	return count > 0, nil
}

// CreateFirstAdmin crea el primer usuario administrador del sistema
// Solo funciona si no existe ningún admin en la base de datos
func CreateFirstAdmin(username, email, password, fullName string) (*models.User, error) {
	collection := database.GetCollection("users")
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	// Verificar que no exista ningún admin
	count, err := collection.CountDocuments(ctx, bson.M{"role": "admin", "status": "active"})
	if err != nil {
		return nil, fmt.Errorf("error verificando usuarios admin: %v", err)
	}

	if count > 0 {
		return nil, fmt.Errorf("ya existe un usuario administrador en el sistema")
	}

	// Hashear contraseña
	hashedPassword, err := hashPassword(password)
	if err != nil {
		return nil, fmt.Errorf("error generando hash de contraseña: %v", err)
	}

	// Crear usuario admin
	newAdmin := models.User{
		ID:        primitive.NewObjectID(),
		Username:  username,
		Email:     email,
		Password:  hashedPassword,
		FullName:  fullName,
		Status:    "active",
		Role:      "admin",
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}

	_, err = collection.InsertOne(ctx, newAdmin)
	if err != nil {
		return nil, fmt.Errorf("error creando usuario admin: %v", err)
	}

	fmt.Printf("✅ First admin user created: %s (%s)\n", newAdmin.Username, newAdmin.Email)

	return &newAdmin, nil
}

// hashPassword función auxiliar para hashear contraseñas
func hashPassword(password string) (string, error) {
	hashedBytes, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return "", err
	}
	return string(hashedBytes), nil
}
