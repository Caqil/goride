package interfaces

import (
	"context"

	"github.com/uber-clone/internal/models"
	"github.com/uber-clone/internal/utils"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

type DriverRepository interface {
	// Basic CRUD operations
	Create(ctx context.Context, driver *models.Driver) error
	GetByID(ctx context.Context, id primitive.ObjectID) (*models.Driver, error)
	GetByUserID(ctx context.Context, userID primitive.ObjectID) (*models.Driver, error)
	Update(ctx context.Context, id primitive.ObjectID, updates map[string]interface{}) error
	Delete(ctx context.Context, id primitive.ObjectID) error

	// Location operations
	UpdateLocation(ctx context.Context, id primitive.ObjectID, location *models.Location) error
	GetNearbyDrivers(ctx context.Context, lat, lng, radiusKM float64, rideType string) ([]*models.Driver, error)
	GetDriversInArea(ctx context.Context, bounds *utils.Bounds) ([]*models.Driver, error)

	// Status operations
	UpdateStatus(ctx context.Context, id primitive.ObjectID, status models.DriverStatus) error
	UpdateAvailability(ctx context.Context, id primitive.ObjectID, available bool) error
	GetAvailableDrivers(ctx context.Context, params *utils.PaginationParams) ([]*models.Driver, int64, error)
	GetOnlineDrivers(ctx context.Context) ([]*models.Driver, error)

	// Document verification
	UpdateDocumentStatus(ctx context.Context, id primitive.ObjectID, docType string, status models.DocumentStatus) error
	GetPendingVerifications(ctx context.Context, params *utils.PaginationParams) ([]*models.Driver, int64, error)

	// Performance metrics
	UpdateRating(ctx context.Context, id primitive.ObjectID, rating float64, totalRatings int64) error
	UpdateEarnings(ctx context.Context, id primitive.ObjectID, amount float64) error
	UpdateRideStats(ctx context.Context, id primitive.ObjectID, totalRides int64) error

	// Search and filtering
	List(ctx context.Context, params *utils.PaginationParams) ([]*models.Driver, int64, error)
	GetByStatus(ctx context.Context, status models.DriverStatus, params *utils.PaginationParams) ([]*models.Driver, int64, error)
	GetByRatingRange(ctx context.Context, minRating, maxRating float64, params *utils.PaginationParams) ([]*models.Driver, int64, error)
	SearchByLicense(ctx context.Context, licenseNumber string) (*models.Driver, error)

	// Analytics
	GetTotalCount(ctx context.Context) (int64, error)
	GetCountByStatus(ctx context.Context, status models.DriverStatus) (int64, error)
	GetAverageRating(ctx context.Context) (float64, error)
	GetTopDrivers(ctx context.Context, limit int) ([]*models.Driver, error)
	GetDriverStats(ctx context.Context, driverID primitive.ObjectID, days int) (map[string]interface{}, error)
}
