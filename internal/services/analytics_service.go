package services

import (
	"context"
	"fmt"
	"time"

	"goride/internal/models"
	"goride/internal/repositories/interfaces"
	"goride/internal/utils"

	"go.mongodb.org/mongo-driver/bson/primitive"
)

type AnalyticsService interface {
	// Event Tracking
	TrackEvent(ctx context.Context, event *models.AnalyticsEvent) error
	TrackUserEvent(ctx context.Context, userID primitive.ObjectID, eventType, eventName string, properties map[string]interface{}) error
	GetEventsByType(ctx context.Context, eventType string, params *utils.PaginationParams) ([]*models.AnalyticsEvent, int64, error)
	GetUserEvents(ctx context.Context, userID primitive.ObjectID, params *utils.PaginationParams) ([]*models.AnalyticsEvent, int64, error)

	// Ride Analytics
	RecordRideAnalytics(ctx context.Context, ride *models.Ride) error
	GetRideAnalytics(ctx context.Context, date time.Time, city string) (*models.RideAnalytics, error)
	GetRideAnalyticsByDateRange(ctx context.Context, startDate, endDate time.Time, city string) ([]*models.RideAnalytics, error)
	CalculateDailyRideStats(ctx context.Context, date time.Time, city string) error

	// Dashboard Metrics
	GetDashboardMetrics(ctx context.Context, days int) (map[string]interface{}, error)
	GetRealtimeMetrics(ctx context.Context) (map[string]interface{}, error)
	GetKPIs(ctx context.Context, days int) (map[string]interface{}, error)

	// User Analytics
	GetUserRegistrationTrends(ctx context.Context, days int) ([]map[string]interface{}, error)
	GetUserActivityTrends(ctx context.Context, days int) ([]map[string]interface{}, error)
	GetUserRetentionStats(ctx context.Context, cohortDate time.Time) (map[string]interface{}, error)
	GetUserEngagementMetrics(ctx context.Context, days int) (map[string]interface{}, error)

	// Ride Analytics
	GetRideVolumeTrends(ctx context.Context, days int, city string) ([]map[string]interface{}, error)
	GetRideCompletionRates(ctx context.Context, days int) (map[string]interface{}, error)
	GetAverageRideMetrics(ctx context.Context, days int) (map[string]interface{}, error)
	GetPopularPickupAreas(ctx context.Context, city string, days int, limit int) ([]map[string]interface{}, error)
	GetPopularDropoffAreas(ctx context.Context, city string, days int, limit int) ([]map[string]interface{}, error)

	// Revenue Analytics
	GetRevenueStats(ctx context.Context, days int) (map[string]interface{}, error)
	GetRevenueTrends(ctx context.Context, days int) ([]map[string]interface{}, error)
	GetDriverEarnings(ctx context.Context, driverID primitive.ObjectID, days int) (map[string]interface{}, error)

	// Performance Analytics
	GetDriverPerformanceMetrics(ctx context.Context, driverID primitive.ObjectID, days int) (map[string]interface{}, error)
	GetSystemPerformanceMetrics(ctx context.Context, days int) (map[string]interface{}, error)
	GetPeakTimeAnalysis(ctx context.Context, city string, days int) (map[string]interface{}, error)

	// Custom Reports
	GenerateCustomReport(ctx context.Context, reportType string, params map[string]interface{}) (map[string]interface{}, error)
	ExportAnalytics(ctx context.Context, startDate, endDate time.Time, format string) ([]byte, error)
}

type analyticsService struct {
	analyticsRepo interfaces.AnalyticsRepository
	rideRepo      interfaces.RideRepository
	userRepo      interfaces.UserRepository
	driverRepo    interfaces.DriverRepository
	paymentRepo   interfaces.PaymentRepository
	cache         CacheService
}

