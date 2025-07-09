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

type auditLogRepository struct {
	collection *mongo.Collection
	cache      CacheService
}

func NewAuditLogRepository(db *mongo.Database, cache CacheService) interfaces.AuditLogRepository {
	return &auditLogRepository{
		collection: db.Collection("audit_logs"),
		cache:      cache,
	}
}

// Basic operations
func (r *auditLogRepository) Create(ctx context.Context, auditLog *models.AuditLog) error {
	auditLog.ID = primitive.NewObjectID()
	auditLog.CreatedAt = time.Now()

	_, err := r.collection.InsertOne(ctx, auditLog)
	if err != nil {
		return fmt.Errorf("failed to create audit log: %w", err)
	}

	return nil
}

func (r *auditLogRepository) GetByID(ctx context.Context, id primitive.ObjectID) (*models.AuditLog, error) {
	var auditLog models.AuditLog
	err := r.collection.FindOne(ctx, bson.M{"_id": id}).Decode(&auditLog)
	if err != nil {
		if err == mongo.ErrNoDocuments {
			return nil, fmt.Errorf("audit log not found")
		}
		return nil, fmt.Errorf("failed to get audit log: %w", err)
	}

	return &auditLog, nil
}

// User actions
func (r *auditLogRepository) GetByUserID(ctx context.Context, userID primitive.ObjectID, params *utils.PaginationParams) ([]*models.AuditLog, int64, error) {
	filter := bson.M{"user_id": userID}
	return r.findAuditLogsWithFilter(ctx, filter, params)
}

func (r *auditLogRepository) GetUserActions(ctx context.Context, userID primitive.ObjectID, action models.AuditAction, params *utils.PaginationParams) ([]*models.AuditLog, int64, error) {
	filter := bson.M{
		"user_id": userID,
		"action":  action,
	}
	return r.findAuditLogsWithFilter(ctx, filter, params)
}

// Resource tracking
func (r *auditLogRepository) GetByResource(ctx context.Context, resource string, params *utils.PaginationParams) ([]*models.AuditLog, int64, error) {
	filter := bson.M{"resource": resource}
	return r.findAuditLogsWithFilter(ctx, filter, params)
}

func (r *auditLogRepository) GetByResourceAndAction(ctx context.Context, resource string, action models.AuditAction, params *utils.PaginationParams) ([]*models.AuditLog, int64, error) {
	filter := bson.M{
		"resource": resource,
		"action":   action,
	}
	return r.findAuditLogsWithFilter(ctx, filter, params)
}

func (r *auditLogRepository) GetResourceHistory(ctx context.Context, resource, resourceID string, params *utils.PaginationParams) ([]*models.AuditLog, int64, error) {
	filter := bson.M{
		"resource":    resource,
		"resource_id": resourceID,
	}
	return r.findAuditLogsWithFilter(ctx, filter, params)
}

// Action filtering
func (r *auditLogRepository) GetByAction(ctx context.Context, action models.AuditAction, params *utils.PaginationParams) ([]*models.AuditLog, int64, error) {
	filter := bson.M{"action": action}
	return r.findAuditLogsWithFilter(ctx, filter, params)
}

// Time-based queries
func (r *auditLogRepository) GetByDateRange(ctx context.Context, startDate, endDate time.Time, params *utils.PaginationParams) ([]*models.AuditLog, int64, error) {
	filter := bson.M{
		"created_at": bson.M{
			"$gte": startDate,
			"$lte": endDate,
		},
	}
	return r.findAuditLogsWithFilter(ctx, filter, params)
}

func (r *auditLogRepository) GetRecentLogs(ctx context.Context, hours int, params *utils.PaginationParams) ([]*models.AuditLog, int64, error) {
	startTime := time.Now().Add(-time.Duration(hours) * time.Hour)
	filter := bson.M{
		"created_at": bson.M{"$gte": startTime},
	}
	return r.findAuditLogsWithFilter(ctx, filter, params)
}

