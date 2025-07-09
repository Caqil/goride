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

type locationRepository struct {
	historyCollection  *mongo.Collection
	geocacheCollection *mongo.Collection
	cache              CacheService
}

func NewLocationRepository(db *mongo.Database, cache CacheService) interfaces.LocationRepository {
	return &locationRepository{
		historyCollection:  db.Collection("location_history"),
		geocacheCollection: db.Collection("geocoding_cache"),
		cache:              cache,
	}
}

// Basic CRUD operations
func (r *locationRepository) Create(ctx context.Context, location *models.LocationHistory) error {
	location.ID = primitive.NewObjectID()
	location.CreatedAt = time.Now()

	_, err := r.historyCollection.InsertOne(ctx, location)
	if err != nil {
		return fmt.Errorf("failed to create location history: %w", err)
	}

	return nil
}

func (r *locationRepository) GetByID(ctx context.Context, id primitive.ObjectID) (*models.LocationHistory, error) {
	var location models.LocationHistory
	err := r.historyCollection.FindOne(ctx, bson.M{"_id": id}).Decode(&location)
	if err != nil {
		if err == mongo.ErrNoDocuments {
			return nil, fmt.Errorf("location history not found")
		}
		return nil, fmt.Errorf("failed to get location history: %w", err)
	}

	return &location, nil
}

func (r *locationRepository) Update(ctx context.Context, id primitive.ObjectID, updates map[string]interface{}) error {
	_, err := r.historyCollection.UpdateOne(
		ctx,
		bson.M{"_id": id},
		bson.M{"$set": updates},
	)
	if err != nil {
		return fmt.Errorf("failed to update location history: %w", err)
	}

	return nil
}

func (r *locationRepository) Delete(ctx context.Context, id primitive.ObjectID) error {
	_, err := r.historyCollection.DeleteOne(ctx, bson.M{"_id": id})
	if err != nil {
		return fmt.Errorf("failed to delete location history: %w", err)
	}

	return nil
}

// User location tracking
func (r *locationRepository) CreateLocationHistory(ctx context.Context, userID primitive.ObjectID, location *models.Location) error {
	locationHistory := &models.LocationHistory{
		UserID:   &userID,
		Location: *location,
		IsActive: true,
	}

	return r.Create(ctx, locationHistory)
}

func (r *locationRepository) GetUserLocationHistory(ctx context.Context, userID primitive.ObjectID, params *utils.PaginationParams) ([]*models.LocationHistory, int64, error) {
	filter := bson.M{"user_id": userID}
	return r.findLocationsWithFilter(ctx, filter, params)
}

func (r *locationRepository) GetLatestUserLocation(ctx context.Context, userID primitive.ObjectID) (*models.LocationHistory, error) {
	// Try cache first
	cacheKey := fmt.Sprintf("latest_location_user_%s", userID.Hex())
	if r.cache != nil {
		var location models.LocationHistory
		if err := r.cache.Get(ctx, cacheKey, &location); err == nil {
			return &location, nil
		}
	}

	var location models.LocationHistory
	err := r.historyCollection.FindOne(
		ctx,
		bson.M{"user_id": userID, "is_active": true},
		options.FindOne().SetSort(bson.D{{Key: "created_at", Value: -1}}),
	).Decode(&location)
	if err != nil {
		if err == mongo.ErrNoDocuments {
			return nil, fmt.Errorf("no location history found for user")
		}
		return nil, fmt.Errorf("failed to get latest user location: %w", err)
	}

	// Cache the result
	if r.cache != nil {
		r.cache.Set(ctx, cacheKey, &location, 5*time.Minute)
	}

	return &location, nil
}

