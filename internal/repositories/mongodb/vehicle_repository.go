package mongodb

import (
	"context"
	"fmt"
	"strings"
	"time"

	"goride/internal/models"
	"goride/internal/repositories/interfaces"
	"goride/internal/utils"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

type vehicleRepository struct {
	collection *mongo.Collection
	cache      CacheService
}

func NewVehicleRepository(db *mongo.Database, cache CacheService) interfaces.VehicleRepository {
	return &vehicleRepository{
		collection: db.Collection("vehicles"),
		cache:      cache,
	}
}

// Basic CRUD operations
func (r *vehicleRepository) Create(ctx context.Context, vehicle *models.Vehicle) error {
	vehicle.ID = primitive.NewObjectID()
	vehicle.CreatedAt = time.Now()
	vehicle.UpdatedAt = time.Now()

	// Normalize license plate to uppercase
	vehicle.LicensePlate = strings.ToUpper(vehicle.LicensePlate)

	_, err := r.collection.InsertOne(ctx, vehicle)
	if err != nil {
		return fmt.Errorf("failed to create vehicle: %w", err)
	}

	// Cache active vehicles
	if vehicle.Status == models.VehicleStatusActive {
		r.cacheVehicle(ctx, vehicle)
	}

	return nil
}

func (r *vehicleRepository) GetByID(ctx context.Context, id primitive.ObjectID) (*models.Vehicle, error) {
	// Try cache first
	if vehicle := r.getVehicleFromCache(ctx, id.Hex()); vehicle != nil {
		return vehicle, nil
	}

	var vehicle models.Vehicle
	err := r.collection.FindOne(ctx, bson.M{"_id": id}).Decode(&vehicle)
	if err != nil {
		if err == mongo.ErrNoDocuments {
			return nil, fmt.Errorf("vehicle not found")
		}
		return nil, fmt.Errorf("failed to get vehicle: %w", err)
	}

	// Cache active vehicles
	if vehicle.Status == models.VehicleStatusActive {
		r.cacheVehicle(ctx, &vehicle)
	}

	return &vehicle, nil
}

func (r *vehicleRepository) Update(ctx context.Context, id primitive.ObjectID, updates map[string]interface{}) error {
	updates["updated_at"] = time.Now()

	// Normalize license plate if being updated
	if licensePlate, exists := updates["license_plate"]; exists {
		if plateStr, ok := licensePlate.(string); ok {
			updates["license_plate"] = strings.ToUpper(plateStr)
		}
	}

	_, err := r.collection.UpdateOne(
		ctx,
		bson.M{"_id": id},
		bson.M{"$set": updates},
	)
	if err != nil {
		return fmt.Errorf("failed to update vehicle: %w", err)
	}

	// Invalidate cache
	r.invalidateVehicleCache(ctx, id.Hex())

	return nil
}

func (r *vehicleRepository) Delete(ctx context.Context, id primitive.ObjectID) error {
	_, err := r.collection.DeleteOne(ctx, bson.M{"_id": id})
	if err != nil {
		return fmt.Errorf("failed to delete vehicle: %w", err)
	}

	// Invalidate cache
	r.invalidateVehicleCache(ctx, id.Hex())

	return nil
}

// Driver association
func (r *vehicleRepository) GetByDriverID(ctx context.Context, driverID primitive.ObjectID) ([]*models.Vehicle, error) {
	filter := bson.M{"driver_id": driverID}

	cursor, err := r.collection.Find(ctx, filter, options.Find().SetSort(bson.D{{Key: "created_at", Value: -1}}))
	if err != nil {
		return nil, fmt.Errorf("failed to find vehicles by driver ID: %w", err)
	}
	defer cursor.Close(ctx)

	var vehicles []*models.Vehicle
	for cursor.Next(ctx) {
		var vehicle models.Vehicle
		if err := cursor.Decode(&vehicle); err != nil {
			return nil, fmt.Errorf("failed to decode vehicle: %w", err)
		}
		vehicles = append(vehicles, &vehicle)
	}

	return vehicles, nil
}

func (r *vehicleRepository) GetActiveVehicleByDriverID(ctx context.Context, driverID primitive.ObjectID) (*models.Vehicle, error) {
	// Try cache first
	cacheKey := fmt.Sprintf("active_vehicle_driver_%s", driverID.Hex())
	if r.cache != nil {
		var vehicle models.Vehicle
		if err := r.cache.Get(ctx, cacheKey, &vehicle); err == nil {
			return &vehicle, nil
		}
	}

	var vehicle models.Vehicle
	err := r.collection.FindOne(ctx, bson.M{
		"driver_id": driverID,
		"status":    models.VehicleStatusActive,
	}).Decode(&vehicle)
	if err != nil {
		if err == mongo.ErrNoDocuments {
			return nil, fmt.Errorf("active vehicle not found for driver")
		}
		return nil, fmt.Errorf("failed to get active vehicle for driver: %w", err)
	}

	// Cache the result
	if r.cache != nil {
		r.cache.Set(ctx, cacheKey, vehicle, 15*time.Minute)
	}

	return &vehicle, nil
}

// Document verification
func (r *vehicleRepository) UpdateDocumentStatus(ctx context.Context, id primitive.ObjectID, docType string, status models.DocumentStatus) error {
	updates := map[string]interface{}{}

	switch docType {
	case "registration":
		updates["registration_status"] = status
	case "insurance":
		updates["insurance_status"] = status
	default:
		return fmt.Errorf("unknown document type: %s", docType)
	}

	return r.Update(ctx, id, updates)
}

func (r *vehicleRepository) GetPendingVerifications(ctx context.Context, params *utils.PaginationParams) ([]*models.Vehicle, int64, error) {
	filter := bson.M{
		"$or": []bson.M{
			{"registration_status": models.DocumentStatusPending},
			{"insurance_status": models.DocumentStatusPending},
		},
	}
	return r.findVehiclesWithFilter(ctx, filter, params)
}

// Vehicle identification
func (r *vehicleRepository) GetByLicensePlate(ctx context.Context, licensePlate string) (*models.Vehicle, error) {
	licensePlate = strings.ToUpper(licensePlate)

	// Try cache first
	cacheKey := fmt.Sprintf("vehicle_plate_%s", licensePlate)
	if r.cache != nil {
		var vehicle models.Vehicle
		if err := r.cache.Get(ctx, cacheKey, &vehicle); err == nil {
			return &vehicle, nil
		}
	}

	var vehicle models.Vehicle
	err := r.collection.FindOne(ctx, bson.M{"license_plate": licensePlate}).Decode(&vehicle)
	if err != nil {
		if err == mongo.ErrNoDocuments {
			return nil, fmt.Errorf("vehicle not found with license plate")
		}
		return nil, fmt.Errorf("failed to get vehicle by license plate: %w", err)
	}

	// Cache the result
	if r.cache != nil {
		r.cache.Set(ctx, cacheKey, vehicle, 30*time.Minute)
	}

	return &vehicle, nil
}

func (r *vehicleRepository) GetByVIN(ctx context.Context, vin string) (*models.Vehicle, error) {
	vin = strings.ToUpper(vin)

	var vehicle models.Vehicle
	err := r.collection.FindOne(ctx, bson.M{"vin": vin}).Decode(&vehicle)
	if err != nil {
		if err == mongo.ErrNoDocuments {
			return nil, fmt.Errorf("vehicle not found with VIN")
		}
		return nil, fmt.Errorf("failed to get vehicle by VIN: %w", err)
	}

	return &vehicle, nil
}

// Status operations
func (r *vehicleRepository) UpdateStatus(ctx context.Context, id primitive.ObjectID, status models.VehicleStatus) error {
	updates := map[string]interface{}{
		"status": status,
	}

	// Set specific timestamps based on status
	switch status {
	case models.VehicleStatusMaintenance:
		updates["last_maintenance"] = time.Now()
	case models.VehicleStatusSuspended:
		updates["suspended_at"] = time.Now()
	}

	return r.Update(ctx, id, updates)
}

func (r *vehicleRepository) GetByStatus(ctx context.Context, status models.VehicleStatus, params *utils.PaginationParams) ([]*models.Vehicle, int64, error) {
	filter := bson.M{"status": status}
	return r.findVehiclesWithFilter(ctx, filter, params)
}

// Vehicle type and features
func (r *vehicleRepository) GetByVehicleType(ctx context.Context, vehicleTypeID primitive.ObjectID, params *utils.PaginationParams) ([]*models.Vehicle, int64, error) {
	filter := bson.M{"vehicle_type_id": vehicleTypeID}
	return r.findVehiclesWithFilter(ctx, filter, params)
}

func (r *vehicleRepository) GetAccessibleVehicles(ctx context.Context, params *utils.PaginationParams) ([]*models.Vehicle, int64, error) {
	filter := bson.M{
		"is_accessible": true,
		"status":        models.VehicleStatusActive,
	}
	return r.findVehiclesWithFilter(ctx, filter, params)
}

func (r *vehicleRepository) GetByFeatures(ctx context.Context, features []string, params *utils.PaginationParams) ([]*models.Vehicle, int64, error) {
	filter := bson.M{
		"features": bson.M{"$all": features},
		"status":   models.VehicleStatusActive,
	}
	return r.findVehiclesWithFilter(ctx, filter, params)
}

// Maintenance
func (r *vehicleRepository) UpdateMaintenanceDate(ctx context.Context, id primitive.ObjectID, lastMaintenance, nextMaintenance *time.Time) error {
	updates := map[string]interface{}{}

	if lastMaintenance != nil {
		updates["last_maintenance"] = *lastMaintenance
	}

	if nextMaintenance != nil {
		updates["next_maintenance"] = *nextMaintenance
	}

	return r.Update(ctx, id, updates)
}

func (r *vehicleRepository) GetVehiclesDueForMaintenance(ctx context.Context) ([]*models.Vehicle, error) {
	filter := bson.M{
		"next_maintenance": bson.M{"$lte": time.Now()},
		"status":           bson.M{"$ne": models.VehicleStatusMaintenance},
	}

	cursor, err := r.collection.Find(ctx, filter, options.Find().SetSort(bson.D{{Key: "next_maintenance", Value: 1}}))
	if err != nil {
		return nil, fmt.Errorf("failed to find vehicles due for maintenance: %w", err)
	}
	defer cursor.Close(ctx)

	var vehicles []*models.Vehicle
	for cursor.Next(ctx) {
		var vehicle models.Vehicle
		if err := cursor.Decode(&vehicle); err != nil {
			return nil, fmt.Errorf("failed to decode vehicle: %w", err)
		}
		vehicles = append(vehicles, &vehicle)
	}

	return vehicles, nil
}

// Statistics
func (r *vehicleRepository) UpdateStats(ctx context.Context, id primitive.ObjectID, totalRides int64, totalDistance float64) error {
	updates := map[string]interface{}{
		"total_rides":    totalRides,
		"total_distance": totalDistance,
	}
	return r.Update(ctx, id, updates)
}

func (r *vehicleRepository) GetVehicleStats(ctx context.Context, vehicleID primitive.ObjectID, days int) (map[string]interface{}, error) {
	vehicle, err := r.GetByID(ctx, vehicleID)
	if err != nil {
		return nil, err
	}

	startDate := time.Now().AddDate(0, 0, -days)

	// Get ride stats from rides collection
	ridesCollection := r.collection.Database().Collection("rides")

	// Get ride counts and stats for the period
	ridePipeline := mongo.Pipeline{
		{{"$match", bson.M{
			"vehicle_id":   vehicleID,
			"requested_at": bson.M{"$gte": startDate},
		}}},
		{{"$group", bson.M{
			"_id":            "$status",
			"count":          bson.M{"$sum": 1},
			"total_distance": bson.M{"$sum": "$actual_distance"},
			"total_duration": bson.M{"$sum": "$actual_duration"},
			"total_earnings": bson.M{"$sum": "$driver_earnings"},
		}}},
	}

	cursor, err := ridesCollection.Aggregate(ctx, ridePipeline)
	if err != nil {
		return nil, fmt.Errorf("failed to get vehicle ride stats: %w", err)
	}
	defer cursor.Close(ctx)

	rideCounts := make(map[string]int64)
	var totalDistance, totalEarnings float64
	var totalDuration int64

	for cursor.Next(ctx) {
		var result struct {
			Status        models.RideStatus `bson:"_id"`
			Count         int64             `bson:"count"`
			TotalDistance float64           `bson:"total_distance"`
			TotalDuration int64             `bson:"total_duration"`
			TotalEarnings float64           `bson:"total_earnings"`
		}

		if err := cursor.Decode(&result); err != nil {
			return nil, fmt.Errorf("failed to decode vehicle ride stats: %w", err)
		}

		rideCounts[string(result.Status)] = result.Count
		if result.Status == models.RideStatusCompleted {
			totalDistance += result.TotalDistance
			totalDuration += result.TotalDuration
			totalEarnings += result.TotalEarnings
		}
	}

	// Calculate metrics
	totalRides := int64(0)
	completedRides := rideCounts["completed"]
	cancelledRides := rideCounts["cancelled"]

	for _, count := range rideCounts {
		totalRides += count
	}

	utilizationRate := float64(0)
	if totalRides > 0 {
		utilizationRate = float64(completedRides) / float64(totalRides) * 100
	}

	avgDistancePerRide := float64(0)
	if completedRides > 0 {
		avgDistancePerRide = totalDistance / float64(completedRides)
	}

	avgEarningsPerRide := float64(0)
	if completedRides > 0 {
		avgEarningsPerRide = totalEarnings / float64(completedRides)
	}

	// Calculate maintenance status
	maintenanceDue := false
	daysUntilMaintenance := int64(0)
	if vehicle.NextMaintenance != nil {
		daysUntilMaintenance = int64(vehicle.NextMaintenance.Sub(time.Now()).Hours() / 24)
		maintenanceDue = daysUntilMaintenance <= 0
	}

	return map[string]interface{}{
		"vehicle_id":             vehicleID,
		"period_days":            days,
		"total_rides":            totalRides,
		"completed_rides":        completedRides,
		"cancelled_rides":        cancelledRides,
		"utilization_rate":       utilizationRate,
		"total_distance":         totalDistance,
		"total_duration_hours":   float64(totalDuration) / 60,
		"total_earnings":         totalEarnings,
		"avg_distance_per_ride":  avgDistancePerRide,
		"avg_earnings_per_ride":  avgEarningsPerRide,
		"overall_total_rides":    vehicle.TotalRides,
		"overall_total_distance": vehicle.TotalDistance,
		"current_status":         vehicle.Status,
		"maintenance_due":        maintenanceDue,
		"days_until_maintenance": daysUntilMaintenance,
		"last_maintenance":       vehicle.LastMaintenance,
		"next_maintenance":       vehicle.NextMaintenance,
		"ride_counts_by_status":  rideCounts,
		"start_date":             startDate,
		"end_date":               time.Now(),
	}, nil
}

// Search and listing
func (r *vehicleRepository) List(ctx context.Context, params *utils.PaginationParams) ([]*models.Vehicle, int64, error) {
	filter := bson.M{}
	return r.findVehiclesWithFilter(ctx, filter, params)
}

func (r *vehicleRepository) SearchByMakeModel(ctx context.Context, make, model string, params *utils.PaginationParams) ([]*models.Vehicle, int64, error) {
	filter := bson.M{}

	if make != "" && model != "" {
		filter = bson.M{
			"$and": []bson.M{
				{"make": bson.M{"$regex": make, "$options": "i"}},
				{"model": bson.M{"$regex": model, "$options": "i"}},
			},
		}
	} else if make != "" {
		filter = bson.M{"make": bson.M{"$regex": make, "$options": "i"}}
	} else if model != "" {
		filter = bson.M{"model": bson.M{"$regex": model, "$options": "i"}}
	}

	return r.findVehiclesWithFilter(ctx, filter, params)
}

// Analytics
func (r *vehicleRepository) GetTotalCount(ctx context.Context) (int64, error) {
	return r.collection.CountDocuments(ctx, bson.M{})
}

func (r *vehicleRepository) GetCountByStatus(ctx context.Context, status models.VehicleStatus) (int64, error) {
	return r.collection.CountDocuments(ctx, bson.M{"status": status})
}

func (r *vehicleRepository) GetCountByType(ctx context.Context, vehicleTypeID primitive.ObjectID) (int64, error) {
	return r.collection.CountDocuments(ctx, bson.M{"vehicle_type_id": vehicleTypeID})
}

// Helper methods
func (r *vehicleRepository) findVehiclesWithFilter(ctx context.Context, filter bson.M, params *utils.PaginationParams) ([]*models.Vehicle, int64, error) {
	// Add search filter if provided
	if params.Search != "" {
		searchFields := []string{"make", "model", "color", "license_plate", "vin"}
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
		return nil, 0, fmt.Errorf("failed to count vehicles: %w", err)
	}

	// Get paginated results
	opts := params.GetSortOptions()
	cursor, err := r.collection.Find(ctx, filter, opts)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to find vehicles: %w", err)
	}
	defer cursor.Close(ctx)

	var vehicles []*models.Vehicle
	for cursor.Next(ctx) {
		var vehicle models.Vehicle
		if err := cursor.Decode(&vehicle); err != nil {
			return nil, 0, fmt.Errorf("failed to decode vehicle: %w", err)
		}
		vehicles = append(vehicles, &vehicle)
	}

	return vehicles, total, nil
}

