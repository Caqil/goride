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

type driverRepository struct {
	collection *mongo.Collection

	cache services.CacheService
}

func NewDriverRepository(db *mongo.Database, cache services.CacheService) interfaces.DriverRepository {
	return &driverRepository{
		collection: db.Collection("drivers"),
		cache:      cache,
	}
}

// Basic CRUD operations
func (r *driverRepository) Create(ctx context.Context, driver *models.Driver) error {
	driver.ID = primitive.NewObjectID()
	driver.CreatedAt = time.Now()
	driver.UpdatedAt = time.Now()

	_, err := r.collection.InsertOne(ctx, driver)
	if err != nil {
		return fmt.Errorf("failed to create driver: %w", err)
	}

	// Cache the driver
	r.cacheDriver(ctx, driver)

	return nil
}

func (r *driverRepository) GetByID(ctx context.Context, id primitive.ObjectID) (*models.Driver, error) {
	// Try cache first
	if driver := r.getDriverFromCache(ctx, id.Hex()); driver != nil {
		return driver, nil
	}

	var driver models.Driver
	err := r.collection.FindOne(ctx, bson.M{"_id": id, "deleted_at": nil}).Decode(&driver)
	if err != nil {
		if err == mongo.ErrNoDocuments {
			return nil, fmt.Errorf("driver not found")
		}
		return nil, fmt.Errorf("failed to get driver: %w", err)
	}

	// Cache the result
	r.cacheDriver(ctx, &driver)

	return &driver, nil
}

func (r *driverRepository) GetByUserID(ctx context.Context, userID primitive.ObjectID) (*models.Driver, error) {
	// Try cache first
	cacheKey := fmt.Sprintf("driver_user_%s", userID.Hex())
	if r.cache != nil {
		var driver models.Driver
		if err := r.cache.Get(ctx, cacheKey, &driver); err == nil {
			return &driver, nil
		}
	}

	var driver models.Driver
	err := r.collection.FindOne(ctx, bson.M{"user_id": userID, "deleted_at": nil}).Decode(&driver)
	if err != nil {
		if err == mongo.ErrNoDocuments {
			return nil, fmt.Errorf("driver not found for user")
		}
		return nil, fmt.Errorf("failed to get driver by user ID: %w", err)
	}

	// Cache the result
	if r.cache != nil {
		r.cache.Set(ctx, cacheKey, &driver, 30*time.Minute)
	}
	r.cacheDriver(ctx, &driver)

	return &driver, nil
}

func (r *driverRepository) Update(ctx context.Context, id primitive.ObjectID, updates map[string]interface{}) error {
	updates["updated_at"] = time.Now()

	_, err := r.collection.UpdateOne(
		ctx,
		bson.M{"_id": id, "deleted_at": nil},
		bson.M{"$set": updates},
	)
	if err != nil {
		return fmt.Errorf("failed to update driver: %w", err)
	}

	// Invalidate cache
	r.invalidateDriverCache(ctx, id.Hex())

	return nil
}

func (r *driverRepository) Delete(ctx context.Context, id primitive.ObjectID) error {
	updates := map[string]interface{}{
		"deleted_at": time.Now(),
	}

	return r.Update(ctx, id, updates)
}

// Location operations
func (r *driverRepository) UpdateLocation(ctx context.Context, id primitive.ObjectID, location *models.Location) error {
	updates := map[string]interface{}{
		"current_location":     location,
		"last_location_update": time.Now(),
	}

	return r.Update(ctx, id, updates)
}

func (r *driverRepository) GetNearbyDrivers(ctx context.Context, lat, lng, radiusKM float64, rideType string) ([]*models.Driver, error) {
	// Convert radius from kilometers to meters for MongoDB
	radiusMeters := radiusKM * 1000

	filter := bson.M{
		"current_location": bson.M{
			"$near": bson.M{
				"$geometry": bson.M{
					"type":        "Point",
					"coordinates": []float64{lng, lat},
				},
				"$maxDistance": radiusMeters,
			},
		},
		"status":       models.DriverStatusOnline,
		"is_available": true,
		"deleted_at":   nil,
	}

	// Add ride type filter if specified
	if rideType != "" {
		filter["ride_types"] = bson.M{"$in": []string{rideType}}
	}

	// Add additional filters for active drivers
	filter["$and"] = []bson.M{
		{"license_status": models.DocumentStatusApproved},
		{"insurance_status": models.DocumentStatusApproved},
		{"background_check_status": models.DocumentStatusApproved},
	}
	filter["last_location_update"] = bson.M{
		"$gte": time.Now().Add(-10 * time.Minute), // Location updated within last 10 minutes
	}

	cursor, err := r.collection.Find(ctx, filter, options.Find().SetLimit(50)) // Limit to 50 nearby drivers
	if err != nil {
		return nil, fmt.Errorf("failed to find nearby drivers: %w", err)
	}
	defer cursor.Close(ctx)

	var drivers []*models.Driver
	for cursor.Next(ctx) {
		var driver models.Driver
		if err := cursor.Decode(&driver); err != nil {
			return nil, fmt.Errorf("failed to decode driver: %w", err)
		}
		drivers = append(drivers, &driver)
	}

	return drivers, nil
}