func (r *locationRepository) GetUserLocationsByDateRange(ctx context.Context, userID primitive.ObjectID, startDate, endDate time.Time) ([]*models.LocationHistory, error) {
	filter := bson.M{
		"user_id": userID,
		"created_at": bson.M{
			"$gte": startDate,
			"$lte": endDate,
		},
	}

	cursor, err := r.historyCollection.Find(ctx, filter, options.Find().SetSort(bson.D{{Key: "created_at", Value: 1}}))
	if err != nil {
		return nil, fmt.Errorf("failed to find user locations by date range: %w", err)
	}
	defer cursor.Close(ctx)

	var locations []*models.LocationHistory
	for cursor.Next(ctx) {
		var location models.LocationHistory
		if err := cursor.Decode(&location); err != nil {
			return nil, fmt.Errorf("failed to decode location: %w", err)
		}
		locations = append(locations, &location)
	}

	return locations, nil
}

// Driver location operations
func (r *locationRepository) UpdateDriverLocation(ctx context.Context, driverID primitive.ObjectID, location *models.Location) error {
	// Create a location history entry
	locationHistory := &models.LocationHistory{
		UserID:   &driverID,
		UserType: models.UserTypeDriver,
		Location: *location,
		IsActive: true,
	}

	// Start a transaction to update both history and invalidate old records
	session, err := r.historyCollection.Database().Client().StartSession()
	if err != nil {
		return fmt.Errorf("failed to start session: %w", err)
	}
	defer session.EndSession(ctx)

	err = mongo.WithSession(ctx, session, func(sc mongo.SessionContext) error {
		// Mark previous driver locations as inactive
		_, err := r.historyCollection.UpdateMany(
			sc,
			bson.M{
				"user_id":   driverID,
				"user_type": models.UserTypeDriver,
				"is_active": true,
			},
			bson.M{"$set": bson.M{"is_active": false}},
		)
		if err != nil {
			return fmt.Errorf("failed to deactivate old locations: %w", err)
		}

		// Insert new location
		return r.Create(sc, locationHistory)
	})

	if err != nil {
		return fmt.Errorf("failed to update driver location: %w", err)
	}

	// Invalidate cache
	if r.cache != nil {
		cacheKey := fmt.Sprintf("latest_location_user_%s", driverID.Hex())
		r.cache.Delete(ctx, cacheKey)
	}

	return nil
}

func (r *locationRepository) GetNearbyUsers(ctx context.Context, lat, lng, radiusKM float64, userType models.UserType) ([]*models.LocationHistory, error) {
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
		"user_type": userType,
		"is_active": true,
		"created_at": bson.M{
			"$gte": time.Now().Add(-30 * time.Minute), // Only locations from last 30 minutes
		},
	}

	cursor, err := r.historyCollection.Find(ctx, filter, options.Find().SetLimit(100))
	if err != nil {
		return nil, fmt.Errorf("failed to find nearby users: %w", err)
	}
	defer cursor.Close(ctx)

	var locations []*models.LocationHistory
	for cursor.Next(ctx) {
		var location models.LocationHistory
		if err := cursor.Decode(&location); err != nil {
			return nil, fmt.Errorf("failed to decode location: %w", err)
		}
		locations = append(locations, &location)
	}

	return locations, nil
}

func (r *locationRepository) GetUsersInArea(ctx context.Context, bounds *utils.Bounds, userType models.UserType) ([]*models.LocationHistory, error) {
	filter := bson.M{
		"location": bson.M{
			"$geoWithin": bson.M{
				"$box": [][]float64{
					{bounds.Southwest.Lng, bounds.Southwest.Lat},
					{bounds.Northeast.Lng, bounds.Northeast.Lat},
				},
			},
		},
		"user_type": userType,
		"is_active": true,
		"created_at": bson.M{
			"$gte": time.Now().Add(-30 * time.Minute),
		},
	}

	cursor, err := r.historyCollection.Find(ctx, filter)
	if err != nil {
		return nil, fmt.Errorf("failed to find users in area: %w", err)
	}
	defer cursor.Close(ctx)

	var locations []*models.LocationHistory
	for cursor.Next(ctx) {
		var location models.LocationHistory
		if err := cursor.Decode(&location); err != nil {
			return nil, fmt.Errorf("failed to decode location: %w", err)
		}
		locations = append(locations, &location)
	}

	return locations, nil
}

