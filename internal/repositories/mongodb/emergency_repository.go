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

type emergencyRepository struct {
	collection *mongo.Collection
	cache      CacheService
}

func NewEmergencyRepository(db *mongo.Database, cache CacheService) interfaces.EmergencyRepository {
	return &emergencyRepository{
		collection: db.Collection("emergencies"),
		cache:      cache,
	}
}

// Basic CRUD operations
func (r *emergencyRepository) Create(ctx context.Context, emergency *models.Emergency) error {
	emergency.ID = primitive.NewObjectID()
	emergency.CreatedAt = time.Now()
	emergency.UpdatedAt = time.Now()

	_, err := r.collection.InsertOne(ctx, emergency)
	if err != nil {
		return fmt.Errorf("failed to create emergency: %w", err)
	}

	// Cache active emergencies
	if emergency.Status == models.EmergencyStatusActive {
		r.cacheEmergency(ctx, emergency)
	}

	return nil
}

func (r *emergencyRepository) GetByID(ctx context.Context, id primitive.ObjectID) (*models.Emergency, error) {
	// Try cache first for active emergencies
	if emergency := r.getEmergencyFromCache(ctx, id.Hex()); emergency != nil {
		return emergency, nil
	}

	var emergency models.Emergency
	err := r.collection.FindOne(ctx, bson.M{"_id": id}).Decode(&emergency)
	if err != nil {
		if err == mongo.ErrNoDocuments {
			return nil, fmt.Errorf("emergency not found")
		}
		return nil, fmt.Errorf("failed to get emergency: %w", err)
	}

	// Cache if active
	if emergency.Status == models.EmergencyStatusActive {
		r.cacheEmergency(ctx, &emergency)
	}

	return &emergency, nil
}

func (r *emergencyRepository) Update(ctx context.Context, id primitive.ObjectID, updates map[string]interface{}) error {
	updates["updated_at"] = time.Now()

	_, err := r.collection.UpdateOne(
		ctx,
		bson.M{"_id": id},
		bson.M{"$set": updates},
	)
	if err != nil {
		return fmt.Errorf("failed to update emergency: %w", err)
	}

	// Invalidate cache
	r.invalidateEmergencyCache(ctx, id.Hex())

	return nil
}

func (r *emergencyRepository) Delete(ctx context.Context, id primitive.ObjectID) error {
	_, err := r.collection.DeleteOne(ctx, bson.M{"_id": id})
	if err != nil {
		return fmt.Errorf("failed to delete emergency: %w", err)
	}

	// Invalidate cache
	r.invalidateEmergencyCache(ctx, id.Hex())

	return nil
}

// User emergencies
func (r *emergencyRepository) GetByUserID(ctx context.Context, userID primitive.ObjectID, params *utils.PaginationParams) ([]*models.Emergency, int64, error) {
	filter := bson.M{"user_id": userID}
	return r.findEmergenciesWithFilter(ctx, filter, params)
}

func (r *emergencyRepository) GetActiveEmergenciesByUser(ctx context.Context, userID primitive.ObjectID) ([]*models.Emergency, error) {
	filter := bson.M{
		"user_id": userID,
		"status":  models.EmergencyStatusActive,
	}

	cursor, err := r.collection.Find(ctx, filter, options.Find().SetSort(bson.D{{Key: "created_at", Value: -1}}))
	if err != nil {
		return nil, fmt.Errorf("failed to find active emergencies by user: %w", err)
	}
	defer cursor.Close(ctx)

	var emergencies []*models.Emergency
	for cursor.Next(ctx) {
		var emergency models.Emergency
		if err := cursor.Decode(&emergency); err != nil {
			return nil, fmt.Errorf("failed to decode emergency: %w", err)
		}
		emergencies = append(emergencies, &emergency)
	}

	return emergencies, nil
}

// Ride emergencies
func (r *emergencyRepository) GetByRideID(ctx context.Context, rideID primitive.ObjectID) ([]*models.Emergency, error) {
	filter := bson.M{"ride_id": rideID}

	cursor, err := r.collection.Find(ctx, filter, options.Find().SetSort(bson.D{{Key: "created_at", Value: -1}}))
	if err != nil {
		return nil, fmt.Errorf("failed to find emergencies by ride: %w", err)
	}
	defer cursor.Close(ctx)

	var emergencies []*models.Emergency
	for cursor.Next(ctx) {
		var emergency models.Emergency
		if err := cursor.Decode(&emergency); err != nil {
			return nil, fmt.Errorf("failed to decode emergency: %w", err)
		}
		emergencies = append(emergencies, &emergency)
	}

	return emergencies, nil
}

