package interfaces

import (
	"context"

	"goride/internal/models"
	"goride/internal/utils"

	"go.mongodb.org/mongo-driver/bson/primitive"
)

type BackgroundCheckRepository interface {
	// CRUD operations
	Create(ctx context.Context, backgroundCheck *models.BackgroundCheck) error
	GetByID(ctx context.Context, id primitive.ObjectID) (*models.BackgroundCheck, error)
	Update(ctx context.Context, id primitive.ObjectID, updates map[string]interface{}) error
	Delete(ctx context.Context, id primitive.ObjectID) error

	// Query operations
	GetByDriverID(ctx context.Context, driverID primitive.ObjectID, params *utils.PaginationParams) ([]*models.BackgroundCheck, int64, error)
	GetByStatus(ctx context.Context, status models.BackgroundCheckStatus, params *utils.PaginationParams) ([]*models.BackgroundCheck, int64, error)
	GetByCheckType(ctx context.Context, checkType models.BackgroundCheckType, params *utils.PaginationParams) ([]*models.BackgroundCheck, int64, error)
	GetExpiredChecks(ctx context.Context, params *utils.PaginationParams) ([]*models.BackgroundCheck, int64, error)
	GetPendingReviews(ctx context.Context, reviewerID *primitive.ObjectID, params *utils.PaginationParams) ([]*models.BackgroundCheck, int64, error)

	// Statistics
	GetStatsByDateRange(ctx context.Context, startDate, endDate string) (*models.BackgroundCheckStats, error)
	GetComplianceStats(ctx context.Context, region string) (map[string]interface{}, error)

	// Bulk operations
	BulkUpdateStatus(ctx context.Context, ids []primitive.ObjectID, status models.BackgroundCheckStatus) error
	GetChecksForPeriodicRecheck(ctx context.Context) ([]*models.BackgroundCheck, error)
}