func (r *locationRepository) GetActiveDriverLocations(ctx context.Context) ([]*models.LocationHistory, error) {
	filter := bson.M{
		"user_type": models.UserTypeDriver,
		"is_active": true,
		"created_at": bson.M{
			"$gte": time.Now().Add(-15 * time.Minute), // Active within last 15 minutes
		},
	}

	cursor, err := r.historyCollection.Find(ctx, filter)
	if err != nil {
		return nil, fmt.Errorf("failed to find active driver locations: %w", err)
	}
	defer cursor.Close(ctx)

	var locations []*models.LocationHistory
	for cursor.Next(ctx) {
		var location models.LocationHistory
		if err := cursor.Decode(&location); err != nil {
			return nil, fmt.Errorf("failed to decode location: %w", err)
		}
		locations = append(locations, &location)
	}

	return locations, nil
}

// Ride location tracking
func (r *locationRepository) CreateRideLocationUpdate(ctx context.Context, rideID primitive.ObjectID, location *models.Location) error {
	locationHistory := &models.LocationHistory{
		RideID:   &rideID,
		Location: *location,
		IsActive: true,
	}

	return r.Create(ctx, locationHistory)
}

func (r *locationRepository) GetRideLocationHistory(ctx context.Context, rideID primitive.ObjectID) ([]*models.LocationHistory, error) {
	filter := bson.M{"ride_id": rideID}

	cursor, err := r.historyCollection.Find(ctx, filter, options.Find().SetSort(bson.D{{Key: "created_at", Value: 1}}))
	if err != nil {
		return nil, fmt.Errorf("failed to find ride location history: %w", err)
	}
	defer cursor.Close(ctx)

	var locations []*models.LocationHistory
	for cursor.Next(ctx) {
		var location models.LocationHistory
		if err := cursor.Decode(&location); err != nil {
			return nil, fmt.Errorf("failed to decode location: %w", err)
		}
		locations = append(locations, &location)
	}

	return locations, nil
}

func (r *locationRepository) GetLatestRideLocation(ctx context.Context, rideID primitive.ObjectID) (*models.LocationHistory, error) {
	var location models.LocationHistory
	err := r.historyCollection.FindOne(
		ctx,
		bson.M{"ride_id": rideID},
		options.FindOne().SetSort(bson.D{{Key: "created_at", Value: -1}}),
	).Decode(&location)
	if err != nil {
		if err == mongo.ErrNoDocuments {
			return nil, fmt.Errorf("no location history found for ride")
		}
		return nil, fmt.Errorf("failed to get latest ride location: %w", err)
	}

	return &location, nil
}

// Search and filtering
func (r *locationRepository) SearchByAddress(ctx context.Context, address string, params *utils.PaginationParams) ([]*models.LocationHistory, int64, error) {
	filter := bson.M{
		"location.address": bson.M{"$regex": address, "$options": "i"},
	}
	return r.findLocationsWithFilter(ctx, filter, params)
}

func (r *locationRepository) GetLocationsByCity(ctx context.Context, city string, params *utils.PaginationParams) ([]*models.LocationHistory, int64, error) {
	filter := bson.M{
		"location.city": city,
	}
	return r.findLocationsWithFilter(ctx, filter, params)
}

func (r *locationRepository) GetFrequentLocations(ctx context.Context, userID primitive.ObjectID, limit int) ([]*models.LocationHistory, error) {
	// Group by location and count frequency
	pipeline := mongo.Pipeline{
		{{"$match", bson.M{
			"user_id":          userID,
			"location.address": bson.M{"$ne": ""},
		}}},
		{{"$group", bson.M{
			"_id": bson.M{
				"address":     "$location.address",
				"coordinates": "$location.coordinates",
			},
			"count":         bson.M{"$sum": 1},
			"last_visit":    bson.M{"$max": "$created_at"},
			"location_data": bson.M{"$first": "$$ROOT"},
		}}},
		{{"$sort", bson.M{"count": -1}}},
		{{"$limit", limit}},
	}

	cursor, err := r.historyCollection.Aggregate(ctx, pipeline)
	if err != nil {
		return nil, fmt.Errorf("failed to get frequent locations: %w", err)
	}
	defer cursor.Close(ctx)

	var locations []*models.LocationHistory
	for cursor.Next(ctx) {
		var result struct {
			LocationData models.LocationHistory `bson:"location_data"`
			Count        int64                  `bson:"count"`
			LastVisit    time.Time              `bson:"last_visit"`
		}

		if err := cursor.Decode(&result); err != nil {
			return nil, fmt.Errorf("failed to decode frequent location: %w", err)
		}

		locations = append(locations, &result.LocationData)
	}

	return locations, nil
}

