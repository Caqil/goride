package mongodb

import (
	"context"
	"fmt"
	"time"

	"goride/internal/models"
	"goride/internal/repositories/interfaces"
	"goride/internal/services"
	"goride/internal/utils"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

type chatRepository struct {
	chatsCollection    *mongo.Collection
	messagesCollection *mongo.Collection
	cache              services.CacheService
}

func NewChatRepository(db *mongo.Database, cache services.CacheService) interfaces.ChatRepository {
	return &chatRepository{
		chatsCollection:    db.Collection("chats"),
		messagesCollection: db.Collection("messages"),
		cache:              cache,
	}
}

// Chat CRUD operations
func (r *chatRepository) CreateChat(ctx context.Context, chat *models.Chat) error {
	chat.ID = primitive.NewObjectID()
	chat.CreatedAt = time.Now()
	chat.UpdatedAt = time.Now()

	_, err := r.chatsCollection.InsertOne(ctx, chat)
	if err != nil {
		return fmt.Errorf("failed to create chat: %w", err)
	}

	// Cache the chat
	r.cacheChat(ctx, chat)

	return nil
}

func (r *chatRepository) GetChatByID(ctx context.Context, id primitive.ObjectID) (*models.Chat, error) {
	// Try cache first
	if chat := r.getChatFromCache(ctx, id.Hex()); chat != nil {
		return chat, nil
	}

	var chat models.Chat
	err := r.chatsCollection.FindOne(ctx, bson.M{"_id": id}).Decode(&chat)
	if err != nil {
		if err == mongo.ErrNoDocuments {
			return nil, fmt.Errorf("chat not found")
		}
		return nil, fmt.Errorf("failed to get chat: %w", err)
	}

	// Cache the result
	r.cacheChat(ctx, &chat)

	return &chat, nil
}

func (r *chatRepository) UpdateChat(ctx context.Context, id primitive.ObjectID, updates map[string]interface{}) error {
	updates["updated_at"] = time.Now()

	_, err := r.chatsCollection.UpdateOne(
		ctx,
		bson.M{"_id": id},
		bson.M{"$set": updates},
	)
	if err != nil {
		return fmt.Errorf("failed to update chat: %w", err)
	}

	// Invalidate cache
	r.invalidateChatCache(ctx, id.Hex())

	return nil
}

func (r *chatRepository) DeleteChat(ctx context.Context, id primitive.ObjectID) error {
	_, err := r.chatsCollection.DeleteOne(ctx, bson.M{"_id": id})
	if err != nil {
		return fmt.Errorf("failed to delete chat: %w", err)
	}

	// Also delete all messages in the chat
	_, err = r.messagesCollection.DeleteMany(ctx, bson.M{"chat_id": id})
	if err != nil {
		return fmt.Errorf("failed to delete chat messages: %w", err)
	}

	// Invalidate cache
	r.invalidateChatCache(ctx, id.Hex())

	return nil
}

// Chat management
func (r *chatRepository) GetChatByRideID(ctx context.Context, rideID primitive.ObjectID) (*models.Chat, error) {
	// Try cache first
	cacheKey := fmt.Sprintf("chat_ride_%s", rideID.Hex())
	if r.cache != nil {
		var chat models.Chat
		if err := r.cache.Get(ctx, cacheKey, &chat); err == nil {
			return &chat, nil
		}
	}

	var chat models.Chat
	err := r.chatsCollection.FindOne(ctx, bson.M{"ride_id": rideID}).Decode(&chat)
	if err != nil {
		if err == mongo.ErrNoDocuments {
			return nil, fmt.Errorf("chat not found for ride")
		}
		return nil, fmt.Errorf("failed to get chat by ride ID: %w", err)
	}

	// Cache the result
	if r.cache != nil {
		r.cache.Set(ctx, cacheKey, &chat, 30*time.Minute)
	}

	return &chat, nil
}

func (r *chatRepository) GetChatsByParticipant(ctx context.Context, participantID primitive.ObjectID, params *utils.PaginationParams) ([]*models.Chat, int64, error) {
	filter := bson.M{
		"participants": bson.M{"$in": []primitive.ObjectID{participantID}},
	}

	return r.findChatsWithFilter(ctx, filter, params)
}

func (r *chatRepository) CloseChat(ctx context.Context, id primitive.ObjectID) error {
	updates := map[string]interface{}{
		"status":    models.ChatStatusClosed,
		"closed_at": time.Now(),
	}

	return r.UpdateChat(ctx, id, updates)
}

// Message CRUD operations
func (r *chatRepository) CreateMessage(ctx context.Context, message *models.Message) error {
	message.ID = primitive.NewObjectID()
	message.CreatedAt = time.Now()
	message.UpdatedAt = time.Now()

	// Start a transaction to update both message and chat
	session, err := r.chatsCollection.Database().Client().StartSession()
	if err != nil {
		return fmt.Errorf("failed to start session: %w", err)
	}
	defer session.EndSession(ctx)

	err = mongo.WithSession(ctx, session, func(sc mongo.SessionContext) error {
		// Insert the message
		_, err := r.messagesCollection.InsertOne(sc, message)
		if err != nil {
			return fmt.Errorf("failed to insert message: %w", err)
		}

		// Update the chat's last message
		_, err = r.chatsCollection.UpdateOne(
			sc,
			bson.M{"_id": message.ChatID},
			bson.M{
				"$set": bson.M{
					"last_message": message,
					"updated_at":   time.Now(),
				},
			},
		)
		if err != nil {
			return fmt.Errorf("failed to update chat: %w", err)
		}

		return nil
	})

	if err != nil {
		return fmt.Errorf("failed to create message: %w", err)
	}

	// Invalidate chat cache
	r.invalidateChatCache(ctx, message.ChatID.Hex())

	return nil
}

func (r *chatRepository) GetMessageByID(ctx context.Context, id primitive.ObjectID) (*models.Message, error) {
	var message models.Message
	err := r.messagesCollection.FindOne(ctx, bson.M{
		"_id":        id,
		"deleted_at": nil,
	}).Decode(&message)
	if err != nil {
		if err == mongo.ErrNoDocuments {
			return nil, fmt.Errorf("message not found")
		}
		return nil, fmt.Errorf("failed to get message: %w", err)
	}

	return &message, nil
}

func (r *chatRepository) UpdateMessage(ctx context.Context, id primitive.ObjectID, updates map[string]interface{}) error {
	updates["updated_at"] = time.Now()

	_, err := r.messagesCollection.UpdateOne(
		ctx,
		bson.M{"_id": id, "deleted_at": nil},
		bson.M{"$set": updates},
	)
	if err != nil {
		return fmt.Errorf("failed to update message: %w", err)
	}

	return nil
}

func (r *chatRepository) DeleteMessage(ctx context.Context, id primitive.ObjectID) error {
	updates := map[string]interface{}{
		"deleted_at": time.Now(),
	}

	return r.UpdateMessage(ctx, id, updates)
}

// Message retrieval
func (r *chatRepository) GetMessagesByChatID(ctx context.Context, chatID primitive.ObjectID, params *utils.PaginationParams) ([]*models.Message, int64, error) {
	filter := bson.M{
		"chat_id":    chatID,
		"deleted_at": nil,
	}

	return r.findMessagesWithFilter(ctx, filter, params)
}

func (r *chatRepository) GetUnreadMessages(ctx context.Context, chatID primitive.ObjectID, userID primitive.ObjectID) ([]*models.Message, error) {
	filter := bson.M{
		"chat_id":         chatID,
		"sender_id":       bson.M{"$ne": userID}, // Not sent by the user
		"deleted_at":      nil,
		"read_by.user_id": bson.M{"$ne": userID}, // Not read by the user
	}

	cursor, err := r.messagesCollection.Find(ctx, filter, options.Find().SetSort(bson.D{{Key: "created_at", Value: 1}}))
	if err != nil {
		return nil, fmt.Errorf("failed to find unread messages: %w", err)
	}
	defer cursor.Close(ctx)

	var messages []*models.Message
	for cursor.Next(ctx) {
		var message models.Message
		if err := cursor.Decode(&message); err != nil {
			return nil, fmt.Errorf("failed to decode message: %w", err)
		}
		messages = append(messages, &message)
	}

	return messages, nil
}

func (r *chatRepository) GetMessagesByType(ctx context.Context, chatID primitive.ObjectID, messageType models.MessageType) ([]*models.Message, error) {
	filter := bson.M{
		"chat_id":    chatID,
		"type":       messageType,
		"deleted_at": nil,
	}

	cursor, err := r.messagesCollection.Find(ctx, filter, options.Find().SetSort(bson.D{{Key: "created_at", Value: -1}}))
	if err != nil {
		return nil, fmt.Errorf("failed to find messages by type: %w", err)
	}
	defer cursor.Close(ctx)

	var messages []*models.Message
	for cursor.Next(ctx) {
		var message models.Message
		if err := cursor.Decode(&message); err != nil {
			return nil, fmt.Errorf("failed to decode message: %w", err)
		}
		messages = append(messages, &message)
	}

	return messages, nil
}

// Message status
func (r *chatRepository) UpdateMessageStatus(ctx context.Context, id primitive.ObjectID, status models.MessageStatus) error {
	updates := map[string]interface{}{
		"status": status,
	}

	return r.UpdateMessage(ctx, id, updates)
}

func (r *chatRepository) MarkMessageAsRead(ctx context.Context, messageID primitive.ObjectID, userID primitive.ObjectID) error {
	readReceipt := models.ReadReceipt{
		UserID: userID,
		ReadAt: time.Now(),
	}

	_, err := r.messagesCollection.UpdateOne(
		ctx,
		bson.M{
			"_id":             messageID,
			"read_by.user_id": bson.M{"$ne": userID}, // Only if not already read
			"deleted_at":      nil,
		},
		bson.M{
			"$push": bson.M{"read_by": readReceipt},
			"$set":  bson.M{"updated_at": time.Now()},
		},
	)
	if err != nil {
		return fmt.Errorf("failed to mark message as read: %w", err)
	}

	return nil
}

func (r *chatRepository) MarkAllMessagesAsRead(ctx context.Context, chatID primitive.ObjectID, userID primitive.ObjectID) error {
	readReceipt := models.ReadReceipt{
		UserID: userID,
		ReadAt: time.Now(),
	}

	_, err := r.messagesCollection.UpdateMany(
		ctx,
		bson.M{
			"chat_id":         chatID,
			"sender_id":       bson.M{"$ne": userID}, // Not sent by the user
			"read_by.user_id": bson.M{"$ne": userID}, // Not already read
			"deleted_at":      nil,
		},
		bson.M{
			"$push": bson.M{"read_by": readReceipt},
			"$set":  bson.M{"updated_at": time.Now()},
		},
	)
	if err != nil {
		return fmt.Errorf("failed to mark all messages as read: %w", err)
	}

	return nil
}

// Search and filtering
func (r *chatRepository) SearchMessages(ctx context.Context, chatID primitive.ObjectID, query string, params *utils.PaginationParams) ([]*models.Message, int64, error) {
	filter := bson.M{
		"chat_id":    chatID,
		"deleted_at": nil,
		"content":    bson.M{"$regex": query, "$options": "i"},
	}

	return r.findMessagesWithFilter(ctx, filter, params)
}

func (r *chatRepository) GetMessagesByDateRange(ctx context.Context, chatID primitive.ObjectID, startDate, endDate time.Time, params *utils.PaginationParams) ([]*models.Message, int64, error) {
	filter := bson.M{
		"chat_id": chatID,
		"created_at": bson.M{
			"$gte": startDate,
			"$lte": endDate,
		},
		"deleted_at": nil,
	}

	return r.findMessagesWithFilter(ctx, filter, params)
}

// Media messages
func (r *chatRepository) GetMediaMessages(ctx context.Context, chatID primitive.ObjectID, mediaType models.MessageType) ([]*models.Message, error) {
	var filter bson.M

	if mediaType != "" {
		filter = bson.M{
			"chat_id":    chatID,
			"type":       mediaType,
			"deleted_at": nil,
		}
	} else {
		// Get all media types (non-text)
		filter = bson.M{
			"chat_id": chatID,
			"type": bson.M{
				"$in": []models.MessageType{
					models.MessageTypeImage,
					models.MessageTypeAudio,
					models.MessageTypeFile,
				},
			},
			"deleted_at": nil,
		}
	}

	cursor, err := r.messagesCollection.Find(ctx, filter, options.Find().SetSort(bson.D{{Key: "created_at", Value: -1}}))
	if err != nil {
		return nil, fmt.Errorf("failed to find media messages: %w", err)
	}
	defer cursor.Close(ctx)

	var messages []*models.Message
	for cursor.Next(ctx) {
		var message models.Message
		if err := cursor.Decode(&message); err != nil {
			return nil, fmt.Errorf("failed to decode message: %w", err)
		}
		messages = append(messages, &message)
	}

	return messages, nil
}

// Analytics
func (r *chatRepository) GetChatStats(ctx context.Context, chatID primitive.ObjectID) (map[string]interface{}, error) {
	// Get total message count
	totalMessages, err := r.messagesCollection.CountDocuments(ctx, bson.M{
		"chat_id":    chatID,
		"deleted_at": nil,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to count total messages: %w", err)
	}

	// Get message counts by participant
	pipeline := mongo.Pipeline{
		{{"$match", bson.M{
			"chat_id":    chatID,
			"deleted_at": nil,
		}}},
		{{"$group", bson.M{
			"_id":   "$sender_id",
			"count": bson.M{"$sum": 1},
		}}},
	}

	cursor, err := r.messagesCollection.Aggregate(ctx, pipeline)
	if err != nil {
		return nil, fmt.Errorf("failed to get message counts by participant: %w", err)
	}
	defer cursor.Close(ctx)

	participantCounts := make(map[string]int64)
	for cursor.Next(ctx) {
		var result struct {
			SenderID primitive.ObjectID `bson:"_id"`
			Count    int64              `bson:"count"`
		}

		if err := cursor.Decode(&result); err != nil {
			return nil, fmt.Errorf("failed to decode participant count: %w", err)
		}

		participantCounts[result.SenderID.Hex()] = result.Count
	}

	// Get message types distribution
	typePipeline := mongo.Pipeline{
		{{"$match", bson.M{
			"chat_id":    chatID,
			"deleted_at": nil,
		}}},
		{{"$group", bson.M{
			"_id":   "$type",
			"count": bson.M{"$sum": 1},
		}}},
	}

	cursor, err = r.messagesCollection.Aggregate(ctx, typePipeline)
	if err != nil {
		return nil, fmt.Errorf("failed to get message types: %w", err)
	}
	defer cursor.Close(ctx)

	typeCounts := make(map[string]int64)
	for cursor.Next(ctx) {
		var result struct {
			Type  models.MessageType `bson:"_id"`
			Count int64              `bson:"count"`
		}

		if err := cursor.Decode(&result); err != nil {
			return nil, fmt.Errorf("failed to decode type count: %w", err)
		}

		typeCounts[string(result.Type)] = result.Count
	}

	// Get chat duration
	chat, err := r.GetChatByID(ctx, chatID)
	if err != nil {
		return nil, fmt.Errorf("failed to get chat: %w", err)
	}

	duration := time.Since(chat.CreatedAt)
	if chat.ClosedAt != nil {
		duration = chat.ClosedAt.Sub(chat.CreatedAt)
	}

	return map[string]interface{}{
		"chat_id":            chatID,
		"total_messages":     totalMessages,
		"participant_counts": participantCounts,
		"message_types":      typeCounts,
		"chat_duration":      duration.Seconds(),
		"is_active":          chat.Status == models.ChatStatusActive,
		"created_at":         chat.CreatedAt,
		"closed_at":          chat.ClosedAt,
	}, nil
}

func (r *chatRepository) GetMessageStats(ctx context.Context, days int) (map[string]interface{}, error) {
	startDate := time.Now().AddDate(0, 0, -days)

	// Total messages
	totalMessages, err := r.messagesCollection.CountDocuments(ctx, bson.M{
		"created_at": bson.M{"$gte": startDate},
		"deleted_at": nil,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to count total messages: %w", err)
	}

	// Messages by type
	typePipeline := mongo.Pipeline{
		{{"$match", bson.M{
			"created_at": bson.M{"$gte": startDate},
			"deleted_at": nil,
		}}},
		{{"$group", bson.M{
			"_id":   "$type",
			"count": bson.M{"$sum": 1},
		}}},
	}

	cursor, err := r.messagesCollection.Aggregate(ctx, typePipeline)
	if err != nil {
		return nil, fmt.Errorf("failed to get message stats by type: %w", err)
	}
	defer cursor.Close(ctx)

	typeCounts := make(map[string]int64)
	for cursor.Next(ctx) {
		var result struct {
			Type  models.MessageType `bson:"_id"`
			Count int64              `bson:"count"`
		}

		if err := cursor.Decode(&result); err != nil {
			return nil, fmt.Errorf("failed to decode message type stats: %w", err)
		}

		typeCounts[string(result.Type)] = result.Count
	}

	// Daily message counts
	dailyPipeline := mongo.Pipeline{
		{{"$match", bson.M{
			"created_at": bson.M{"$gte": startDate},
			"deleted_at": nil,
		}}},
		{{"$group", bson.M{
			"_id": bson.M{
				"date": bson.M{"$dateToString": bson.M{
					"format": "%Y-%m-%d",
					"date":   "$created_at",
				}},
			},
			"count": bson.M{"$sum": 1},
		}}},
		{{"$sort", bson.M{"_id.date": 1}}},
	}

	cursor, err = r.messagesCollection.Aggregate(ctx, dailyPipeline)
	if err != nil {
		return nil, fmt.Errorf("failed to get daily message stats: %w", err)
	}
	defer cursor.Close(ctx)

	dailyCounts := make(map[string]int64)
	for cursor.Next(ctx) {
		var result struct {
			ID struct {
				Date string `bson:"date"`
			} `bson:"_id"`
			Count int64 `bson:"count"`
		}

		if err := cursor.Decode(&result); err != nil {
			return nil, fmt.Errorf("failed to decode daily message stats: %w", err)
		}

		dailyCounts[result.ID.Date] = result.Count
	}

	// Active chats
	activeChats, err := r.chatsCollection.CountDocuments(ctx, bson.M{
		"status":     models.ChatStatusActive,
		"updated_at": bson.M{"$gte": startDate},
	})
	if err != nil {
		return nil, fmt.Errorf("failed to count active chats: %w", err)
	}

	return map[string]interface{}{
		"total_messages": totalMessages,
		"type_counts":    typeCounts,
		"daily_counts":   dailyCounts,
		"active_chats":   activeChats,
		"period_days":    days,
		"start_date":     startDate,
		"end_date":       time.Now(),
	}, nil
}

func (r *chatRepository) GetUserChatActivity(ctx context.Context, userID primitive.ObjectID, days int) (map[string]interface{}, error) {
	startDate := time.Now().AddDate(0, 0, -days)

	// Messages sent by user
	messagesSent, err := r.messagesCollection.CountDocuments(ctx, bson.M{
		"sender_id":  userID,
		"created_at": bson.M{"$gte": startDate},
		"deleted_at": nil,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to count messages sent: %w", err)
	}

	// Chats participated in
	chatsParticipated, err := r.chatsCollection.CountDocuments(ctx, bson.M{
		"participants": bson.M{"$in": []primitive.ObjectID{userID}},
		"updated_at":   bson.M{"$gte": startDate},
	})
	if err != nil {
		return nil, fmt.Errorf("failed to count chats participated: %w", err)
	}

	// Messages by type
	typePipeline := mongo.Pipeline{
		{{"$match", bson.M{
			"sender_id":  userID,
			"created_at": bson.M{"$gte": startDate},
			"deleted_at": nil,
		}}},
		{{"$group", bson.M{
			"_id":   "$type",
			"count": bson.M{"$sum": 1},
		}}},
	}

	cursor, err := r.messagesCollection.Aggregate(ctx, typePipeline)
	if err != nil {
		return nil, fmt.Errorf("failed to get user message types: %w", err)
	}
	defer cursor.Close(ctx)

	typeCounts := make(map[string]int64)
	for cursor.Next(ctx) {
		var result struct {
			Type  models.MessageType `bson:"_id"`
			Count int64              `bson:"count"`
		}

		if err := cursor.Decode(&result); err != nil {
			return nil, fmt.Errorf("failed to decode user message type: %w", err)
		}

		typeCounts[string(result.Type)] = result.Count
	}

	// Average response time
	avgResponseTime, err := r.calculateAverageResponseTime(ctx, userID, startDate)
	if err != nil {
		// Don't fail the whole request if this calculation fails
		avgResponseTime = 0
	}

	return map[string]interface{}{
		"user_id":            userID,
		"messages_sent":      messagesSent,
		"chats_participated": chatsParticipated,
		"message_types":      typeCounts,
		"avg_response_time":  avgResponseTime,
		"period_days":        days,
		"start_date":         startDate,
		"end_date":           time.Now(),
	}, nil
}

// Helper methods
func (r *chatRepository) findChatsWithFilter(ctx context.Context, filter bson.M, params *utils.PaginationParams) ([]*models.Chat, int64, error) {
	// Add search filter if provided
	if params.Search != "" {
		// For chats, we might search in the last message content
		searchFilter := bson.M{
			"last_message.content": bson.M{"$regex": params.Search, "$options": "i"},
		}
		filter = bson.M{
			"$and": []bson.M{filter, searchFilter},
		}
	}

	// Get total count
	total, err := r.chatsCollection.CountDocuments(ctx, filter)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to count chats: %w", err)
	}

	// Get paginated results
	opts := params.GetSortOptions()
	// Default sort by updated_at descending for chats
	if params.Sort == "updated_at" || params.Sort == "" {
		opts.SetSort(bson.D{{Key: "updated_at", Value: -1}})
	}

	cursor, err := r.chatsCollection.Find(ctx, filter, opts)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to find chats: %w", err)
	}
	defer cursor.Close(ctx)

	var chats []*models.Chat
	for cursor.Next(ctx) {
		var chat models.Chat
		if err := cursor.Decode(&chat); err != nil {
			return nil, 0, fmt.Errorf("failed to decode chat: %w", err)
		}
		chats = append(chats, &chat)
	}

	return chats, total, nil
}

func (r *chatRepository) findMessagesWithFilter(ctx context.Context, filter bson.M, params *utils.PaginationParams) ([]*models.Message, int64, error) {
	// Add search filter if provided
	if params.Search != "" {
		searchFields := []string{"content"}
		filter = bson.M{
			"$and": []bson.M{
				filter,
				params.GetSearchFilter(searchFields),
			},
		}
	}

	// Get total count
	total, err := r.messagesCollection.CountDocuments(ctx, filter)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to count messages: %w", err)
	}

	// Get paginated results
	opts := params.GetSortOptions()
	// Default sort by created_at ascending for messages (chronological order)
	if params.Sort == "created_at" || params.Sort == "" {
		opts.SetSort(bson.D{{Key: "created_at", Value: 1}})
	}

	cursor, err := r.messagesCollection.Find(ctx, filter, opts)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to find messages: %w", err)
	}
	defer cursor.Close(ctx)

	var messages []*models.Message
	for cursor.Next(ctx) {
		var message models.Message
		if err := cursor.Decode(&message); err != nil {
			return nil, 0, fmt.Errorf("failed to decode message: %w", err)
		}
		messages = append(messages, &message)
	}

	return messages, total, nil
}

