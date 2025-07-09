package interfaces

import (
	"context"

	"goride/internal/models"
	"goride/internal/utils"

	"go.mongodb.org/mongo-driver/bson/primitive"
)

type UserRepository interface {
	// Basic CRUD operations
	Create(ctx context.Context, user *models.User) error
	GetByID(ctx context.Context, id primitive.ObjectID) (*models.User, error)
	Update(ctx context.Context, id primitive.ObjectID, updates map[string]interface{}) error
	Delete(ctx context.Context, id primitive.ObjectID) error

	// Authentication operations
	GetByEmail(ctx context.Context, email string) (*models.User, error)
	GetByPhone(ctx context.Context, phone string) (*models.User, error)
	GetBySocialID(ctx context.Context, provider string, socialID string) (*models.User, error)

	// Verification operations
	UpdateEmailVerification(ctx context.Context, id primitive.ObjectID, verified bool) error
	UpdatePhoneVerification(ctx context.Context, id primitive.ObjectID, verified bool) error
	UpdateLastLogin(ctx context.Context, id primitive.ObjectID) error
	UpdateLastActive(ctx context.Context, id primitive.ObjectID) error

	// Search and listing
	List(ctx context.Context, params *utils.PaginationParams) ([]*models.User, int64, error)
	SearchByName(ctx context.Context, name string, params *utils.PaginationParams) ([]*models.User, int64, error)
	GetByStatus(ctx context.Context, status models.UserStatus, params *utils.PaginationParams) ([]*models.User, int64, error)
	GetByType(ctx context.Context, userType models.UserType, params *utils.PaginationParams) ([]*models.User, int64, error)

	// Statistics
	GetTotalCount(ctx context.Context) (int64, error)
	GetCountByStatus(ctx context.Context, status models.UserStatus) (int64, error)
	GetCountByType(ctx context.Context, userType models.UserType) (int64, error)
	GetRegistrationStats(ctx context.Context, days int) (map[string]int64, error)
}
