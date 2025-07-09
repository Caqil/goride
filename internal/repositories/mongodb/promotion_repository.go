package mongodb

import (
	"context"
	"fmt"
	"strings"
	"time"

	"goride/internal/models"
	"goride/internal/repositories/interfaces"
	"goride/internal/utils"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

type promotionRepository struct {
	collection *mongo.Collection
	cache      CacheService
}

func NewPromotionRepository(db *mongo.Database, cache CacheService) interfaces.PromotionRepository {
	return &promotionRepository{
		collection: db.Collection("promotions"),
		cache:      cache,
	}
}

// Basic CRUD operations
func (r *promotionRepository) Create(ctx context.Context, promotion *models.Promotion) error {
	promotion.ID = primitive.NewObjectID()
	promotion.CreatedAt = time.Now()
	promotion.UpdatedAt = time.Now()

	// Ensure promotion code is uppercase
	promotion.Code = strings.ToUpper(promotion.Code)

	_, err := r.collection.InsertOne(ctx, promotion)
	if err != nil {
		return fmt.Errorf("failed to create promotion: %w", err)
	}

	// Cache active promotions
	if promotion.Status == models.PromotionStatusActive {
		r.cachePromotion(ctx, promotion)
	}

	return nil
}

func (r *promotionRepository) GetByID(ctx context.Context, id primitive.ObjectID) (*models.Promotion, error) {
	// Try cache first for active promotions
	if promotion := r.getPromotionFromCache(ctx, id.Hex()); promotion != nil {
		return promotion, nil
	}

	var promotion models.Promotion
	err := r.collection.FindOne(ctx, bson.M{"_id": id}).Decode(&promotion)
	if err != nil {
		if err == mongo.ErrNoDocuments {
			return nil, fmt.Errorf("promotion not found")
		}
		return nil, fmt.Errorf("failed to get promotion: %w", err)
	}

	// Cache active promotions
	if promotion.Status == models.PromotionStatusActive {
		r.cachePromotion(ctx, &promotion)
	}

	return &promotion, nil
}

func (r *promotionRepository) Update(ctx context.Context, id primitive.ObjectID, updates map[string]interface{}) error {
	updates["updated_at"] = time.Now()

	// Ensure promotion code is uppercase if being updated
	if code, exists := updates["code"]; exists {
		if codeStr, ok := code.(string); ok {
			updates["code"] = strings.ToUpper(codeStr)
		}
	}

	_, err := r.collection.UpdateOne(
		ctx,
		bson.M{"_id": id},
		bson.M{"$set": updates},
	)
	if err != nil {
		return fmt.Errorf("failed to update promotion: %w", err)
	}

	// Invalidate cache
	r.invalidatePromotionCache(ctx, id.Hex())

	return nil
}

func (r *promotionRepository) Delete(ctx context.Context, id primitive.ObjectID) error {
	_, err := r.collection.DeleteOne(ctx, bson.M{"_id": id})
	if err != nil {
		return fmt.Errorf("failed to delete promotion: %w", err)
	}

	// Invalidate cache
	r.invalidatePromotionCache(ctx, id.Hex())

	return nil
}

// Code operations
func (r *promotionRepository) GetByCode(ctx context.Context, code string) (*models.Promotion, error) {
	code = strings.ToUpper(code)

	// Try cache first
	cacheKey := fmt.Sprintf("promotion_code_%s", code)
	if r.cache != nil {
		var promotion models.Promotion
		if err := r.cache.Get(ctx, cacheKey, &promotion); err == nil {
			return &promotion, nil
		}
	}

	var promotion models.Promotion
	err := r.collection.FindOne(ctx, bson.M{"code": code}).Decode(&promotion)
	if err != nil {
		if err == mongo.ErrNoDocuments {
			return nil, fmt.Errorf("promotion not found with code")
		}
		return nil, fmt.Errorf("failed to get promotion by code: %w", err)
	}

	// Cache active promotions
	if promotion.Status == models.PromotionStatusActive {
		r.cache.Set(ctx, cacheKey, promotion, 30*time.Minute)
	}

	return &promotion, nil
}

func (r *promotionRepository) ValidateCode(ctx context.Context, code string, userID primitive.ObjectID, rideType string) (*models.Promotion, error) {
	promotion, err := r.GetByCode(ctx, code)
	if err != nil {
		return nil, err
	}

	// Check if promotion is active
	if promotion.Status != models.PromotionStatusActive {
		return nil, fmt.Errorf("promotion is not active")
	}

	// Check validity dates
	now := time.Now()
	if now.Before(promotion.ValidFrom) {
		return nil, fmt.Errorf("promotion is not yet valid")
	}
	if now.After(promotion.ValidUntil) {
		return nil, fmt.Errorf("promotion has expired")
	}

	// Check usage limits
	if promotion.UsageLimit > 0 && promotion.UsedCount >= promotion.UsageLimit {
		return nil, fmt.Errorf("promotion usage limit reached")
	}

	// Check user-specific usage limit
	if promotion.UserLimit > 0 {
		userUsage, err := r.getUserPromotionUsage(ctx, userID, promotion.ID)
		if err != nil {
			return nil, fmt.Errorf("failed to check user promotion usage: %w", err)
		}
		if userUsage >= int64(promotion.UserLimit) {
			return nil, fmt.Errorf("user promotion usage limit reached")
		}
	}

	// Check applicable user types
	if len(promotion.ApplicableUserTypes) > 0 {
		// Get user type from users collection
		userCollection := r.collection.Database().Collection("users")
		var user struct {
			UserType models.UserType `bson:"user_type"`
		}
		err := userCollection.FindOne(ctx, bson.M{"_id": userID}).Decode(&user)
		if err != nil {
			return nil, fmt.Errorf("failed to get user type: %w", err)
		}

		userTypeValid := false
		for _, applicableType := range promotion.ApplicableUserTypes {
			if applicableType == user.UserType {
				userTypeValid = true
				break
			}
		}
		if !userTypeValid {
			return nil, fmt.Errorf("promotion not applicable to user type")
		}
	}

	// Check applicable ride types
	if len(promotion.ApplicableRideTypes) > 0 && rideType != "" {
		rideTypeValid := false
		for _, applicableRideType := range promotion.ApplicableRideTypes {
			if string(applicableRideType) == rideType {
				rideTypeValid = true
				break
			}
		}
		if !rideTypeValid {
			return nil, fmt.Errorf("promotion not applicable to ride type")
		}
	}

	// Check if it's first ride only
	if promotion.IsFirstRideOnly {
		rideCount, err := r.getUserRideCount(ctx, userID)
		if err != nil {
			return nil, fmt.Errorf("failed to check user ride count: %w", err)
		}
		if rideCount > 0 {
			return nil, fmt.Errorf("promotion is only for first ride")
		}
	}

	return promotion, nil
}

// Status operations
func (r *promotionRepository) GetActivePromotions(ctx context.Context, params *utils.PaginationParams) ([]*models.Promotion, int64, error) {
	filter := bson.M{
		"status":      models.PromotionStatusActive,
		"valid_from":  bson.M{"$lte": time.Now()},
		"valid_until": bson.M{"$gte": time.Now()},
	}
	return r.findPromotionsWithFilter(ctx, filter, params)
}

func (r *promotionRepository) GetByStatus(ctx context.Context, status models.PromotionStatus, params *utils.PaginationParams) ([]*models.Promotion, int64, error) {
	filter := bson.M{"status": status}
	return r.findPromotionsWithFilter(ctx, filter, params)
}

func (r *promotionRepository) UpdateStatus(ctx context.Context, id primitive.ObjectID, status models.PromotionStatus) error {
	updates := map[string]interface{}{
		"status": status,
	}
	return r.Update(ctx, id, updates)
}

// Usage tracking
func (r *promotionRepository) IncrementUsage(ctx context.Context, id primitive.ObjectID) error {
	_, err := r.collection.UpdateOne(
		ctx,
		bson.M{"_id": id},
		bson.M{
			"$inc": bson.M{"used_count": 1},
			"$set": bson.M{"updated_at": time.Now()},
		},
	)
	if err != nil {
		return fmt.Errorf("failed to increment promotion usage: %w", err)
	}

	// Invalidate cache
	r.invalidatePromotionCache(ctx, id.Hex())

	return nil
}

func (r *promotionRepository) GetUsageStats(ctx context.Context, id primitive.ObjectID) (map[string]interface{}, error) {
	promotion, err := r.GetByID(ctx, id)
	if err != nil {
		return nil, err
	}

	// Get daily usage over the last 30 days
	thirtyDaysAgo := time.Now().AddDate(0, 0, -30)

	pipeline := mongo.Pipeline{
		{{"$match", bson.M{
			"promotion_id": id,
			"created_at":   bson.M{"$gte": thirtyDaysAgo},
		}}},
		{{"$group", bson.M{
			"_id": bson.M{
				"date": bson.M{
					"$dateToString": bson.M{
						"format": "%Y-%m-%d",
						"date":   "$created_at",
					},
				},
			},
			"usage_count": bson.M{"$sum": 1},
		}}},
		{{"$sort", bson.D{{Key: "_id.date", Value: 1}}}},
	}

	ridesCollection := r.collection.Database().Collection("rides")
	cursor, err := ridesCollection.Aggregate(ctx, pipeline)
	if err != nil {
		return nil, fmt.Errorf("failed to get usage stats: %w", err)
	}
	defer cursor.Close(ctx)

	dailyUsage := make([]map[string]interface{}, 0)
	for cursor.Next(ctx) {
		var result struct {
			ID struct {
				Date string `bson:"date"`
			} `bson:"_id"`
			UsageCount int64 `bson:"usage_count"`
		}

		if err := cursor.Decode(&result); err != nil {
			return nil, fmt.Errorf("failed to decode usage stats: %w", err)
		}

		dailyUsage = append(dailyUsage, map[string]interface{}{
			"date":        result.ID.Date,
			"usage_count": result.UsageCount,
		})
	}

	usageRate := float64(0)
	if promotion.UsageLimit > 0 {
		usageRate = float64(promotion.UsedCount) / float64(promotion.UsageLimit) * 100
	}

	return map[string]interface{}{
		"promotion_id":   id,
		"total_used":     promotion.UsedCount,
		"usage_limit":    promotion.UsageLimit,
		"usage_rate":     usageRate,
		"daily_usage":    dailyUsage,
		"is_active":      promotion.Status == models.PromotionStatusActive,
		"days_remaining": int(promotion.ValidUntil.Sub(time.Now()).Hours() / 24),
	}, nil
}

// Type and applicability
func (r *promotionRepository) GetByType(ctx context.Context, promotionType models.PromotionType, params *utils.PaginationParams) ([]*models.Promotion, int64, error) {
	filter := bson.M{"type": promotionType}
	return r.findPromotionsWithFilter(ctx, filter, params)
}

func (r *promotionRepository) GetApplicablePromotions(ctx context.Context, userType models.UserType, rideType string, amount float64) ([]*models.Promotion, error) {
	filter := bson.M{
		"status":          models.PromotionStatusActive,
		"valid_from":      bson.M{"$lte": time.Now()},
		"valid_until":     bson.M{"$gte": time.Now()},
		"min_ride_amount": bson.M{"$lte": amount},
		"$or": []bson.M{
			{"usage_limit": 0}, // No limit
			{"$expr": bson.M{"$lt": []interface{}{"$used_count", "$usage_limit"}}},
		},
	}

	// Check user type applicability
	if userType != "" {
		filter["$or"] = append(filter["$or"].([]bson.M), bson.M{
			"applicable_user_types": bson.M{"$size": 0}, // No restrictions
		}, bson.M{
			"applicable_user_types": bson.M{"$in": []models.UserType{userType}},
		})
	}

	// Check ride type applicability
	if rideType != "" {
		filter["$or"] = append(filter["$or"].([]bson.M), bson.M{
			"applicable_ride_types": bson.M{"$size": 0}, // No restrictions
		}, bson.M{
			"applicable_ride_types": bson.M{"$in": []string{rideType}},
		})
	}

	cursor, err := r.collection.Find(ctx, filter, options.Find().SetSort(bson.D{{Key: "created_at", Value: -1}}))
	if err != nil {
		return nil, fmt.Errorf("failed to find applicable promotions: %w", err)
	}
	defer cursor.Close(ctx)

	var promotions []*models.Promotion
	for cursor.Next(ctx) {
		var promotion models.Promotion
		if err := cursor.Decode(&promotion); err != nil {
			return nil, fmt.Errorf("failed to decode promotion: %w", err)
		}
		promotions = append(promotions, &promotion)
	}

	return promotions, nil
}

// Time-based queries
func (r *promotionRepository) GetValidPromotions(ctx context.Context, checkTime time.Time) ([]*models.Promotion, error) {
	filter := bson.M{
		"status":      models.PromotionStatusActive,
		"valid_from":  bson.M{"$lte": checkTime},
		"valid_until": bson.M{"$gte": checkTime},
	}

	cursor, err := r.collection.Find(ctx, filter)
	if err != nil {
		return nil, fmt.Errorf("failed to find valid promotions: %w", err)
	}
	defer cursor.Close(ctx)

	var promotions []*models.Promotion
	for cursor.Next(ctx) {
		var promotion models.Promotion
		if err := cursor.Decode(&promotion); err != nil {
			return nil, fmt.Errorf("failed to decode promotion: %w", err)
		}
		promotions = append(promotions, &promotion)
	}

	return promotions, nil
}

func (r *promotionRepository) GetExpiredPromotions(ctx context.Context) ([]*models.Promotion, error) {
	filter := bson.M{
		"status":      models.PromotionStatusActive,
		"valid_until": bson.M{"$lt": time.Now()},
	}

	cursor, err := r.collection.Find(ctx, filter)
	if err != nil {
		return nil, fmt.Errorf("failed to find expired promotions: %w", err)
	}
	defer cursor.Close(ctx)

	var promotions []*models.Promotion
	for cursor.Next(ctx) {
		var promotion models.Promotion
		if err := cursor.Decode(&promotion); err != nil {
			return nil, fmt.Errorf("failed to decode promotion: %w", err)
		}
		promotions = append(promotions, &promotion)
	}

	return promotions, nil
}

func (r *promotionRepository) GetPromotionsByDateRange(ctx context.Context, startDate, endDate time.Time, params *utils.PaginationParams) ([]*models.Promotion, int64, error) {
	filter := bson.M{
		"created_at": bson.M{
			"$gte": startDate,
			"$lte": endDate,
		},
	}
	return r.findPromotionsWithFilter(ctx, filter, params)
}

// Search and filtering
func (r *promotionRepository) SearchPromotions(ctx context.Context, query string, params *utils.PaginationParams) ([]*models.Promotion, int64, error) {
	filter := bson.M{
		"$or": []bson.M{
			{"code": bson.M{"$regex": query, "$options": "i"}},
			{"title": bson.M{"$regex": query, "$options": "i"}},
			{"description": bson.M{"$regex": query, "$options": "i"}},
		},
	}
	return r.findPromotionsWithFilter(ctx, filter, params)
}

func (r *promotionRepository) GetPromotionsForCity(ctx context.Context, city string, params *utils.PaginationParams) ([]*models.Promotion, int64, error) {
	filter := bson.M{
		"$or": []bson.M{
			{"target_cities": bson.M{"$size": 0}}, // No city restrictions
			{"target_cities": bson.M{"$in": []string{city}}},
		},
	}
	return r.findPromotionsWithFilter(ctx, filter, params)
}

// Analytics
func (r *promotionRepository) GetPromotionStats(ctx context.Context, days int) (map[string]interface{}, error) {
	startDate := time.Now().AddDate(0, 0, -days)

	pipeline := mongo.Pipeline{
		{{"$match", bson.M{"created_at": bson.M{"$gte": startDate}}}},
		{{"$group", bson.M{
			"_id":         "$status",
			"count":       bson.M{"$sum": 1},
			"total_usage": bson.M{"$sum": "$used_count"},
			"avg_usage":   bson.M{"$avg": "$used_count"},
		}}},
	}

	cursor, err := r.collection.Aggregate(ctx, pipeline)
	if err != nil {
		return nil, fmt.Errorf("failed to get promotion stats: %w", err)
	}
	defer cursor.Close(ctx)

	stats := make(map[string]interface{})
	var totalPromotions int64

	for cursor.Next(ctx) {
		var result struct {
			ID         models.PromotionStatus `bson:"_id"`
			Count      int64                  `bson:"count"`
			TotalUsage int64                  `bson:"total_usage"`
			AvgUsage   float64                `bson:"avg_usage"`
		}

		if err := cursor.Decode(&result); err != nil {
			return nil, fmt.Errorf("failed to decode promotion stats: %w", err)
		}

		stats[string(result.ID)] = map[string]interface{}{
			"count":       result.Count,
			"total_usage": result.TotalUsage,
			"avg_usage":   result.AvgUsage,
		}

		totalPromotions += result.Count
	}

	stats["summary"] = map[string]interface{}{
		"total_promotions": totalPromotions,
		"period_days":      days,
		"start_date":       startDate,
		"end_date":         time.Now(),
	}

	return stats, nil
}

func (r *promotionRepository) GetTopPromotions(ctx context.Context, limit int, days int) ([]*models.Promotion, error) {
	startDate := time.Now().AddDate(0, 0, -days)

	filter := bson.M{
		"created_at": bson.M{"$gte": startDate},
		"used_count": bson.M{"$gt": 0},
	}

	opts := options.Find().
		SetSort(bson.D{{Key: "used_count", Value: -1}}).
		SetLimit(int64(limit))

	cursor, err := r.collection.Find(ctx, filter, opts)
	if err != nil {
		return nil, fmt.Errorf("failed to find top promotions: %w", err)
	}
	defer cursor.Close(ctx)

	var promotions []*models.Promotion
	for cursor.Next(ctx) {
		var promotion models.Promotion
		if err := cursor.Decode(&promotion); err != nil {
			return nil, fmt.Errorf("failed to decode promotion: %w", err)
		}
		promotions = append(promotions, &promotion)
	}

	return promotions, nil
}

func (r *promotionRepository) GetPromotionEffectiveness(ctx context.Context, id primitive.ObjectID, days int) (map[string]interface{}, error) {
	startDate := time.Now().AddDate(0, 0, -days)

	// Get promotion details
	promotion, err := r.GetByID(ctx, id)
	if err != nil {
		return nil, err
	}

	// Get rides that used this promotion
	ridesCollection := r.collection.Database().Collection("rides")

	pipeline := mongo.Pipeline{
		{{"$match", bson.M{
			"promotion_id": id,
			"created_at":   bson.M{"$gte": startDate},
		}}},
		{{"$group", bson.M{
			"_id":            nil,
			"total_rides":    bson.M{"$sum": 1},
			"total_discount": bson.M{"$sum": "$discount_amount"},
			"total_revenue":  bson.M{"$sum": "$fare_amount"},
			"avg_ride_value": bson.M{"$avg": "$fare_amount"},
		}}},
	}

	cursor, err := ridesCollection.Aggregate(ctx, pipeline)
	if err != nil {
		return nil, fmt.Errorf("failed to get promotion effectiveness: %w", err)
	}
	defer cursor.Close(ctx)

	var result struct {
		TotalRides    int64   `bson:"total_rides"`
		TotalDiscount float64 `bson:"total_discount"`
		TotalRevenue  float64 `bson:"total_revenue"`
		AvgRideValue  float64 `bson:"avg_ride_value"`
	}

	if cursor.Next(ctx) {
		if err := cursor.Decode(&result); err != nil {
			return nil, fmt.Errorf("failed to decode promotion effectiveness: %w", err)
		}
	}

	// Calculate ROI and other metrics
	roi := float64(0)
	if result.TotalDiscount > 0 {
		roi = (result.TotalRevenue - result.TotalDiscount) / result.TotalDiscount * 100
	}

	conversionRate := float64(0)
	if promotion.UsageLimit > 0 {
		conversionRate = float64(result.TotalRides) / float64(promotion.UsageLimit) * 100
	}

	return map[string]interface{}{
		"promotion_id":    id,
		"promotion_code":  promotion.Code,
		"total_rides":     result.TotalRides,
		"total_discount":  result.TotalDiscount,
		"total_revenue":   result.TotalRevenue,
		"avg_ride_value":  result.AvgRideValue,
		"roi_percentage":  roi,
		"conversion_rate": conversionRate,
		"cost_per_ride":   result.TotalDiscount / float64(result.TotalRides),
		"period_days":     days,
		"start_date":      startDate,
		"end_date":        time.Now(),
	}, nil
}

// Helper methods
func (r *promotionRepository) findPromotionsWithFilter(ctx context.Context, filter bson.M, params *utils.PaginationParams) ([]*models.Promotion, int64, error) {
	// Add search filter if provided
	if params.Search != "" {
		searchFields := []string{"code", "title", "description"}
		searchFilter := params.GetSearchFilter(searchFields)
		if len(searchFilter) > 0 {
			filter = bson.M{
				"$and": []bson.M{filter, searchFilter},
			}
		}
	}

	// Get total count
	total, err := r.collection.CountDocuments(ctx, filter)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to count promotions: %w", err)
	}

	// Get paginated results
	opts := params.GetSortOptions()
	cursor, err := r.collection.Find(ctx, filter, opts)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to find promotions: %w", err)
	}
	defer cursor.Close(ctx)

	var promotions []*models.Promotion
	for cursor.Next(ctx) {
		var promotion models.Promotion
		if err := cursor.Decode(&promotion); err != nil {
			return nil, 0, fmt.Errorf("failed to decode promotion: %w", err)
		}
		promotions = append(promotions, &promotion)
	}

	return promotions, total, nil
}

