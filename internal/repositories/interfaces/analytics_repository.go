package interfaces

import (
	"context"
	"time"

	"github.com/uber-clone/internal/models"
	"github.com/uber-clone/internal/utils"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

type AnalyticsRepository interface {
	// Event tracking
	CreateEvent(ctx context.Context, event *models.AnalyticsEvent) error
	GetEventsByType(ctx context.Context, eventType string, params *utils.PaginationParams) ([]*models.AnalyticsEvent, int64, error)
	GetEventsByUser(ctx context.Context, userID primitive.ObjectID, params *utils.PaginationParams) ([]*models.AnalyticsEvent, int64, error)
	GetEventsByDateRange(ctx context.Context, startDate, endDate time.Time, eventType string) ([]*models.AnalyticsEvent, error)
	
	// Ride analytics
	CreateRideAnalytics(ctx context.Context, analytics *models.RideAnalytics) error
	GetRideAnalyticsByDate(ctx context.Context, date time.Time, city string) (*models.RideAnalytics, error)
	GetRideAnalyticsByDateRange(ctx context.Context, startDate, endDate time.Time, city string) ([]*models.RideAnalytics, error)
	UpdateRideAnalytics(ctx context.Context, date time.Time, city string, updates map[string]interface{}) error
	
	// Dashboard metrics
	GetDashboardMetrics(ctx context.Context, days int) (map[string]interface{}, error)
	GetRealtimeMetrics(ctx context.Context) (map[string]interface{}, error)
	
	// User analytics
	GetUserRegistrationTrends(ctx context.Context, days int) ([]map[string]interface{}, error)
	GetUserActivityTrends(ctx context.Context, days int) ([]map[string]interface{}, error)
	GetUserRetentionStats(ctx context.Context, cohortDate time.Time) (map[string]interface{}, error)
	
	// Ride analytics
	GetRideVolumeTrends(ctx context.Context, days int, city string) ([]map[string]interface{}, error)
	GetRideCompletionRates(ctx context.Context, days int) (map[string]interface{}, error)
	GetAverageRideMetrics(ctx context.Context, days int) (map[string]interface{}, error)
	GetPopularPickupAreas(ctx context.Context, city string, days int, limit int) ([]map[string]interface{}, error)
	GetPopularDropoffAreas(ctx context.Context, city string, days int, limit int) ([]map[string]interface{}, error)
	
	// Revenue analytics
	GetRevenueStats(ctx context.Context, days int) (map[string]interface{}, error)
	GetRevenueTrends(ctx context.Context, days int) ([]map[string]interface{}, error)
	GetRevenueByCity(ctx context.Context, days int) ([]map[string]interface{}, error)
	GetRevenueByRideType(ctx context.Context, days int) ([]map[string]interface{}, error)
	
	// Driver analytics
	GetDriverMetrics(ctx context.Context, days int) (map[string]interface{}, error)
	GetDriverUtilization(ctx context.Context, days int) (map[string]interface{}, error)
	GetDriverEarningsStats(ctx context.Context, days int) (map[string]interface{}, error)
	GetTopDrivers(ctx context.Context, metric string, days int, limit int) ([]map[string]interface{}, error)
	
	// Performance analytics
	GetSystemPerformanceMetrics(ctx context.Context, hours int) ([]map[string]interface{}, error)
	GetAPIResponseTimes(ctx context.Context, hours int) ([]map[string]interface{}, error)
	GetErrorRates(ctx context.Context, hours int) ([]map[string]interface{}, error)
	
	// Custom reports
	GenerateCustomReport(ctx context.Context, reportType string, parameters map[string]interface{}, dateRange []time.Time) ([]map[string]interface{}, error)
	GetReportData(ctx context.Context, reportID string) (map[string]interface{}, error)
}