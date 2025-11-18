package services

import (
	"context"
	"time"

	"omniapi/internal/database"
	"omniapi/internal/models"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo/options"
)

// MessageService servicio para operaciones de mensajes
type MessageService struct {
	collection string
}

// NewMessageService crea una nueva instancia del servicio de mensajes
func NewMessageService() *MessageService {
	return &MessageService{
		collection: "messages",
	}
}

// Create crea un nuevo mensaje
func (ms *MessageService) Create(message *models.Message) error {
	collection := database.GetCollection(ms.collection)
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	// Establecer ID y timestamp
	message.ID = primitive.NewObjectID()
	message.CreatedAt = time.Now()

	if message.Channel == "" {
		message.Channel = "general"
	}

	_, err := collection.InsertOne(ctx, message)
	return err
}

// GetByID obtiene un mensaje por ID
func (ms *MessageService) GetByID(id string) (*models.Message, error) {
	collection := database.GetCollection(ms.collection)
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	objectID, err := primitive.ObjectIDFromHex(id)
	if err != nil {
		return nil, err
	}

	var message models.Message
	err = collection.FindOne(ctx, bson.M{"_id": objectID}).Decode(&message)
	if err != nil {
		return nil, err
	}

	return &message, nil
}

// GetByChannel obtiene mensajes de un canal específico
func (ms *MessageService) GetByChannel(channel string, page, perPage int) ([]models.Message, int64, error) {
	collection := database.GetCollection(ms.collection)
	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()

	filter := bson.M{"channel": channel}
	skip := (page - 1) * perPage

	// Contar total
	total, err := collection.CountDocuments(ctx, filter)
	if err != nil {
		return nil, 0, err
	}

	// Opciones de consulta (ordenar por fecha descendente)
	opts := options.Find()
	opts.SetSkip(int64(skip))
	opts.SetLimit(int64(perPage))
	opts.SetSort(bson.D{{Key: "created_at", Value: -1}})

	cursor, err := collection.Find(ctx, filter, opts)
	if err != nil {
		return nil, 0, err
	}
	defer cursor.Close(ctx)

	var messages []models.Message
	if err = cursor.All(ctx, &messages); err != nil {
		return nil, 0, err
	}

	return messages, total, nil
}

// GetRecentMessages obtiene los mensajes más recientes
func (ms *MessageService) GetRecentMessages(limit int) ([]models.Message, error) {
	collection := database.GetCollection(ms.collection)
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	opts := options.Find()
	opts.SetLimit(int64(limit))
	opts.SetSort(bson.D{{Key: "created_at", Value: -1}})

	cursor, err := collection.Find(ctx, bson.M{}, opts)
	if err != nil {
		return nil, err
	}
	defer cursor.Close(ctx)

	var messages []models.Message
	if err = cursor.All(ctx, &messages); err != nil {
		return nil, err
	}

	return messages, nil
}

// GetUserMessages obtiene mensajes de un usuario específico
func (ms *MessageService) GetUserMessages(userID string, page, perPage int) ([]models.Message, int64, error) {
	objectID, err := primitive.ObjectIDFromHex(userID)
	if err != nil {
		return nil, 0, err
	}

	filter := bson.M{"from_user": objectID}
	return ms.getMessagesWithFilter(filter, page, perPage)
}

// GetPrivateMessages obtiene mensajes privados entre dos usuarios
func (ms *MessageService) GetPrivateMessages(user1ID, user2ID string, page, perPage int) ([]models.Message, int64, error) {
	objID1, err := primitive.ObjectIDFromHex(user1ID)
	if err != nil {
		return nil, 0, err
	}

	objID2, err := primitive.ObjectIDFromHex(user2ID)
	if err != nil {
		return nil, 0, err
	}

	filter := bson.M{
		"$or": []bson.M{
			{"from_user": objID1, "to_user": objID2},
			{"from_user": objID2, "to_user": objID1},
		},
	}

	return ms.getMessagesWithFilter(filter, page, perPage)
}

// MarkAsRead marca un mensaje como leído por un usuario
func (ms *MessageService) MarkAsRead(messageID, userID string) error {
	collection := database.GetCollection(ms.collection)
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	msgObjectID, err := primitive.ObjectIDFromHex(messageID)
	if err != nil {
		return err
	}

	userObjectID, err := primitive.ObjectIDFromHex(userID)
	if err != nil {
		return err
	}

	_, err = collection.UpdateOne(
		ctx,
		bson.M{"_id": msgObjectID},
		bson.M{
			"$addToSet": bson.M{
				"read_by": userObjectID,
			},
		},
	)

	return err
}

// Delete elimina un mensaje
func (ms *MessageService) Delete(id string) error {
	collection := database.GetCollection(ms.collection)
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	objectID, err := primitive.ObjectIDFromHex(id)
	if err != nil {
		return err
	}

	_, err = collection.DeleteOne(ctx, bson.M{"_id": objectID})
	return err
}

// GetMessageStats obtiene estadísticas de mensajes
func (ms *MessageService) GetMessageStats() (map[string]interface{}, error) {
	collection := database.GetCollection(ms.collection)
	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()

	// Contar mensajes por tipo
	pipeline := []bson.M{
		{
			"$group": bson.M{
				"_id":   "$type",
				"count": bson.M{"$sum": 1},
			},
		},
	}

	cursor, err := collection.Aggregate(ctx, pipeline)
	if err != nil {
		return nil, err
	}
	defer cursor.Close(ctx)

	var typeStats []bson.M
	if err = cursor.All(ctx, &typeStats); err != nil {
		return nil, err
	}

	// Contar total de mensajes
	total, err := collection.CountDocuments(ctx, bson.M{})
	if err != nil {
		return nil, err
	}

	// Contar mensajes de hoy
	today := time.Now().Truncate(24 * time.Hour)
	todayCount, err := collection.CountDocuments(ctx, bson.M{
		"created_at": bson.M{"$gte": today},
	})
	if err != nil {
		return nil, err
	}

	return map[string]interface{}{
		"total_messages":   total,
		"messages_today":   todayCount,
		"messages_by_type": typeStats,
	}, nil
}

// Función auxiliar para obtener mensajes con filtro
func (ms *MessageService) getMessagesWithFilter(filter bson.M, page, perPage int) ([]models.Message, int64, error) {
	collection := database.GetCollection(ms.collection)
	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()

	skip := (page - 1) * perPage

	// Contar total
	total, err := collection.CountDocuments(ctx, filter)
	if err != nil {
		return nil, 0, err
	}

	// Opciones de consulta
	opts := options.Find()
	opts.SetSkip(int64(skip))
	opts.SetLimit(int64(perPage))
	opts.SetSort(bson.D{{Key: "created_at", Value: -1}})

	cursor, err := collection.Find(ctx, filter, opts)
	if err != nil {
		return nil, 0, err
	}
	defer cursor.Close(ctx)

	var messages []models.Message
	if err = cursor.All(ctx, &messages); err != nil {
		return nil, 0, err
	}

	return messages, total, nil
}
