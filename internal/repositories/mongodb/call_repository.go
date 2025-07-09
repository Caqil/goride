package mongodb

import (
	"context"
	"fmt"
	"time"

	"goride/internal/models"
	"goride/internal/repositories/interfaces"
	"goride/internal/utils"
	"goride/pkg/cache"
	"goride/pkg/database"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

type callRepository struct {
	collection *mongo.Collection
	cache      *cache.RedisCache
}

func NewCallRepository(db *database.MongoDB, cache *cache.RedisCache) interfaces.CallRepository {
	return &callRepository{
		collection: db.Collection("calls"),
		cache:      cache,
	}
}

// Call CRUD operations
func (r *callRepository) CreateCall(ctx context.Context, call *models.Call) error {
	call.ID = primitive.NewObjectID()
	call.CreatedAt = time.Now()
	call.UpdatedAt = time.Now()

	_, err := r.collection.InsertOne(ctx, call)
	if err != nil {
		return fmt.Errorf("failed to create call: %w", err)
	}

	// Invalidate cache
	r.invalidateCallCache(ctx, call.CallerID, call.CalleeID)

	return nil
}

func (r *callRepository) GetCallByID(ctx context.Context, id primitive.ObjectID) (*models.Call, error) {
	// Try cache first
	cacheKey := fmt.Sprintf("call:%s", id.Hex())
	var call models.Call
	if r.cache != nil {
		if err := r.cache.Get(ctx, cacheKey, &call); err == nil {
			return &call, nil
		}
	}

	// Fetch from database
	err := r.collection.FindOne(ctx, bson.M{"_id": id}).Decode(&call)
	if err != nil {
		if err == mongo.ErrNoDocuments {
			return nil, fmt.Errorf("call not found")
		}
		return nil, fmt.Errorf("failed to get call: %w", err)
	}

	// Cache the result
	if r.cache != nil {
		r.cache.Set(ctx, cacheKey, call, time.Hour)
	}

	return &call, nil
}

func (r *callRepository) GetCallBySID(ctx context.Context, callSID string) (*models.Call, error) {
	// Try cache first
	cacheKey := fmt.Sprintf("call_sid:%s", callSID)
	var call models.Call
	if r.cache != nil {
		if err := r.cache.Get(ctx, cacheKey, &call); err == nil {
			return &call, nil
		}
	}

	// Fetch from database
	err := r.collection.FindOne(ctx, bson.M{"call_sid": callSID}).Decode(&call)
	if err != nil {
		if err == mongo.ErrNoDocuments {
			return nil, fmt.Errorf("call not found")
		}
		return nil, fmt.Errorf("failed to get call: %w", err)
	}

	// Cache the result
	if r.cache != nil {
		r.cache.Set(ctx, cacheKey, call, time.Hour)
		r.cache.Set(ctx, fmt.Sprintf("call:%s", call.ID.Hex()), call, time.Hour)
	}

	return &call, nil
}

func (r *callRepository) UpdateCall(ctx context.Context, id primitive.ObjectID, updates map[string]interface{}) error {
	updates["updated_at"] = time.Now()

	_, err := r.collection.UpdateOne(
		ctx,
		bson.M{"_id": id},
		bson.M{"$set": updates},
	)
	if err != nil {
		return fmt.Errorf("failed to update call: %w", err)
	}

	// Invalidate cache
	if r.cache != nil {
		r.cache.Delete(ctx, fmt.Sprintf("call:%s", id.Hex()))
	}

	return nil
}

func (r *callRepository) DeleteCall(ctx context.Context, id primitive.ObjectID) error {
	_, err := r.collection.DeleteOne(ctx, bson.M{"_id": id})
	if err != nil {
		return fmt.Errorf("failed to delete call: %w", err)
	}

	// Invalidate cache
	if r.cache != nil {
		r.cache.Delete(ctx, fmt.Sprintf("call:%s", id.Hex()))
	}

	return nil
}

// Call queries
func (r *callRepository) GetCallsByUser(ctx context.Context, userID primitive.ObjectID, params *utils.PaginationParams) ([]*models.Call, int64, error) {
	filter := bson.M{
		"$or": []bson.M{
			{"caller_id": userID},
			{"callee_id": userID},
		},
	}

	return r.findCalls(ctx, filter, params)
}

func (r *callRepository) GetCallsByRide(ctx context.Context, rideID primitive.ObjectID) ([]*models.Call, error) {
	filter := bson.M{"ride_id": rideID}

	cursor, err := r.collection.Find(ctx, filter, options.Find().SetSort(bson.M{"created_at": -1}))
	if err != nil {
		return nil, fmt.Errorf("failed to get calls by ride: %w", err)
	}
	defer cursor.Close(ctx)

	var calls []*models.Call
	if err := cursor.All(ctx, &calls); err != nil {
		return nil, fmt.Errorf("failed to decode calls: %w", err)
	}

	return calls, nil
}

func (r *callRepository) GetCallsByStatus(ctx context.Context, status models.CallStatus, params *utils.PaginationParams) ([]*models.Call, int64, error) {
	filter := bson.M{"status": status}
	return r.findCalls(ctx, filter, params)
}

func (r *callRepository) GetCallsByType(ctx context.Context, callType models.CallType, params *utils.PaginationParams) ([]*models.Call, int64, error) {
	filter := bson.M{"type": callType}
	return r.findCalls(ctx, filter, params)
}

func (r *callRepository) GetCallsByDateRange(ctx context.Context, startDate, endDate time.Time, params *utils.PaginationParams) ([]*models.Call, int64, error) {
	filter := bson.M{
		"created_at": bson.M{
			"$gte": startDate,
			"$lte": endDate,
		},
	}
	return r.findCalls(ctx, filter, params)
}

// Emergency calls
func (r *callRepository) GetEmergencyCalls(ctx context.Context, params *utils.PaginationParams) ([]*models.Call, int64, error) {
	filter := bson.M{"is_emergency": true}
	return r.findCalls(ctx, filter, params)
}

func (r *callRepository) GetUserEmergencyCalls(ctx context.Context, userID primitive.ObjectID) ([]*models.Call, error) {
	filter := bson.M{
		"caller_id":    userID,
		"is_emergency": true,
	}

	cursor, err := r.collection.Find(ctx, filter, options.Find().SetSort(bson.M{"created_at": -1}))
	if err != nil {
		return nil, fmt.Errorf("failed to get emergency calls: %w", err)
	}
	defer cursor.Close(ctx)

	var calls []*models.Call
	if err := cursor.All(ctx, &calls); err != nil {
		return nil, fmt.Errorf("failed to decode emergency calls: %w", err)
	}

	return calls, nil
}

// Active calls
func (r *callRepository) GetActiveCalls(ctx context.Context) ([]*models.Call, error) {
	filter := bson.M{
		"status": bson.M{
			"$in": []models.CallStatus{
				models.CallStatusInitiated,
				models.CallStatusRinging,
				models.CallStatusAnswered,
			},
		},
	}

	cursor, err := r.collection.Find(ctx, filter, options.Find().SetSort(bson.M{"created_at": -1}))
	if err != nil {
		return nil, fmt.Errorf("failed to get active calls: %w", err)
	}
	defer cursor.Close(ctx)

	var calls []*models.Call
	if err := cursor.All(ctx, &calls); err != nil {
		return nil, fmt.Errorf("failed to decode active calls: %w", err)
	}

	return calls, nil
}

func (r *callRepository) GetActiveCallsByUser(ctx context.Context, userID primitive.ObjectID) ([]*models.Call, error) {
	filter := bson.M{
		"$or": []bson.M{
			{"caller_id": userID},
			{"callee_id": userID},
		},
		"status": bson.M{
			"$in": []models.CallStatus{
				models.CallStatusInitiated,
				models.CallStatusRinging,
				models.CallStatusAnswered,
			},
		},
	}

	cursor, err := r.collection.Find(ctx, filter, options.Find().SetSort(bson.M{"created_at": -1}))
	if err != nil {
		return nil, fmt.Errorf("failed to get active calls by user: %w", err)
	}
	defer cursor.Close(ctx)

	var calls []*models.Call
	if err := cursor.All(ctx, &calls); err != nil {
		return nil, fmt.Errorf("failed to decode active calls: %w", err)
	}

	return calls, nil
}

func (r *callRepository) GetActiveCallsByRide(ctx context.Context, rideID primitive.ObjectID) ([]*models.Call, error) {
	filter := bson.M{
		"ride_id": rideID,
		"status": bson.M{
			"$in": []models.CallStatus{
				models.CallStatusInitiated,
				models.CallStatusRinging,
				models.CallStatusAnswered,
			},
		},
	}

	cursor, err := r.collection.Find(ctx, filter, options.Find().SetSort(bson.M{"created_at": -1}))
	if err != nil {
		return nil, fmt.Errorf("failed to get active calls by ride: %w", err)
	}
	defer cursor.Close(ctx)

	var calls []*models.Call
	if err := cursor.All(ctx, &calls); err != nil {
		return nil, fmt.Errorf("failed to decode active calls: %w", err)
	}

	return calls, nil
}

// Call analytics
func (r *callRepository) GetCallStats(ctx context.Context, startDate, endDate time.Time) (*models.CallAnalytics, error) {
	pipeline := []bson.M{
		{
			"$match": bson.M{
				"created_at": bson.M{
					"$gte": startDate,
					"$lte": endDate,
				},
			},
		},
		{
			"$group": bson.M{
				"_id":         nil,
				"total_calls": bson.M{"$sum": 1},
				"successful_calls": bson.M{
					"$sum": bson.M{
						"$cond": []interface{}{
							bson.M{"$eq": []interface{}{"$status", models.CallStatusCompleted}},
							1,
							0,
						},
					},
				},
				"failed_calls": bson.M{
					"$sum": bson.M{
						"$cond": []interface{}{
							bson.M{"$in": []interface{}{"$status", []models.CallStatus{
								models.CallStatusFailed,
								models.CallStatusCancelled,
								models.CallStatusNoAnswer,
							}}},
							1,
							0,
						},
					},
				},
				"total_duration":   bson.M{"$sum": "$duration"},
				"average_duration": bson.M{"$avg": "$duration"},
				"average_quality":  bson.M{"$avg": "$quality_rating"},
				"emergency_calls": bson.M{
					"$sum": bson.M{
						"$cond": []interface{}{
							bson.M{"$eq": []interface{}{"$is_emergency", true}},
							1,
							0,
						},
					},
				},
			},
		},
	}

	cursor, err := r.collection.Aggregate(ctx, pipeline)
	if err != nil {
		return nil, fmt.Errorf("failed to get call stats: %w", err)
	}
	defer cursor.Close(ctx)

	var result struct {
		TotalCalls      int64   `bson:"total_calls"`
		SuccessfulCalls int64   `bson:"successful_calls"`
		FailedCalls     int64   `bson:"failed_calls"`
		TotalDuration   int64   `bson:"total_duration"`
		AverageDuration float64 `bson:"average_duration"`
		AverageQuality  float64 `bson:"average_quality"`
		EmergencyCalls  int64   `bson:"emergency_calls"`
	}

	if cursor.Next(ctx) {
		if err := cursor.Decode(&result); err != nil {
			return nil, fmt.Errorf("failed to decode call stats: %w", err)
		}
	}

	// Get calls by type and status
	callsByType, err := r.getCallsByType(ctx, startDate, endDate)
	if err != nil {
		return nil, err
	}

	callsByStatus, err := r.getCallsByStatus(ctx, startDate, endDate)
	if err != nil {
		return nil, err
	}

	return &models.CallAnalytics{
		TotalCalls:      result.TotalCalls,
		SuccessfulCalls: result.SuccessfulCalls,
		FailedCalls:     result.FailedCalls,
		AverageDuration: result.AverageDuration,
		TotalDuration:   result.TotalDuration,
		CallsByType:     callsByType,
		CallsByStatus:   callsByStatus,
		AverageQuality:  result.AverageQuality,
		EmergencyCalls:  result.EmergencyCalls,
	}, nil
}

func (r *callRepository) GetUserCallStats(ctx context.Context, userID primitive.ObjectID, days int) (map[string]interface{}, error) {
	startDate := time.Now().AddDate(0, 0, -days)
	endDate := time.Now()

	pipeline := []bson.M{
		{
			"$match": bson.M{
				"$or": []bson.M{
					{"caller_id": userID},
					{"callee_id": userID},
				},
				"created_at": bson.M{
					"$gte": startDate,
					"$lte": endDate,
				},
			},
		},
		{
			"$group": bson.M{
				"_id":              nil,
				"total_calls":      bson.M{"$sum": 1},
				"total_duration":   bson.M{"$sum": "$duration"},
				"average_duration": bson.M{"$avg": "$duration"},
				"average_quality":  bson.M{"$avg": "$quality_rating"},
			},
		},
	}

	cursor, err := r.collection.Aggregate(ctx, pipeline)
	if err != nil {
		return nil, fmt.Errorf("failed to get user call stats: %w", err)
	}
	defer cursor.Close(ctx)

	var result map[string]interface{}
	if cursor.Next(ctx) {
		if err := cursor.Decode(&result); err != nil {
			return nil, fmt.Errorf("failed to decode user call stats: %w", err)
		}
	} else {
		result = map[string]interface{}{
			"total_calls":      0,
			"total_duration":   0,
			"average_duration": 0.0,
			"average_quality":  0.0,
		}
	}

	return result, nil
}

func (r *callRepository) GetCallDurationStats(ctx context.Context, days int) (map[string]interface{}, error) {
	startDate := time.Now().AddDate(0, 0, -days)
	endDate := time.Now()

	pipeline := []bson.M{
		{
			"$match": bson.M{
				"created_at": bson.M{
					"$gte": startDate,
					"$lte": endDate,
				},
				"status": models.CallStatusCompleted,
			},
		},
		{
			"$group": bson.M{
				"_id":              nil,
				"min_duration":     bson.M{"$min": "$duration"},
				"max_duration":     bson.M{"$max": "$duration"},
				"average_duration": bson.M{"$avg": "$duration"},
				"total_duration":   bson.M{"$sum": "$duration"},
			},
		},
	}

	cursor, err := r.collection.Aggregate(ctx, pipeline)
	if err != nil {
		return nil, fmt.Errorf("failed to get call duration stats: %w", err)
	}
	defer cursor.Close(ctx)

	var result map[string]interface{}
	if cursor.Next(ctx) {
		if err := cursor.Decode(&result); err != nil {
			return nil, fmt.Errorf("failed to decode call duration stats: %w", err)
		}
	}

	return result, nil
}

func (r *callRepository) GetCallQualityStats(ctx context.Context, days int) (map[string]interface{}, error) {
	startDate := time.Now().AddDate(0, 0, -days)
	endDate := time.Now()

	pipeline := []bson.M{
		{
			"$match": bson.M{
				"created_at": bson.M{
					"$gte": startDate,
					"$lte": endDate,
				},
				"quality_rating": bson.M{"$gt": 0},
			},
		},
		{
			"$group": bson.M{
				"_id":             nil,
				"average_quality": bson.M{"$avg": "$quality_rating"},
				"min_quality":     bson.M{"$min": "$quality_rating"},
				"max_quality":     bson.M{"$max": "$quality_rating"},
				"total_ratings":   bson.M{"$sum": 1},
			},
		},
	}

	cursor, err := r.collection.Aggregate(ctx, pipeline)
	if err != nil {
		return nil, fmt.Errorf("failed to get call quality stats: %w", err)
	}
	defer cursor.Close(ctx)

	var result map[string]interface{}
	if cursor.Next(ctx) {
		if err := cursor.Decode(&result); err != nil {
			return nil, fmt.Errorf("failed to decode call quality stats: %w", err)
		}
	}

	return result, nil
}

// Search and filtering
func (r *callRepository) SearchCalls(ctx context.Context, query string, params *utils.PaginationParams) ([]*models.Call, int64, error) {
	filter := bson.M{
		"$or": []bson.M{
			{"call_sid": bson.M{"$regex": query, "$options": "i"}},
			{"from_number": bson.M{"$regex": query, "$options": "i"}},
			{"to_number": bson.M{"$regex": query, "$options": "i"}},
			{"proxy_number": bson.M{"$regex": query, "$options": "i"}},
		},
	}
	return r.findCalls(ctx, filter, params)
}

func (r *callRepository) GetFailedCalls(ctx context.Context, days int) ([]*models.Call, error) {
	startDate := time.Now().AddDate(0, 0, -days)
	filter := bson.M{
		"status": bson.M{
			"$in": []models.CallStatus{
				models.CallStatusFailed,
				models.CallStatusCancelled,
				models.CallStatusNoAnswer,
			},
		},
		"created_at": bson.M{"$gte": startDate},
	}

	cursor, err := r.collection.Find(ctx, filter, options.Find().SetSort(bson.M{"created_at": -1}))
	if err != nil {
		return nil, fmt.Errorf("failed to get failed calls: %w", err)
	}
	defer cursor.Close(ctx)

	var calls []*models.Call
	if err := cursor.All(ctx, &calls); err != nil {
		return nil, fmt.Errorf("failed to decode failed calls: %w", err)
	}

	return calls, nil
}

func (r *callRepository) GetHighCostCalls(ctx context.Context, threshold float64, days int) ([]*models.Call, error) {
	startDate := time.Now().AddDate(0, 0, -days)
	filter := bson.M{
		"cost":       bson.M{"$gte": threshold},
		"created_at": bson.M{"$gte": startDate},
	}

	cursor, err := r.collection.Find(ctx, filter, options.Find().SetSort(bson.M{"cost": -1}))
	if err != nil {
		return nil, fmt.Errorf("failed to get high cost calls: %w", err)
	}
	defer cursor.Close(ctx)

	var calls []*models.Call
	if err := cursor.All(ctx, &calls); err != nil {
		return nil, fmt.Errorf("failed to decode high cost calls: %w", err)
	}

	return calls, nil
}

// Proxy number management
func (r *callRepository) GetProxyNumberUsage(ctx context.Context, proxyNumber string) ([]*models.Call, error) {
	filter := bson.M{"proxy_number": proxyNumber}

	cursor, err := r.collection.Find(ctx, filter, options.Find().SetSort(bson.M{"created_at": -1}))
	if err != nil {
		return nil, fmt.Errorf("failed to get proxy number usage: %w", err)
	}
	defer cursor.Close(ctx)

	var calls []*models.Call
	if err := cursor.All(ctx, &calls); err != nil {
		return nil, fmt.Errorf("failed to decode proxy number usage: %w", err)
	}

	return calls, nil
}

func (r *callRepository) GetAvailableProxyNumbers(ctx context.Context) ([]string, error) {
	// This would be implemented based on your proxy number management strategy
	// For now, return empty slice
	return []string{}, nil
}

func (r *callRepository) AssignProxyNumber(ctx context.Context, rideID primitive.ObjectID, proxyNumber string) error {
	// Implementation would depend on your proxy number management strategy
	return nil
}

func (r *callRepository) ReleaseProxyNumber(ctx context.Context, proxyNumber string) error {
	// Implementation would depend on your proxy number management strategy
	return nil
}

// Call recordings
func (r *callRepository) UpdateCallRecording(ctx context.Context, callID primitive.ObjectID, recordingURL string) error {
	updates := map[string]interface{}{
		"recording_url": recordingURL,
		"updated_at":    time.Now(),
	}

	return r.UpdateCall(ctx, callID, updates)
}

func (r *callRepository) GetCallsWithRecordings(ctx context.Context, params *utils.PaginationParams) ([]*models.Call, int64, error) {
	filter := bson.M{
		"recording_url": bson.M{"$ne": ""},
	}
	return r.findCalls(ctx, filter, params)
}

func (r *callRepository) DeleteCallRecording(ctx context.Context, callID primitive.ObjectID) error {
	updates := map[string]interface{}{
		"recording_url": "",
		"updated_at":    time.Now(),
	}

	return r.UpdateCall(ctx, callID, updates)
}

// Helper methods
func (r *callRepository) findCalls(ctx context.Context, filter bson.M, params *utils.PaginationParams) ([]*models.Call, int64, error) {
	// Get total count
	total, err := r.collection.CountDocuments(ctx, filter)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to count calls: %w", err)
	}

	// Build options
	opts := options.Find()
	opts.SetSort(bson.M{"created_at": -1})

	if params != nil {
		// Calculate skip based on page and page size
		skip := (params.Page - 1) * params.PageSize
		opts.SetSkip(int64(skip))
		opts.SetLimit(int64(params.PageSize))
	}

	// Find calls
	cursor, err := r.collection.Find(ctx, filter, opts)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to find calls: %w", err)
	}
	defer cursor.Close(ctx)

	var calls []*models.Call
	if err := cursor.All(ctx, &calls); err != nil {
		return nil, 0, fmt.Errorf("failed to decode calls: %w", err)
	}

	return calls, total, nil
}