func (r *chatRepository) calculateAverageResponseTime(ctx context.Context, userID primitive.ObjectID, startDate time.Time) (float64, error) {
	// This is a simplified calculation
	// In a real implementation, you'd need more sophisticated logic to calculate response times
	pipeline := mongo.Pipeline{
		{{"$match", bson.M{
			"sender_id":  userID,
			"created_at": bson.M{"$gte": startDate},
			"deleted_at": nil,
		}}},
		{{"$sort", bson.D{{Key: "chat_id", Value: 1}, {Key: "created_at", Value: 1}}}},
		{{"$group", bson.M{
			"_id":      "$chat_id",
			"messages": bson.M{"$push": "$$ROOT"},
		}}},
	}

	cursor, err := r.messagesCollection.Aggregate(ctx, pipeline)
	if err != nil {
		return 0, fmt.Errorf("failed to calculate response time: %w", err)
	}
	defer cursor.Close(ctx)

	// This would need more complex logic to properly calculate response times
	// For now, return a placeholder
	return 0, nil
}

// Cache operations
func (r *chatRepository) cacheChat(ctx context.Context, chat *models.Chat) {
	if r.cache != nil {
		cacheKey := fmt.Sprintf("chat:%s", chat.ID.Hex())
		r.cache.Set(ctx, cacheKey, chat, 15*time.Minute)
	}
}

func (r *chatRepository) getChatFromCache(ctx context.Context, chatID string) *models.Chat {
	if r.cache == nil {
		return nil
	}

	cacheKey := fmt.Sprintf("chat:%s", chatID)
	var chat models.Chat
	err := r.cache.Get(ctx, cacheKey, &chat)
	if err != nil {
		return nil
	}

	return &chat
}

func (r *chatRepository) invalidateChatCache(ctx context.Context, chatID string) {
	if r.cache != nil {
		cacheKey := fmt.Sprintf("chat:%s", chatID)
		r.cache.Delete(ctx, cacheKey)
	}
}