// Cache operations
func (r *vehicleRepository) cacheVehicle(ctx context.Context, vehicle *models.Vehicle) {
	if r.cache != nil && vehicle.Status == models.VehicleStatusActive {
		cacheKey := fmt.Sprintf("vehicle:%s", vehicle.ID.Hex())
		r.cache.Set(ctx, cacheKey, vehicle, 15*time.Minute)

		// Also cache by license plate
		if vehicle.LicensePlate != "" {
			plateKey := fmt.Sprintf("vehicle_plate_%s", vehicle.LicensePlate)
			r.cache.Set(ctx, plateKey, vehicle, 30*time.Minute)
		}

		// Also cache as active vehicle for driver
		if !vehicle.DriverID.IsZero() {
			driverKey := fmt.Sprintf("active_vehicle_driver_%s", vehicle.DriverID.Hex())
			r.cache.Set(ctx, driverKey, vehicle, 15*time.Minute)
		}
	}
}

func (r *vehicleRepository) getVehicleFromCache(ctx context.Context, vehicleID string) *models.Vehicle {
	if r.cache == nil {
		return nil
	}

	cacheKey := fmt.Sprintf("vehicle:%s", vehicleID)
	var vehicle models.Vehicle
	err := r.cache.Get(ctx, cacheKey, &vehicle)
	if err != nil {
		return nil
	}

	return &vehicle
}

func (r *vehicleRepository) invalidateVehicleCache(ctx context.Context, vehicleID string) {
	if r.cache != nil {
		cacheKey := fmt.Sprintf("vehicle:%s", vehicleID)
		r.cache.Delete(ctx, cacheKey)

		// Note: We can't easily invalidate the license plate and driver caches
		// without additional lookups. This is a trade-off for performance vs cache consistency
	}
}