func (r *emergencyRepository) GetActiveEmergencyForRide(ctx context.Context, rideID primitive.ObjectID) (*models.Emergency, error) {
	var emergency models.Emergency
	err := r.collection.FindOne(ctx, bson.M{
		"ride_id": rideID,
		"status":  models.EmergencyStatusActive,
	}).Decode(&emergency)
	if err != nil {
		if err == mongo.ErrNoDocuments {
			return nil, fmt.Errorf("no active emergency found for ride")
		}
		return nil, fmt.Errorf("failed to get active emergency for ride: %w", err)
	}

	return &emergency, nil
}

// Status operations
func (r *emergencyRepository) UpdateStatus(ctx context.Context, id primitive.ObjectID, status models.EmergencyStatus) error {
	updates := map[string]interface{}{
		"status": status,
	}

	if status == models.EmergencyStatusResolved {
		updates["resolved_at"] = time.Now()
	}

	return r.Update(ctx, id, updates)
}

func (r *emergencyRepository) ResolveEmergency(ctx context.Context, id primitive.ObjectID, resolvedBy primitive.ObjectID, notes string) error {
	updates := map[string]interface{}{
		"status":      models.EmergencyStatusResolved,
		"resolved_by": resolvedBy,
		"notes":       notes,
		"resolved_at": time.Now(),
	}

	return r.Update(ctx, id, updates)
}

func (r *emergencyRepository) GetByStatus(ctx context.Context, status models.EmergencyStatus, params *utils.PaginationParams) ([]*models.Emergency, int64, error) {
	filter := bson.M{"status": status}
	return r.findEmergenciesWithFilter(ctx, filter, params)
}

// Type filtering
func (r *emergencyRepository) GetByType(ctx context.Context, emergencyType models.EmergencyType, params *utils.PaginationParams) ([]*models.Emergency, int64, error) {
	filter := bson.M{"type": emergencyType}
	return r.findEmergenciesWithFilter(ctx, filter, params)
}

func (r *emergencyRepository) GetActiveEmergencies(ctx context.Context) ([]*models.Emergency, error) {
	filter := bson.M{"status": models.EmergencyStatusActive}

	cursor, err := r.collection.Find(ctx, filter, options.Find().SetSort(bson.D{{Key: "created_at", Value: -1}}))
	if err != nil {
		return nil, fmt.Errorf("failed to find active emergencies: %w", err)
	}
	defer cursor.Close(ctx)

	var emergencies []*models.Emergency
	for cursor.Next(ctx) {
		var emergency models.Emergency
		if err := cursor.Decode(&emergency); err != nil {
			return nil, fmt.Errorf("failed to decode emergency: %w", err)
		}
		emergencies = append(emergencies, &emergency)
	}

	return emergencies, nil
}

// Location-based queries
func (r *emergencyRepository) GetEmergenciesInArea(ctx context.Context, bounds *utils.Bounds) ([]*models.Emergency, error) {
	filter := bson.M{
		"location": bson.M{
			"$geoWithin": bson.M{
				"$box": [][]float64{
					{bounds.Southwest.Lng, bounds.Southwest.Lat},
					{bounds.Northeast.Lng, bounds.Northeast.Lat},
				},
			},
		},
		"status": models.EmergencyStatusActive,
	}

	cursor, err := r.collection.Find(ctx, filter, options.Find().SetSort(bson.D{{Key: "created_at", Value: -1}}))
	if err != nil {
		return nil, fmt.Errorf("failed to find emergencies in area: %w", err)
	}
	defer cursor.Close(ctx)

	var emergencies []*models.Emergency
	for cursor.Next(ctx) {
		var emergency models.Emergency
		if err := cursor.Decode(&emergency); err != nil {
			return nil, fmt.Errorf("failed to decode emergency: %w", err)
		}
		emergencies = append(emergencies, &emergency)
	}

	return emergencies, nil
}

func (r *emergencyRepository) GetNearbyEmergencies(ctx context.Context, lat, lng, radiusKM float64) ([]*models.Emergency, error) {
	// Convert radius from kilometers to meters
	radiusMeters := radiusKM * 1000

	filter := bson.M{
		"location": bson.M{
			"$near": bson.M{
				"$geometry": bson.M{
					"type":        "Point",
					"coordinates": []float64{lng, lat},
				},
				"$maxDistance": radiusMeters,
			},
		},
		"status": models.EmergencyStatusActive,
	}

	cursor, err := r.collection.Find(ctx, filter, options.Find().SetLimit(50))
	if err != nil {
		return nil, fmt.Errorf("failed to find nearby emergencies: %w", err)
	}
	defer cursor.Close(ctx)

	var emergencies []*models.Emergency
	for cursor.Next(ctx) {
		var emergency models.Emergency
		if err := cursor.Decode(&emergency); err != nil {
			return nil, fmt.Errorf("failed to decode emergency: %w", err)
		}
		emergencies = append(emergencies, &emergency)
	}

	return emergencies, nil
}

