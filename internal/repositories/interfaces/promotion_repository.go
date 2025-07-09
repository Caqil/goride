package interfaces

import (
	"context"
	"time"

	"goride/internal/models"
	"goride/internal/utils"

	"go.mongodb.org/mongo-driver/bson/primitive"
)

type PromotionRepository interface {
	// Basic CRUD operations
	Create(ctx context.Context, promotion *models.Promotion) error
	GetByID(ctx context.Context, id primitive.ObjectID) (*models.Promotion, error)
	Update(ctx context.Context, id primitive.ObjectID, updates map[string]interface{}) error
	Delete(ctx context.Context, id primitive.ObjectID) error

	// Code operations
	GetByCode(ctx context.Context, code string) (*models.Promotion, error)
	ValidateCode(ctx context.Context, code string, userID primitive.ObjectID, rideType string) (*models.Promotion, error)

	// Status operations
	GetActivePromotions(ctx context.Context, params *utils.PaginationParams) ([]*models.Promotion, int64, error)
	GetByStatus(ctx context.Context, status models.PromotionStatus, params *utils.PaginationParams) ([]*models.Promotion, int64, error)
	UpdateStatus(ctx context.Context, id primitive.ObjectID, status models.PromotionStatus) error

	// Usage tracking
	IncrementUsage(ctx context.Context, id primitive.ObjectID) error
	GetUsageStats(ctx context.Context, id primitive.ObjectID) (map[string]interface{}, error)

	// Type and applicability
	GetByType(ctx context.Context, promotionType models.PromotionType, params *utils.PaginationParams) ([]*models.Promotion, int64, error)
	GetApplicablePromotions(ctx context.Context, userType models.UserType, rideType string, amount float64) ([]*models.Promotion, error)

	// Time-based queries
	GetValidPromotions(ctx context.Context, checkTime time.Time) ([]*models.Promotion, error)
	GetExpiredPromotions(ctx context.Context) ([]*models.Promotion, error)
	GetPromotionsByDateRange(ctx context.Context, startDate, endDate time.Time, params *utils.PaginationParams) ([]*models.Promotion, int64, error)

	// Search and filtering
	SearchPromotions(ctx context.Context, query string, params *utils.PaginationParams) ([]*models.Promotion, int64, error)
	GetPromotionsForCity(ctx context.Context, city string, params *utils.PaginationParams) ([]*models.Promotion, int64, error)

	// Analytics
	GetPromotionStats(ctx context.Context, days int) (map[string]interface{}, error)
	GetTopPromotions(ctx context.Context, limit int, days int) ([]*models.Promotion, error)
	GetPromotionEffectiveness(ctx context.Context, id primitive.ObjectID, days int) (map[string]interface{}, error)
}