// Security monitoring
func (r *auditLogRepository) GetSecurityEvents(ctx context.Context, params *utils.PaginationParams) ([]*models.AuditLog, int64, error) {
	filter := bson.M{
		"action": bson.M{
			"$in": []models.AuditAction{
				models.AuditActionLogin,
				models.AuditActionLogout,
				models.AuditActionDelete,
			},
		},
	}

	// Add additional security-related filters
	securityFilters := bson.M{
		"$or": []bson.M{
			filter,
			{"metadata.security_event": true},
			{"metadata.failed_login": true},
			{"metadata.suspicious": true},
		},
	}

	return r.findAuditLogsWithFilter(ctx, securityFilters, params)
}

func (r *auditLogRepository) GetFailedLoginAttempts(ctx context.Context, ipAddress string, hours int) ([]*models.AuditLog, error) {
	startTime := time.Now().Add(-time.Duration(hours) * time.Hour)

	filter := bson.M{
		"action":     models.AuditActionLogin,
		"ip_address": ipAddress,
		"created_at": bson.M{"$gte": startTime},
		"$or": []bson.M{
			{"metadata.success": false},
			{"metadata.failed": true},
			{"metadata.login_failed": true},
		},
	}

	cursor, err := r.collection.Find(ctx, filter, options.Find().SetSort(bson.D{{Key: "created_at", Value: -1}}))
	if err != nil {
		return nil, fmt.Errorf("failed to find failed login attempts: %w", err)
	}
	defer cursor.Close(ctx)

	var logs []*models.AuditLog
	for cursor.Next(ctx) {
		var log models.AuditLog
		if err := cursor.Decode(&log); err != nil {
			return nil, fmt.Errorf("failed to decode audit log: %w", err)
		}
		logs = append(logs, &log)
	}

	return logs, nil
}

func (r *auditLogRepository) GetSuspiciousActivities(ctx context.Context, params *utils.PaginationParams) ([]*models.AuditLog, int64, error) {
	// Define suspicious activity patterns
	filter := bson.M{
		"$or": []bson.M{
			{"metadata.suspicious": true},
			{"metadata.anomaly_detected": true},
			{"metadata.risk_score": bson.M{"$gte": 0.7}},
			{
				"action":   models.AuditActionDelete,
				"resource": bson.M{"$in": []string{"user", "driver", "payment", "ride"}},
			},
			{
				"action":                    models.AuditActionLogin,
				"metadata.location_changed": true,
			},
		},
	}

	return r.findAuditLogsWithFilter(ctx, filter, params)
}

// IP and location tracking
func (r *auditLogRepository) GetByIPAddress(ctx context.Context, ipAddress string, params *utils.PaginationParams) ([]*models.AuditLog, int64, error) {
	filter := bson.M{"ip_address": ipAddress}
	return r.findAuditLogsWithFilter(ctx, filter, params)
}

func (r *auditLogRepository) GetLoginsByLocation(ctx context.Context, userID primitive.ObjectID, days int) ([]map[string]interface{}, error) {
	startDate := time.Now().AddDate(0, 0, -days)

	pipeline := mongo.Pipeline{
		{{Key: "$match", Value: bson.M{
			"user_id":    userID,
			"action":     models.AuditActionLogin,
			"created_at": bson.M{"$gte": startDate},
		}}},
		{{Key: "$group", Value: bson.M{
			"_id": bson.M{
				"ip_address": "$ip_address",
				"city":       "$location.city",
				"country":    "$location.country",
			},
			"login_count": bson.M{"$sum": 1},
			"last_login":  bson.M{"$max": "$created_at"},
			"first_login": bson.M{"$min": "$created_at"},
		}}},
		{{Key: "$sort", Value: bson.M{"login_count": -1}}},
	}

	cursor, err := r.collection.Aggregate(ctx, pipeline)
	if err != nil {
		return nil, fmt.Errorf("failed to get logins by location: %w", err)
	}
	defer cursor.Close(ctx)

	var results []map[string]interface{}
	for cursor.Next(ctx) {
		var result struct {
			ID struct {
				IPAddress string `bson:"ip_address"`
				City      string `bson:"city"`
				Country   string `bson:"country"`
			} `bson:"_id"`
			LoginCount int64     `bson:"login_count"`
			LastLogin  time.Time `bson:"last_login"`
			FirstLogin time.Time `bson:"first_login"`
		}

		if err := cursor.Decode(&result); err != nil {
			return nil, fmt.Errorf("failed to decode login location: %w", err)
		}

		results = append(results, map[string]interface{}{
			"ip_address":  result.ID.IPAddress,
			"city":        result.ID.City,
			"country":     result.ID.Country,
			"login_count": result.LoginCount,
			"last_login":  result.LastLogin,
			"first_login": result.FirstLogin,
		})
	}

	return results, nil
}

