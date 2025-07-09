package interfaces

import (
	"context"
	"time"

	"github.com/uber-clone/internal/models"
	"github.com/uber-clone/internal/utils"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

type EmergencyRepository interface {
	// Basic CRUD operations
	Create(ctx context.Context, emergency *models.Emergency) error
	GetByID(ctx context.Context, id primitive.ObjectID) (*models.Emergency, error)
	Update(ctx context.Context, id primitive.ObjectID, updates map[string]interface{}) error
	Delete(ctx context.Context, id primitive.ObjectID) error
	
	// User emergencies
	GetByUserID(ctx context.Context, userID primitive.ObjectID, params *utils.PaginationParams) ([]*models.Emergency, int64, error)
	GetActiveEmergenciesByUser(ctx context.Context, userID primitive.ObjectID) ([]*models.Emergency, error)
	
	// Ride emergencies
	GetByRideID(ctx context.Context, rideID primitive.ObjectID) ([]*models.Emergency, error)
	GetActiveEmergencyForRide(ctx context.Context, rideID primitive.ObjectID) (*models.Emergency, error)
	
	// Status operations
	UpdateStatus(ctx context.Context, id primitive.ObjectID, status models.EmergencyStatus) error
	ResolveEmergency(ctx context.Context, id primitive.ObjectID, resolvedBy primitive.ObjectID, notes string) error
	GetByStatus(ctx context.Context, status models.EmergencyStatus, params *utils.PaginationParams) ([]*models.Emergency, int64, error)
	
	// Type filtering
	GetByType(ctx context.Context, emergencyType models.EmergencyType, params *utils.PaginationParams) ([]*models.Emergency, int64, error)
	GetActiveEmergencies(ctx context.Context) ([]*models.Emergency, error)
	
	// Location-based queries
	GetEmergenciesInArea(ctx context.Context, bounds *utils.Bounds) ([]*models.Emergency, error)
	GetNearbyEmergencies(ctx context.Context, lat, lng, radiusKM float64) ([]*models.Emergency, error)
	
	// Time-based queries
	GetEmergenciesByDateRange(ctx context.Context, startDate, endDate time.Time, params *utils.PaginationParams) ([]*models.Emergency, int64, error)
	GetRecentEmergencies(ctx context.Context, hours int) ([]*models.Emergency, error)
	GetUnresolvedEmergencies(ctx context.Context, olderThan time.Duration) ([]*models.Emergency, error)
	
	// Response tracking
	UpdateResponseTime(ctx context.Context, id primitive.ObjectID, responseTime time.Time) error
	GetAverageResponseTime(ctx context.Context, days int) (time.Duration, error)
	
	// Analytics
	GetEmergencyStats(ctx context.Context, days int) (map[string]interface{}, error)
	GetEmergencyTrends(ctx context.Context, days int) ([]map[string]interface{}, error)
	GetResponseTimeStats(ctx context.Context, days int) (map[string]interface{}, error)
}
