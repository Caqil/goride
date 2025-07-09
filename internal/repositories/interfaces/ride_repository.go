package interfaces

import (
	"context"
	"time"

	"goride/internal/models"
	"goride/internal/utils"

	"go.mongodb.org/mongo-driver/bson/primitive"
)

type RideRepository interface {
	// Basic CRUD operations
	Create(ctx context.Context, ride *models.Ride) error
	GetByID(ctx context.Context, id primitive.ObjectID) (*models.Ride, error)
	GetByRideNumber(ctx context.Context, rideNumber string) (*models.Ride, error)
	Update(ctx context.Context, id primitive.ObjectID, updates map[string]interface{}) error
	Delete(ctx context.Context, id primitive.ObjectID) error

	// Status operations
	UpdateStatus(ctx context.Context, id primitive.ObjectID, status models.RideStatus) error
	AssignDriver(ctx context.Context, id primitive.ObjectID, driverID, vehicleID primitive.ObjectID) error
	StartRide(ctx context.Context, id primitive.ObjectID) error
	CompleteRide(ctx context.Context, id primitive.ObjectID, actualDistance float64, actualDuration int, actualFare float64) error
	CancelRide(ctx context.Context, id primitive.ObjectID, reason, cancelledBy string) error

	// Route operations
	UpdateRoute(ctx context.Context, id primitive.ObjectID, route *models.Route) error
	AddWaypoint(ctx context.Context, id primitive.ObjectID, waypoint *models.Location) error
	RemoveWaypoint(ctx context.Context, id primitive.ObjectID, waypointIndex int) error

	// Search and filtering
	GetByRider(ctx context.Context, riderID primitive.ObjectID, params *utils.PaginationParams) ([]*models.Ride, int64, error)
	GetByDriver(ctx context.Context, driverID primitive.ObjectID, params *utils.PaginationParams) ([]*models.Ride, int64, error)
	GetByStatus(ctx context.Context, status models.RideStatus, params *utils.PaginationParams) ([]*models.Ride, int64, error)
	GetActiveRides(ctx context.Context) ([]*models.Ride, error)
	GetPendingRides(ctx context.Context) ([]*models.Ride, error)

	// Time-based queries
	GetRidesByDateRange(ctx context.Context, startDate, endDate time.Time, params *utils.PaginationParams) ([]*models.Ride, int64, error)
	GetScheduledRides(ctx context.Context, scheduledTime time.Time) ([]*models.Ride, error)
	GetRidesInProgress(ctx context.Context) ([]*models.Ride, error)

	// Location-based queries
	GetRidesInArea(ctx context.Context, bounds *utils.Bounds, params *utils.PaginationParams) ([]*models.Ride, int64, error)
	GetNearbyRides(ctx context.Context, lat, lng, radiusKM float64) ([]*models.Ride, error)

	// Analytics and statistics
	GetRideStats(ctx context.Context, startDate, endDate time.Time) (map[string]interface{}, error)
	GetRevenueStats(ctx context.Context, startDate, endDate time.Time) (map[string]interface{}, error)
	GetPopularRoutes(ctx context.Context, limit int, days int) ([]map[string]interface{}, error)
	GetPeakHours(ctx context.Context, days int) ([]map[string]interface{}, error)

	// Ratings
	UpdateRiderRating(ctx context.Context, id primitive.ObjectID, rating float64) error
	UpdateDriverRating(ctx context.Context, id primitive.ObjectID, rating float64) error
}
