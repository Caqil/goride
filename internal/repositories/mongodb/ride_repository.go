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

type rideRepository struct {
	collection *mongo.Collection
	cache      services.CacheService
}

func NewRideRepository(db *mongo.Database, cache services.CacheService) interfaces.RideRepository {
	return &rideRepository{
		collection: db.Collection("rides"),
		cache:      cache,
	}
}

// Basic CRUD operations
func (r *rideRepository) Create(ctx context.Context, ride *models.Ride) error {
	ride.ID = primitive.NewObjectID()
	ride.RequestedAt = time.Now()

	_, err := r.collection.InsertOne(ctx, ride)
	if err != nil {
		return fmt.Errorf("failed to create ride: %w", err)
	}

	// Cache active rides for quick access
	if ride.Status == models.RideStatusRequested || ride.Status == models.RideStatusAccepted {
		r.cacheRide(ctx, ride)
	}

	return nil
}

func (r *rideRepository) GetByID(ctx context.Context, id primitive.ObjectID) (*models.Ride, error) {
	// Try cache first for active rides
	if ride := r.getRideFromCache(ctx, id.Hex()); ride != nil {
		return ride, nil
	}

	var ride models.Ride
	err := r.collection.FindOne(ctx, bson.M{"_id": id}).Decode(&ride)
	if err != nil {
		if err == mongo.ErrNoDocuments {
			return nil, fmt.Errorf("ride not found")
		}
		return nil, fmt.Errorf("failed to get ride: %w", err)
	}

	// Cache active rides
	if ride.Status == models.RideStatusRequested || ride.Status == models.RideStatusAccepted || ride.Status == models.RideStatusInProgress {
		r.cacheRide(ctx, &ride)
	}

	return &ride, nil
}

func (r *rideRepository) GetByRideNumber(ctx context.Context, rideNumber string) (*models.Ride, error) {
	var ride models.Ride
	err := r.collection.FindOne(ctx, bson.M{"ride_number": rideNumber}).Decode(&ride)
	if err != nil {
		if err == mongo.ErrNoDocuments {
			return nil, fmt.Errorf("ride not found with number")
		}
		return nil, fmt.Errorf("failed to get ride by number: %w", err)
	}

	return &ride, nil
}

func (r *rideRepository) Update(ctx context.Context, id primitive.ObjectID, updates map[string]interface{}) error {
	_, err := r.collection.UpdateOne(
		ctx,
		bson.M{"_id": id},
		bson.M{"$set": updates},
	)
	if err != nil {
		return fmt.Errorf("failed to update ride: %w", err)
	}

	// Invalidate cache
	r.invalidateRideCache(ctx, id.Hex())

	return nil
}

func (r *rideRepository) Delete(ctx context.Context, id primitive.ObjectID) error {
	_, err := r.collection.DeleteOne(ctx, bson.M{"_id": id})
	if err != nil {
		return fmt.Errorf("failed to delete ride: %w", err)
	}

	// Invalidate cache
	r.invalidateRideCache(ctx, id.Hex())

	return nil
}

// Status operations
func (r *rideRepository) UpdateStatus(ctx context.Context, id primitive.ObjectID, status models.RideStatus) error {
	updates := map[string]interface{}{
		"status": status,
	}

	switch status {
	case models.RideStatusAccepted:
		updates["accepted_at"] = time.Now()
	case models.RideStatusDriverArrived:
		updates["driver_arrived_at"] = time.Now()
	case models.RideStatusInProgress:
		updates["started_at"] = time.Now()
	case models.RideStatusCompleted:
		updates["completed_at"] = time.Now()
	case models.RideStatusCancelled:
		updates["cancelled_at"] = time.Now()
	}

	return r.Update(ctx, id, updates)
}

func (r *rideRepository) AssignDriver(ctx context.Context, id primitive.ObjectID, driverID, vehicleID primitive.ObjectID) error {
	updates := map[string]interface{}{
		"driver_id":   driverID,
		"vehicle_id":  vehicleID,
		"status":      models.RideStatusAccepted,
		"accepted_at": time.Now(),
	}

	return r.Update(ctx, id, updates)
}