func NewAnalyticsService(
	analyticsRepo interfaces.AnalyticsRepository,
	rideRepo interfaces.RideRepository,
	userRepo interfaces.UserRepository,
	driverRepo interfaces.DriverRepository,
	paymentRepo interfaces.PaymentRepository,
	cache CacheService,
) AnalyticsService {
	return &analyticsService{
		analyticsRepo: analyticsRepo,
		rideRepo:      rideRepo,
		userRepo:      userRepo,
		driverRepo:    driverRepo,
		paymentRepo:   paymentRepo,
		cache:         cache,
	}
}

// Event Tracking
func (s *analyticsService) TrackEvent(ctx context.Context, event *models.AnalyticsEvent) error {
	event.ID = primitive.NewObjectID()
	event.CreatedAt = time.Now()

	// Validate event
	if event.EventType == "" || event.EventName == "" {
		return fmt.Errorf("event type and name are required")
	}

	// Store event
	if err := s.analyticsRepo.CreateEvent(ctx, event); err != nil {
		return fmt.Errorf("failed to track event: %w", err)
	}

	// Update real-time metrics in cache
	s.updateRealtimeMetrics(ctx, event)

	return nil
}

func (s *analyticsService) TrackUserEvent(ctx context.Context, userID primitive.ObjectID, eventType, eventName string, properties map[string]interface{}) error {
	event := &models.AnalyticsEvent{
		UserID:     &userID,
		EventType:  eventType,
		EventName:  eventName,
		Properties: properties,
		CreatedAt:  time.Now(),
	}

	return s.TrackEvent(ctx, event)
}

func (s *analyticsService) GetEventsByType(ctx context.Context, eventType string, params *utils.PaginationParams) ([]*models.AnalyticsEvent, int64, error) {
	return s.analyticsRepo.GetEventsByType(ctx, eventType, params)
}

func (s *analyticsService) GetUserEvents(ctx context.Context, userID primitive.ObjectID, params *utils.PaginationParams) ([]*models.AnalyticsEvent, int64, error) {
	return s.analyticsRepo.GetEventsByUser(ctx, userID, params)
}

// Ride Analytics
func (s *analyticsService) RecordRideAnalytics(ctx context.Context, ride *models.Ride) error {
	// Track ride completion event
	eventProperties := map[string]interface{}{
		"ride_id":          ride.ID,
		"ride_type":        ride.RideType,
		"duration":         ride.ActualDuration,
		"distance":         ride.ActualDistance,
		"fare":             ride.ActualFare,
		"surge_multiplier": ride.SurgeMultiplier,
	}

	if err := s.TrackUserEvent(ctx, ride.RiderID, "ride", "ride_completed", eventProperties); err != nil {
		return err
	}

	// Update daily ride analytics
	date := time.Date(ride.CompletedAt.Year(), ride.CompletedAt.Month(), ride.CompletedAt.Day(), 0, 0, 0, 0, ride.CompletedAt.Location())
	city := ride.PickupLocation.City

	return s.CalculateDailyRideStats(ctx, date, city)
}

func (s *analyticsService) GetRideAnalytics(ctx context.Context, date time.Time, city string) (*models.RideAnalytics, error) {
	return s.analyticsRepo.GetRideAnalyticsByDate(ctx, date, city)
}

func (s *analyticsService) GetRideAnalyticsByDateRange(ctx context.Context, startDate, endDate time.Time, city string) ([]*models.RideAnalytics, error) {
	return s.analyticsRepo.GetRideAnalyticsByDateRange(ctx, startDate, endDate, city)
}