func (r *driverRepository) GetDriversInArea(ctx context.Context, bounds *utils.Bounds) ([]*models.Driver, error) {
	filter := bson.M{
		"current_location": bson.M{
			"$geoWithin": bson.M{
				"$box": [][]float64{
					{bounds.Southwest.Lng, bounds.Southwest.Lat},
					{bounds.Northeast.Lng, bounds.Northeast.Lat},
				},
			},
		},
		"status":       models.DriverStatusOnline,
		"is_available": true,
		"deleted_at":   nil,
	}

	cursor, err := r.collection.Find(ctx, filter)
	if err != nil {
		return nil, fmt.Errorf("failed to find drivers in area: %w", err)
	}
	defer cursor.Close(ctx)

	var drivers []*models.Driver
	for cursor.Next(ctx) {
		var driver models.Driver
		if err := cursor.Decode(&driver); err != nil {
			return nil, fmt.Errorf("failed to decode driver: %w", err)
		}
		drivers = append(drivers, &driver)
	}

	return drivers, nil
}

// Status operations
func (r *driverRepository) UpdateStatus(ctx context.Context, id primitive.ObjectID, status models.DriverStatus) error {
	updates := map[string]interface{}{
		"status": status,
	}

	// If going offline, set availability to false
	if status == models.DriverStatusOffline {
		updates["is_available"] = false
	}

	return r.Update(ctx, id, updates)
}

func (r *driverRepository) UpdateAvailability(ctx context.Context, id primitive.ObjectID, available bool) error {
	updates := map[string]interface{}{
		"is_available": available,
	}

	// If setting available to true, ensure driver is online
	if available {
		// First check if driver is online
		driver, err := r.GetByID(ctx, id)
		if err != nil {
			return err
		}

		if driver.Status == models.DriverStatusOffline {
			updates["status"] = models.DriverStatusOnline
		}
	}

	return r.Update(ctx, id, updates)
}

func (r *driverRepository) GetAvailableDrivers(ctx context.Context, params *utils.PaginationParams) ([]*models.Driver, int64, error) {
	filter := bson.M{
		"status":       models.DriverStatusOnline,
		"is_available": true,
		"deleted_at":   nil,
	}

	return r.findDriversWithFilter(ctx, filter, params)
}

func (r *driverRepository) GetOnlineDrivers(ctx context.Context) ([]*models.Driver, error) {
	filter := bson.M{
		"status":     models.DriverStatusOnline,
		"deleted_at": nil,
	}

	cursor, err := r.collection.Find(ctx, filter)
	if err != nil {
		return nil, fmt.Errorf("failed to find online drivers: %w", err)
	}
	defer cursor.Close(ctx)

	var drivers []*models.Driver
	for cursor.Next(ctx) {
		var driver models.Driver
		if err := cursor.Decode(&driver); err != nil {
			return nil, fmt.Errorf("failed to decode driver: %w", err)
		}
		drivers = append(drivers, &driver)
	}

	return drivers, nil
}

// Document verification
func (r *driverRepository) UpdateDocumentStatus(ctx context.Context, id primitive.ObjectID, docType string, status models.DocumentStatus) error {
	updates := map[string]interface{}{}

	// Update the specific document status based on document type
	switch docType {
	case "license":
		updates["license_status"] = status
	case "insurance":
		updates["insurance_status"] = status
	case "background_check":
		updates["background_check_status"] = status
		if status == models.DocumentStatusApproved {
			updates["background_check_date"] = time.Now()
		}
	default:
		return fmt.Errorf("unsupported document type: %s", docType)
	}

	// Get current driver to check if all documents are approved
	driver, err := r.GetByID(ctx, id)
	if err != nil {
		return err
	}

	// Check if all required documents are approved
	allApproved := true

	// Check license status
	licenseStatus := driver.LicenseStatus
	if docType == "license" {
		licenseStatus = status
	}
	if licenseStatus != models.DocumentStatusApproved {
		allApproved = false
	}

	// Check insurance status
	insuranceStatus := driver.InsuranceStatus
	if docType == "insurance" {
		insuranceStatus = status
	}
	if insuranceStatus != models.DocumentStatusApproved {
		allApproved = false
	}

	// Check background check status
	backgroundStatus := driver.BackgroundCheckStatus
	if docType == "background_check" {
		backgroundStatus = status
	}
	if backgroundStatus != models.DocumentStatusApproved {
		allApproved = false
	}

	// If all documents are approved, update approval timestamp
	if allApproved {
		updates["approved_at"] = time.Now()
	}

	return r.Update(ctx, id, updates)
}