// Geocoding cache
func (r *locationRepository) CacheGeocodingResult(ctx context.Context, address string, location *models.Location) error {
	geocacheEntry := map[string]interface{}{
		"_id":        utils.GenerateHashFromString(address), // Use address hash as ID
		"address":    address,
		"location":   location,
		"created_at": time.Now(),
		"expires_at": time.Now().Add(24 * time.Hour), // Cache for 24 hours
	}

	_, err := r.geocacheCollection.ReplaceOne(
		ctx,
		bson.M{"_id": geocacheEntry["_id"]},
		geocacheEntry,
		options.Replace().SetUpsert(true),
	)
	if err != nil {
		return fmt.Errorf("failed to cache geocoding result: %w", err)
	}

	return nil
}

func (r *locationRepository) GetCachedGeocodingResult(ctx context.Context, address string) (*models.Location, error) {
	addressHash := utils.GenerateHashFromString(address)

	var result struct {
		Location  models.Location `bson:"location"`
		ExpiresAt time.Time       `bson:"expires_at"`
	}

	err := r.geocacheCollection.FindOne(ctx, bson.M{
		"_id":        addressHash,
		"expires_at": bson.M{"$gt": time.Now()}, // Not expired
	}).Decode(&result)
	if err != nil {
		if err == mongo.ErrNoDocuments {
			return nil, fmt.Errorf("geocoding cache miss")
		}
		return nil, fmt.Errorf("failed to get cached geocoding result: %w", err)
	}

	return &result.Location, nil
}

// Analytics
func (r *locationRepository) GetLocationStats(ctx context.Context, days int) (map[string]interface{}, error) {
	startDate := time.Now().AddDate(0, 0, -days)

	// Total location updates
	totalUpdates, err := r.historyCollection.CountDocuments(ctx, bson.M{
		"created_at": bson.M{"$gte": startDate},
	})
	if err != nil {
		return nil, fmt.Errorf("failed to count total updates: %w", err)
	}

	// Updates by user type
	typePipeline := mongo.Pipeline{
		{{"$match", bson.M{
			"created_at": bson.M{"$gte": startDate},
			"user_type":  bson.M{"$exists": true},
		}}},
		{{"$group", bson.M{
			"_id":   "$user_type",
			"count": bson.M{"$sum": 1},
		}}},
	}

	cursor, err := r.historyCollection.Aggregate(ctx, typePipeline)
	if err != nil {
		return nil, fmt.Errorf("failed to get updates by type: %w", err)
	}
	defer cursor.Close(ctx)

	updatesByType := make(map[string]int64)
	for cursor.Next(ctx) {
		var result struct {
			UserType models.UserType `bson:"_id"`
			Count    int64           `bson:"count"`
		}

		if err := cursor.Decode(&result); err != nil {
			return nil, fmt.Errorf("failed to decode type stats: %w", err)
		}

		updatesByType[string(result.UserType)] = result.Count
	}

	// Active users (users with location updates in the period)
	activeUsers, err := r.historyCollection.Distinct(ctx, "user_id", bson.M{
		"created_at": bson.M{"$gte": startDate},
		"user_id":    bson.M{"$ne": nil},
	})
	if err != nil {
		return nil, fmt.Errorf("failed to count active users: %w", err)
	}

	// Top cities by activity
	cityPipeline := mongo.Pipeline{
		{{"$match", bson.M{
			"created_at":    bson.M{"$gte": startDate},
			"location.city": bson.M{"$ne": ""},
		}}},
		{{"$group", bson.M{
			"_id":   "$location.city",
			"count": bson.M{"$sum": 1},
		}}},
		{{"$sort", bson.M{"count": -1}}},
		{{"$limit", 10}},
	}

	cursor, err = r.historyCollection.Aggregate(ctx, cityPipeline)
	if err != nil {
		return nil, fmt.Errorf("failed to get top cities: %w", err)
	}
	defer cursor.Close(ctx)

	topCities := make([]map[string]interface{}, 0)
	for cursor.Next(ctx) {
		var result struct {
			City  string `bson:"_id"`
			Count int64  `bson:"count"`
		}

		if err := cursor.Decode(&result); err != nil {
			return nil, fmt.Errorf("failed to decode city stats: %w", err)
		}

		topCities = append(topCities, map[string]interface{}{
			"city":  result.City,
			"count": result.Count,
		})
	}

	return map[string]interface{}{
		"total_updates":   totalUpdates,
		"updates_by_type": updatesByType,
		"active_users":    len(activeUsers),
		"top_cities":      topCities,
		"period_days":     days,
		"start_date":      startDate,
		"end_date":        time.Now(),
	}, nil
}