func (s *analyticsService) CalculateDailyRideStats(ctx context.Context, date time.Time, city string) error {
	// Get rides for the day
	startOfDay := time.Date(date.Year(), date.Month(), date.Day(), 0, 0, 0, 0, date.Location())
	endOfDay := startOfDay.Add(24 * time.Hour)

	rides, _, err := s.rideRepo.GetRidesByDateRange(ctx, startOfDay, endOfDay, nil)
	if err != nil {
		return err
	}

	// Filter by city if specified
	var cityRides []*models.Ride
	for _, ride := range rides {
		if city == "" || ride.PickupLocation.City == city {
			cityRides = append(cityRides, ride)
		}
	}

	// Calculate statistics
	stats := s.calculateRideStatistics(cityRides)

	// Create or update ride analytics
	analytics := &models.RideAnalytics{
		Date:                date,
		City:                city,
		TotalRides:          stats["total_rides"].(int64),
		CompletedRides:      stats["completed_rides"].(int64),
		CancelledRides:      stats["cancelled_rides"].(int64),
		TotalRevenue:        stats["total_revenue"].(float64),
		AverageRideValue:    stats["average_ride_value"].(float64),
		AverageRideTime:     stats["average_ride_time"].(float64),
		AverageRideDistance: stats["average_ride_distance"].(float64),
		PeakHours:           stats["peak_hours"].([]string),
		PopularRoutes:       stats["popular_routes"].([]models.RouteStats),
	}

	// Check if analytics already exist for this date/city
	existing, err := s.analyticsRepo.GetRideAnalyticsByDate(ctx, date, city)
	if err == nil && existing != nil {
		// Update existing
		updates := map[string]interface{}{
			"total_rides":           analytics.TotalRides,
			"completed_rides":       analytics.CompletedRides,
			"cancelled_rides":       analytics.CancelledRides,
			"total_revenue":         analytics.TotalRevenue,
			"average_ride_value":    analytics.AverageRideValue,
			"average_ride_time":     analytics.AverageRideTime,
			"average_ride_distance": analytics.AverageRideDistance,
			"peak_hours":            analytics.PeakHours,
			"popular_routes":        analytics.PopularRoutes,
			"updated_at":            time.Now(),
		}
		return s.analyticsRepo.UpdateRideAnalytics(ctx, date, city, updates)
	}

	// Create new analytics
	return s.analyticsRepo.CreateRideAnalytics(ctx, analytics)
}

// Dashboard Metrics
func (s *analyticsService) GetDashboardMetrics(ctx context.Context, days int) (map[string]interface{}, error) {
	// Try cache first
	cacheKey := fmt.Sprintf("dashboard_metrics_%d", days)
	if s.cache != nil {
		var metrics map[string]interface{}
		if err := s.cache.Get(ctx, cacheKey, &metrics); err == nil {
			return metrics, nil
		}
	}

	metrics, err := s.analyticsRepo.GetDashboardMetrics(ctx, days)
	if err != nil {
		return nil, err
	}

	// Add calculated metrics
	kpis, err := s.GetKPIs(ctx, days)
	if err == nil {
		for k, v := range kpis {
			metrics[k] = v
		}
	}

	// Cache for 5 minutes
	if s.cache != nil {
		s.cache.Set(ctx, cacheKey, metrics, 5*time.Minute)
	}

	return metrics, nil
}

func (s *analyticsService) GetRealtimeMetrics(ctx context.Context) (map[string]interface{}, error) {
	return s.analyticsRepo.GetRealtimeMetrics(ctx)
}

func (s *analyticsService) GetKPIs(ctx context.Context, days int) (map[string]interface{}, error) {
	time.Now().AddDate(0, 0, -days)
	time.Now()

	kpis := make(map[string]interface{})

	// User KPIs
	userStats, err := s.userRepo.GetRegistrationStats(ctx, days)
	if err == nil {
		kpis["new_users"] = userStats["total"]
		kpis["user_growth_rate"] = s.calculateGrowthRate(userStats)
	}

	// Ride KPIs
	completionRates, err := s.GetRideCompletionRates(ctx, days)
	if err == nil {
		kpis["ride_completion_rate"] = completionRates["completion_rate"]
		kpis["cancellation_rate"] = completionRates["cancellation_rate"]
	}

	// Revenue KPIs
	revenueStats, err := s.GetRevenueStats(ctx, days)
	if err == nil {
		kpis["total_revenue"] = revenueStats["total_revenue"]
		kpis["average_order_value"] = revenueStats["average_order_value"]
		kpis["revenue_growth_rate"] = revenueStats["growth_rate"]
	}

	return kpis, nil
}

