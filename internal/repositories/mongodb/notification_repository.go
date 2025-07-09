package mongodb

import (
	"context"
	"fmt"
	"time"

	"goride/internal/models"
	"goride/internal/repositories/interfaces"
	"goride/internal/utils"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

type notificationRepository struct {
	collection *mongo.Collection
	cache      CacheService
}

func NewNotificationRepository(db *mongo.Database, cache CacheService) interfaces.NotificationRepository {
	return &notificationRepository{
		collection: db.Collection("notifications"),
		cache:      cache,
	}
}

// Basic CRUD operations
func (r *notificationRepository) Create(ctx context.Context, notification *models.Notification) error {
	notification.ID = primitive.NewObjectID()
	notification.CreatedAt = time.Now()
	notification.UpdatedAt = time.Now()

	_, err := r.collection.InsertOne(ctx, notification)
	if err != nil {
		return fmt.Errorf("failed to create notification: %w", err)
	}

	// Invalidate user's unread count cache
	r.invalidateUnreadCountCache(ctx, notification.UserID)

	return nil
}

func (r *notificationRepository) GetByID(ctx context.Context, id primitive.ObjectID) (*models.Notification, error) {
	var notification models.Notification
	err := r.collection.FindOne(ctx, bson.M{"_id": id}).Decode(&notification)
	if err != nil {
		if err == mongo.ErrNoDocuments {
			return nil, fmt.Errorf("notification not found")
		}
		return nil, fmt.Errorf("failed to get notification: %w", err)
	}

	return &notification, nil
}

func (r *notificationRepository) Update(ctx context.Context, id primitive.ObjectID, updates map[string]interface{}) error {
	updates["updated_at"] = time.Now()

	result, err := r.collection.UpdateOne(
		ctx,
		bson.M{"_id": id},
		bson.M{"$set": updates},
	)
	if err != nil {
		return fmt.Errorf("failed to update notification: %w", err)
	}

	if result.ModifiedCount == 0 {
		return fmt.Errorf("notification not found or no changes made")
	}

	// If status was updated, invalidate cache
	if _, exists := updates["status"]; exists {
		// Get the notification to find user_id for cache invalidation
		notification, err := r.GetByID(ctx, id)
		if err == nil {
			r.invalidateUnreadCountCache(ctx, notification.UserID)
		}
	}

	return nil
}

func (r *notificationRepository) Delete(ctx context.Context, id primitive.ObjectID) error {
	// Get notification first to invalidate cache
	notification, err := r.GetByID(ctx, id)
	if err != nil {
		return err
	}

	_, err = r.collection.DeleteOne(ctx, bson.M{"_id": id})
	if err != nil {
		return fmt.Errorf("failed to delete notification: %w", err)
	}

	// Invalidate user's unread count cache
	r.invalidateUnreadCountCache(ctx, notification.UserID)

	return nil
}

// User notifications
func (r *notificationRepository) GetByUserID(ctx context.Context, userID primitive.ObjectID, params *utils.PaginationParams) ([]*models.Notification, int64, error) {
	filter := bson.M{"user_id": userID}
	return r.findNotificationsWithFilter(ctx, filter, params)
}

func (r *notificationRepository) GetUnreadByUserID(ctx context.Context, userID primitive.ObjectID) ([]*models.Notification, error) {
	filter := bson.M{
		"user_id": userID,
		"status":  models.NotificationStatusUnread,
	}

	cursor, err := r.collection.Find(ctx, filter, options.Find().SetSort(bson.D{{Key: "created_at", Value: -1}}))
	if err != nil {
		return nil, fmt.Errorf("failed to find unread notifications: %w", err)
	}
	defer cursor.Close(ctx)

	var notifications []*models.Notification
	for cursor.Next(ctx) {
		var notification models.Notification
		if err := cursor.Decode(&notification); err != nil {
			return nil, fmt.Errorf("failed to decode notification: %w", err)
		}
		notifications = append(notifications, &notification)
	}

	return notifications, nil
}

func (r *notificationRepository) GetUnreadCount(ctx context.Context, userID primitive.ObjectID) (int64, error) {
	// Try cache first
	cacheKey := fmt.Sprintf("unread_count_user_%s", userID.Hex())
	if r.cache != nil {
		var count int64
		if err := r.cache.Get(ctx, cacheKey, &count); err == nil {
			return count, nil
		}
	}

	count, err := r.collection.CountDocuments(ctx, bson.M{
		"user_id": userID,
		"status":  models.NotificationStatusUnread,
	})
	if err != nil {
		return 0, fmt.Errorf("failed to count unread notifications: %w", err)
	}

	// Cache the result
	if r.cache != nil {
		r.cache.Set(ctx, cacheKey, count, 5*time.Minute)
	}

	return count, nil
}

// Status operations
func (r *notificationRepository) MarkAsRead(ctx context.Context, id primitive.ObjectID) error {
	updates := map[string]interface{}{
		"status":  models.NotificationStatusRead,
		"read_at": time.Now(),
	}

	return r.Update(ctx, id, updates)
}

func (r *notificationRepository) MarkAllAsRead(ctx context.Context, userID primitive.ObjectID) error {
	updates := bson.M{
		"$set": bson.M{
			"status":     models.NotificationStatusRead,
			"read_at":    time.Now(),
			"updated_at": time.Now(),
		},
	}

	_, err := r.collection.UpdateMany(
		ctx,
		bson.M{
			"user_id": userID,
			"status":  models.NotificationStatusUnread,
		},
		updates,
	)
	if err != nil {
		return fmt.Errorf("failed to mark all notifications as read: %w", err)
	}

	// Invalidate unread count cache
	r.invalidateUnreadCountCache(ctx, userID)

	return nil
}

func (r *notificationRepository) UpdateStatus(ctx context.Context, id primitive.ObjectID, status models.NotificationStatus) error {
	updates := map[string]interface{}{
		"status": status,
	}

	if status == models.NotificationStatusRead {
		updates["read_at"] = time.Now()
	}

	return r.Update(ctx, id, updates)
}

// Type filtering
func (r *notificationRepository) GetByType(ctx context.Context, notificationType models.NotificationType, params *utils.PaginationParams) ([]*models.Notification, int64, error) {
	filter := bson.M{"type": notificationType}
	return r.findNotificationsWithFilter(ctx, filter, params)
}

func (r *notificationRepository) GetByUserAndType(ctx context.Context, userID primitive.ObjectID, notificationType models.NotificationType, params *utils.PaginationParams) ([]*models.Notification, int64, error) {
	filter := bson.M{
		"user_id": userID,
		"type":    notificationType,
	}
	return r.findNotificationsWithFilter(ctx, filter, params)
}

// Time-based operations
func (r *notificationRepository) GetExpiredNotifications(ctx context.Context) ([]*models.Notification, error) {
	filter := bson.M{
		"expires_at": bson.M{
			"$exists": true,
			"$lte":    time.Now(),
		},
		"status": bson.M{"$ne": models.NotificationStatusExpired},
	}

	cursor, err := r.collection.Find(ctx, filter)
	if err != nil {
		return nil, fmt.Errorf("failed to find expired notifications: %w", err)
	}
	defer cursor.Close(ctx)

	var notifications []*models.Notification
	for cursor.Next(ctx) {
		var notification models.Notification
		if err := cursor.Decode(&notification); err != nil {
			return nil, fmt.Errorf("failed to decode notification: %w", err)
		}
		notifications = append(notifications, &notification)
	}

	return notifications, nil
}

func (r *notificationRepository) DeleteExpiredNotifications(ctx context.Context) error {
	filter := bson.M{
		"expires_at": bson.M{
			"$exists": true,
			"$lte":    time.Now(),
		},
	}

	result, err := r.collection.DeleteMany(ctx, filter)
	if err != nil {
		return fmt.Errorf("failed to delete expired notifications: %w", err)
	}

	fmt.Printf("Deleted %d expired notifications\n", result.DeletedCount)
	return nil
}

func (r *notificationRepository) GetNotificationsByDateRange(ctx context.Context, startDate, endDate time.Time, params *utils.PaginationParams) ([]*models.Notification, int64, error) {
	filter := bson.M{
		"created_at": bson.M{
			"$gte": startDate,
			"$lte": endDate,
		},
	}
	return r.findNotificationsWithFilter(ctx, filter, params)
}

// Bulk operations
func (r *notificationRepository) CreateBatch(ctx context.Context, notifications []*models.Notification) error {
	if len(notifications) == 0 {
		return nil
	}

	// Prepare documents for bulk insert
	docs := make([]interface{}, len(notifications))
	userIDs := make(map[primitive.ObjectID]bool)

	for i, notification := range notifications {
		notification.ID = primitive.NewObjectID()
		notification.CreatedAt = time.Now()
		notification.UpdatedAt = time.Now()
		docs[i] = notification
		userIDs[notification.UserID] = true
	}

	_, err := r.collection.InsertMany(ctx, docs)
	if err != nil {
		return fmt.Errorf("failed to create notifications batch: %w", err)
	}

	// Invalidate unread count cache for all affected users
	for userID := range userIDs {
		r.invalidateUnreadCountCache(ctx, userID)
	}

	return nil
}

func (r *notificationRepository) DeleteByUserID(ctx context.Context, userID primitive.ObjectID) error {
	_, err := r.collection.DeleteMany(ctx, bson.M{"user_id": userID})
	if err != nil {
		return fmt.Errorf("failed to delete notifications by user ID: %w", err)
	}

	// Invalidate cache
	r.invalidateUnreadCountCache(ctx, userID)

	return nil
}

func (r *notificationRepository) DeleteOldNotifications(ctx context.Context, days int) error {
	cutoffDate := time.Now().AddDate(0, 0, -days)

	result, err := r.collection.DeleteMany(ctx, bson.M{
		"created_at": bson.M{"$lt": cutoffDate},
		"status": bson.M{"$in": []models.NotificationStatus{
			models.NotificationStatusRead,
			models.NotificationStatusExpired,
		}},
	})
	if err != nil {
		return fmt.Errorf("failed to delete old notifications: %w", err)
	}

	fmt.Printf("Deleted %d old notifications\n", result.DeletedCount)
	return nil
}

// Analytics
func (r *notificationRepository) GetNotificationStats(ctx context.Context, days int) (map[string]interface{}, error) {
	startDate := time.Now().AddDate(0, 0, -days)

	// Total notifications
	totalNotifications, err := r.collection.CountDocuments(ctx, bson.M{
		"created_at": bson.M{"$gte": startDate},
	})
	if err != nil {
		return nil, fmt.Errorf("failed to count total notifications: %w", err)
	}

	// Notifications by status
	statusPipeline := mongo.Pipeline{
		{{"$match", bson.M{
			"created_at": bson.M{"$gte": startDate},
		}}},
		{{"$group", bson.M{
			"_id":   "$status",
			"count": bson.M{"$sum": 1},
		}}},
	}

	cursor, err := r.collection.Aggregate(ctx, statusPipeline)
	if err != nil {
		return nil, fmt.Errorf("failed to get notifications by status: %w", err)
	}
	defer cursor.Close(ctx)

	statusCounts := make(map[string]int64)
	for cursor.Next(ctx) {
		var result struct {
			Status models.NotificationStatus `bson:"_id"`
			Count  int64                     `bson:"count"`
		}

		if err := cursor.Decode(&result); err != nil {
			return nil, fmt.Errorf("failed to decode status count: %w", err)
		}

		statusCounts[string(result.Status)] = result.Count
	}

	// Notifications by type
	typePipeline := mongo.Pipeline{
		{{"$match", bson.M{
			"created_at": bson.M{"$gte": startDate},
		}}},
		{{"$group", bson.M{
			"_id":   "$type",
			"count": bson.M{"$sum": 1},
		}}},
	}

	cursor, err = r.collection.Aggregate(ctx, typePipeline)
	if err != nil {
		return nil, fmt.Errorf("failed to get notifications by type: %w", err)
	}
	defer cursor.Close(ctx)

	typeCounts := make(map[string]int64)
	for cursor.Next(ctx) {
		var result struct {
			Type  models.NotificationType `bson:"_id"`
			Count int64                   `bson:"count"`
		}

		if err := cursor.Decode(&result); err != nil {
			return nil, fmt.Errorf("failed to decode type count: %w", err)
		}

		typeCounts[string(result.Type)] = result.Count
	}

	// Read rate calculation
	readCount := statusCounts[string(models.NotificationStatusRead)]
	readRate := float64(0)
	if totalNotifications > 0 {
		readRate = float64(readCount) / float64(totalNotifications) * 100
	}

	return map[string]interface{}{
		"total_notifications": totalNotifications,
		"status_counts":       statusCounts,
		"type_counts":         typeCounts,
		"read_rate":           readRate,
		"period_days":         days,
		"start_date":          startDate,
		"end_date":            time.Now(),
	}, nil
}

func (r *notificationRepository) GetDeliveryStats(ctx context.Context, days int) (map[string]interface{}, error) {
	startDate := time.Now().AddDate(0, 0, -days)

	pipeline := mongo.Pipeline{
		{{"$match", bson.M{
			"created_at": bson.M{"$gte": startDate},
		}}},
		{{"$group", bson.M{
			"_id":        nil,
			"total_sent": bson.M{"$sum": 1},
			"delivered": bson.M{"$sum": bson.M{
				"$cond": []interface{}{
					bson.M{"$eq": []interface{}{"$delivery_status", "delivered"}},
					1, 0,
				},
			}},
			"failed": bson.M{"$sum": bson.M{
				"$cond": []interface{}{
					bson.M{"$eq": []interface{}{"$delivery_status", "failed"}},
					1, 0,
				},
			}},
			"pending": bson.M{"$sum": bson.M{
				"$cond": []interface{}{
					bson.M{"$eq": []interface{}{"$delivery_status", "pending"}},
					1, 0,
				},
			}},
		}}},
	}

	cursor, err := r.collection.Aggregate(ctx, pipeline)
	if err != nil {
		return nil, fmt.Errorf("failed to get delivery stats: %w", err)
	}
	defer cursor.Close(ctx)

	var result struct {
		TotalSent int64 `bson:"total_sent"`
		Delivered int64 `bson:"delivered"`
		Failed    int64 `bson:"failed"`
		Pending   int64 `bson:"pending"`
	}

	if cursor.Next(ctx) {
		if err := cursor.Decode(&result); err != nil {
			return nil, fmt.Errorf("failed to decode delivery stats: %w", err)
		}
	}

	deliveryRate := float64(0)
	failureRate := float64(0)

	if result.TotalSent > 0 {
		deliveryRate = float64(result.Delivered) / float64(result.TotalSent) * 100
		failureRate = float64(result.Failed) / float64(result.TotalSent) * 100
	}

	return map[string]interface{}{
		"total_sent":    result.TotalSent,
		"delivered":     result.Delivered,
		"failed":        result.Failed,
		"pending":       result.Pending,
		"delivery_rate": deliveryRate,
		"failure_rate":  failureRate,
		"period_days":   days,
		"start_date":    startDate,
		"end_date":      time.Now(),
	}, nil
}

func (r *notificationRepository) GetEngagementStats(ctx context.Context, userID primitive.ObjectID, days int) (map[string]interface{}, error) {
	startDate := time.Now().AddDate(0, 0, -days)

	pipeline := mongo.Pipeline{
		{{"$match", bson.M{
			"user_id":    userID,
			"created_at": bson.M{"$gte": startDate},
		}}},
		{{"$group", bson.M{
			"_id":            "$type",
			"total_received": bson.M{"$sum": 1},
			"total_read": bson.M{"$sum": bson.M{
				"$cond": []interface{}{
					bson.M{"$eq": []interface{}{"$status", models.NotificationStatusRead}},
					1, 0,
				},
			}},
			"avg_read_time": bson.M{"$avg": bson.M{
				"$subtract": []interface{}{"$read_at", "$created_at"},
			}},
		}}},
	}

	cursor, err := r.collection.Aggregate(ctx, pipeline)
	if err != nil {
		return nil, fmt.Errorf("failed to get engagement stats: %w", err)
	}
	defer cursor.Close(ctx)

	engagementByType := make(map[string]map[string]interface{})
	totalReceived := int64(0)
	totalRead := int64(0)

	for cursor.Next(ctx) {
		var result struct {
			Type          models.NotificationType `bson:"_id"`
			TotalReceived int64                   `bson:"total_received"`
			TotalRead     int64                   `bson:"total_read"`
			AvgReadTime   float64                 `bson:"avg_read_time"`
		}

		if err := cursor.Decode(&result); err != nil {
			return nil, fmt.Errorf("failed to decode engagement stats: %w", err)
		}

		readRate := float64(0)
		if result.TotalReceived > 0 {
			readRate = float64(result.TotalRead) / float64(result.TotalReceived) * 100
		}

		engagementByType[string(result.Type)] = map[string]interface{}{
			"total_received": result.TotalReceived,
			"total_read":     result.TotalRead,
			"read_rate":      readRate,
			"avg_read_time":  result.AvgReadTime / 1000, // Convert to seconds
		}

		totalReceived += result.TotalReceived
		totalRead += result.TotalRead
	}

	overallReadRate := float64(0)
	if totalReceived > 0 {
		overallReadRate = float64(totalRead) / float64(totalReceived) * 100
	}

	return map[string]interface{}{
		"user_id":            userID,
		"total_received":     totalReceived,
		"total_read":         totalRead,
		"overall_read_rate":  overallReadRate,
		"engagement_by_type": engagementByType,
		"period_days":        days,
		"start_date":         startDate,
		"end_date":           time.Now(),
	}, nil
}

// Helper methods
func (r *notificationRepository) findNotificationsWithFilter(ctx context.Context, filter bson.M, params *utils.PaginationParams) ([]*models.Notification, int64, error) {
	// Add search filter if provided
	if params.Search != "" {
		searchFields := []string{"title", "message"}
		filter = bson.M{
			"$and": []bson.M{
				filter,
				params.GetSearchFilter(searchFields),
			},
		}
	}

	// Get total count
	total, err := r.collection.CountDocuments(ctx, filter)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to count notifications: %w", err)
	}

	// Get paginated results
	opts := params.GetSortOptions()
	// Default sort by created_at descending for notifications
	if params.Sort == "created_at" || params.Sort == "" {
		opts.SetSort(bson.D{{Key: "created_at", Value: -1}})
	}

	cursor, err := r.collection.Find(ctx, filter, opts)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to find notifications: %w", err)
	}
	defer cursor.Close(ctx)

	var notifications []*models.Notification
	for cursor.Next(ctx) {
		var notification models.Notification
		if err := cursor.Decode(&notification); err != nil {
			return nil, 0, fmt.Errorf("failed to decode notification: %w", err)
		}
		notifications = append(notifications, &notification)
	}

	return notifications, total, nil
}

// Cache operations
func (r *notificationRepository) invalidateUnreadCountCache(ctx context.Context, userID primitive.ObjectID) {
	if r.cache != nil {
		cacheKey := fmt.Sprintf("unread_count_user_%s", userID.Hex())
		r.cache.Delete(ctx, cacheKey)
	}
}