func (r *rideRepository) StartRide(ctx context.Context, id primitive.ObjectID) error {
	updates := map[string]interface{}{
		"status":     models.RideStatusInProgress,
		"started_at": time.Now(),
	}

	return r.Update(ctx, id, updates)
}

func (r *rideRepository) CompleteRide(ctx context.Context, id primitive.ObjectID, actualDistance float64, actualDuration int, actualFare float64) error {
	updates := map[string]interface{}{
		"status":          models.RideStatusCompleted,
		"completed_at":    time.Now(),
		"actual_distance": actualDistance,
		"actual_duration": actualDuration,
		"actual_fare":     actualFare,
	}

	return r.Update(ctx, id, updates)
}

func (r *rideRepository) CancelRide(ctx context.Context, id primitive.ObjectID, reason, cancelledBy string) error {
	updates := map[string]interface{}{
		"status":              models.RideStatusCancelled,
		"cancelled_at":        time.Now(),
		"cancellation_reason": reason,
		"cancelled_by":        cancelledBy,
	}

	return r.Update(ctx, id, updates)
}

// Route operations
func (r *rideRepository) UpdateRoute(ctx context.Context, id primitive.ObjectID, route *models.Route) error {
	updates := map[string]interface{}{
		"route":              route,
		"estimated_distance": route.Distance,
		"estimated_duration": route.Duration / 60, // Convert seconds to minutes
	}

	return r.Update(ctx, id, updates)
}

func (r *rideRepository) AddWaypoint(ctx context.Context, id primitive.ObjectID, waypoint *models.Location) error {
	_, err := r.collection.UpdateOne(
		ctx,
		bson.M{"_id": id},
		bson.M{"$push": bson.M{"waypoints": waypoint}},
	)
	if err != nil {
		return fmt.Errorf("failed to add waypoint: %w", err)
	}

	// Invalidate cache
	r.invalidateRideCache(ctx, id.Hex())

	return nil
}

func (r *rideRepository) RemoveWaypoint(ctx context.Context, id primitive.ObjectID, waypointIndex int) error {
	// First get the ride to check waypoints length
	ride, err := r.GetByID(ctx, id)
	if err != nil {
		return err
	}

	if waypointIndex < 0 || waypointIndex >= len(ride.Waypoints) {
		return fmt.Errorf("waypoint index out of range")
	}

	// Remove waypoint at specific index
	pipeline := mongo.Pipeline{
		{{"$set", bson.M{
			"waypoints": bson.M{
				"$concatArrays": []interface{}{
					bson.M{"$slice": []interface{}{"$waypoints", waypointIndex}},
					bson.M{"$slice": []interface{}{"$waypoints", waypointIndex + 1, bson.M{"$size": "$waypoints"}}},
				},
			},
		}}},
	}

	_, err = r.collection.UpdateOne(ctx, bson.M{"_id": id}, pipeline)
	if err != nil {
		return fmt.Errorf("failed to remove waypoint: %w", err)
	}

	// Invalidate cache
	r.invalidateRideCache(ctx, id.Hex())

	return nil
}

// Search and filtering
func (r *rideRepository) GetByRider(ctx context.Context, riderID primitive.ObjectID, params *utils.PaginationParams) ([]*models.Ride, int64, error) {
	filter := bson.M{"rider_id": riderID}
	return r.findRidesWithFilter(ctx, filter, params)
}

func (r *rideRepository) GetByDriver(ctx context.Context, driverID primitive.ObjectID, params *utils.PaginationParams) ([]*models.Ride, int64, error) {
	filter := bson.M{"driver_id": driverID}
	return r.findRidesWithFilter(ctx, filter, params)
}

func (r *rideRepository) GetByStatus(ctx context.Context, status models.RideStatus, params *utils.PaginationParams) ([]*models.Ride, int64, error) {
	filter := bson.M{"status": status}
	return r.findRidesWithFilter(ctx, filter, params)
}