// User Analytics
func (s *analyticsService) GetUserRegistrationTrends(ctx context.Context, days int) ([]map[string]interface{}, error) {
	return s.analyticsRepo.GetUserRegistrationTrends(ctx, days)
}

func (s *analyticsService) GetUserActivityTrends(ctx context.Context, days int) ([]map[string]interface{}, error) {
	return s.analyticsRepo.GetUserActivityTrends(ctx, days)
}

func (s *analyticsService) GetUserRetentionStats(ctx context.Context, cohortDate time.Time) (map[string]interface{}, error) {
	return s.analyticsRepo.GetUserRetentionStats(ctx, cohortDate)
}

func (s *analyticsService) GetUserEngagementMetrics(ctx context.Context, days int) (map[string]interface{}, error) {
	startDate := time.Now().AddDate(0, 0, -days)
	endDate := time.Now()

	// Get user activity events
	events, err := s.analyticsRepo.GetEventsByDateRange(ctx, startDate, endDate, "user_activity")
	if err != nil {
		return nil, err
	}

	// Calculate engagement metrics
	metrics := map[string]interface{}{
		"total_sessions":           len(events),
		"unique_users":             s.countUniqueUsers(events),
		"average_session_duration": s.calculateAverageSessionDuration(events),
		"bounce_rate":              s.calculateBounceRate(events),
		"retention_rate":           s.calculateRetentionRate(events),
	}

	return metrics, nil
}

// Ride Analytics
func (s *analyticsService) GetRideVolumeTrends(ctx context.Context, days int, city string) ([]map[string]interface{}, error) {
	return s.analyticsRepo.GetRideVolumeTrends(ctx, days, city)
}

func (s *analyticsService) GetRideCompletionRates(ctx context.Context, days int) (map[string]interface{}, error) {
	return s.analyticsRepo.GetRideCompletionRates(ctx, days)
}

func (s *analyticsService) GetAverageRideMetrics(ctx context.Context, days int) (map[string]interface{}, error) {
	return s.analyticsRepo.GetAverageRideMetrics(ctx, days)
}

func (s *analyticsService) GetPopularPickupAreas(ctx context.Context, city string, days int, limit int) ([]map[string]interface{}, error) {
	return s.analyticsRepo.GetPopularPickupAreas(ctx, city, days, limit)
}

func (s *analyticsService) GetPopularDropoffAreas(ctx context.Context, city string, days int, limit int) ([]map[string]interface{}, error) {
	return s.analyticsRepo.GetPopularDropoffAreas(ctx, city, days, limit)
}

// Revenue Analytics
func (s *analyticsService) GetRevenueStats(ctx context.Context, days int) (map[string]interface{}, error) {
	return s.analyticsRepo.GetRevenueStats(ctx, days)
}

func (s *analyticsService) GetRevenueTrends(ctx context.Context, days int) ([]map[string]interface{}, error) {
	return s.analyticsRepo.GetRevenueTrends(ctx, days)
}

func (s *analyticsService) GetDriverEarnings(ctx context.Context, driverID primitive.ObjectID, days int) (map[string]interface{}, error) {
	startDate := time.Now().AddDate(0, 0, -days)
	endDate := time.Now()

	return s.paymentRepo.GetDriverEarnings(ctx, driverID, startDate, endDate)
}

