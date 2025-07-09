package services

import (
	"context"
	"fmt"
	"time"

	"goride/internal/models"
	"goride/internal/repositories/interfaces"
	"goride/internal/utils"

	"go.mongodb.org/mongo-driver/bson/primitive"
)

type AccessibilityService interface {
	// Accessibility Options Management
	CreateAccessibilityOption(ctx context.Context, option *models.AccessibilityOption) error
	GetAccessibilityOption(ctx context.Context, id primitive.ObjectID) (*models.AccessibilityOption, error)
	UpdateAccessibilityOption(ctx context.Context, id primitive.ObjectID, updates map[string]interface{}) error
	DeleteAccessibilityOption(ctx context.Context, id primitive.ObjectID) error
	ListAccessibilityOptions(ctx context.Context, params *utils.PaginationParams) ([]*models.AccessibilityOption, int64, error)
	GetActiveAccessibilityOptions(ctx context.Context) ([]*models.AccessibilityOption, error)

	// User Accessibility Management
	SetUserAccessibilityNeeds(ctx context.Context, userID primitive.ObjectID, needs []string) error
	GetUserAccessibilityNeeds(ctx context.Context, userID primitive.ObjectID) ([]string, error)
	ValidateAccessibilityNeeds(ctx context.Context, needs []string) error

	// Driver Vehicle Accessibility
	SetVehicleAccessibilityFeatures(ctx context.Context, vehicleID primitive.ObjectID, features []string) error
	GetAccessibleVehicles(ctx context.Context, requiredFeatures []string) ([]*models.Vehicle, error)

	// Ride Matching with Accessibility
	FindAccessibleDrivers(ctx context.Context, riderID primitive.ObjectID, location *models.Location, radiusKM float64) ([]*models.Driver, error)
	ValidateRideAccessibility(ctx context.Context, rideRequest *models.RideRequest) error

	// Analytics
	GetAccessibilityUsageStats(ctx context.Context, days int) (map[string]interface{}, error)
	GetAccessibilityComplianceReport(ctx context.Context, startDate, endDate time.Time) (map[string]interface{}, error)
}

type accessibilityService struct {
	riderRepo   interfaces.RiderRepository
	driverRepo  interfaces.DriverRepository
	vehicleRepo interfaces.VehicleRepository
	userRepo    interfaces.UserRepository
	cache       CacheService
}

func NewAccessibilityService(
	riderRepo interfaces.RiderRepository,
	driverRepo interfaces.DriverRepository,
	vehicleRepo interfaces.VehicleRepository,
	userRepo interfaces.UserRepository,
	cache CacheService,
) AccessibilityService {
	return &accessibilityService{
		riderRepo:   riderRepo,
		driverRepo:  driverRepo,
		vehicleRepo: vehicleRepo,
		userRepo:    userRepo,
		cache:       cache,
	}
}

// Accessibility Options Management
func (s *accessibilityService) CreateAccessibilityOption(ctx context.Context, option *models.AccessibilityOption) error {
	option.ID = primitive.NewObjectID()
	option.CreatedAt = time.Now()
	option.UpdatedAt = time.Now()
	option.IsActive = true

	// Validate required fields
	if option.Name == "" || option.Code == "" {
		return fmt.Errorf("name and code are required")
	}

	// Store in cache-like structure (you would implement this based on your cache service)
	return s.cacheAccessibilityOption(ctx, option)
}

func (s *accessibilityService) GetAccessibilityOption(ctx context.Context, id primitive.ObjectID) (*models.AccessibilityOption, error) {
	// Try cache first
	if option := s.getAccessibilityOptionFromCache(ctx, id.Hex()); option != nil {
		return option, nil
	}

	return nil, fmt.Errorf("accessibility option not found")
}

func (s *accessibilityService) UpdateAccessibilityOption(ctx context.Context, id primitive.ObjectID, updates map[string]interface{}) error {
	updates["updated_at"] = time.Now()

	// Invalidate cache
	s.invalidateAccessibilityOptionCache(ctx, id.Hex())

	return nil
}

func (s *accessibilityService) DeleteAccessibilityOption(ctx context.Context, id primitive.ObjectID) error {
	updates := map[string]interface{}{
		"is_active":  false,
		"updated_at": time.Now(),
	}

	return s.UpdateAccessibilityOption(ctx, id, updates)
}

func (s *accessibilityService) ListAccessibilityOptions(ctx context.Context, params *utils.PaginationParams) ([]*models.AccessibilityOption, int64, error) {
	// This would typically query from database
	options := s.getAllAccessibilityOptionsFromCache(ctx)
	total := int64(len(options))

	// Apply pagination
	start := (params.Page - 1) * params.PageSize
	end := start + params.PageSize

	if start >= len(options) {
		return []*models.AccessibilityOption{}, total, nil
	}

	if end > len(options) {
		end = len(options)
	}

	return options[start:end], total, nil
}

func (s *accessibilityService) GetActiveAccessibilityOptions(ctx context.Context) ([]*models.AccessibilityOption, error) {
	options := s.getAllAccessibilityOptionsFromCache(ctx)
	var activeOptions []*models.AccessibilityOption

	for _, option := range options {
		if option.IsActive {
			activeOptions = append(activeOptions, option)
		}
	}

	return activeOptions, nil
}

