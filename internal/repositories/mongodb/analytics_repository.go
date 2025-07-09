package mongodb

import (
	"context"
	"fmt"
	"time"

	"goride/internal/models"
	"goride/internal/repositories/interfaces"
	"goride/internal/utils"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

type analyticsRepository struct {
	eventsCollection        *mongo.Collection
	rideAnalyticsCollection *mongo.Collection
	cache                   CacheService
}

func NewAnalyticsRepository(db *mongo.Database, cache CacheService) interfaces.AnalyticsRepository {
	return &analyticsRepository{
		eventsCollection:        db.Collection("analytics_events"),
		rideAnalyticsCollection: db.Collection("ride_analytics"),
		cache:                   cache,
	}
}

// Event tracking
func (r *analyticsRepository) CreateEvent(ctx context.Context, event *models.AnalyticsEvent) error {
	event.ID = primitive.NewObjectID()
	event.CreatedAt = time.Now()

	_, err := r.eventsCollection.InsertOne(ctx, event)
	if err != nil {
		return fmt.Errorf("failed to create analytics event: %w", err)
	}

	return nil
}

func (r *analyticsRepository) GetEventsByType(ctx context.Context, eventType string, params *utils.PaginationParams) ([]*models.AnalyticsEvent, int64, error) {
	filter := bson.M{"event_type": eventType}
	return r.findEventsWithFilter(ctx, filter, params)
}

func (r *analyticsRepository) GetEventsByUser(ctx context.Context, userID primitive.ObjectID, params *utils.PaginationParams) ([]*models.AnalyticsEvent, int64, error) {
	filter := bson.M{"user_id": userID}
	return r.findEventsWithFilter(ctx, filter, params)
}

func (r *analyticsRepository) GetEventsByDateRange(ctx context.Context, startDate, endDate time.Time, eventType string) ([]*models.AnalyticsEvent, error) {
	filter := bson.M{
		"created_at": bson.M{
			"$gte": startDate,
			"$lte": endDate,
		},
	}

	if eventType != "" {
		filter["event_type"] = eventType
	}

	cursor, err := r.eventsCollection.Find(ctx, filter)
	if err != nil {
		return nil, fmt.Errorf("failed to find events: %w", err)
	}
	defer cursor.Close(ctx)

	var events []*models.AnalyticsEvent
	for cursor.Next(ctx) {
		var event models.AnalyticsEvent
		if err := cursor.Decode(&event); err != nil {
			return nil, fmt.Errorf("failed to decode event: %w", err)
		}
		events = append(events, &event)
	}

	return events, nil
}

// Ride analytics
func (r *analyticsRepository) CreateRideAnalytics(ctx context.Context, analytics *models.RideAnalytics) error {
	analytics.ID = primitive.NewObjectID()
	analytics.CreatedAt = time.Now()
	analytics.UpdatedAt = time.Now()

	_, err := r.rideAnalyticsCollection.InsertOne(ctx, analytics)
	if err != nil {
		return fmt.Errorf("failed to create ride analytics: %w", err)
	}

	return nil
}

func (r *analyticsRepository) GetRideAnalyticsByDate(ctx context.Context, date time.Time, city string) (*models.RideAnalytics, error) {
	filter := bson.M{
		"date": bson.M{
			"$gte": utils.GetStartOfDay(date),
			"$lt":  utils.GetEndOfDay(date),
		},
		"city": city,
	}

	var analytics models.RideAnalytics
	err := r.rideAnalyticsCollection.FindOne(ctx, filter).Decode(&analytics)
	if err != nil {
		if err == mongo.ErrNoDocuments {
			return nil, fmt.Errorf("ride analytics not found")
		}
		return nil, fmt.Errorf("failed to get ride analytics: %w", err)
	}

	return &analytics, nil
}

func (r *analyticsRepository) GetRideAnalyticsByDateRange(ctx context.Context, startDate, endDate time.Time, city string) ([]*models.RideAnalytics, error) {
	filter := bson.M{
		"date": bson.M{
			"$gte": startDate,
			"$lte": endDate,
		},
	}

	if city != "" {
		filter["city"] = city
	}

	cursor, err := r.rideAnalyticsCollection.Find(ctx, filter, options.Find().SetSort(bson.D{{Key: "date", Value: 1}}))
	if err != nil {
		return nil, fmt.Errorf("failed to find ride analytics: %w", err)
	}
	defer cursor.Close(ctx)

	var analytics []*models.RideAnalytics
	for cursor.Next(ctx) {
		var item models.RideAnalytics
		if err := cursor.Decode(&item); err != nil {
			return nil, fmt.Errorf("failed to decode ride analytics: %w", err)
		}
		analytics = append(analytics, &item)
	}

	return analytics, nil
}

func (r *analyticsRepository) UpdateRideAnalytics(ctx context.Context, date time.Time, city string, updates map[string]interface{}) error {
	filter := bson.M{
		"date": bson.M{
			"$gte": utils.GetStartOfDay(date),
			"$lt":  utils.GetEndOfDay(date),
		},
		"city": city,
	}

	updates["updated_at"] = time.Now()

	_, err := r.rideAnalyticsCollection.UpdateOne(
		ctx,
		filter,
		bson.M{"$set": updates},
		options.Update().SetUpsert(true),
	)
	if err != nil {
		return fmt.Errorf("failed to update ride analytics: %w", err)
	}

	return nil
}

// Dashboard metrics
func (r *analyticsRepository) GetDashboardMetrics(ctx context.Context, days int) (map[string]interface{}, error) {
	cacheKey := fmt.Sprintf("dashboard_metrics_%d", days)

	// Try cache first
	if r.cache != nil {
		var metrics map[string]interface{}
		if err := r.cache.Get(ctx, cacheKey, &metrics); err == nil {
			return metrics, nil
		}
	}

	startDate := time.Now().AddDate(0, 0, -days)

	// Aggregate multiple metrics
	pipeline := mongo.Pipeline{
		{{Key: "$match", Value: bson.M{
			"date": bson.M{"$gte": startDate},
		}}},
		{{Key: "$group", Value: bson.M{
			"_id":               nil,
			"total_rides":       bson.M{"$sum": "$total_rides"},
			"completed_rides":   bson.M{"$sum": "$completed_rides"},
			"cancelled_rides":   bson.M{"$sum": "$cancelled_rides"},
			"total_revenue":     bson.M{"$sum": "$total_revenue"},
			"avg_ride_value":    bson.M{"$avg": "$average_ride_value"},
			"avg_ride_time":     bson.M{"$avg": "$average_ride_time"},
			"avg_ride_distance": bson.M{"$avg": "$average_ride_distance"},
		}}},
	}

	cursor, err := r.rideAnalyticsCollection.Aggregate(ctx, pipeline)
	if err != nil {
		return nil, fmt.Errorf("failed to get dashboard metrics: %w", err)
	}
	defer cursor.Close(ctx)

	var result struct {
		TotalRides      int64   `bson:"total_rides"`
		CompletedRides  int64   `bson:"completed_rides"`
		CancelledRides  int64   `bson:"cancelled_rides"`
		TotalRevenue    float64 `bson:"total_revenue"`
		AvgRideValue    float64 `bson:"avg_ride_value"`
		AvgRideTime     float64 `bson:"avg_ride_time"`
		AvgRideDistance float64 `bson:"avg_ride_distance"`
	}

	if cursor.Next(ctx) {
		if err := cursor.Decode(&result); err != nil {
			return nil, fmt.Errorf("failed to decode dashboard metrics: %w", err)
		}
	}

	metrics := map[string]interface{}{
		"total_rides":       result.TotalRides,
		"completed_rides":   result.CompletedRides,
		"cancelled_rides":   result.CancelledRides,
		"completion_rate":   float64(result.CompletedRides) / float64(result.TotalRides) * 100,
		"cancellation_rate": float64(result.CancelledRides) / float64(result.TotalRides) * 100,
		"total_revenue":     result.TotalRevenue,
		"avg_ride_value":    result.AvgRideValue,
		"avg_ride_time":     result.AvgRideTime,
		"avg_ride_distance": result.AvgRideDistance,
		"period_days":       days,
	}

	// Cache the result
	if r.cache != nil {
		r.cache.Set(ctx, cacheKey, metrics, 5*time.Minute)
	}

	return metrics, nil
}

func (r *analyticsRepository) GetRealtimeMetrics(ctx context.Context) (map[string]interface{}, error) {
	// Get real-time metrics from various collections
	today := time.Now()
	startOfDay := utils.GetStartOfDay(today)

	// Today's rides
	ridesFilter := bson.M{
		"created_at": bson.M{"$gte": startOfDay},
	}

	ridesCollection := r.rideAnalyticsCollection.Database().Collection("rides")
	totalRidesToday, _ := ridesCollection.CountDocuments(ctx, ridesFilter)

	completedRidesToday, _ := ridesCollection.CountDocuments(ctx, bson.M{
		"created_at": bson.M{"$gte": startOfDay},
		"status":     "completed",
	})

	inProgressRides, _ := ridesCollection.CountDocuments(ctx, bson.M{
		"status": bson.M{"$in": []string{"requested", "accepted", "driver_arrived", "in_progress"}},
	})

	// Active drivers
	driversCollection := r.rideAnalyticsCollection.Database().Collection("drivers")
	onlineDrivers, _ := driversCollection.CountDocuments(ctx, bson.M{
		"status":       "online",
		"is_available": true,
	})

	return map[string]interface{}{
		"rides_today":       totalRidesToday,
		"completed_today":   completedRidesToday,
		"rides_in_progress": inProgressRides,
		"active_drivers":    onlineDrivers,
		"timestamp":         time.Now(),
	}, nil
}

// User analytics
func (r *analyticsRepository) GetUserRegistrationTrends(ctx context.Context, days int) ([]map[string]interface{}, error) {
	startDate := time.Now().AddDate(0, 0, -days)

	usersCollection := r.eventsCollection.Database().Collection("users")

	pipeline := mongo.Pipeline{
		{{Key: "$match", Value: bson.M{
			"created_at": bson.M{"$gte": startDate},
			"deleted_at": nil,
		}}},
		{{Key: "$group", Value: bson.M{
			"_id": bson.M{
				"date": bson.M{"$dateToString": bson.M{
					"format": "%Y-%m-%d",
					"date":   "$created_at",
				}},
				"user_type": "$user_type",
			},
			"count": bson.M{"$sum": 1},
		}}},
		{{Key: "$sort", Value: bson.M{"_id.date": 1}}},
	}

	cursor, err := usersCollection.Aggregate(ctx, pipeline)
	if err != nil {
		return nil, fmt.Errorf("failed to get user registration trends: %w", err)
	}
	defer cursor.Close(ctx)

	var results []map[string]interface{}
	for cursor.Next(ctx) {
		var result struct {
			ID struct {
				Date     string `bson:"date"`
				UserType string `bson:"user_type"`
			} `bson:"_id"`
			Count int64 `bson:"count"`
		}

		if err := cursor.Decode(&result); err != nil {
			return nil, fmt.Errorf("failed to decode registration trend: %w", err)
		}

		results = append(results, map[string]interface{}{
			"date":      result.ID.Date,
			"user_type": result.ID.UserType,
			"count":     result.Count,
		})
	}

	return results, nil
}

func (r *analyticsRepository) GetUserActivityTrends(ctx context.Context, days int) ([]map[string]interface{}, error) {
	startDate := time.Now().AddDate(0, 0, -days)

	pipeline := mongo.Pipeline{
		{{"$match", bson.M{
			"created_at": bson.M{"$gte": startDate},
			"event_type": "user_activity",
		}}},
		{{"$group", bson.M{
			"_id": bson.M{
				"date": bson.M{"$dateToString": bson.M{
					"format": "%Y-%m-%d",
					"date":   "$created_at",
				}},
				"activity": "$event_name",
			},
			"count": bson.M{"$sum": 1},
		}}},
		{{"$sort", bson.M{"_id.date": 1}}},
	}

	cursor, err := r.eventsCollection.Aggregate(ctx, pipeline)
	if err != nil {
		return nil, fmt.Errorf("failed to get user activity trends: %w", err)
	}
	defer cursor.Close(ctx)

	var results []map[string]interface{}
	for cursor.Next(ctx) {
		var result struct {
			ID struct {
				Date     string `bson:"date"`
				Activity string `bson:"activity"`
			} `bson:"_id"`
			Count int64 `bson:"count"`
		}

		if err := cursor.Decode(&result); err != nil {
			return nil, fmt.Errorf("failed to decode activity trend: %w", err)
		}

		results = append(results, map[string]interface{}{
			"date":     result.ID.Date,
			"activity": result.ID.Activity,
			"count":    result.Count,
		})
	}

	return results, nil
}

func (r *analyticsRepository) GetUserRetentionStats(ctx context.Context, cohortDate time.Time) (map[string]interface{}, error) {
	// Complex retention analysis - simplified version
	usersCollection := r.eventsCollection.Database().Collection("users")

	cohortStart := utils.GetStartOfDay(cohortDate)
	cohortEnd := utils.GetEndOfDay(cohortDate)

	// Get users from cohort
	cohortFilter := bson.M{
		"created_at": bson.M{
			"$gte": cohortStart,
			"$lt":  cohortEnd,
		},
		"deleted_at": nil,
	}

	cohortSize, err := usersCollection.CountDocuments(ctx, cohortFilter)
	if err != nil {
		return nil, fmt.Errorf("failed to count cohort users: %w", err)
	}

	// Calculate retention for different periods
	periods := []int{1, 7, 14, 30, 90}
	retention := make(map[string]interface{})

	for _, period := range periods {
		checkDate := cohortDate.AddDate(0, 0, period)

		// Count users who were active on the check date
		activeFilter := bson.M{
			"created_at": bson.M{
				"$gte": cohortStart,
				"$lt":  cohortEnd,
			},
			"last_active_at": bson.M{
				"$gte": utils.GetStartOfDay(checkDate),
			},
			"deleted_at": nil,
		}

		activeCount, _ := usersCollection.CountDocuments(ctx, activeFilter)

		retentionRate := float64(0)
		if cohortSize > 0 {
			retentionRate = float64(activeCount) / float64(cohortSize) * 100
		}

		retention[fmt.Sprintf("day_%d", period)] = retentionRate
	}

	return map[string]interface{}{
		"cohort_date": cohortDate,
		"cohort_size": cohortSize,
		"retention":   retention,
	}, nil
}

// Ride analytics methods
func (r *analyticsRepository) GetRideVolumeTrends(ctx context.Context, days int, city string) ([]map[string]interface{}, error) {
	startDate := time.Now().AddDate(0, 0, -days)

	filter := bson.M{
		"date": bson.M{"$gte": startDate},
	}

	if city != "" {
		filter["city"] = city
	}

	pipeline := mongo.Pipeline{
		{{"$match", filter}},
		{{"$group", bson.M{
			"_id": bson.M{
				"date": bson.M{"$dateToString": bson.M{
					"format": "%Y-%m-%d",
					"date":   "$date",
				}},
			},
			"total_rides":     bson.M{"$sum": "$total_rides"},
			"completed_rides": bson.M{"$sum": "$completed_rides"},
			"cancelled_rides": bson.M{"$sum": "$cancelled_rides"},
		}}},
		{{"$sort", bson.M{"_id.date": 1}}},
	}

	cursor, err := r.rideAnalyticsCollection.Aggregate(ctx, pipeline)
	if err != nil {
		return nil, fmt.Errorf("failed to get ride volume trends: %w", err)
	}
	defer cursor.Close(ctx)

	var results []map[string]interface{}
	for cursor.Next(ctx) {
		var result struct {
			ID struct {
				Date string `bson:"date"`
			} `bson:"_id"`
			TotalRides     int64 `bson:"total_rides"`
			CompletedRides int64 `bson:"completed_rides"`
			CancelledRides int64 `bson:"cancelled_rides"`
		}

		if err := cursor.Decode(&result); err != nil {
			return nil, fmt.Errorf("failed to decode ride volume trend: %w", err)
		}

		results = append(results, map[string]interface{}{
			"date":            result.ID.Date,
			"total_rides":     result.TotalRides,
			"completed_rides": result.CompletedRides,
			"cancelled_rides": result.CancelledRides,
		})
	}

	return results, nil
}

func (r *analyticsRepository) GetRideCompletionRates(ctx context.Context, days int) (map[string]interface{}, error) {
	startDate := time.Now().AddDate(0, 0, -days)

	pipeline := mongo.Pipeline{
		{{"$match", bson.M{
			"date": bson.M{"$gte": startDate},
		}}},
		{{"$group", bson.M{
			"_id":             nil,
			"total_rides":     bson.M{"$sum": "$total_rides"},
			"completed_rides": bson.M{"$sum": "$completed_rides"},
			"cancelled_rides": bson.M{"$sum": "$cancelled_rides"},
		}}},
	}

	cursor, err := r.rideAnalyticsCollection.Aggregate(ctx, pipeline)
	if err != nil {
		return nil, fmt.Errorf("failed to get ride completion rates: %w", err)
	}
	defer cursor.Close(ctx)

	var result struct {
		TotalRides     int64 `bson:"total_rides"`
		CompletedRides int64 `bson:"completed_rides"`
		CancelledRides int64 `bson:"cancelled_rides"`
	}

	if cursor.Next(ctx) {
		if err := cursor.Decode(&result); err != nil {
			return nil, fmt.Errorf("failed to decode completion rates: %w", err)
		}
	}

	completionRate := float64(0)
	cancellationRate := float64(0)

	if result.TotalRides > 0 {
		completionRate = float64(result.CompletedRides) / float64(result.TotalRides) * 100
		cancellationRate = float64(result.CancelledRides) / float64(result.TotalRides) * 100
	}

	return map[string]interface{}{
		"total_rides":       result.TotalRides,
		"completed_rides":   result.CompletedRides,
		"cancelled_rides":   result.CancelledRides,
		"completion_rate":   completionRate,
		"cancellation_rate": cancellationRate,
		"period_days":       days,
	}, nil
}

func (r *analyticsRepository) GetAverageRideMetrics(ctx context.Context, days int) (map[string]interface{}, error) {
	startDate := time.Now().AddDate(0, 0, -days)

	pipeline := mongo.Pipeline{
		{{"$match", bson.M{
			"date": bson.M{"$gte": startDate},
		}}},
		{{"$group", bson.M{
			"_id":               nil,
			"avg_ride_value":    bson.M{"$avg": "$average_ride_value"},
			"avg_ride_time":     bson.M{"$avg": "$average_ride_time"},
			"avg_ride_distance": bson.M{"$avg": "$average_ride_distance"},
			"total_revenue":     bson.M{"$sum": "$total_revenue"},
			"total_rides":       bson.M{"$sum": "$total_rides"},
		}}},
	}

	cursor, err := r.rideAnalyticsCollection.Aggregate(ctx, pipeline)
	if err != nil {
		return nil, fmt.Errorf("failed to get average ride metrics: %w", err)
	}
	defer cursor.Close(ctx)

	var result struct {
		AvgRideValue    float64 `bson:"avg_ride_value"`
		AvgRideTime     float64 `bson:"avg_ride_time"`
		AvgRideDistance float64 `bson:"avg_ride_distance"`
		TotalRevenue    float64 `bson:"total_revenue"`
		TotalRides      int64   `bson:"total_rides"`
	}

	if cursor.Next(ctx) {
		if err := cursor.Decode(&result); err != nil {
			return nil, fmt.Errorf("failed to decode average metrics: %w", err)
		}
	}

	return map[string]interface{}{
		"avg_ride_value":    result.AvgRideValue,
		"avg_ride_time":     result.AvgRideTime,
		"avg_ride_distance": result.AvgRideDistance,
		"total_revenue":     result.TotalRevenue,
		"total_rides":       result.TotalRides,
		"period_days":       days,
	}, nil
}

func (r *analyticsRepository) GetPopularPickupAreas(ctx context.Context, city string, days int, limit int) ([]map[string]interface{}, error) {
	startDate := time.Now().AddDate(0, 0, -days)

	ridesCollection := r.eventsCollection.Database().Collection("rides")

	pipeline := mongo.Pipeline{
		{{"$match", bson.M{
			"created_at":           bson.M{"$gte": startDate},
			"pickup_location.city": city,
		}}},
		{{"$group", bson.M{
			"_id":        "$pickup_location.address",
			"ride_count": bson.M{"$sum": 1},
			"avg_fare":   bson.M{"$avg": "$fare.total"},
			"location":   bson.M{"$first": "$pickup_location"},
		}}},
		{{"$sort", bson.M{"ride_count": -1}}},
		{{"$limit", limit}},
	}

	cursor, err := ridesCollection.Aggregate(ctx, pipeline)
	if err != nil {
		return nil, fmt.Errorf("failed to get popular pickup areas: %w", err)
	}
	defer cursor.Close(ctx)

	var results []map[string]interface{}
	for cursor.Next(ctx) {
		var result struct {
			Address   string                 `bson:"_id"`
			RideCount int64                  `bson:"ride_count"`
			AvgFare   float64                `bson:"avg_fare"`
			Location  map[string]interface{} `bson:"location"`
		}

		if err := cursor.Decode(&result); err != nil {
			return nil, fmt.Errorf("failed to decode pickup area: %w", err)
		}

		results = append(results, map[string]interface{}{
			"address":    result.Address,
			"ride_count": result.RideCount,
			"avg_fare":   result.AvgFare,
			"location":   result.Location,
		})
	}

	return results, nil
}

func (r *analyticsRepository) GetPopularDropoffAreas(ctx context.Context, city string, days int, limit int) ([]map[string]interface{}, error) {
	startDate := time.Now().AddDate(0, 0, -days)

	ridesCollection := r.eventsCollection.Database().Collection("rides")

	pipeline := mongo.Pipeline{
		{{"$match", bson.M{
			"created_at":            bson.M{"$gte": startDate},
			"dropoff_location.city": city,
		}}},
		{{"$group", bson.M{
			"_id":        "$dropoff_location.address",
			"ride_count": bson.M{"$sum": 1},
			"avg_fare":   bson.M{"$avg": "$fare.total"},
			"location":   bson.M{"$first": "$dropoff_location"},
		}}},
		{{"$sort", bson.M{"ride_count": -1}}},
		{{"$limit", limit}},
	}

	cursor, err := ridesCollection.Aggregate(ctx, pipeline)
	if err != nil {
		return nil, fmt.Errorf("failed to get popular dropoff areas: %w", err)
	}
	defer cursor.Close(ctx)

	var results []map[string]interface{}
	for cursor.Next(ctx) {
		var result struct {
			Address   string                 `bson:"_id"`
			RideCount int64                  `bson:"ride_count"`
			AvgFare   float64                `bson:"avg_fare"`
			Location  map[string]interface{} `bson:"location"`
		}

		if err := cursor.Decode(&result); err != nil {
			return nil, fmt.Errorf("failed to decode dropoff area: %w", err)
		}

		results = append(results, map[string]interface{}{
			"address":    result.Address,
			"ride_count": result.RideCount,
			"avg_fare":   result.AvgFare,
			"location":   result.Location,
		})
	}

	return results, nil
}

// Revenue analytics
func (r *analyticsRepository) GetRevenueStats(ctx context.Context, days int) (map[string]interface{}, error) {
	startDate := time.Now().AddDate(0, 0, -days)

	pipeline := mongo.Pipeline{
		{{"$match", bson.M{
			"date": bson.M{"$gte": startDate},
		}}},
		{{"$group", bson.M{
			"_id":                 nil,
			"total_revenue":       bson.M{"$sum": "$total_revenue"},
			"avg_revenue_per_day": bson.M{"$avg": "$total_revenue"},
			"max_daily_revenue":   bson.M{"$max": "$total_revenue"},
			"min_daily_revenue":   bson.M{"$min": "$total_revenue"},
			"total_rides":         bson.M{"$sum": "$total_rides"},
		}}},
	}

	cursor, err := r.rideAnalyticsCollection.Aggregate(ctx, pipeline)
	if err != nil {
		return nil, fmt.Errorf("failed to get revenue stats: %w", err)
	}
	defer cursor.Close(ctx)

	var result struct {
		TotalRevenue     float64 `bson:"total_revenue"`
		AvgRevenuePerDay float64 `bson:"avg_revenue_per_day"`
		MaxDailyRevenue  float64 `bson:"max_daily_revenue"`
		MinDailyRevenue  float64 `bson:"min_daily_revenue"`
		TotalRides       int64   `bson:"total_rides"`
	}

	if cursor.Next(ctx) {
		if err := cursor.Decode(&result); err != nil {
			return nil, fmt.Errorf("failed to decode revenue stats: %w", err)
		}
	}

	return map[string]interface{}{
		"total_revenue":       result.TotalRevenue,
		"avg_revenue_per_day": result.AvgRevenuePerDay,
		"max_daily_revenue":   result.MaxDailyRevenue,
		"min_daily_revenue":   result.MinDailyRevenue,
		"total_rides":         result.TotalRides,
		"avg_revenue_per_ride": func() float64 {
			if result.TotalRides > 0 {
				return result.TotalRevenue / float64(result.TotalRides)
			}
			return 0
		}(),
		"period_days": days,
	}, nil
}

func (r *analyticsRepository) GetRevenueTrends(ctx context.Context, days int) ([]map[string]interface{}, error) {
	startDate := time.Now().AddDate(0, 0, -days)

	pipeline := mongo.Pipeline{
		{{"$match", bson.M{
			"date": bson.M{"$gte": startDate},
		}}},
		{{"$group", bson.M{
			"_id": bson.M{
				"date": bson.M{"$dateToString": bson.M{
					"format": "%Y-%m-%d",
					"date":   "$date",
				}},
			},
			"total_revenue":  bson.M{"$sum": "$total_revenue"},
			"total_rides":    bson.M{"$sum": "$total_rides"},
			"avg_ride_value": bson.M{"$avg": "$average_ride_value"},
		}}},
		{{"$sort", bson.M{"_id.date": 1}}},
	}

	cursor, err := r.rideAnalyticsCollection.Aggregate(ctx, pipeline)
	if err != nil {
		return nil, fmt.Errorf("failed to get revenue trends: %w", err)
	}
	defer cursor.Close(ctx)

	var results []map[string]interface{}
	for cursor.Next(ctx) {
		var result struct {
			ID struct {
				Date string `bson:"date"`
			} `bson:"_id"`
			TotalRevenue float64 `bson:"total_revenue"`
			TotalRides   int64   `bson:"total_rides"`
			AvgRideValue float64 `bson:"avg_ride_value"`
		}

		if err := cursor.Decode(&result); err != nil {
			return nil, fmt.Errorf("failed to decode revenue trend: %w", err)
		}

		results = append(results, map[string]interface{}{
			"date":           result.ID.Date,
			"total_revenue":  result.TotalRevenue,
			"total_rides":    result.TotalRides,
			"avg_ride_value": result.AvgRideValue,
		})
	}

	return results, nil
}

func (r *analyticsRepository) GetRevenueByCity(ctx context.Context, days int) ([]map[string]interface{}, error) {
	startDate := time.Now().AddDate(0, 0, -days)

	pipeline := mongo.Pipeline{
		{{"$match", bson.M{
			"date": bson.M{"$gte": startDate},
		}}},
		{{"$group", bson.M{
			"_id":            "$city",
			"total_revenue":  bson.M{"$sum": "$total_revenue"},
			"total_rides":    bson.M{"$sum": "$total_rides"},
			"avg_ride_value": bson.M{"$avg": "$average_ride_value"},
		}}},
		{{"$sort", bson.M{"total_revenue": -1}}},
	}

	cursor, err := r.rideAnalyticsCollection.Aggregate(ctx, pipeline)
	if err != nil {
		return nil, fmt.Errorf("failed to get revenue by city: %w", err)
	}
	defer cursor.Close(ctx)

	var results []map[string]interface{}
	for cursor.Next(ctx) {
		var result struct {
			City         string  `bson:"_id"`
			TotalRevenue float64 `bson:"total_revenue"`
			TotalRides   int64   `bson:"total_rides"`
			AvgRideValue float64 `bson:"avg_ride_value"`
		}

		if err := cursor.Decode(&result); err != nil {
			return nil, fmt.Errorf("failed to decode city revenue: %w", err)
		}

		results = append(results, map[string]interface{}{
			"city":           result.City,
			"total_revenue":  result.TotalRevenue,
			"total_rides":    result.TotalRides,
			"avg_ride_value": result.AvgRideValue,
		})
	}

	return results, nil
}

func (r *analyticsRepository) GetRevenueByRideType(ctx context.Context, days int) ([]map[string]interface{}, error) {
	startDate := time.Now().AddDate(0, 0, -days)

	ridesCollection := r.eventsCollection.Database().Collection("rides")

	pipeline := mongo.Pipeline{
		{{"$match", bson.M{
			"created_at": bson.M{"$gte": startDate},
			"status":     "completed",
		}}},
		{{"$group", bson.M{
			"_id":           "$ride_type",
			"total_revenue": bson.M{"$sum": "$fare.total"},
			"total_rides":   bson.M{"$sum": 1},
			"avg_fare":      bson.M{"$avg": "$fare.total"},
		}}},
		{{"$sort", bson.M{"total_revenue": -1}}},
	}

	cursor, err := ridesCollection.Aggregate(ctx, pipeline)
	if err != nil {
		return nil, fmt.Errorf("failed to get revenue by ride type: %w", err)
	}
	defer cursor.Close(ctx)

	var results []map[string]interface{}
	for cursor.Next(ctx) {
		var result struct {
			RideType     string  `bson:"_id"`
			TotalRevenue float64 `bson:"total_revenue"`
			TotalRides   int64   `bson:"total_rides"`
			AvgFare      float64 `bson:"avg_fare"`
		}

		if err := cursor.Decode(&result); err != nil {
			return nil, fmt.Errorf("failed to decode ride type revenue: %w", err)
		}

		results = append(results, map[string]interface{}{
			"ride_type":     result.RideType,
			"total_revenue": result.TotalRevenue,
			"total_rides":   result.TotalRides,
			"avg_fare":      result.AvgFare,
		})
	}

	return results, nil
}

// Driver analytics
func (r *analyticsRepository) GetDriverMetrics(ctx context.Context, days int) (map[string]interface{}, error) {
	startDate := time.Now().AddDate(0, 0, -days)

	driversCollection := r.eventsCollection.Database().Collection("drivers")

	pipeline := mongo.Pipeline{
		{{"$match", bson.M{
			"updated_at": bson.M{"$gte": startDate},
		}}},
		{{"$group", bson.M{
			"_id":           nil,
			"total_drivers": bson.M{"$sum": 1},
			"active_drivers": bson.M{"$sum": bson.M{
				"$cond": []interface{}{
					bson.M{"$eq": []interface{}{"$status", "online"}},
					1, 0,
				},
			}},
			"avg_rating":     bson.M{"$avg": "$rating"},
			"total_earnings": bson.M{"$sum": "$total_earnings"},
			"total_rides":    bson.M{"$sum": "$total_rides"},
		}}},
	}

	cursor, err := driversCollection.Aggregate(ctx, pipeline)
	if err != nil {
		return nil, fmt.Errorf("failed to get driver metrics: %w", err)
	}
	defer cursor.Close(ctx)

	var result struct {
		TotalDrivers  int64   `bson:"total_drivers"`
		ActiveDrivers int64   `bson:"active_drivers"`
		AvgRating     float64 `bson:"avg_rating"`
		TotalEarnings float64 `bson:"total_earnings"`
		TotalRides    int64   `bson:"total_rides"`
	}

	if cursor.Next(ctx) {
		if err := cursor.Decode(&result); err != nil {
			return nil, fmt.Errorf("failed to decode driver metrics: %w", err)
		}
	}

	return map[string]interface{}{
		"total_drivers":  result.TotalDrivers,
		"active_drivers": result.ActiveDrivers,
		"avg_rating":     result.AvgRating,
		"total_earnings": result.TotalEarnings,
		"total_rides":    result.TotalRides,
		"avg_earnings_per_driver": func() float64 {
			if result.TotalDrivers > 0 {
				return result.TotalEarnings / float64(result.TotalDrivers)
			}
			return 0
		}(),
		"period_days": days,
	}, nil
}

func (r *analyticsRepository) GetDriverUtilization(ctx context.Context, days int) (map[string]interface{}, error) {
	startDate := time.Now().AddDate(0, 0, -days)

	driversCollection := r.eventsCollection.Database().Collection("drivers")

	pipeline := mongo.Pipeline{
		{{"$match", bson.M{
			"updated_at": bson.M{"$gte": startDate},
		}}},
		{{"$group", bson.M{
			"_id":                   nil,
			"avg_online_hours":      bson.M{"$avg": "$online_hours"},
			"avg_acceptance_rate":   bson.M{"$avg": "$acceptance_rate"},
			"avg_completion_rate":   bson.M{"$avg": "$completion_rate"},
			"avg_cancellation_rate": bson.M{"$avg": "$cancellation_rate"},
		}}},
	}

	cursor, err := driversCollection.Aggregate(ctx, pipeline)
	if err != nil {
		return nil, fmt.Errorf("failed to get driver utilization: %w", err)
	}
	defer cursor.Close(ctx)

	var result struct {
		AvgOnlineHours      float64 `bson:"avg_online_hours"`
		AvgAcceptanceRate   float64 `bson:"avg_acceptance_rate"`
		AvgCompletionRate   float64 `bson:"avg_completion_rate"`
		AvgCancellationRate float64 `bson:"avg_cancellation_rate"`
	}

	if cursor.Next(ctx) {
		if err := cursor.Decode(&result); err != nil {
			return nil, fmt.Errorf("failed to decode driver utilization: %w", err)
		}
	}

	return map[string]interface{}{
		"avg_online_hours":      result.AvgOnlineHours,
		"avg_acceptance_rate":   result.AvgAcceptanceRate,
		"avg_completion_rate":   result.AvgCompletionRate,
		"avg_cancellation_rate": result.AvgCancellationRate,
		"period_days":           days,
	}, nil
}

func (r *analyticsRepository) GetDriverEarningsStats(ctx context.Context, days int) (map[string]interface{}, error) {
	startDate := time.Now().AddDate(0, 0, -days)

	driversCollection := r.eventsCollection.Database().Collection("drivers")

	pipeline := mongo.Pipeline{
		{{"$match", bson.M{
			"updated_at": bson.M{"$gte": startDate},
		}}},
		{{"$group", bson.M{
			"_id":            nil,
			"total_earnings": bson.M{"$sum": "$total_earnings"},
			"avg_earnings":   bson.M{"$avg": "$total_earnings"},
			"max_earnings":   bson.M{"$max": "$total_earnings"},
			"min_earnings":   bson.M{"$min": "$total_earnings"},
			"total_drivers":  bson.M{"$sum": 1},
		}}},
	}

	cursor, err := driversCollection.Aggregate(ctx, pipeline)
	if err != nil {
		return nil, fmt.Errorf("failed to get driver earnings stats: %w", err)
	}
	defer cursor.Close(ctx)

	var result struct {
		TotalEarnings float64 `bson:"total_earnings"`
		AvgEarnings   float64 `bson:"avg_earnings"`
		MaxEarnings   float64 `bson:"max_earnings"`
		MinEarnings   float64 `bson:"min_earnings"`
		TotalDrivers  int64   `bson:"total_drivers"`
	}

	if cursor.Next(ctx) {
		if err := cursor.Decode(&result); err != nil {
			return nil, fmt.Errorf("failed to decode earnings stats: %w", err)
		}
	}

	return map[string]interface{}{
		"total_earnings": result.TotalEarnings,
		"avg_earnings":   result.AvgEarnings,
		"max_earnings":   result.MaxEarnings,
		"min_earnings":   result.MinEarnings,
		"total_drivers":  result.TotalDrivers,
		"period_days":    days,
	}, nil
}

func (r *analyticsRepository) GetTopDrivers(ctx context.Context, metric string, days int, limit int) ([]map[string]interface{}, error) {
	startDate := time.Now().AddDate(0, 0, -days)

	driversCollection := r.eventsCollection.Database().Collection("drivers")
	r.eventsCollection.Database().Collection("users")

	// Define sort field based on metric
	sortField := "$total_earnings" // default
	switch metric {
	case "rides":
		sortField = "$total_rides"
	case "rating":
		sortField = "$rating"
	case "earnings":
		sortField = "$total_earnings"
	}

	pipeline := mongo.Pipeline{
		{{Key: "$match", Value: bson.M{
			"updated_at": bson.M{"$gte": startDate},
		}}},
		{{Key: "$lookup", Value: bson.M{
			"from":         "users",
			"localField":   "user_id",
			"foreignField": "_id",
			"as":           "user",
		}}},
		{{Key: "$unwind", Value: "$user"}},
		{{"$project", bson.M{
			"user_id":         1,
			"rating":          1,
			"total_rides":     1,
			"total_earnings":  1,
			"acceptance_rate": 1,
			"completion_rate": 1,
			"user.first_name": 1,
			"user.last_name":  1,
			"user.email":      1,
		}}},
		{{"$sort", bson.M{sortField: -1}}},
		{{"$limit", limit}},
	}

	cursor, err := driversCollection.Aggregate(ctx, pipeline)
	if err != nil {
		return nil, fmt.Errorf("failed to get top drivers: %w", err)
	}
	defer cursor.Close(ctx)

	var results []map[string]interface{}
	for cursor.Next(ctx) {
		var result map[string]interface{}
		if err := cursor.Decode(&result); err != nil {
			return nil, fmt.Errorf("failed to decode top driver: %w", err)
		}
		results = append(results, result)
	}

	return results, nil
}

// Performance analytics
func (r *analyticsRepository) GetSystemPerformanceMetrics(ctx context.Context, hours int) ([]map[string]interface{}, error) {
	startTime := time.Now().Add(-time.Duration(hours) * time.Hour)

	pipeline := mongo.Pipeline{
		{{Key: "$match", Value: bson.M{
			"created_at": bson.M{"$gte": startTime},
			"event_type": "system_performance",
		}}},
		{{"$group", bson.M{
			"_id": bson.M{
				"hour": bson.M{"$dateToString": bson.M{
					"format": "%Y-%m-%d %H:00",
					"date":   "$created_at",
				}},
			},
			"avg_response_time": bson.M{"$avg": bson.M{"$toDouble": "$properties.response_time"}},
			"error_count": bson.M{"$sum": bson.M{
				"$cond": []interface{}{
					bson.M{"$eq": []interface{}{"$event_name", "error"}},
					1, 0,
				},
			}},
			"total_requests": bson.M{"$sum": 1},
		}}},
		{{"$sort", bson.M{"_id.hour": 1}}},
	}

	cursor, err := r.eventsCollection.Aggregate(ctx, pipeline)
	if err != nil {
		return nil, fmt.Errorf("failed to get system performance metrics: %w", err)
	}
	defer cursor.Close(ctx)

	var results []map[string]interface{}
	for cursor.Next(ctx) {
		var result struct {
			ID struct {
				Hour string `bson:"hour"`
			} `bson:"_id"`
			AvgResponseTime float64 `bson:"avg_response_time"`
			ErrorCount      int64   `bson:"error_count"`
			TotalRequests   int64   `bson:"total_requests"`
		}

		if err := cursor.Decode(&result); err != nil {
			return nil, fmt.Errorf("failed to decode performance metric: %w", err)
		}

		errorRate := float64(0)
		if result.TotalRequests > 0 {
			errorRate = float64(result.ErrorCount) / float64(result.TotalRequests) * 100
		}

		results = append(results, map[string]interface{}{
			"hour":              result.ID.Hour,
			"avg_response_time": result.AvgResponseTime,
			"error_count":       result.ErrorCount,
			"total_requests":    result.TotalRequests,
			"error_rate":        errorRate,
		})
	}

	return results, nil
}

func (r *analyticsRepository) GetAPIResponseTimes(ctx context.Context, hours int) ([]map[string]interface{}, error) {
	startTime := time.Now().Add(-time.Duration(hours) * time.Hour)

	pipeline := mongo.Pipeline{
		{{Key: "$match", Value: bson.M{
			"created_at": bson.M{"$gte": startTime},
			"event_type": "api_call",
		}}},
		{{"$group", bson.M{
			"_id":               "$properties.endpoint",
			"avg_response_time": bson.M{"$avg": bson.M{"$toDouble": "$properties.response_time"}},
			"max_response_time": bson.M{"$max": bson.M{"$toDouble": "$properties.response_time"}},
			"min_response_time": bson.M{"$min": bson.M{"$toDouble": "$properties.response_time"}},
			"total_calls":       bson.M{"$sum": 1},
		}}},
		{{"$sort", bson.M{"avg_response_time": -1}}},
	}

	cursor, err := r.eventsCollection.Aggregate(ctx, pipeline)
	if err != nil {
		return nil, fmt.Errorf("failed to get API response times: %w", err)
	}
	defer cursor.Close(ctx)

	var results []map[string]interface{}
	for cursor.Next(ctx) {
		var result struct {
			Endpoint        string  `bson:"_id"`
			AvgResponseTime float64 `bson:"avg_response_time"`
			MaxResponseTime float64 `bson:"max_response_time"`
			MinResponseTime float64 `bson:"min_response_time"`
			TotalCalls      int64   `bson:"total_calls"`
		}

		if err := cursor.Decode(&result); err != nil {
			return nil, fmt.Errorf("failed to decode API response time: %w", err)
		}

		results = append(results, map[string]interface{}{
			"endpoint":          result.Endpoint,
			"avg_response_time": result.AvgResponseTime,
			"max_response_time": result.MaxResponseTime,
			"min_response_time": result.MinResponseTime,
			"total_calls":       result.TotalCalls,
		})
	}

	return results, nil
}

func (r *analyticsRepository) GetErrorRates(ctx context.Context, hours int) ([]map[string]interface{}, error) {
	startTime := time.Now().Add(-time.Duration(hours) * time.Hour)

	pipeline := mongo.Pipeline{
		{{"$match", bson.M{
			"created_at": bson.M{"$gte": startTime},
			"event_type": "api_call",
		}}},
		{{"$group", bson.M{
			"_id":         "$properties.endpoint",
			"total_calls": bson.M{"$sum": 1},
			"error_calls": bson.M{"$sum": bson.M{
				"$cond": []interface{}{
					bson.M{"$gte": []interface{}{bson.M{"$toInt": "$properties.status_code"}, 400}},
					1, 0,
				},
			}},
		}}},
		{{"$project", bson.M{
			"endpoint":    "$_id",
			"total_calls": 1,
			"error_calls": 1,
			"error_rate": bson.M{
				"$multiply": []interface{}{
					bson.M{"$divide": []interface{}{"$error_calls", "$total_calls"}},
					100,
				},
			},
		}}},
		{{"$sort", bson.M{"error_rate": -1}}},
	}

	cursor, err := r.eventsCollection.Aggregate(ctx, pipeline)
	if err != nil {
		return nil, fmt.Errorf("failed to get error rates: %w", err)
	}
	defer cursor.Close(ctx)

	var results []map[string]interface{}
	for cursor.Next(ctx) {
		var result map[string]interface{}
		if err := cursor.Decode(&result); err != nil {
			return nil, fmt.Errorf("failed to decode error rate: %w", err)
		}
		results = append(results, result)
	}

	return results, nil
}

// Custom reports
func (r *analyticsRepository) GenerateCustomReport(ctx context.Context, reportType string, parameters map[string]interface{}, dateRange []time.Time) ([]map[string]interface{}, error) {
	// Implementation depends on specific report types
	// This is a simplified version
	switch reportType {
	case "ride_summary":
		return r.generateRideSummaryReport(ctx, parameters, dateRange)
	case "driver_performance":
		return r.generateDriverPerformanceReport(ctx, parameters, dateRange)
	case "revenue_analysis":
		return r.generateRevenueAnalysisReport(ctx, parameters, dateRange)
	default:
		return nil, fmt.Errorf("unsupported report type: %s", reportType)
	}
}

func (r *analyticsRepository) GetReportData(ctx context.Context, reportID string) (map[string]interface{}, error) {
	// This would typically fetch from a reports collection
	// For now, return a placeholder
	return map[string]interface{}{
		"report_id": reportID,
		"status":    "completed",
		"data":      "Report data would be here",
	}, nil
}

// Helper methods
func (r *analyticsRepository) findEventsWithFilter(ctx context.Context, filter bson.M, params *utils.PaginationParams) ([]*models.AnalyticsEvent, int64, error) {
	// Get total count
	total, err := r.eventsCollection.CountDocuments(ctx, filter)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to count events: %w", err)
	}

	// Get paginated results
	opts := params.GetSortOptions()
	cursor, err := r.eventsCollection.Find(ctx, filter, opts)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to find events: %w", err)
	}
	defer cursor.Close(ctx)

	var events []*models.AnalyticsEvent
	for cursor.Next(ctx) {
		var event models.AnalyticsEvent
		if err := cursor.Decode(&event); err != nil {
			return nil, 0, fmt.Errorf("failed to decode event: %w", err)
		}
		events = append(events, &event)
	}

	return events, total, nil
}

// Custom report generators
func (r *analyticsRepository) generateRideSummaryReport(ctx context.Context, parameters map[string]interface{}, dateRange []time.Time) ([]map[string]interface{}, error) {
	// Implementation for ride summary report
	return []map[string]interface{}{}, nil
}

func (r *analyticsRepository) generateDriverPerformanceReport(ctx context.Context, parameters map[string]interface{}, dateRange []time.Time) ([]map[string]interface{}, error) {
	// Implementation for driver performance report
	return []map[string]interface{}{}, nil
}

func (r *analyticsRepository) generateRevenueAnalysisReport(ctx context.Context, parameters map[string]interface{}, dateRange []time.Time) ([]map[string]interface{}, error) {
	// Implementation for revenue analysis report
	return []map[string]interface{}{}, nil
}