// Time-based queries
func (r *emergencyRepository) GetEmergenciesByDateRange(ctx context.Context, startDate, endDate time.Time, params *utils.PaginationParams) ([]*models.Emergency, int64, error) {
	filter := bson.M{
		"created_at": bson.M{
			"$gte": startDate,
			"$lte": endDate,
		},
	}
	return r.findEmergenciesWithFilter(ctx, filter, params)
}

func (r *emergencyRepository) GetRecentEmergencies(ctx context.Context, hours int) ([]*models.Emergency, error) {
	startTime := time.Now().Add(-time.Duration(hours) * time.Hour)
	filter := bson.M{
		"created_at": bson.M{"$gte": startTime},
	}

	cursor, err := r.collection.Find(ctx, filter, options.Find().SetSort(bson.D{{Key: "created_at", Value: -1}}))
	if err != nil {
		return nil, fmt.Errorf("failed to find recent emergencies: %w", err)
	}
	defer cursor.Close(ctx)

	var emergencies []*models.Emergency
	for cursor.Next(ctx) {
		var emergency models.Emergency
		if err := cursor.Decode(&emergency); err != nil {
			return nil, fmt.Errorf("failed to decode emergency: %w", err)
		}
		emergencies = append(emergencies, &emergency)
	}

	return emergencies, nil
}

func (r *emergencyRepository) GetUnresolvedEmergencies(ctx context.Context, olderThan time.Duration) ([]*models.Emergency, error) {
	cutoffTime := time.Now().Add(-olderThan)
	filter := bson.M{
		"status":     models.EmergencyStatusActive,
		"created_at": bson.M{"$lt": cutoffTime},
	}

	cursor, err := r.collection.Find(ctx, filter, options.Find().SetSort(bson.D{{Key: "created_at", Value: 1}}))
	if err != nil {
		return nil, fmt.Errorf("failed to find unresolved emergencies: %w", err)
	}
	defer cursor.Close(ctx)

	var emergencies []*models.Emergency
	for cursor.Next(ctx) {
		var emergency models.Emergency
		if err := cursor.Decode(&emergency); err != nil {
			return nil, fmt.Errorf("failed to decode emergency: %w", err)
		}
		emergencies = append(emergencies, &emergency)
	}

	return emergencies, nil
}

// Response tracking
func (r *emergencyRepository) UpdateResponseTime(ctx context.Context, id primitive.ObjectID, responseTime time.Time) error {
	updates := map[string]interface{}{
		"response_time": responseTime,
	}

	return r.Update(ctx, id, updates)
}

func (r *emergencyRepository) GetAverageResponseTime(ctx context.Context, days int) (time.Duration, error) {
	startDate := time.Now().AddDate(0, 0, -days)

	pipeline := mongo.Pipeline{
		{{"$match", bson.M{
			"created_at":    bson.M{"$gte": startDate},
			"response_time": bson.M{"$exists": true, "$ne": nil},
		}}},
		{{"$project", bson.M{
			"response_duration": bson.M{
				"$subtract": []interface{}{"$response_time", "$created_at"},
			},
		}}},
		{{"$group", bson.M{
			"_id":               nil,
			"avg_response_time": bson.M{"$avg": "$response_duration"},
		}}},
	}

	cursor, err := r.collection.Aggregate(ctx, pipeline)
	if err != nil {
		return 0, fmt.Errorf("failed to calculate average response time: %w", err)
	}
	defer cursor.Close(ctx)

	var result struct {
		AvgResponseTime float64 `bson:"avg_response_time"`
	}

	if cursor.Next(ctx) {
		if err := cursor.Decode(&result); err != nil {
			return 0, fmt.Errorf("failed to decode average response time: %w", err)
		}
	}

	// Convert milliseconds to duration
	return time.Duration(result.AvgResponseTime) * time.Millisecond, nil
}

