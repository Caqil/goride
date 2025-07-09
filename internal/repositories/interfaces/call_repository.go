package interfaces

import (
	"context"
	"time"

	"goride/internal/models"
	"goride/internal/utils"

	"go.mongodb.org/mongo-driver/bson/primitive"
)

type CallRepository interface {
	// Call CRUD operations
	CreateCall(ctx context.Context, call *models.Call) error
	GetCallByID(ctx context.Context, id primitive.ObjectID) (*models.Call, error)
	GetCallBySID(ctx context.Context, callSID string) (*models.Call, error)
	UpdateCall(ctx context.Context, id primitive.ObjectID, updates map[string]interface{}) error
	DeleteCall(ctx context.Context, id primitive.ObjectID) error

	// Call queries
	GetCallsByUser(ctx context.Context, userID primitive.ObjectID, params *utils.PaginationParams) ([]*models.Call, int64, error)
	GetCallsByRide(ctx context.Context, rideID primitive.ObjectID) ([]*models.Call, error)
	GetCallsByStatus(ctx context.Context, status models.CallStatus, params *utils.PaginationParams) ([]*models.Call, int64, error)
	GetCallsByType(ctx context.Context, callType models.CallType, params *utils.PaginationParams) ([]*models.Call, int64, error)
	GetCallsByDateRange(ctx context.Context, startDate, endDate time.Time, params *utils.PaginationParams) ([]*models.Call, int64, error)

	// Emergency calls
	GetEmergencyCalls(ctx context.Context, params *utils.PaginationParams) ([]*models.Call, int64, error)
	GetUserEmergencyCalls(ctx context.Context, userID primitive.ObjectID) ([]*models.Call, error)

	// Active calls
	GetActiveCalls(ctx context.Context) ([]*models.Call, error)
	GetActiveCallsByUser(ctx context.Context, userID primitive.ObjectID) ([]*models.Call, error)
	GetActiveCallsByRide(ctx context.Context, rideID primitive.ObjectID) ([]*models.Call, error)

	// Call analytics
	GetCallStats(ctx context.Context, startDate, endDate time.Time) (*models.CallAnalytics, error)
	GetUserCallStats(ctx context.Context, userID primitive.ObjectID, days int) (map[string]interface{}, error)
	GetCallDurationStats(ctx context.Context, days int) (map[string]interface{}, error)
	GetCallQualityStats(ctx context.Context, days int) (map[string]interface{}, error)

	// Search and filtering
	SearchCalls(ctx context.Context, query string, params *utils.PaginationParams) ([]*models.Call, int64, error)
	GetFailedCalls(ctx context.Context, days int) ([]*models.Call, error)
	GetHighCostCalls(ctx context.Context, threshold float64, days int) ([]*models.Call, error)

	// Proxy number management
	GetProxyNumberUsage(ctx context.Context, proxyNumber string) ([]*models.Call, error)
	GetAvailableProxyNumbers(ctx context.Context) ([]string, error)
	AssignProxyNumber(ctx context.Context, rideID primitive.ObjectID, proxyNumber string) error
	ReleaseProxyNumber(ctx context.Context, proxyNumber string) error

	// Call recordings
	UpdateCallRecording(ctx context.Context, callID primitive.ObjectID, recordingURL string) error
	GetCallsWithRecordings(ctx context.Context, params *utils.PaginationParams) ([]*models.Call, int64, error)
	DeleteCallRecording(ctx context.Context, callID primitive.ObjectID) error
}
