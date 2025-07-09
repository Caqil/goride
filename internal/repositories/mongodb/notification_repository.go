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

	// Invalidate cache
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
		"$or": []bson.M{
			{"expires_at": nil},
			{"expires_at": bson.M{"$gt": time.Now()}},
		},
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
	cacheKey := fmt.Sprintf("unread_count_%s", userID.Hex())
	if r.cache != nil {
		var count int64
		if err := r.cache.Get(ctx, cacheKey, &count); err == nil {
			return count, nil
		}
	}

	filter := bson.M{
		"user_id": userID,
		"status":  models.NotificationStatusUnread,
		"$or": []bson.M{
			{"expires_at": nil},
			{"expires_at": bson.M{"$gt": time.Now()}},
		},
	}

	count, err := r.collection.CountDocuments(ctx, filter)
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
	filter := bson.M{
		"user_id": userID,
		"status":  models.NotificationStatusUnread,
	}
	updates := bson.M{
		"$set": bson.M{
			"status":     models.NotificationStatusRead,
			"read_at":    time.Now(),
			"updated_at": time.Now(),
		},
	}

	_, err := r.collection.UpdateMany(ctx, filter, updates)
	if err != nil {
		return fmt.Errorf("failed to mark all notifications as read: %w", err)
	}

	// Invalidate cache
	r.invalidateUnreadCountCache(ctx, userID)

	return nil
}

func (r *notificationRepository) UpdateStatus(ctx context.Context, id primitive.ObjectID, status models.NotificationStatus) error {
	updates := map[string]interface{}{
		"status": status,
	}

	if status == models.NotificationStatusRead {
		updates["read_at"] = time.Now()
	} else if status == models.NotificationStatusSent {
		updates["sent_at"] = time.Now()
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
		"expires_at": bson.M{"$lt": time.Now()},
		"status":     bson.M{"$ne": models.NotificationStatusExpired},
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
		"expires_at": bson.M{"$lt": time.Now().AddDate(0, 0, -30)}, // Delete after 30 days of expiry
	}

	_, err := r.collection.DeleteMany(ctx, filter)
	if err != nil {
		return fmt.Errorf("failed to delete expired notifications: %w", err)
	}

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

	documents := make([]interface{}, len(notifications))
	userIDs := make(map[primitive.ObjectID]bool)

	for i, notification := range notifications {
		notification.ID = primitive.NewObjectID()
		notification.CreatedAt = time.Now()
		notification.UpdatedAt = time.Now()
		documents[i] = notification
		userIDs[notification.UserID] = true
	}

	_, err := r.collection.InsertMany(ctx, documents)
	if err != nil {
		return fmt.Errorf("failed to create batch notifications: %w", err)
	}

	// Invalidate cache for all affected users
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
	filter := bson.M{
		"created_at": bson.M{"$lt": cutoffDate},
		"status":     bson.M{"$in": []models.NotificationStatus{models.NotificationStatusRead, models.NotificationStatusExpired}},
	}

	_, err := r.collection.DeleteMany(ctx, filter)
	if err != nil {
		return fmt.Errorf("failed to delete old notifications: %w", err)
	}

	return nil
}

// Analytics
func (r *notificationRepository) GetNotificationStats(ctx context.Context, days int) (map[string]interface{}, error) {
	startDate := time.Now().AddDate(0, 0, -days)

	pipeline := mongo.Pipeline{
		{{"$match", bson.M{"created_at": bson.M{"$gte": startDate}}}},
		{{"$group", bson.M{
			"_id":   "$type",
			"total": bson.M{"$sum": 1},
			"sent": bson.M{"$sum": bson.M{
				"$cond": []interface{}{
					bson.M{"$eq": []interface{}{"$status", models.NotificationStatusSent}},
					1,
					0,
				},
			}},
			"read": bson.M{"$sum": bson.M{
				"$cond": []interface{}{
					bson.M{"$eq": []interface{}{"$status", models.NotificationStatusRead}},
					1,
					0,
				},
			}},
			"failed": bson.M{"$sum": bson.M{
				"$cond": []interface{}{
					bson.M{"$eq": []interface{}{"$status", models.NotificationStatusFailed}},
					1,
					0,
				},
			}},
		}}},
	}

	cursor, err := r.collection.Aggregate(ctx, pipeline)
	if err != nil {
		return nil, fmt.Errorf("failed to get notification stats: %w", err)
	}
	defer cursor.Close(ctx)

	stats := make(map[string]interface{})
	var totalNotifications int64

	for cursor.Next(ctx) {
		var result struct {
			ID     models.NotificationType `bson:"_id"`
			Total  int64                   `bson:"total"`
			Sent   int64                   `bson:"sent"`
			Read   int64                   `bson:"read"`
			Failed int64                   `bson:"failed"`
		}

		if err := cursor.Decode(&result); err != nil {
			return nil, fmt.Errorf("failed to decode notification stats: %w", err)
		}

		stats[string(result.ID)] = map[string]interface{}{
			"total":        result.Total,
			"sent":         result.Sent,
			"read":         result.Read,
			"failed":       result.Failed,
			"read_rate":    float64(result.Read) / float64(result.Total) * 100,
			"failure_rate": float64(result.Failed) / float64(result.Total) * 100,
		}

		totalNotifications += result.Total
	}

	stats["summary"] = map[string]interface{}{
		"total_notifications": totalNotifications,
		"period_days":         days,
		"start_date":          startDate,
		"end_date":            time.Now(),
	}

	return stats, nil
}