func (r *driverRepository) GetPendingVerifications(ctx context.Context, params *utils.PaginationParams) ([]*models.Driver, int64, error) {
	filter := bson.M{
		"$or": []bson.M{
			{"license_status": models.DocumentStatusPending},
			{"insurance_status": models.DocumentStatusPending},
			{"background_check_status": models.DocumentStatusPending},
		},
		"deleted_at": nil,
	}

	return r.findDriversWithFilter(ctx, filter, params)
}

// Performance metrics
func (r *driverRepository) UpdateRating(ctx context.Context, id primitive.ObjectID, rating float64, totalRatings int64) error {
	updates := map[string]interface{}{
		"rating":        rating,
		"total_ratings": totalRatings,
	}

	return r.Update(ctx, id, updates)
}

func (r *driverRepository) UpdateEarnings(ctx context.Context, id primitive.ObjectID, amount float64) error {
	_, err := r.collection.UpdateOne(
		ctx,
		bson.M{"_id": id, "deleted_at": nil},
		bson.M{
			"$inc": bson.M{"total_earnings": amount},
			"$set": bson.M{"updated_at": time.Now()},
		},
	)
	if err != nil {
		return fmt.Errorf("failed to update driver earnings: %w", err)
	}

	// Invalidate cache
	r.invalidateDriverCache(ctx, id.Hex())

	return nil
}

func (r *driverRepository) UpdateRideStats(ctx context.Context, id primitive.ObjectID, totalRides int64) error {
	updates := map[string]interface{}{
		"total_rides": totalRides,
	}

	return r.Update(ctx, id, updates)
}

// Search and filtering
func (r *driverRepository) List(ctx context.Context, params *utils.PaginationParams) ([]*models.Driver, int64, error) {
	filter := bson.M{"deleted_at": nil}
	return r.findDriversWithFilter(ctx, filter, params)
}

func (r *driverRepository) GetByStatus(ctx context.Context, status models.DriverStatus, params *utils.PaginationParams) ([]*models.Driver, int64, error) {
	filter := bson.M{
		"status":     status,
		"deleted_at": nil,
	}
	return r.findDriversWithFilter(ctx, filter, params)
}

func (r *driverRepository) GetByRatingRange(ctx context.Context, minRating, maxRating float64, params *utils.PaginationParams) ([]*models.Driver, int64, error) {
	filter := bson.M{
		"rating": bson.M{
			"$gte": minRating,
			"$lte": maxRating,
		},
		"deleted_at": nil,
	}
	return r.findDriversWithFilter(ctx, filter, params)
}

func (r *driverRepository) SearchByLicense(ctx context.Context, licenseNumber string) (*models.Driver, error) {
	var driver models.Driver
	err := r.collection.FindOne(ctx, bson.M{
		"license_number": licenseNumber,
		"deleted_at":     nil,
	}).Decode(&driver)
	if err != nil {
		if err == mongo.ErrNoDocuments {
			return nil, fmt.Errorf("driver not found with license number")
		}
		return nil, fmt.Errorf("failed to search driver by license: %w", err)
	}

	return &driver, nil
}

// Analytics
func (r *driverRepository) GetTotalCount(ctx context.Context) (int64, error) {
	return r.collection.CountDocuments(ctx, bson.M{"deleted_at": nil})
}

func (r *driverRepository) GetCountByStatus(ctx context.Context, status models.DriverStatus) (int64, error) {
	return r.collection.CountDocuments(ctx, bson.M{
		"status":     status,
		"deleted_at": nil,
	})
}