func (r *rideRepository) GetActiveRides(ctx context.Context) ([]*models.Ride, error) {
	filter := bson.M{
		"status": bson.M{"$in": []models.RideStatus{
			models.RideStatusRequested,
			models.RideStatusAccepted,
			models.RideStatusDriverArrived,
			models.RideStatusInProgress,
		}},
	}

	cursor, err := r.collection.Find(ctx, filter, options.Find().SetSort(bson.D{{Key: "requested_at", Value: -1}}))
	if err != nil {
		return nil, fmt.Errorf("failed to find active rides: %w", err)
	}
	defer cursor.Close(ctx)

	var rides []*models.Ride
	for cursor.Next(ctx) {
		var ride models.Ride
		if err := cursor.Decode(&ride); err != nil {
			return nil, fmt.Errorf("failed to decode ride: %w", err)
		}
		rides = append(rides, &ride)
	}

	return rides, nil
}

func (r *rideRepository) GetPendingRides(ctx context.Context) ([]*models.Ride, error) {
	filter := bson.M{"status": models.RideStatusRequested}

	cursor, err := r.collection.Find(ctx, filter, options.Find().SetSort(bson.D{{Key: "requested_at", Value: 1}}))
	if err != nil {
		return nil, fmt.Errorf("failed to find pending rides: %w", err)
	}
	defer cursor.Close(ctx)

	var rides []*models.Ride
	for cursor.Next(ctx) {
		var ride models.Ride
		if err := cursor.Decode(&ride); err != nil {
			return nil, fmt.Errorf("failed to decode ride: %w", err)
		}
		rides = append(rides, &ride)
	}

	return rides, nil
}

// Time-based queries
func (r *rideRepository) GetRidesByDateRange(ctx context.Context, startDate, endDate time.Time, params *utils.PaginationParams) ([]*models.Ride, int64, error) {
	filter := bson.M{
		"requested_at": bson.M{
			"$gte": startDate,
			"$lte": endDate,
		},
	}
	return r.findRidesWithFilter(ctx, filter, params)
}

func (r *rideRepository) GetScheduledRides(ctx context.Context, scheduledTime time.Time) ([]*models.Ride, error) {
	// Get rides scheduled within 30 minutes of the given time
	startTime := scheduledTime.Add(-15 * time.Minute)
	endTime := scheduledTime.Add(15 * time.Minute)

	filter := bson.M{
		"scheduled_time": bson.M{
			"$gte": startTime,
			"$lte": endTime,
		},
		"status": models.RideStatusRequested,
	}

	cursor, err := r.collection.Find(ctx, filter, options.Find().SetSort(bson.D{{Key: "scheduled_time", Value: 1}}))
	if err != nil {
		return nil, fmt.Errorf("failed to find scheduled rides: %w", err)
	}
	defer cursor.Close(ctx)

	var rides []*models.Ride
	for cursor.Next(ctx) {
		var ride models.Ride
		if err := cursor.Decode(&ride); err != nil {
			return nil, fmt.Errorf("failed to decode ride: %w", err)
		}
		rides = append(rides, &ride)
	}

	return rides, nil
}

func (r *rideRepository) GetRidesInProgress(ctx context.Context) ([]*models.Ride, error) {
	filter := bson.M{"status": models.RideStatusInProgress}

	cursor, err := r.collection.Find(ctx, filter, options.Find().SetSort(bson.D{{Key: "started_at", Value: 1}}))
	if err != nil {
		return nil, fmt.Errorf("failed to find rides in progress: %w", err)
	}
	defer cursor.Close(ctx)

	var rides []*models.Ride
	for cursor.Next(ctx) {
		var ride models.Ride
		if err := cursor.Decode(&ride); err != nil {
			return nil, fmt.Errorf("failed to decode ride: %w", err)
		}
		rides = append(rides, &ride)
	}

	return rides, nil
}

// Location-based queries
func (r *rideRepository) GetRidesInArea(ctx context.Context, bounds *utils.Bounds, params *utils.PaginationParams) ([]*models.Ride, int64, error) {
	filter := bson.M{
		"pickup_location": bson.M{
			"$geoWithin": bson.M{
				"$box": [][]float64{
					{bounds.Southwest.Lng, bounds.Southwest.Lat},
					{bounds.Northeast.Lng, bounds.Northeast.Lat},
				},
			},
		},
	}
	return r.findRidesWithFilter(ctx, filter, params)
}