// Data compliance
func (r *auditLogRepository) GetDataAccessLogs(ctx context.Context, resource string, days int) ([]*models.AuditLog, error) {
	startDate := time.Now().AddDate(0, 0, -days)

	filter := bson.M{
		"resource":   resource,
		"action":     models.AuditActionRead,
		"created_at": bson.M{"$gte": startDate},
	}

	cursor, err := r.collection.Find(ctx, filter, options.Find().SetSort(bson.D{{Key: "created_at", Value: -1}}))
	if err != nil {
		return nil, fmt.Errorf("failed to find data access logs: %w", err)
	}
	defer cursor.Close(ctx)

	var logs []*models.AuditLog
	for cursor.Next(ctx) {
		var log models.AuditLog
		if err := cursor.Decode(&log); err != nil {
			return nil, fmt.Errorf("failed to decode audit log: %w", err)
		}
		logs = append(logs, &log)
	}

	return logs, nil
}

func (r *auditLogRepository) GetDataModificationLogs(ctx context.Context, startDate, endDate time.Time, params *utils.PaginationParams) ([]*models.AuditLog, int64, error) {
	filter := bson.M{
		"action": bson.M{
			"$in": []models.AuditAction{
				models.AuditActionCreate,
				models.AuditActionUpdate,
				models.AuditActionDelete,
			},
		},
		"created_at": bson.M{
			"$gte": startDate,
			"$lte": endDate,
		},
	}

	return r.findAuditLogsWithFilter(ctx, filter, params)
}

// Cleanup operations
func (r *auditLogRepository) DeleteOldLogs(ctx context.Context, days int) error {
	cutoffDate := time.Now().AddDate(0, 0, -days)

	result, err := r.collection.DeleteMany(ctx, bson.M{
		"created_at": bson.M{"$lt": cutoffDate},
	})
	if err != nil {
		return fmt.Errorf("failed to delete old logs: %w", err)
	}

	fmt.Printf("Deleted %d old audit logs\n", result.DeletedCount)
	return nil
}

func (r *auditLogRepository) ArchiveLogs(ctx context.Context, beforeDate time.Time) error {
	// Implementation would move logs to an archive collection or external storage
	// For now, we'll mark them as archived
	result, err := r.collection.UpdateMany(
		ctx,
		bson.M{"created_at": bson.M{"$lt": beforeDate}},
		bson.M{"$set": bson.M{"archived": true, "archived_at": time.Now()}},
	)
	if err != nil {
		return fmt.Errorf("failed to archive logs: %w", err)
	}

	fmt.Printf("Archived %d audit logs\n", result.ModifiedCount)
	return nil
}

