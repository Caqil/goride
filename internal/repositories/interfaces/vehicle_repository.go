package interfaces

import (
	"context"

	"github.com/uber-clone/internal/models"
	"github.com/uber-clone/internal/utils"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

type VehicleRepository interface {
	// Basic CRUD operations
	Create(ctx context.Context, vehicle *models.Vehicle) error
	GetByID(ctx context.Context, id primitive.ObjectID) (*models.Vehicle, error)
	Update(ctx context.Context, id primitive.ObjectID, updates map[string]interface{}) error
	Delete(ctx context.Context, id primitive.ObjectID) error
	
	// Driver association
	GetByDriverID(ctx context.Context, driverID primitive.ObjectID) ([]*models.Vehicle, error)
	GetActiveVehicleByDriverID(ctx context.Context, driverID primitive.ObjectID) (*models.Vehicle, error)
	
	// Document verification
	UpdateDocumentStatus(ctx context.Context, id primitive.ObjectID, docType string, status models.DocumentStatus) error
	GetPendingVerifications(ctx context.Context, params *utils.PaginationParams) ([]*models.Vehicle, int64, error)
	
	// Vehicle identification
	GetByLicensePlate(ctx context.Context, licensePlate string) (*models.Vehicle, error)
	GetByVIN(ctx context.Context, vin string) (*models.Vehicle, error)
	
	// Status operations
	UpdateStatus(ctx context.Context, id primitive.ObjectID, status models.VehicleStatus) error
	GetByStatus(ctx context.Context, status models.VehicleStatus, params *utils.PaginationParams) ([]*models.Vehicle, int64, error)
	
	// Vehicle type and features
	GetByVehicleType(ctx context.Context, vehicleTypeID primitive.ObjectID, params *utils.PaginationParams) ([]*models.Vehicle, int64, error)
	GetAccessibleVehicles(ctx context.Context, params *utils.PaginationParams) ([]*models.Vehicle, int64, error)
	GetByFeatures(ctx context.Context, features []string, params *utils.PaginationParams) ([]*models.Vehicle, int64, error)
	
	// Maintenance
	UpdateMaintenanceDate(ctx context.Context, id primitive.ObjectID, lastMaintenance, nextMaintenance *time.Time) error
	GetVehiclesDueForMaintenance(ctx context.Context) ([]*models.Vehicle, error)
	
	// Statistics
	UpdateStats(ctx context.Context, id primitive.ObjectID, totalRides int64, totalDistance float64) error
	GetVehicleStats(ctx context.Context, vehicleID primitive.ObjectID, days int) (map[string]interface{}, error)
	
	// Search and listing
	List(ctx context.Context, params *utils.PaginationParams) ([]*models.Vehicle, int64, error)
	SearchByMakeModel(ctx context.Context, make, model string, params *utils.PaginationParams) ([]*models.Vehicle, int64, error)
	
	// Analytics
	GetTotalCount(ctx context.Context) (int64, error)
	GetCountByStatus(ctx context.Context, status models.VehicleStatus) (int64, error)
	GetCountByType(ctx context.Context, vehicleTypeID primitive.ObjectID) (int64, error)
}