func (r *locationRepository) GetPopularAreas(ctx context.Context, city string, days int, limit int) ([]map[string]interface{}, error) {
	startDate := time.Now().AddDate(0, 0, -days)

	filter := bson.M{
		"created_at":       bson.M{"$gte": startDate},
		"location.address": bson.M{"$ne": ""},
	}

	if city != "" {
		filter["location.city"] = city
	}

	pipeline := mongo.Pipeline{
		{{"$match", filter}},
		{{"$group", bson.M{
			"_id": bson.M{
				"address":     "$location.address",
				"coordinates": "$location.coordinates",
			},
			"visit_count":   bson.M{"$sum": 1},
			"unique_users":  bson.M{"$addToSet": "$user_id"},
			"location_data": bson.M{"$first": "$location"},
		}}},
		{{"$project", bson.M{
			"address":       "$_id.address",
			"coordinates":   "$_id.coordinates",
			"visit_count":   1,
			"unique_users":  bson.M{"$size": "$unique_users"},
			"location_data": 1,
		}}},
		{{"$sort", bson.M{"visit_count": -1}}},
		{{"$limit", limit}},
	}

	cursor, err := r.historyCollection.Aggregate(ctx, pipeline)
	if err != nil {
		return nil, fmt.Errorf("failed to get popular areas: %w", err)
	}
	defer cursor.Close(ctx)

	var results []map[string]interface{}
	for cursor.Next(ctx) {
		var result struct {
			Address      string                 `bson:"address"`
			Coordinates  []float64              `bson:"coordinates"`
			VisitCount   int64                  `bson:"visit_count"`
			UniqueUsers  int64                  `bson:"unique_users"`
			LocationData map[string]interface{} `bson:"location_data"`
		}

		if err := cursor.Decode(&result); err != nil {
			return nil, fmt.Errorf("failed to decode popular area: %w", err)
		}

		results = append(results, map[string]interface{}{
			"address":       result.Address,
			"coordinates":   result.Coordinates,
			"visit_count":   result.VisitCount,
			"unique_users":  result.UniqueUsers,
			"location_data": result.LocationData,
		})
	}

	return results, nil
}