func (r *notificationRepository) GetDeliveryStats(ctx context.Context, days int) (map[string]interface{}, error) {
	startDate := time.Now().AddDate(0, 0, -days)

	pipeline := mongo.Pipeline{
		{{"$match", bson.M{"created_at": bson.M{"$gte": startDate}}}},
		{{"$group", bson.M{
			"_id": bson.M{
				"date": bson.M{
					"$dateToString": bson.M{
						"format": "%Y-%m-%d",
						"date":   "$created_at",
					},
				},
			},
			"created": bson.M{"$sum": 1},
			"sent": bson.M{"$sum": bson.M{
				"$cond": []interface{}{
					bson.M{"$eq": []interface{}{"$status", models.NotificationStatusSent}},
					1,
					0,
				},
			}},
			"failed": bson.M{"$sum": bson.M{
				"$cond": []interface{}{
					bson.M{"$eq": []interface{}{"$status", models.NotificationStatusFailed}},
					1,
					0,
				},
			}},
		}}},
		{{"$sort", bson.D{{Key: "_id.date", Value: 1}}}},
	}

	cursor, err := r.collection.Aggregate(ctx, pipeline)
	if err != nil {
		return nil, fmt.Errorf("failed to get delivery stats: %w", err)
	}
	defer cursor.Close(ctx)

	daily := make([]map[string]interface{}, 0)
	var totalCreated, totalSent, totalFailed int64

	for cursor.Next(ctx) {
		var result struct {
			ID struct {
				Date string `bson:"date"`
			} `bson:"_id"`
			Created int64 `bson:"created"`
			Sent    int64 `bson:"sent"`
			Failed  int64 `bson:"failed"`
		}

		if err := cursor.Decode(&result); err != nil {
			return nil, fmt.Errorf("failed to decode delivery stats: %w", err)
		}

		deliveryRate := float64(0)
		if result.Created > 0 {
			deliveryRate = float64(result.Sent) / float64(result.Created) * 100
		}

		daily = append(daily, map[string]interface{}{
			"date":          result.ID.Date,
			"created":       result.Created,
			"sent":          result.Sent,
			"failed":        result.Failed,
			"delivery_rate": deliveryRate,
		})

		totalCreated += result.Created
		totalSent += result.Sent
		totalFailed += result.Failed
	}

	overallDeliveryRate := float64(0)
	if totalCreated > 0 {
		overallDeliveryRate = float64(totalSent) / float64(totalCreated) * 100
	}

	return map[string]interface{}{
		"daily": daily,
		"summary": map[string]interface{}{
			"total_created":         totalCreated,
			"total_sent":            totalSent,
			"total_failed":          totalFailed,
			"overall_delivery_rate": overallDeliveryRate,
			"period_days":           days,
		},
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
			"_id":   "$type",
			"total": bson.M{"$sum": 1},
			"read": bson.M{"$sum": bson.M{
				"$cond": []interface{}{
					bson.M{"$eq": []interface{}{"$status", models.NotificationStatusRead}},
					1,
					0,
				},
			}},
			"avg_read_time": bson.M{"$avg": bson.M{
				"$cond": []interface{}{
					bson.M{"$ne": []interface{}{"$read_at", nil}},
					bson.M{"$subtract": []interface{}{"$read_at", "$created_at"}},
					nil,
				},
			}},
		}}},
	}

	cursor, err := r.collection.Aggregate(ctx, pipeline)
	if err != nil {
		return nil, fmt.Errorf("failed to get engagement stats: %w", err)
	}
	defer cursor.Close(ctx)

	stats := make(map[string]interface{})
	var totalNotifications, totalRead int64

	for cursor.Next(ctx) {
		var result struct {
			ID          models.NotificationType `bson:"_id"`
			Total       int64                   `bson:"total"`
			Read        int64                   `bson:"read"`
			AvgReadTime *float64                `bson:"avg_read_time"`
		}

		if err := cursor.Decode(&result); err != nil {
			return nil, fmt.Errorf("failed to decode engagement stats: %w", err)
		}

		readRate := float64(0)
		if result.Total > 0 {
			readRate = float64(result.Read) / float64(result.Total) * 100
		}

		avgReadTimeMinutes := float64(0)
		if result.AvgReadTime != nil {
			avgReadTimeMinutes = *result.AvgReadTime / (1000 * 60) // Convert milliseconds to minutes
		}

		stats[string(result.ID)] = map[string]interface{}{
			"total":                 result.Total,
			"read":                  result.Read,
			"read_rate":             readRate,
			"avg_read_time_minutes": avgReadTimeMinutes,
		}

		totalNotifications += result.Total
		totalRead += result.Read
	}

	overallReadRate := float64(0)
	if totalNotifications > 0 {
		overallReadRate = float64(totalRead) / float64(totalNotifications) * 100
	}

	stats["summary"] = map[string]interface{}{
		"total_notifications": totalNotifications,
		"total_read":          totalRead,
		"overall_read_rate":   overallReadRate,
		"period_days":         days,
		"user_id":             userID,
	}

	return stats, nil
}

// Helper methods
func (r *notificationRepository) findNotificationsWithFilter(ctx context.Context, filter bson.M, params *utils.PaginationParams) ([]*models.Notification, int64, error) {
	// Add search filter if provided
	if params.Search != "" {
		searchFields := []string{"title", "message"}
		searchFilter := params.GetSearchFilter(searchFields)
		if len(searchFilter) > 0 {
			filter = bson.M{
				"$and": []bson.M{filter, searchFilter},
			}
		}
	}

	// Get total count
	total, err := r.collection.CountDocuments(ctx, filter)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to count notifications: %w", err)
	}

	// Get paginated results
	opts := params.GetSortOptions()
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
		cacheKey := fmt.Sprintf("unread_count_%s", userID.Hex())
		r.cache.Delete(ctx, cacheKey)
	}
}