func (r *driverRepository) GetAverageRating(ctx context.Context) (float64, error) {
	pipeline := mongo.Pipeline{
		bson.D{{Key: "$match", Value: bson.D{
			{Key: "deleted_at", Value: nil},
			{Key: "total_ratings", Value: bson.D{{Key: "$gt", Value: 0}}}, // Only drivers with ratings
		}}},
		bson.D{{Key: "$group", Value: bson.D{
			{Key: "_id", Value: nil},
			{Key: "avg_rating", Value: bson.D{{Key: "$avg", Value: "$rating"}}},
		}}},
	}

	cursor, err := r.collection.Aggregate(ctx, pipeline)
	if err != nil {
		return 0, fmt.Errorf("failed to calculate average rating: %w", err)
	}
	defer cursor.Close(ctx)

	var result struct {
		AvgRating float64 `bson:"avg_rating"`
	}

	if cursor.Next(ctx) {
		if err := cursor.Decode(&result); err != nil {
			return 0, fmt.Errorf("failed to decode average rating: %w", err)
		}
	}

	return result.AvgRating, nil
}

func (r *driverRepository) GetTopDrivers(ctx context.Context, limit int) ([]*models.Driver, error) {
	filter := bson.M{
		"deleted_at":    nil,
		"total_ratings": bson.M{"$gte": 10}, // At least 10 ratings
	}

	opts := options.Find().
		SetSort(bson.D{{Key: "rating", Value: -1}}).
		SetLimit(int64(limit))

	cursor, err := r.collection.Find(ctx, filter, opts)
	if err != nil {
		return nil, fmt.Errorf("failed to find top drivers: %w", err)
	}
	defer cursor.Close(ctx)

	var drivers []*models.Driver
	for cursor.Next(ctx) {
		var driver models.Driver
		if err := cursor.Decode(&driver); err != nil {
			return nil, fmt.Errorf("failed to decode driver: %w", err)
		}
		drivers = append(drivers, &driver)
	}

	return drivers, nil
}

func (r *driverRepository) GetDriverStats(ctx context.Context, driverID primitive.ObjectID, days int) (map[string]interface{}, error) {
	driver, err := r.GetByID(ctx, driverID)
	if err != nil {
		return nil, err
	}

	// Get rides for this driver in the specified period
	startDate := time.Now().AddDate(0, 0, -days)
	ridesCollection := r.collection.Database().Collection("rides")

	// Count rides by status
	ridesPipeline := mongo.Pipeline{
		bson.D{{Key: "$match", Value: bson.D{
			{Key: "driver_id", Value: driverID},
			{Key: "created_at", Value: bson.D{{Key: "$gte", Value: startDate}}},
		}}},
		bson.D{{Key: "$group", Value: bson.D{
			{Key: "_id", Value: "$status"},
			{Key: "count", Value: bson.D{{Key: "$sum", Value: 1}}},
			{Key: "total_earnings", Value: bson.D{{Key: "$sum", Value: "$fare.driver_amount"}}},
		}}},
	}

	cursor, err := ridesCollection.Aggregate(ctx, ridesPipeline)
	if err != nil {
		return nil, fmt.Errorf("failed to get driver ride stats: %w", err)
	}
	defer cursor.Close(ctx)

	rideCounts := make(map[string]int64)
	var totalEarnings float64

	for cursor.Next(ctx) {
		var result struct {
			Status        string  `bson:"_id"`
			Count         int64   `bson:"count"`
			TotalEarnings float64 `bson:"total_earnings"`
		}

		if err := cursor.Decode(&result); err != nil {
			return nil, fmt.Errorf("failed to decode ride stats: %w", err)
		}

		rideCounts[result.Status] = result.Count
		if result.Status == "completed" {
			totalEarnings = result.TotalEarnings
		}
	}

	// Calculate metrics
	totalRides := int64(0)
	completedRides := rideCounts["completed"]
	cancelledRides := rideCounts["cancelled"]

	for _, count := range rideCounts {
		totalRides += count
	}

	completionRate := float64(0)
	if totalRides > 0 {
		completionRate = float64(completedRides) / float64(totalRides) * 100
	}

	cancellationRate := float64(0)
	if totalRides > 0 {
		cancellationRate = float64(cancelledRides) / float64(totalRides) * 100
	}

	// Calculate average earnings per ride
	avgEarningsPerRide := float64(0)
	if completedRides > 0 {
		avgEarningsPerRide = totalEarnings / float64(completedRides)
	}

	// Get recent ratings
	recentRatingsPipeline := mongo.Pipeline{
		bson.D{{Key: "$match", Value: bson.D{
			{Key: "driver_id", Value: driverID},
			{Key: "created_at", Value: bson.D{{Key: "$gte", Value: startDate}}},
			{Key: "driver_rating", Value: bson.D{{Key: "$exists", Value: true}, {Key: "$ne", Value: nil}}},
		}}},
		bson.D{{Key: "$group", Value: bson.D{
			{Key: "_id", Value: nil},
			{Key: "avg_rating", Value: bson.D{{Key: "$avg", Value: "$driver_rating"}}},
			{Key: "rating_count", Value: bson.D{{Key: "$sum", Value: 1}}},
		}}},
	}

	cursor, err = ridesCollection.Aggregate(ctx, recentRatingsPipeline)
	if err != nil {
		return nil, fmt.Errorf("failed to get recent ratings: %w", err)
	}
	defer cursor.Close(ctx)

	var recentRating float64
	var recentRatingCount int64

	if cursor.Next(ctx) {
		var result struct {
			AvgRating   float64 `bson:"avg_rating"`
			RatingCount int64   `bson:"rating_count"`
		}

		if err := cursor.Decode(&result); err == nil {
			recentRating = result.AvgRating
			recentRatingCount = result.RatingCount
		}
	}

	return map[string]interface{}{
		"driver_id":              driverID,
		"period_days":            days,
		"total_rides":            totalRides,
		"completed_rides":        completedRides,
		"cancelled_rides":        cancelledRides,
		"completion_rate":        completionRate,
		"cancellation_rate":      cancellationRate,
		"total_earnings":         totalEarnings,
		"avg_earnings_per_ride":  avgEarningsPerRide,
		"recent_avg_rating":      recentRating,
		"recent_rating_count":    recentRatingCount,
		"overall_rating":         driver.Rating,
		"overall_total_rides":    driver.TotalRides,
		"overall_total_earnings": driver.TotalEarnings,
		"acceptance_rate":        driver.AcceptanceRate,
		"ride_counts_by_status":  rideCounts,
		"start_date":             startDate,
		"end_date":               time.Now(),
	}, nil
}

