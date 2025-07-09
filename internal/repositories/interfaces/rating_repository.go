package interfaces

import (
	"context"

	"github.com/uber-clone/internal/models"
	"github.com/uber-clone/internal/utils"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

type RatingRepository interface {
	// Basic CRUD operations
	Create(ctx context.Context, rating *models.Rating) error
	GetByID(ctx context.Context, id primitive.ObjectID) (*models.Rating, error)
	Update(ctx context.Context, id primitive.ObjectID, updates map[string]interface{}) error
	Delete(ctx context.Context, id primitive.ObjectID) error
	
	// Ride ratings
	GetByRideID(ctx context.Context, rideID primitive.ObjectID) ([]*models.Rating, error)
	GetRideRating(ctx context.Context, rideID primitive.ObjectID, raterType models.UserType) (*models.Rating, error)
	
	// User ratings
	GetByRaterID(ctx context.Context, raterID primitive.ObjectID, params *utils.PaginationParams) ([]*models.Rating, int64, error)
	GetByRatedID(ctx context.Context, ratedID primitive.ObjectID, params *utils.PaginationParams) ([]*models.Rating, int64, error)
	
	// Rating statistics
	GetAverageRating(ctx context.Context, ratedID primitive.ObjectID) (float64, error)
	GetRatingDistribution(ctx context.Context, ratedID primitive.ObjectID) (map[int]int64, error)
	GetRatingCount(ctx context.Context, ratedID primitive.ObjectID) (int64, error)
	
	// Rating analysis
	GetRatingsByRange(ctx context.Context, minRating, maxRating float64, params *utils.PaginationParams) ([]*models.Rating, int64, error)
	GetRecentRatings(ctx context.Context, ratedID primitive.ObjectID, days int) ([]*models.Rating, error)
	GetRatingTrends(ctx context.Context, ratedID primitive.ObjectID, days int) ([]map[string]interface{}, error)
	
	// Tag analysis
	GetRatingsByTag(ctx context.Context, tag string, params *utils.PaginationParams) ([]*models.Rating, int64, error)
	GetPopularTags(ctx context.Context, ratedID primitive.ObjectID, limit int) ([]map[string]interface{}, error)
	
	// Search and filtering
	SearchRatings(ctx context.Context, query string, params *utils.PaginationParams) ([]*models.Rating, int64, error)
	GetRatingsByType(ctx context.Context, raterType models.UserType, params *utils.PaginationParams) ([]*models.Rating, int64, error)
	
	// Analytics
	GetRatingStats(ctx context.Context, days int) (map[string]interface{}, error)
	GetLowRatedUsers(ctx context.Context, threshold float64, userType models.UserType) ([]*models.Rating, error)
}
