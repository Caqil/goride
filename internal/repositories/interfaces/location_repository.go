package interfaces

import (
	"context"
	"time"

	"goride/internal/models"
	"goride/internal/utils"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

type LocationRepository interface {
	// Basic CRUD operations
	Create(ctx context.Context, location *models.LocationHistory) error
	GetByID(ctx context.Context, id primitive.ObjectID) (*models.LocationHistory, error)
	Update(ctx context.Context, id primitive.ObjectID, updates map[string]interface{}) error
	Delete(ctx context.Context, id primitive.ObjectID) error

	// User location tracking
	CreateLocationHistory(ctx context.Context, userID primitive.ObjectID, location *models.Location) error
	GetUserLocationHistory(ctx context.Context, userID primitive.ObjectID, params *utils.PaginationParams) ([]*models.LocationHistory, int64, error)
	GetLatestUserLocation(ctx context.Context, userID primitive.ObjectID) (*models.LocationHistory, error)
	GetUserLocationsByDateRange(ctx context.Context, userID primitive.ObjectID, startDate, endDate time.Time) ([]*models.LocationHistory, error)

	// Driver location operations
	UpdateDriverLocation(ctx context.Context, driverID primitive.ObjectID, location *models.Location) error
	GetNearbyUsers(ctx context.Context, lat, lng, radiusKM float64, userType models.UserType) ([]*models.LocationHistory, error)
	GetUsersInArea(ctx context.Context, bounds *utils.Bounds, userType models.UserType) ([]*models.LocationHistory, error)
	GetActiveDriverLocations(ctx context.Context) ([]*models.LocationHistory, error)

	// Ride location tracking
	CreateRideLocationUpdate(ctx context.Context, rideID primitive.ObjectID, location *models.Location) error
	GetRideLocationHistory(ctx context.Context, rideID primitive.ObjectID) ([]*models.LocationHistory, error)
	GetLatestRideLocation(ctx context.Context, rideID primitive.ObjectID) (*models.LocationHistory, error)

	// Search and filtering
	SearchByAddress(ctx context.Context, address string, params *utils.PaginationParams) ([]*models.LocationHistory, int64, error)
	GetLocationsByCity(ctx context.Context, city string, params *utils.PaginationParams) ([]*models.LocationHistory, int64, error)
	GetFrequentLocations(ctx context.Context, userID primitive.ObjectID, limit int) ([]*models.LocationHistory, error)

	// Geocoding cache
	CacheGeocodingResult(ctx context.Context, address string, location *models.Location) error
	GetCachedGeocodingResult(ctx context.Context, address string) (*models.Location, error)

	// Analytics
	GetLocationStats(ctx context.Context, days int) (map[string]interface{}, error)
	GetPopularAreas(ctx context.Context, city string, days int, limit int) ([]map[string]interface{}, error)
	GetUserMovementPattern(ctx context.Context, userID primitive.ObjectID, days int) (map[string]interface{}, error)

	// Cleanup operations
	DeleteOldLocations(ctx context.Context, days int) error
	ArchiveLocationHistory(ctx context.Context, beforeDate time.Time) error
}