func (r *rideRepository) GetNearbyRides(ctx context.Context, lat, lng, radiusKM float64) ([]*models.Ride, error) {
	// Convert radius from kilometers to meters
	radiusMeters := radiusKM * 1000

	filter := bson.M{
		"pickup_location": bson.M{
			"$near": bson.M{
				"$geometry": bson.M{
					"type":        "Point",
					"coordinates": []float64{lng, lat},
				},
				"$maxDistance": radiusMeters,
			},
		},
		"status": bson.M{"$in": []models.RideStatus{
			models.RideStatusRequested,
			models.RideStatusAccepted,
			models.RideStatusInProgress,
		}},
	}

	cursor, err := r.collection.Find(ctx, filter, options.Find().SetSort(bson.D{{Key: "requested_at", Value: -1}}))
	if err != nil {
		return nil, fmt.Errorf("failed to find nearby rides: %w", err)
	}
	defer cursor.Close(ctx)

	var rides []*models.Ride
	for cursor.Next(ctx) {
		var ride models.Ride
		if err := cursor.Decode(&ride); err != nil {
			return nil, fmt.Errorf("failed to decode ride: %w", err)
		}
		rides = append(rides, &ride)
	}

	return rides, nil
}

// Analytics and statistics
func (r *rideRepository) GetRideStats(ctx context.Context, startDate, endDate time.Time) (map[string]interface{}, error) {
	pipeline := mongo.Pipeline{
		{{"$match", bson.M{
			"requested_at": bson.M{
				"$gte": startDate,
				"$lte": endDate,
			},
		}}},
		{{"$group", bson.M{
			"_id":            "$status",
			"count":          bson.M{"$sum": 1},
			"total_distance": bson.M{"$sum": "$actual_distance"},
			"total_duration": bson.M{"$sum": "$actual_duration"},
			"avg_distance":   bson.M{"$avg": "$actual_distance"},
			"avg_duration":   bson.M{"$avg": "$actual_duration"},
		}}},
	}

	cursor, err := r.collection.Aggregate(ctx, pipeline)
	if err != nil {
		return nil, fmt.Errorf("failed to get ride stats: %w", err)
	}
	defer cursor.Close(ctx)

	stats := make(map[string]interface{})
	var totalRides int64

	for cursor.Next(ctx) {
		var result struct {
			ID            models.RideStatus `bson:"_id"`
			Count         int64             `bson:"count"`
			TotalDistance float64           `bson:"total_distance"`
			TotalDuration int               `bson:"total_duration"`
			AvgDistance   float64           `bson:"avg_distance"`
			AvgDuration   float64           `bson:"avg_duration"`
		}

		if err := cursor.Decode(&result); err != nil {
			return nil, fmt.Errorf("failed to decode ride stats: %w", err)
		}

		stats[string(result.ID)] = map[string]interface{}{
			"count":          result.Count,
			"total_distance": result.TotalDistance,
			"total_duration": result.TotalDuration,
			"avg_distance":   result.AvgDistance,
			"avg_duration":   result.AvgDuration,
		}

		totalRides += result.Count
	}

	// Calculate completion rate
	completedCount := int64(0)
	if completed, exists := stats[string(models.RideStatusCompleted)]; exists {
		if countMap, ok := completed.(map[string]interface{}); ok {
			if count, ok := countMap["count"].(int64); ok {
				completedCount = count
			}
		}
	}

	completionRate := float64(0)
	if totalRides > 0 {
		completionRate = float64(completedCount) / float64(totalRides) * 100
	}

	stats["summary"] = map[string]interface{}{
		"total_rides":     totalRides,
		"completion_rate": completionRate,
		"start_date":      startDate,
		"end_date":        endDate,
	}

	return stats, nil
}