func (r *callRepository) getCallsByType(ctx context.Context, startDate, endDate time.Time) (map[models.CallType]int64, error) {
	pipeline := []bson.M{
		{
			"$match": bson.M{
				"created_at": bson.M{
					"$gte": startDate,
					"$lte": endDate,
				},
			},
		},
		{
			"$group": bson.M{
				"_id":   "$type",
				"count": bson.M{"$sum": 1},
			},
		},
	}

	cursor, err := r.collection.Aggregate(ctx, pipeline)
	if err != nil {
		return nil, fmt.Errorf("failed to get calls by type: %w", err)
	}
	defer cursor.Close(ctx)

	result := make(map[models.CallType]int64)
	for cursor.Next(ctx) {
		var doc struct {
			ID    models.CallType `bson:"_id"`
			Count int64           `bson:"count"`
		}
		if err := cursor.Decode(&doc); err != nil {
			continue
		}
		result[doc.ID] = doc.Count
	}

	return result, nil
}

func (r *callRepository) getCallsByStatus(ctx context.Context, startDate, endDate time.Time) (map[models.CallStatus]int64, error) {
	pipeline := []bson.M{
		{
			"$match": bson.M{
				"created_at": bson.M{
					"$gte": startDate,
					"$lte": endDate,
				},
			},
		},
		{
			"$group": bson.M{
				"_id":   "$status",
				"count": bson.M{"$sum": 1},
			},
		},
	}

	cursor, err := r.collection.Aggregate(ctx, pipeline)
	if err != nil {
		return nil, fmt.Errorf("failed to get calls by status: %w", err)
	}
	defer cursor.Close(ctx)

	result := make(map[models.CallStatus]int64)
	for cursor.Next(ctx) {
		var doc struct {
			ID    models.CallStatus `bson:"_id"`
			Count int64             `bson:"count"`
		}
		if err := cursor.Decode(&doc); err != nil {
			continue
		}
		result[doc.ID] = doc.Count
	}

	return result, nil
}

func (r *callRepository) invalidateCallCache(ctx context.Context, callerID, calleeID primitive.ObjectID) {
	if r.cache == nil {
		return
	}

	// Invalidate specific user caches
	r.cache.Delete(ctx, fmt.Sprintf("calls:user:%s", callerID.Hex()))
	r.cache.Delete(ctx, fmt.Sprintf("calls:user:%s", calleeID.Hex()))
}