// User Accessibility Management
func (s *accessibilityService) SetUserAccessibilityNeeds(ctx context.Context, userID primitive.ObjectID, needs []string) error {
	// Validate accessibility needs
	if err := s.ValidateAccessibilityNeeds(ctx, needs); err != nil {
		return err
	}

	// Update rider profile
	updates := map[string]interface{}{
		"accessibility_needs": needs,
		"updated_at":          time.Now(),
	}

	// Get rider first, then update by rider ID
	rider, err := s.riderRepo.GetByUserID(ctx, userID)
	if err != nil {
		return err
	}

	return s.riderRepo.Update(ctx, rider.ID, updates)
}

func (s *accessibilityService) GetUserAccessibilityNeeds(ctx context.Context, userID primitive.ObjectID) ([]string, error) {
	rider, err := s.riderRepo.GetByUserID(ctx, userID)
	if err != nil {
		return nil, err
	}

	return rider.AccessibilityNeeds, nil
}

func (s *accessibilityService) ValidateAccessibilityNeeds(ctx context.Context, needs []string) error {
	validNeeds := map[string]bool{
		"wheelchair_accessible": true,
		"hearing_impaired":      true,
		"visually_impaired":     true,
		"service_animal":        true,
		"mobility_aid":          true,
		"cognitive_assistance":  true,
		"extra_time":            true,
		"large_print":           true,
		"sign_language":         true,
	}

	for _, need := range needs {
		if !validNeeds[need] {
			return fmt.Errorf("invalid accessibility need: %s", need)
		}
	}

	return nil
}

// Driver Vehicle Accessibility
func (s *accessibilityService) SetVehicleAccessibilityFeatures(ctx context.Context, vehicleID primitive.ObjectID, features []string) error {
	// Validate features
	validFeatures := map[string]bool{
		"wheelchair_ramp":      true,
		"wheelchair_lift":      true,
		"hand_controls":        true,
		"hearing_loop":         true,
		"braille_controls":     true,
		"voice_guidance":       true,
		"service_animal_space": true,
		"lowered_floor":        true,
		"wide_doors":           true,
		"accessible_seating":   true,
	}

	for _, feature := range features {
		if !validFeatures[feature] {
			return fmt.Errorf("invalid accessibility feature: %s", feature)
		}
	}

	updates := map[string]interface{}{
		"accessibility_features": features,
		"updated_at":             time.Now(),
	}

	return s.vehicleRepo.Update(ctx, vehicleID, updates)
}

func (s *accessibilityService) GetAccessibleVehicles(ctx context.Context, requiredFeatures []string) ([]*models.Vehicle, error) {
	// Get vehicles by features - since GetByAccessibilityFeatures doesn't exist,
	// we'll use GetByFeatures which should cover accessibility features
	params := &utils.PaginationParams{
		Page:     1,
		PageSize: 1000, // Get a large number to find all accessible vehicles
	}
	vehicles, _, err := s.vehicleRepo.GetByFeatures(ctx, requiredFeatures, params)
	if err != nil {
		return nil, err
	}

	// Filter for only accessible vehicles
	var accessibleVehicles []*models.Vehicle
	for _, vehicle := range vehicles {
		if vehicle.IsAccessible {
			accessibleVehicles = append(accessibleVehicles, vehicle)
		}
	}

	return accessibleVehicles, nil
}

// Ride Matching with Accessibility
func (s *accessibilityService) FindAccessibleDrivers(ctx context.Context, riderID primitive.ObjectID, location *models.Location, radiusKM float64) ([]*models.Driver, error) {
	// Get rider's accessibility needs
	accessibilityNeeds, err := s.GetUserAccessibilityNeeds(ctx, riderID)
	if err != nil {
		return nil, err
	}

	// If no accessibility needs, return regular nearby drivers
	if len(accessibilityNeeds) == 0 {
		return s.driverRepo.GetNearbyDrivers(ctx, location.Latitude(), location.Longitude(), radiusKM, "")
	}

	// Map accessibility needs to vehicle features
	requiredFeatures := s.mapNeedsToFeatures(accessibilityNeeds)

	// Get accessible vehicles
	accessibleVehicles, err := s.GetAccessibleVehicles(ctx, requiredFeatures)
	if err != nil {
		return nil, err
	}

	// Get drivers for these vehicles
	var accessibleDrivers []*models.Driver
	for _, vehicle := range accessibleVehicles {
		// Get driver by the vehicle's driver ID
		driver, err := s.driverRepo.GetByID(ctx, vehicle.DriverID)
		if err != nil {
			continue
		}

		// Check if driver is nearby and available
		if driver.IsAvailable && s.isDriverNearby(driver, location, radiusKM) {
			accessibleDrivers = append(accessibleDrivers, driver)
		}
	}

	return accessibleDrivers, nil
}

