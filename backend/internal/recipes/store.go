package recipes

import (
	"context"
	"fmt"
	"sync"
	"time"

	"omniapi/internal/database"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

// FieldMapping mapeo de un campo de origen a destino
type FieldMapping struct {
	From      string `bson:"from" json:"from"`                               // Path origen (JSONPath-like)
	To        string `bson:"to" json:"to"`                                   // Campo destino
	Type      string `bson:"type" json:"type"`                               // string, number, boolean, array, object
	Transform string `bson:"transform,omitempty" json:"transform,omitempty"` // Transformación opcional
}

// StaticField campo con valor estático
type StaticField struct {
	Field string `bson:"field" json:"field"` // Nombre del campo
	Value string `bson:"value" json:"value"` // Valor (puede ser $NOW para timestamp)
}

// Recipe representa una receta de transformación de datos
type Recipe struct {
	ID            primitive.ObjectID `bson:"_id,omitempty" json:"id,omitempty"`
	Name          string             `bson:"name" json:"name"`
	Description   string             `bson:"description,omitempty" json:"description,omitempty"`
	Provider      string             `bson:"provider,omitempty" json:"provider,omitempty"`
	EndpointID    string             `bson:"endpoint_id,omitempty" json:"endpoint_id,omitempty"`
	InstanceID    string             `bson:"instance_id,omitempty" json:"instance_id,omitempty"`
	SourcePath    string             `bson:"source_path,omitempty" json:"source_path,omitempty"` // Path al array de datos a iterar
	FieldMappings []FieldMapping     `bson:"field_mappings" json:"field_mappings"`
	StaticFields  []StaticField      `bson:"static_fields,omitempty" json:"static_fields,omitempty"`
	Enabled       bool               `bson:"enabled" json:"enabled"`
	CreatedAt     time.Time          `bson:"created_at" json:"created_at"`
	UpdatedAt     time.Time          `bson:"updated_at" json:"updated_at"`
}

// Store gestiona el almacenamiento de recetas
type Store struct {
	collection *mongo.Collection
	mu         sync.RWMutex
}

var (
	storeInstance *Store
	storeOnce     sync.Once
)

// GetStore retorna la instancia singleton del Store
func GetStore() *Store {
	storeOnce.Do(func() {
		storeInstance = &Store{}
		storeInstance.init()
	})
	return storeInstance
}

// init inicializa la conexión a MongoDB
func (s *Store) init() {
	db := database.Database
	if db != nil {
		s.collection = db.Collection("recipes")

		// Crear índices
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()

		indexes := []mongo.IndexModel{
			{
				Keys:    bson.D{{Key: "name", Value: 1}},
				Options: options.Index().SetUnique(true),
			},
			{
				Keys: bson.D{{Key: "provider", Value: 1}},
			},
			{
				Keys: bson.D{{Key: "endpoint_id", Value: 1}},
			},
			{
				Keys: bson.D{{Key: "instance_id", Value: 1}},
			},
		}

		_, err := s.collection.Indexes().CreateMany(ctx, indexes)
		if err != nil {
			fmt.Printf("⚠️  Warning: could not create recipe indexes: %v\n", err)
		}

		fmt.Println("📋 Recipe Store initialized")
	}
}

// List retorna todas las recetas
func (s *Store) List() ([]Recipe, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	if s.collection == nil {
		return []Recipe{}, nil
	}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	cursor, err := s.collection.Find(ctx, bson.M{}, options.Find().SetSort(bson.D{{Key: "created_at", Value: -1}}))
	if err != nil {
		return nil, fmt.Errorf("error listing recipes: %w", err)
	}
	defer cursor.Close(ctx)

	var recipes []Recipe
	if err := cursor.All(ctx, &recipes); err != nil {
		return nil, fmt.Errorf("error decoding recipes: %w", err)
	}

	if recipes == nil {
		recipes = []Recipe{}
	}

	return recipes, nil
}

// Get retorna una receta por ID
func (s *Store) Get(id string) (*Recipe, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	if s.collection == nil {
		return nil, fmt.Errorf("recipe not found")
	}

	objectID, err := primitive.ObjectIDFromHex(id)
	if err != nil {
		return nil, fmt.Errorf("invalid recipe id: %w", err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	var recipe Recipe
	err = s.collection.FindOne(ctx, bson.M{"_id": objectID}).Decode(&recipe)
	if err != nil {
		if err == mongo.ErrNoDocuments {
			return nil, fmt.Errorf("recipe not found")
		}
		return nil, fmt.Errorf("error getting recipe: %w", err)
	}

	return &recipe, nil
}

// GetByInstanceID retorna recetas por instance_id
func (s *Store) GetByInstanceID(instanceID string) ([]Recipe, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	if s.collection == nil {
		return []Recipe{}, nil
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	cursor, err := s.collection.Find(ctx, bson.M{"instance_id": instanceID, "enabled": true})
	if err != nil {
		return nil, fmt.Errorf("error finding recipes: %w", err)
	}
	defer cursor.Close(ctx)

	var recipes []Recipe
	if err := cursor.All(ctx, &recipes); err != nil {
		return nil, fmt.Errorf("error decoding recipes: %w", err)
	}

	return recipes, nil
}

// Create crea una nueva receta
func (s *Store) Create(recipe *Recipe) (*Recipe, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.collection == nil {
		return nil, fmt.Errorf("database not available")
	}

	recipe.ID = primitive.NewObjectID()
	recipe.CreatedAt = time.Now()
	recipe.UpdatedAt = time.Now()
	recipe.Enabled = true

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	_, err := s.collection.InsertOne(ctx, recipe)
	if err != nil {
		return nil, fmt.Errorf("error creating recipe: %w", err)
	}

	fmt.Printf("📋 Recipe created: %s\n", recipe.Name)
	return recipe, nil
}

// Update actualiza una receta existente
func (s *Store) Update(id string, recipe *Recipe) (*Recipe, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.collection == nil {
		return nil, fmt.Errorf("database not available")
	}

	objectID, err := primitive.ObjectIDFromHex(id)
	if err != nil {
		return nil, fmt.Errorf("invalid recipe id: %w", err)
	}

	recipe.UpdatedAt = time.Now()

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	update := bson.M{
		"$set": bson.M{
			"name":           recipe.Name,
			"description":    recipe.Description,
			"provider":       recipe.Provider,
			"endpoint_id":    recipe.EndpointID,
			"instance_id":    recipe.InstanceID,
			"source_path":    recipe.SourcePath,
			"field_mappings": recipe.FieldMappings,
			"static_fields":  recipe.StaticFields,
			"enabled":        recipe.Enabled,
			"updated_at":     recipe.UpdatedAt,
		},
	}

	result := s.collection.FindOneAndUpdate(
		ctx,
		bson.M{"_id": objectID},
		update,
		options.FindOneAndUpdate().SetReturnDocument(options.After),
	)

	var updated Recipe
	if err := result.Decode(&updated); err != nil {
		if err == mongo.ErrNoDocuments {
			return nil, fmt.Errorf("recipe not found")
		}
		return nil, fmt.Errorf("error updating recipe: %w", err)
	}

	fmt.Printf("📋 Recipe updated: %s\n", updated.Name)
	return &updated, nil
}

// Delete elimina una receta
func (s *Store) Delete(id string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.collection == nil {
		return fmt.Errorf("database not available")
	}

	objectID, err := primitive.ObjectIDFromHex(id)
	if err != nil {
		return fmt.Errorf("invalid recipe id: %w", err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	result, err := s.collection.DeleteOne(ctx, bson.M{"_id": objectID})
	if err != nil {
		return fmt.Errorf("error deleting recipe: %w", err)
	}

	if result.DeletedCount == 0 {
		return fmt.Errorf("recipe not found")
	}

	fmt.Printf("📋 Recipe deleted: %s\n", id)
	return nil
}