func (r *rideRepository) GetRevenueStats(ctx context.Context, startDate, endDate time.Time) (map[string]interface{}, error) {
	pipeline := mongo.Pipeline{
		{{"$match", bson.M{
			"status": models.RideStatusCompleted,
			"completed_at": bson.M{
				"$gte": startDate,
				"$lte": endDate,
			},
		}}},
		{{"$group", bson.M{
			"_id":           nil,
			"total_revenue": bson.M{"$sum": "$actual_fare"},
			"total_rides":   bson.M{"$sum": 1},
			"avg_fare":      bson.M{"$avg": "$actual_fare"},
			"min_fare":      bson.M{"$min": "$actual_fare"},
			"max_fare":      bson.M{"$max": "$actual_fare"},
		}}},
	}

	cursor, err := r.collection.Aggregate(ctx, pipeline)
	if err != nil {
		return nil, fmt.Errorf("failed to get revenue stats: %w", err)
	}
	defer cursor.Close(ctx)

	var result struct {
		TotalRevenue float64 `bson:"total_revenue"`
		TotalRides   int64   `bson:"total_rides"`
		AvgFare      float64 `bson:"avg_fare"`
		MinFare      float64 `bson:"min_fare"`
		MaxFare      float64 `bson:"max_fare"`
	}

	if cursor.Next(ctx) {
		if err := cursor.Decode(&result); err != nil {
			return nil, fmt.Errorf("failed to decode revenue stats: %w", err)
		}
	}

	return map[string]interface{}{
		"total_revenue": result.TotalRevenue,
		"total_rides":   result.TotalRides,
		"avg_fare":      result.AvgFare,
		"min_fare":      result.MinFare,
		"max_fare":      result.MaxFare,
		"start_date":    startDate,
		"end_date":      endDate,
	}, nil
}

func (r *rideRepository) GetPopularRoutes(ctx context.Context, limit int, days int) ([]map[string]interface{}, error) {
	startDate := time.Now().AddDate(0, 0, -days)

	pipeline := mongo.Pipeline{
		{{"$match", bson.M{
			"status":       models.RideStatusCompleted,
			"completed_at": bson.M{"$gte": startDate},
		}}},
		{{"$group", bson.M{
			"_id": bson.M{
				"pickup_city":  "$pickup_location.city",
				"dropoff_city": "$dropoff_location.city",
			},
			"count":        bson.M{"$sum": 1},
			"avg_distance": bson.M{"$avg": "$actual_distance"},
			"avg_duration": bson.M{"$avg": "$actual_duration"},
			"avg_fare":     bson.M{"$avg": "$actual_fare"},
		}}},
		{{"$sort", bson.D{{Key: "count", Value: -1}}}},
		{{"$limit", limit}},
	}

	cursor, err := r.collection.Aggregate(ctx, pipeline)
	if err != nil {
		return nil, fmt.Errorf("failed to get popular routes: %w", err)
	}
	defer cursor.Close(ctx)

	var routes []map[string]interface{}

	for cursor.Next(ctx) {
		var result struct {
			ID struct {
				PickupCity  string `bson:"pickup_city"`
				DropoffCity string `bson:"dropoff_city"`
			} `bson:"_id"`
			Count       int64   `bson:"count"`
			AvgDistance float64 `bson:"avg_distance"`
			AvgDuration float64 `bson:"avg_duration"`
			AvgFare     float64 `bson:"avg_fare"`
		}

		if err := cursor.Decode(&result); err != nil {
			return nil, fmt.Errorf("failed to decode popular route: %w", err)
		}

		routes = append(routes, map[string]interface{}{
			"pickup_city":  result.ID.PickupCity,
			"dropoff_city": result.ID.DropoffCity,
			"ride_count":   result.Count,
			"avg_distance": result.AvgDistance,
			"avg_duration": result.AvgDuration,
			"avg_fare":     result.AvgFare,
		})
	}

	return routes, nil
}

