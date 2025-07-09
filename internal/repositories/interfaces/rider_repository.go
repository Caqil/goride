package interfaces

import (
	"context"

	"goride/internal/models"
	"goride/internal/utils"

	"go.mongodb.org/mongo-driver/bson/primitive"
)

type RiderRepository interface {
	// Basic CRUD operations
	Create(ctx context.Context, rider *models.Rider) error
	GetByID(ctx context.Context, id primitive.ObjectID) (*models.Rider, error)
	GetByUserID(ctx context.Context, userID primitive.ObjectID) (*models.Rider, error)
	Update(ctx context.Context, id primitive.ObjectID, updates map[string]interface{}) error
	Delete(ctx context.Context, id primitive.ObjectID) error

	// Payment methods
	AddPaymentMethod(ctx context.Context, id primitive.ObjectID, paymentMethodID primitive.ObjectID) error
	RemovePaymentMethod(ctx context.Context, id primitive.ObjectID, paymentMethodID primitive.ObjectID) error
	SetDefaultPaymentMethod(ctx context.Context, id primitive.ObjectID, paymentMethodID primitive.ObjectID) error

	// Favorite locations
	AddFavoriteLocation(ctx context.Context, id primitive.ObjectID, location *models.FavoriteLocation) error
	RemoveFavoriteLocation(ctx context.Context, id primitive.ObjectID, locationName string) error
	UpdateFavoriteLocation(ctx context.Context, id primitive.ObjectID, locationName string, updates map[string]interface{}) error

	// Ride statistics
	UpdateRating(ctx context.Context, id primitive.ObjectID, rating float64, totalRatings int64) error
	UpdateRideStats(ctx context.Context, id primitive.ObjectID, totalRides int64, totalSpent float64) error
	UpdateLoyaltyPoints(ctx context.Context, id primitive.ObjectID, points int64) error

	// Referral system
	GetByReferralCode(ctx context.Context, referralCode string) (*models.Rider, error)
	UpdateReferralStats(ctx context.Context, id primitive.ObjectID, referredBy primitive.ObjectID) error

	// Search and listing
	List(ctx context.Context, params *utils.PaginationParams) ([]*models.Rider, int64, error)
	GetByRatingRange(ctx context.Context, minRating, maxRating float64, params *utils.PaginationParams) ([]*models.Rider, int64, error)
	GetHighValueRiders(ctx context.Context, minSpent float64, params *utils.PaginationParams) ([]*models.Rider, int64, error)

	// Analytics
	GetTotalCount(ctx context.Context) (int64, error)
	GetAverageRating(ctx context.Context) (float64, error)
	GetTopRiders(ctx context.Context, limit int) ([]*models.Rider, error)
	GetRiderStats(ctx context.Context, riderID primitive.ObjectID, days int) (map[string]interface{}, error)
}