// Helper methods
func (r *driverRepository) findDriversWithFilter(ctx context.Context, filter bson.M, params *utils.PaginationParams) ([]*models.Driver, int64, error) {
	// Add search filter if provided
	if params.Search != "" {
		// Create OR conditions for different search fields
		searchConditions := []bson.M{}

		// Search in license number
		searchConditions = append(searchConditions, bson.M{
			"license_number": bson.M{"$regex": params.Search, "$options": "i"},
		})

		// Search in vehicle info if available
		searchConditions = append(searchConditions, bson.M{
			"$or": []bson.M{
				{"vehicle_make": bson.M{"$regex": params.Search, "$options": "i"}},
				{"vehicle_model": bson.M{"$regex": params.Search, "$options": "i"}},
			},
		})

		filter = bson.M{
			"$and": []bson.M{
				filter,
				{"$or": searchConditions},
			},
		}
	}

	// Get total count
	total, err := r.collection.CountDocuments(ctx, filter)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to count drivers: %w", err)
	}

	// Get paginated results
	opts := params.GetSortOptions()
	cursor, err := r.collection.Find(ctx, filter, opts)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to find drivers: %w", err)
	}
	defer cursor.Close(ctx)

	var drivers []*models.Driver
	for cursor.Next(ctx) {
		var driver models.Driver
		if err := cursor.Decode(&driver); err != nil {
			return nil, 0, fmt.Errorf("failed to decode driver: %w", err)
		}
		drivers = append(drivers, &driver)
	}

	return drivers, total, nil
}

// Cache operations
func (r *driverRepository) cacheDriver(ctx context.Context, driver *models.Driver) {
	if r.cache != nil {
		cacheKey := fmt.Sprintf("driver:%s", driver.ID.Hex())
		r.cache.Set(ctx, cacheKey, driver, 15*time.Minute)
	}
}

func (r *driverRepository) getDriverFromCache(ctx context.Context, driverID string) *models.Driver {
	if r.cache == nil {
		return nil
	}

	cacheKey := fmt.Sprintf("driver:%s", driverID)
	var driver models.Driver
	err := r.cache.Get(ctx, cacheKey, &driver)
	if err != nil {
		return nil
	}

	return &driver
}

func (r *driverRepository) invalidateDriverCache(ctx context.Context, driverID string) {
	if r.cache != nil {
		cacheKey := fmt.Sprintf("driver:%s", driverID)
		r.cache.Delete(ctx, cacheKey)

		// Also invalidate user-based cache if we have access to user_id
		// This would require getting the driver first, but we'll skip it for now
		// to avoid additional database calls
	}
}