// Performance Analytics
func (s *analyticsService) GetDriverPerformanceMetrics(ctx context.Context, driverID primitive.ObjectID, days int) (map[string]interface{}, error) {
	driver, err := s.driverRepo.GetByID(ctx, driverID)
	if err != nil {
		return nil, err
	}

	metrics := map[string]interface{}{
		"rating":            driver.Rating,
		"total_rides":       driver.TotalRides,
		"acceptance_rate":   driver.AcceptanceRate,
		"cancellation_rate": driver.CancellationRate,
		"completion_rate":   driver.CompletionRate,
		"online_hours":      driver.OnlineHours,
	}

	// Get earnings for the period
	earnings, err := s.GetDriverEarnings(ctx, driverID, days)
	if err == nil {
		metrics["earnings"] = earnings
	}

	return metrics, nil
}

func (s *analyticsService) GetSystemPerformanceMetrics(ctx context.Context, days int) (map[string]interface{}, error) {
	metrics := map[string]interface{}{
		"average_response_time": 0.0,
		"uptime_percentage":     99.9,
		"error_rate":            0.1,
		"throughput":            0,
	}

	// This would calculate real system performance metrics
	// from logs and monitoring data
	return metrics, nil
}

func (s *analyticsService) GetPeakTimeAnalysis(ctx context.Context, city string, days int) (map[string]interface{}, error) {
	startDate := time.Now().AddDate(0, 0, -days)
	endDate := time.Now()

	analytics, err := s.analyticsRepo.GetRideAnalyticsByDateRange(ctx, startDate, endDate, city)
	if err != nil {
		return nil, err
	}

	// Analyze peak hours across all days
	hourCounts := make(map[int]int)
	for _, dayAnalytics := range analytics {
		for _, hour := range dayAnalytics.PeakHours {
			if h, err := time.Parse("15", hour); err == nil {
				hourCounts[h.Hour()]++
			}
		}
	}

	// Find most common peak hours
	var peakHours []int
	maxCount := 0
	for hour, count := range hourCounts {
		if count > maxCount {
			maxCount = count
			peakHours = []int{hour}
		} else if count == maxCount {
			peakHours = append(peakHours, hour)
		}
	}

	return map[string]interface{}{
		"peak_hours":        peakHours,
		"peak_frequency":    maxCount,
		"hour_distribution": hourCounts,
	}, nil
}

// Custom Reports
func (s *analyticsService) GenerateCustomReport(ctx context.Context, reportType string, params map[string]interface{}) (map[string]interface{}, error) {
	switch reportType {
	case "driver_performance":
		return s.generateDriverPerformanceReport(ctx, params)
	case "revenue_analysis":
		return s.generateRevenueAnalysisReport(ctx, params)
	case "user_behavior":
		return s.generateUserBehaviorReport(ctx, params)
	default:
		return nil, fmt.Errorf("unsupported report type: %s", reportType)
	}
}

func (s *analyticsService) ExportAnalytics(ctx context.Context, startDate, endDate time.Time, format string) ([]byte, error) {
	// This would export analytics data in the specified format (CSV, JSON, etc.)
	return nil, fmt.Errorf("export functionality not implemented")
}