func (r *rideRepository) GetPeakHours(ctx context.Context, days int) ([]map[string]interface{}, error) {
	startDate := time.Now().AddDate(0, 0, -days)

	pipeline := mongo.Pipeline{
		{{"$match", bson.M{
			"requested_at": bson.M{"$gte": startDate},
		}}},
		{{"$group", bson.M{
			"_id":   bson.M{"$hour": "$requested_at"},
			"count": bson.M{"$sum": 1},
			"completed_count": bson.M{"$sum": bson.M{
				"$cond": []interface{}{
					bson.M{"$eq": []interface{}{"$status", models.RideStatusCompleted}},
					1,
					0,
				},
			}},
		}}},
		{{"$sort", bson.D{{Key: "_id", Value: 1}}}},
	}

	cursor, err := r.collection.Aggregate(ctx, pipeline)
	if err != nil {
		return nil, fmt.Errorf("failed to get peak hours: %w", err)
	}
	defer cursor.Close(ctx)

	var peakHours []map[string]interface{}

	for cursor.Next(ctx) {
		var result struct {
			ID             int   `bson:"_id"`
			Count          int64 `bson:"count"`
			CompletedCount int64 `bson:"completed_count"`
		}

		if err := cursor.Decode(&result); err != nil {
			return nil, fmt.Errorf("failed to decode peak hour: %w", err)
		}

		completionRate := float64(0)
		if result.Count > 0 {
			completionRate = float64(result.CompletedCount) / float64(result.Count) * 100
		}

		peakHours = append(peakHours, map[string]interface{}{
			"hour":            result.ID,
			"total_requests":  result.Count,
			"completed_rides": result.CompletedCount,
			"completion_rate": completionRate,
		})
	}

	return peakHours, nil
}

// Ratings
func (r *rideRepository) UpdateRiderRating(ctx context.Context, id primitive.ObjectID, rating float64) error {
	updates := map[string]interface{}{
		"rider_rating": rating,
	}
	return r.Update(ctx, id, updates)
}

func (r *rideRepository) UpdateDriverRating(ctx context.Context, id primitive.ObjectID, rating float64) error {
	updates := map[string]interface{}{
		"driver_rating": rating,
	}
	return r.Update(ctx, id, updates)
}

// Helper methods
func (r *rideRepository) findRidesWithFilter(ctx context.Context, filter bson.M, params *utils.PaginationParams) ([]*models.Ride, int64, error) {
	// Add search filter if provided
	if params.Search != "" {
		searchFields := []string{"ride_number", "pickup_location.address", "dropoff_location.address"}
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
		return nil, 0, fmt.Errorf("failed to count rides: %w", err)
	}

	// Get paginated results
	opts := params.GetSortOptions()
	cursor, err := r.collection.Find(ctx, filter, opts)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to find rides: %w", err)
	}
	defer cursor.Close(ctx)

	var rides []*models.Ride
	for cursor.Next(ctx) {
		var ride models.Ride
		if err := cursor.Decode(&ride); err != nil {
			return nil, 0, fmt.Errorf("failed to decode ride: %w", err)
		}
		rides = append(rides, &ride)
	}

	return rides, total, nil
}

// Cache operations
func (r *rideRepository) cacheRide(ctx context.Context, ride *models.Ride) {
	if r.cache != nil {
		cacheKey := fmt.Sprintf("ride:%s", ride.ID.Hex())
		r.cache.Set(ctx, cacheKey, ride, 15*time.Minute)

		// Also cache by ride number
		if ride.RideNumber != "" {
			numberKey := fmt.Sprintf("ride_number_%s", ride.RideNumber)
			r.cache.Set(ctx, numberKey, ride, 15*time.Minute)
		}
	}
}

func (r *rideRepository) getRideFromCache(ctx context.Context, rideID string) *models.Ride {
	if r.cache == nil {
		return nil
	}

	cacheKey := fmt.Sprintf("ride:%s", rideID)
	var ride models.Ride
	err := r.cache.Get(ctx, cacheKey, &ride)
	if err != nil {
		return nil
	}

	return &ride
}

func (r *rideRepository) invalidateRideCache(ctx context.Context, rideID string) {
	if r.cache != nil {
		cacheKey := fmt.Sprintf("ride:%s", rideID)
		r.cache.Delete(ctx, cacheKey)

		// Note: We can't easily invalidate the ride number cache without knowing the ride number
		// This is a trade-off for performance vs cache consistency
	}
}