// Analytics
func (r *emergencyRepository) GetEmergencyStats(ctx context.Context, days int) (map[string]interface{}, error) {
	startDate := time.Now().AddDate(0, 0, -days)

	// Total emergencies
	totalEmergencies, err := r.collection.CountDocuments(ctx, bson.M{
		"created_at": bson.M{"$gte": startDate},
	})
	if err != nil {
		return nil, fmt.Errorf("failed to count total emergencies: %w", err)
	}

	// Emergencies by status
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
		return nil, fmt.Errorf("failed to get emergencies by status: %w", err)
	}
	defer cursor.Close(ctx)

	statusCounts := make(map[string]int64)
	for cursor.Next(ctx) {
		var result struct {
			Status models.EmergencyStatus `bson:"_id"`
			Count  int64                  `bson:"count"`
		}

		if err := cursor.Decode(&result); err != nil {
			return nil, fmt.Errorf("failed to decode status count: %w", err)
		}

		statusCounts[string(result.Status)] = result.Count
	}

	// Emergencies by type
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
		return nil, fmt.Errorf("failed to get emergencies by type: %w", err)
	}
	defer cursor.Close(ctx)

	typeCounts := make(map[string]int64)
	for cursor.Next(ctx) {
		var result struct {
			Type  models.EmergencyType `bson:"_id"`
			Count int64                `bson:"count"`
		}

		if err := cursor.Decode(&result); err != nil {
			return nil, fmt.Errorf("failed to decode type count: %w", err)
		}

		typeCounts[string(result.Type)] = result.Count
	}

	// Resolution rate
	resolvedCount := statusCounts[string(models.EmergencyStatusResolved)]
	resolutionRate := float64(0)
	if totalEmergencies > 0 {
		resolutionRate = float64(resolvedCount) / float64(totalEmergencies) * 100
	}

	// Active emergencies
	activeCount := statusCounts[string(models.EmergencyStatusActive)]

	return map[string]interface{}{
		"total_emergencies": totalEmergencies,
		"status_counts":     statusCounts,
		"type_counts":       typeCounts,
		"resolution_rate":   resolutionRate,
		"active_count":      activeCount,
		"resolved_count":    resolvedCount,
		"period_days":       days,
		"start_date":        startDate,
		"end_date":          time.Now(),
	}, nil
}

func (r *emergencyRepository) GetEmergencyTrends(ctx context.Context, days int) ([]map[string]interface{}, error) {
	startDate := time.Now().AddDate(0, 0, -days)

	pipeline := mongo.Pipeline{
		{{"$match", bson.M{
			"created_at": bson.M{"$gte": startDate},
		}}},
		{{"$group", bson.M{
			"_id": bson.M{
				"date": bson.M{"$dateToString": bson.M{
					"format": "%Y-%m-%d",
					"date":   "$created_at",
				}},
				"type": "$type",
			},
			"count": bson.M{"$sum": 1},
		}}},
		{{"$sort", bson.M{"_id.date": 1}}},
	}

	cursor, err := r.collection.Aggregate(ctx, pipeline)
	if err != nil {
		return nil, fmt.Errorf("failed to get emergency trends: %w", err)
	}
	defer cursor.Close(ctx)

	var results []map[string]interface{}
	for cursor.Next(ctx) {
		var result struct {
			ID struct {
				Date string               `bson:"date"`
				Type models.EmergencyType `bson:"type"`
			} `bson:"_id"`
			Count int64 `bson:"count"`
		}

		if err := cursor.Decode(&result); err != nil {
			return nil, fmt.Errorf("failed to decode emergency trend: %w", err)
		}

		results = append(results, map[string]interface{}{
			"date":  result.ID.Date,
			"type":  string(result.ID.Type),
			"count": result.Count,
		})
	}

	return results, nil
}