// Helper methods
func (s *analyticsService) calculateRideStatistics(rides []*models.Ride) map[string]interface{} {
	var totalRides, completedRides, cancelledRides int64
	var totalRevenue, totalTime, totalDistance float64
	hourCounts := make(map[int]int)
	routeCounts := make(map[string]*models.RouteStats)

	for _, ride := range rides {
		totalRides++

		if ride.Status == models.RideStatusCompleted {
			completedRides++
			totalRevenue += ride.ActualFare
			totalTime += float64(ride.ActualDuration)
			totalDistance += ride.ActualDistance

			// Track peak hours
			if ride.CompletedAt != nil {
				hour := ride.CompletedAt.Hour()
				hourCounts[hour]++
			}

			// Track popular routes
			routeKey := fmt.Sprintf("%s-%s", ride.PickupLocation.City, ride.DropoffLocation.City)
			if stats, exists := routeCounts[routeKey]; exists {
				stats.RideCount++
				stats.AverageFare = (stats.AverageFare + ride.ActualFare) / 2
			} else {
				routeCounts[routeKey] = &models.RouteStats{
					PickupArea:  ride.PickupLocation.City,
					DropoffArea: ride.DropoffLocation.City,
					RideCount:   1,
					AverageFare: ride.ActualFare,
				}
			}
		} else if ride.Status == models.RideStatusCancelled {
			cancelledRides++
		}
	}

	// Calculate averages
	var averageRideValue, averageRideTime, averageRideDistance float64
	if completedRides > 0 {
		averageRideValue = totalRevenue / float64(completedRides)
		averageRideTime = totalTime / float64(completedRides)
		averageRideDistance = totalDistance / float64(completedRides)
	}

	// Find peak hours (hours with most rides)
	var peakHours []string
	maxCount := 0
	for hour, count := range hourCounts {
		if count > maxCount {
			maxCount = count
			peakHours = []string{fmt.Sprintf("%02d", hour)}
		} else if count == maxCount {
			peakHours = append(peakHours, fmt.Sprintf("%02d", hour))
		}
	}

	// Convert route stats to slice
	var popularRoutes []models.RouteStats
	for _, stats := range routeCounts {
		popularRoutes = append(popularRoutes, *stats)
	}

	return map[string]interface{}{
		"total_rides":           totalRides,
		"completed_rides":       completedRides,
		"cancelled_rides":       cancelledRides,
		"total_revenue":         totalRevenue,
		"average_ride_value":    averageRideValue,
		"average_ride_time":     averageRideTime,
		"average_ride_distance": averageRideDistance,
		"peak_hours":            peakHours,
		"popular_routes":        popularRoutes,
	}
}

func (s *analyticsService) calculateGrowthRate(stats map[string]int64) float64 {
	if current, exists := stats["current"]; exists {
		if previous, exists := stats["previous"]; exists && previous > 0 {
			return ((float64(current) - float64(previous)) / float64(previous)) * 100
		}
	}
	return 0.0
}

func (s *analyticsService) countUniqueUsers(events []*models.AnalyticsEvent) int {
	users := make(map[string]bool)
	for _, event := range events {
		if event.UserID != nil {
			users[event.UserID.Hex()] = true
		}
	}
	return len(users)
}

func (s *analyticsService) calculateAverageSessionDuration(events []*models.AnalyticsEvent) float64 {
	// This would calculate session duration based on events
	return 0.0 // Placeholder
}

func (s *analyticsService) calculateBounceRate(events []*models.AnalyticsEvent) float64 {
	// This would calculate bounce rate based on session events
	return 0.0 // Placeholder
}

func (s *analyticsService) calculateRetentionRate(events []*models.AnalyticsEvent) float64 {
	// This would calculate retention rate based on user activity
	return 0.0 // Placeholder
}

func (s *analyticsService) updateRealtimeMetrics(ctx context.Context, event *models.AnalyticsEvent) {
	if s.cache == nil {
		return
	}

	// Update real-time counters
	key := fmt.Sprintf("realtime_events_%s", event.EventType)
	s.cache.Increment(ctx, key, 1, 24*time.Hour)
}

func (s *analyticsService) generateDriverPerformanceReport(ctx context.Context, params map[string]interface{}) (map[string]interface{}, error) {
	// Implementation for driver performance report
	return map[string]interface{}{
		"report_type":  "driver_performance",
		"generated_at": time.Now(),
	}, nil
}

func (s *analyticsService) generateRevenueAnalysisReport(ctx context.Context, params map[string]interface{}) (map[string]interface{}, error) {
	// Implementation for revenue analysis report
	return map[string]interface{}{
		"report_type":  "revenue_analysis",
		"generated_at": time.Now(),
	}, nil
}

func (s *analyticsService) generateUserBehaviorReport(ctx context.Context, params map[string]interface{}) (map[string]interface{}, error) {
	// Implementation for user behavior report
	return map[string]interface{}{
		"report_type":  "user_behavior",
		"generated_at": time.Now(),
	}, nil
}