// Analytics
func (r *auditLogRepository) GetAuditStats(ctx context.Context, days int) (map[string]interface{}, error) {
	startDate := time.Now().AddDate(0, 0, -days)

	pipeline := mongo.Pipeline{
		{{Key: "$match", Value: bson.M{
			"created_at": bson.M{"$gte": startDate},
		}}},
		{{Key: "$group", Value: bson.M{
			"_id":   "$action",
			"count": bson.M{"$sum": 1},
		}}},
		{{Key: "$sort", Value: bson.M{"count": -1}}},
	}

	cursor, err := r.collection.Aggregate(ctx, pipeline)
	if err != nil {
		return nil, fmt.Errorf("failed to get audit stats: %w", err)
	}
	defer cursor.Close(ctx)

	actionCounts := make(map[string]int64)
	totalLogs := int64(0)

	for cursor.Next(ctx) {
		var result struct {
			Action models.AuditAction `bson:"_id"`
			Count  int64              `bson:"count"`
		}

		if err := cursor.Decode(&result); err != nil {
			return nil, fmt.Errorf("failed to decode audit stats: %w", err)
		}

		actionCounts[string(result.Action)] = result.Count
		totalLogs += result.Count
	}

	// Get unique users count
	uniqueUsers, _ := r.collection.Distinct(ctx, "user_id", bson.M{
		"created_at": bson.M{"$gte": startDate},
		"user_id":    bson.M{"$ne": nil},
	})

	// Get unique IP addresses count
	uniqueIPs, _ := r.collection.Distinct(ctx, "ip_address", bson.M{
		"created_at": bson.M{"$gte": startDate},
		"ip_address": bson.M{"$ne": ""},
	})

	return map[string]interface{}{
		"total_logs":    totalLogs,
		"unique_users":  len(uniqueUsers),
		"unique_ips":    len(uniqueIPs),
		"action_counts": actionCounts,
		"period_days":   days,
		"start_date":    startDate,
		"end_date":      time.Now(),
	}, nil
}

func (r *auditLogRepository) GetUserActivitySummary(ctx context.Context, userID primitive.ObjectID, days int) (map[string]interface{}, error) {
	startDate := time.Now().AddDate(0, 0, -days)

	pipeline := mongo.Pipeline{
		{{Key: "$match", Value: bson.M{
			"user_id":    userID,
			"created_at": bson.M{"$gte": startDate},
		}}},
		{{Key: "$group", Value: bson.M{
			"_id":              nil,
			"total_actions":    bson.M{"$sum": 1},
			"unique_resources": bson.M{"$addToSet": "$resource"},
			"actions_by_type":  bson.M{"$push": "$action"},
			"last_activity":    bson.M{"$max": "$created_at"},
			"first_activity":   bson.M{"$min": "$created_at"},
		}}},
	}

	cursor, err := r.collection.Aggregate(ctx, pipeline)
	if err != nil {
		return nil, fmt.Errorf("failed to get user activity summary: %w", err)
	}
	defer cursor.Close(ctx)

	var result struct {
		TotalActions    int64                `bson:"total_actions"`
		UniqueResources []string             `bson:"unique_resources"`
		ActionsByType   []models.AuditAction `bson:"actions_by_type"`
		LastActivity    time.Time            `bson:"last_activity"`
		FirstActivity   time.Time            `bson:"first_activity"`
	}

	if cursor.Next(ctx) {
		if err := cursor.Decode(&result); err != nil {
			return nil, fmt.Errorf("failed to decode user activity summary: %w", err)
		}
	}

	// Count actions by type
	actionCounts := make(map[string]int64)
	for _, action := range result.ActionsByType {
		actionCounts[string(action)]++
	}

	// Get session info
	sessionCount, _ := r.collection.CountDocuments(ctx, bson.M{
		"user_id":    userID,
		"action":     models.AuditActionLogin,
		"created_at": bson.M{"$gte": startDate},
	})

	return map[string]interface{}{
		"user_id":          userID,
		"total_actions":    result.TotalActions,
		"unique_resources": result.UniqueResources,
		"resource_count":   len(result.UniqueResources),
		"action_counts":    actionCounts,
		"session_count":    sessionCount,
		"last_activity":    result.LastActivity,
		"first_activity":   result.FirstActivity,
		"period_days":      days,
	}, nil
}