func (r *promotionRepository) getUserPromotionUsage(ctx context.Context, userID, promotionID primitive.ObjectID) (int64, error) {
	// Count how many times this user has used this promotion
	ridesCollection := r.collection.Database().Collection("rides")

	count, err := ridesCollection.CountDocuments(ctx, bson.M{
		"rider_id":     userID,
		"promotion_id": promotionID,
	})
	if err != nil {
		return 0, fmt.Errorf("failed to count user promotion usage: %w", err)
	}

	return count, nil
}

func (r *promotionRepository) getUserRideCount(ctx context.Context, userID primitive.ObjectID) (int64, error) {
	ridesCollection := r.collection.Database().Collection("rides")

	count, err := ridesCollection.CountDocuments(ctx, bson.M{
		"rider_id": userID,
		"status":   bson.M{"$in": []string{"completed", "ongoing"}},
	})
	if err != nil {
		return 0, fmt.Errorf("failed to count user rides: %w", err)
	}

	return count, nil
}

// Cache operations
func (r *promotionRepository) cachePromotion(ctx context.Context, promotion *models.Promotion) {
	if r.cache != nil && promotion.Status == models.PromotionStatusActive {
		cacheKey := fmt.Sprintf("promotion:%s", promotion.ID.Hex())
		r.cache.Set(ctx, cacheKey, promotion, 30*time.Minute)

		// Also cache by code
		codeKey := fmt.Sprintf("promotion_code_%s", promotion.Code)
		r.cache.Set(ctx, codeKey, promotion, 30*time.Minute)
	}
}

func (r *promotionRepository) getPromotionFromCache(ctx context.Context, promotionID string) *models.Promotion {
	if r.cache == nil {
		return nil
	}

	cacheKey := fmt.Sprintf("promotion:%s", promotionID)
	var promotion models.Promotion
	err := r.cache.Get(ctx, cacheKey, &promotion)
	if err != nil {
		return nil
	}

	return &promotion
}

func (r *promotionRepository) invalidatePromotionCache(ctx context.Context, promotionID string) {
	if r.cache != nil {
		cacheKey := fmt.Sprintf("promotion:%s", promotionID)
		r.cache.Delete(ctx, cacheKey)

		// Note: We can't easily invalidate the code cache without knowing the code
		// This is a trade-off for performance vs cache consistency
	}
}