func (r *locationRepository) GetUserMovementPattern(ctx context.Context, userID primitive.ObjectID, days int) (map[string]interface{}, error) {
	startDate := time.Now().AddDate(0, 0, -days)

	// Get user's location history
	locations, err := r.GetUserLocationsByDateRange(ctx, userID, startDate, time.Now())
	if err != nil {
		return nil, err
	}

	if len(locations) == 0 {
		return map[string]interface{}{
			"user_id":          userID,
			"total_locations":  0,
			"movement_pattern": "no_data",
		}, nil
	}

	// Calculate movement statistics
	var totalDistance float64
	var maxDistance float64
	frequentAreas := make(map[string]int)

	for i, location := range locations {
		// Count frequent areas
		if location.Location.City != "" {
			frequentAreas[location.Location.City]++
		}

		// Calculate distance between consecutive points
		if i > 0 {
			prevLocation := locations[i-1]
			distance := utils.CalculateDistance(
				prevLocation.Location.Latitude(),
				prevLocation.Location.Longitude(),
				location.Location.Latitude(),
				location.Location.Longitude(),
			)
			totalDistance += distance
			if distance > maxDistance {
				maxDistance = distance
			}
		}
	}

	// Determine movement pattern
	movementPattern := "stationary"
	avgDailyDistance := totalDistance / float64(days)

	if avgDailyDistance > 50 {
		movementPattern = "highly_mobile"
	} else if avgDailyDistance > 20 {
		movementPattern = "moderately_mobile"
	} else if avgDailyDistance > 5 {
		movementPattern = "locally_mobile"
	}

	// Find most frequent area
	var mostFrequentArea string
	var maxCount int
	for area, count := range frequentAreas {
		if count > maxCount {
			maxCount = count
			mostFrequentArea = area
		}
	}

	return map[string]interface{}{
		"user_id":             userID,
		"total_locations":     len(locations),
		"total_distance_km":   totalDistance,
		"avg_daily_distance":  avgDailyDistance,
		"max_single_distance": maxDistance,
		"movement_pattern":    movementPattern,
		"frequent_areas":      frequentAreas,
		"most_frequent_area":  mostFrequentArea,
		"period_days":         days,
		"start_date":          startDate,
		"end_date":            time.Now(),
	}, nil
}

// Cleanup operations
func (r *locationRepository) DeleteOldLocations(ctx context.Context, days int) error {
	cutoffDate := time.Now().AddDate(0, 0, -days)

	result, err := r.historyCollection.DeleteMany(ctx, bson.M{
		"created_at": bson.M{"$lt": cutoffDate},
		"is_active":  false, // Only delete inactive location records
	})
	if err != nil {
		return fmt.Errorf("failed to delete old locations: %w", err)
	}

	fmt.Printf("Deleted %d old location records\n", result.DeletedCount)
	return nil
}

func (r *locationRepository) ArchiveLocationHistory(ctx context.Context, beforeDate time.Time) error {
	// Mark old records as archived
	result, err := r.historyCollection.UpdateMany(
		ctx,
		bson.M{
			"created_at": bson.M{"$lt": beforeDate},
			"is_active":  false,
		},
		bson.M{"$set": bson.M{"archived": true, "archived_at": time.Now()}},
	)
	if err != nil {
		return fmt.Errorf("failed to archive location history: %w", err)
	}

	fmt.Printf("Archived %d location records\n", result.ModifiedCount)
	return nil
}

// Helper methods
func (r *locationRepository) findLocationsWithFilter(ctx context.Context, filter bson.M, params *utils.PaginationParams) ([]*models.LocationHistory, int64, error) {
	// Add search filter if provided
	if params.Search != "" {
		searchFields := []string{"location.address", "location.city"}
		filter = bson.M{
			"$and": []bson.M{
				filter,
				params.GetSearchFilter(searchFields),
			},
		}
	}

	// Get total count
	total, err := r.historyCollection.CountDocuments(ctx, filter)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to count locations: %w", err)
	}

	// Get paginated results
	opts := params.GetSortOptions()
	// Default sort by created_at descending for location history
	if params.Sort == "created_at" || params.Sort == "" {
		opts.SetSort(bson.D{{Key: "created_at", Value: -1}})
	}

	cursor, err := r.historyCollection.Find(ctx, filter, opts)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to find locations: %w", err)
	}
	defer cursor.Close(ctx)

	var locations []*models.LocationHistory
	for cursor.Next(ctx) {
		var location models.LocationHistory
		if err := cursor.Decode(&location); err != nil {
			return nil, 0, fmt.Errorf("failed to decode location: %w", err)
		}
		locations = append(locations, &location)
	}

	return locations, total, nil
}