func (s *accessibilityService) ValidateRideAccessibility(ctx context.Context, rideRequest *models.RideRequest) error {
	// Get rider's accessibility needs
	accessibilityNeeds, err := s.GetUserAccessibilityNeeds(ctx, rideRequest.RiderID)
	if err != nil {
		return err
	}

	// If no accessibility needs, no validation needed
	if len(accessibilityNeeds) == 0 {
		return nil
	}

	// Check if ride type supports accessibility needs
	accessibleRideTypes := map[models.RideType]bool{
		models.RideTypeAccessible: true,
		models.RideTypeStandard:   false, // depends on vehicle
		models.RideTypePremium:    false, // depends on vehicle
		models.RideTypeXL:         true,  // typically more accessible
	}

	if !accessibleRideTypes[rideRequest.RideType] && rideRequest.RideType != models.RideTypeAccessible {
		return fmt.Errorf("selected ride type may not support accessibility requirements")
	}

	return nil
}

// Analytics
func (s *accessibilityService) GetAccessibilityUsageStats(ctx context.Context, days int) (map[string]interface{}, error) {
	stats := map[string]interface{}{
		"total_accessible_rides":  0,
		"accessibility_requests":  0,
		"most_requested_features": []string{},
		"compliance_percentage":   0.0,
		"response_time_avg":       0.0,
		"satisfaction_rating_avg": 0.0,
	}

	// This would calculate real statistics from database
	// For now, returning placeholder data
	return stats, nil
}

func (s *accessibilityService) GetAccessibilityComplianceReport(ctx context.Context, startDate, endDate time.Time) (map[string]interface{}, error) {
	report := map[string]interface{}{
		"period":                  fmt.Sprintf("%s to %s", startDate.Format("2006-01-02"), endDate.Format("2006-01-02")),
		"total_vehicles":          0,
		"accessible_vehicles":     0,
		"compliance_percentage":   0.0,
		"accessible_rides_served": 0,
		"accessibility_denials":   0,
		"improvement_areas":       []string{},
	}

	return report, nil
}

// Helper methods
func (s *accessibilityService) mapNeedsToFeatures(needs []string) []string {
	needToFeatureMap := map[string]string{
		"wheelchair_accessible": "wheelchair_ramp",
		"hearing_impaired":      "hearing_loop",
		"visually_impaired":     "voice_guidance",
		"service_animal":        "service_animal_space",
		"mobility_aid":          "accessible_seating",
	}

	var features []string
	for _, need := range needs {
		if feature, exists := needToFeatureMap[need]; exists {
			features = append(features, feature)
		}
	}

	return features
}

func (s *accessibilityService) isDriverNearby(driver *models.Driver, location *models.Location, radiusKM float64) bool {
	if driver.CurrentLocation == nil {
		return false
	}

	distance := utils.CalculateDistance(
		location.Latitude(), location.Longitude(),
		driver.CurrentLocation.Latitude(), driver.CurrentLocation.Longitude(),
	)

	return distance <= radiusKM
}

// Cache helper methods (implement based on your cache service)
func (s *accessibilityService) cacheAccessibilityOption(ctx context.Context, option *models.AccessibilityOption) error {
	if s.cache != nil {
		key := fmt.Sprintf("accessibility_option:%s", option.ID.Hex())
		return s.cache.Set(ctx, key, option, 24*time.Hour)
	}
	return nil
}

func (s *accessibilityService) getAccessibilityOptionFromCache(ctx context.Context, id string) *models.AccessibilityOption {
	if s.cache != nil {
		key := fmt.Sprintf("accessibility_option:%s", id)
		var option models.AccessibilityOption
		if err := s.cache.Get(ctx, key, &option); err == nil {
			return &option
		}
	}
	return nil
}

func (s *accessibilityService) invalidateAccessibilityOptionCache(ctx context.Context, id string) {
	if s.cache != nil {
		key := fmt.Sprintf("accessibility_option:%s", id)
		s.cache.Delete(ctx, key)
	}
}

func (s *accessibilityService) getAllAccessibilityOptionsFromCache(ctx context.Context) []*models.AccessibilityOption {
	// This would implement cache retrieval of all options
	// For now, returning default options
	return []*models.AccessibilityOption{
		{
			ID:          primitive.NewObjectID(),
			Name:        "Wheelchair Accessible",
			Code:        "wheelchair_accessible",
			Description: "Vehicle has wheelchair ramp or lift",
			Icon:        "wheelchair",
			IsActive:    true,
			CreatedAt:   time.Now(),
			UpdatedAt:   time.Now(),
		},
		{
			ID:          primitive.NewObjectID(),
			Name:        "Hearing Impaired Support",
			Code:        "hearing_impaired",
			Description: "Vehicle has hearing loop system",
			Icon:        "hearing",
			IsActive:    true,
			CreatedAt:   time.Now(),
			UpdatedAt:   time.Now(),
		},
		{
			ID:          primitive.NewObjectID(),
			Name:        "Service Animal Friendly",
			Code:        "service_animal",
			Description: "Vehicle accommodates service animals",
			Icon:        "pet",
			IsActive:    true,
			CreatedAt:   time.Now(),
			UpdatedAt:   time.Now(),
		},
	}
}