func (r *emergencyRepository) GetResponseTimeStats(ctx context.Context, days int) (map[string]interface{}, error) {
	startDate := time.Now().AddDate(0, 0, -days)

	pipeline := mongo.Pipeline{
		{{"$match", bson.M{
			"created_at":    bson.M{"$gte": startDate},
			"response_time": bson.M{"$exists": true, "$ne": nil},
		}}},
		{{"$project", bson.M{
			"type": 1,
			"response_duration": bson.M{
				"$divide": []interface{}{
					bson.M{"$subtract": []interface{}{"$response_time", "$created_at"}},
					1000, // Convert to seconds
				},
			},
		}}},
		{{"$group", bson.M{
			"_id":               nil,
			"avg_response_time": bson.M{"$avg": "$response_duration"},
			"min_response_time": bson.M{"$min": "$response_duration"},
			"max_response_time": bson.M{"$max": "$response_duration"},
			"total_responses":   bson.M{"$sum": 1},
		}}},
	}

	cursor, err := r.collection.Aggregate(ctx, pipeline)
	if err != nil {
		return nil, fmt.Errorf("failed to get response time stats: %w", err)
	}
	defer cursor.Close(ctx)

	var result struct {
		AvgResponseTime float64 `bson:"avg_response_time"`
		MinResponseTime float64 `bson:"min_response_time"`
		MaxResponseTime float64 `bson:"max_response_time"`
		TotalResponses  int64   `bson:"total_responses"`
	}

	if cursor.Next(ctx) {
		if err := cursor.Decode(&result); err != nil {
			return nil, fmt.Errorf("failed to decode response time stats: %w", err)
		}
	}

	// Response time by type
	typePipeline := mongo.Pipeline{
		{{"$match", bson.M{
			"created_at":    bson.M{"$gte": startDate},
			"response_time": bson.M{"$exists": true, "$ne": nil},
		}}},
		{{"$project", bson.M{
			"type": 1,
			"response_duration": bson.M{
				"$divide": []interface{}{
					bson.M{"$subtract": []interface{}{"$response_time", "$created_at"}},
					1000,
				},
			},
		}}},
		{{"$group", bson.M{
			"_id":               "$type",
			"avg_response_time": bson.M{"$avg": "$response_duration"},
			"count":             bson.M{"$sum": 1},
		}}},
	}

	cursor, err = r.collection.Aggregate(ctx, typePipeline)
	if err != nil {
		return nil, fmt.Errorf("failed to get response time by type: %w", err)
	}
	defer cursor.Close(ctx)

	responseTimeByType := make(map[string]map[string]interface{})
	for cursor.Next(ctx) {
		var typeResult struct {
			Type            models.EmergencyType `bson:"_id"`
			AvgResponseTime float64              `bson:"avg_response_time"`
			Count           int64                `bson:"count"`
		}

		if err := cursor.Decode(&typeResult); err != nil {
			return nil, fmt.Errorf("failed to decode type response time: %w", err)
		}

		responseTimeByType[string(typeResult.Type)] = map[string]interface{}{
			"avg_response_time": typeResult.AvgResponseTime,
			"count":             typeResult.Count,
		}
	}

	return map[string]interface{}{
		"avg_response_time":     result.AvgResponseTime,
		"min_response_time":     result.MinResponseTime,
		"max_response_time":     result.MaxResponseTime,
		"total_responses":       result.TotalResponses,
		"response_time_by_type": responseTimeByType,
		"period_days":           days,
		"start_date":            startDate,
		"end_date":              time.Now(),
	}, nil
}

// Helper methods
func (r *emergencyRepository) findEmergenciesWithFilter(ctx context.Context, filter bson.M, params *utils.PaginationParams) ([]*models.Emergency, int64, error) {
	// Add search filter if provided
	if params.Search != "" {
		searchFields := []string{"description", "notes", "emergency_number"}
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
		return nil, 0, fmt.Errorf("failed to count emergencies: %w", err)
	}

	// Get paginated results
	opts := params.GetSortOptions()
	// Default sort by created_at descending for emergencies
	if params.Sort == "created_at" || params.Sort == "" {
		opts.SetSort(bson.D{{Key: "created_at", Value: -1}})
	}

	cursor, err := r.collection.Find(ctx, filter, opts)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to find emergencies: %w", err)
	}
	defer cursor.Close(ctx)

	var emergencies []*models.Emergency
	for cursor.Next(ctx) {
		var emergency models.Emergency
		if err := cursor.Decode(&emergency); err != nil {
			return nil, 0, fmt.Errorf("failed to decode emergency: %w", err)
		}
		emergencies = append(emergencies, &emergency)
	}

	return emergencies, total, nil
}

// Cache operations
func (r *emergencyRepository) cacheEmergency(ctx context.Context, emergency *models.Emergency) {
	if r.cache != nil && emergency.Status == models.EmergencyStatusActive {
		cacheKey := fmt.Sprintf("emergency:%s", emergency.ID.Hex())
		r.cache.Set(ctx, cacheKey, emergency, 5*time.Minute) // Shorter TTL for emergencies
	}
}

func (r *emergencyRepository) getEmergencyFromCache(ctx context.Context, emergencyID string) *models.Emergency {
	if r.cache == nil {
		return nil
	}

	cacheKey := fmt.Sprintf("emergency:%s", emergencyID)
	var emergency models.Emergency
	err := r.cache.Get(ctx, cacheKey, &emergency)
	if err != nil {
		return nil
	}

	return &emergency
}

func (r *emergencyRepository) invalidateEmergencyCache(ctx context.Context, emergencyID string) {
	if r.cache != nil {
		cacheKey := fmt.Sprintf("emergency:%s", emergencyID)
		r.cache.Delete(ctx, cacheKey)
	}
}