func (r *auditLogRepository) GetSystemUsageStats(ctx context.Context, days int) (map[string]interface{}, error) {
	startDate := time.Now().AddDate(0, 0, -days)

	// Get hourly usage pattern
	hourlyPipeline := mongo.Pipeline{
		{{Key: "$match", Value: bson.M{
			"created_at": bson.M{"$gte": startDate},
		}}},
		{{Key: "$group", Value: bson.M{
			"_id": bson.M{
				"hour": bson.M{"$hour": "$created_at"},
			},
			"count": bson.M{"$sum": 1},
		}}},
		{{Key: "$sort", Value: bson.M{"_id.hour": 1}}},
	}

	cursor, err := r.collection.Aggregate(ctx, hourlyPipeline)
	if err != nil {
		return nil, fmt.Errorf("failed to get hourly usage: %w", err)
	}

	hourlyUsage := make(map[int]int64)
	for cursor.Next(ctx) {
		var result struct {
			ID struct {
				Hour int `bson:"hour"`
			} `bson:"_id"`
			Count int64 `bson:"count"`
		}

		if err := cursor.Decode(&result); err != nil {
			cursor.Close(ctx)
			return nil, fmt.Errorf("failed to decode hourly usage: %w", err)
		}

		hourlyUsage[result.ID.Hour] = result.Count
	}
	cursor.Close(ctx)

	// Get resource usage
	resourcePipeline := mongo.Pipeline{
		{{Key: "$match", Value: bson.M{
			"created_at": bson.M{"$gte": startDate},
		}}},
		{{Key: "$group", Value: bson.M{
			"_id":   "$resource",
			"count": bson.M{"$sum": 1},
		}}},
		{{Key: "$sort", Value: bson.M{"count": -1}}},
	}

	cursor, err = r.collection.Aggregate(ctx, resourcePipeline)
	if err != nil {
		return nil, fmt.Errorf("failed to get resource usage: %w", err)
	}
	defer cursor.Close(ctx)

	resourceUsage := make(map[string]int64)
	for cursor.Next(ctx) {
		var result struct {
			Resource string `bson:"_id"`
			Count    int64  `bson:"count"`
		}

		if err := cursor.Decode(&result); err != nil {
			return nil, fmt.Errorf("failed to decode resource usage: %w", err)
		}

		resourceUsage[result.Resource] = result.Count
	}

	// Get peak usage hour
	var peakHour int
	var maxCount int64
	for hour, count := range hourlyUsage {
		if count > maxCount {
			maxCount = count
			peakHour = hour
		}
	}

	return map[string]interface{}{
		"hourly_usage":   hourlyUsage,
		"resource_usage": resourceUsage,
		"peak_hour":      peakHour,
		"peak_count":     maxCount,
		"period_days":    days,
		"start_date":     startDate,
		"end_date":       time.Now(),
	}, nil
}

// Helper methods
func (r *auditLogRepository) findAuditLogsWithFilter(ctx context.Context, filter bson.M, params *utils.PaginationParams) ([]*models.AuditLog, int64, error) {
	// Add search filter if provided
	if params.Search != "" {
		searchFields := []string{"resource", "action", "ip_address", "user_agent"}
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
		return nil, 0, fmt.Errorf("failed to count audit logs: %w", err)
	}

	// Get paginated results
	opts := params.GetSortOptions()
	// Default sort by created_at descending for audit logs
	if params.Sort == "created_at" || params.Sort == "" {
		opts.SetSort(bson.D{{Key: "created_at", Value: -1}})
	}

	cursor, err := r.collection.Find(ctx, filter, opts)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to find audit logs: %w", err)
	}
	defer cursor.Close(ctx)

	var logs []*models.AuditLog
	for cursor.Next(ctx) {
		var log models.AuditLog
		if err := cursor.Decode(&log); err != nil {
			return nil, 0, fmt.Errorf("failed to decode audit log: %w", err)
		}
		logs = append(logs, &log)
	}

	return logs, total, nil
}
