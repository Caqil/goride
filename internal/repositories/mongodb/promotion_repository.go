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

	// Cache the result if active
	if r.cache != nil && promotion.Status == models.PromotionStatusActive {
		r.cache.Set(ctx, cacheKey, &promotion, 30*time.Minute)
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

	// Check if promotion is still valid (not expired)
	now := time.Now()
	if promotion.StartDate.After(now) {
		return nil, fmt.Errorf("promotion has not started yet")
	}
	if promotion.EndDate.Before(now) {
		return nil, fmt.Errorf("promotion has expired")
	}

	// Check usage limits
	if promotion.MaxUses > 0 && promotion.CurrentUses >= promotion.MaxUses {
		return nil, fmt.Errorf("promotion usage limit reached")
	}

	if promotion.MaxUsesPerUser > 0 {
		userUsage, err := r.getUserPromotionUsage(ctx, userID, promotion.ID)
		if err != nil {
			return nil, fmt.Errorf("failed to check user promotion usage: %w", err)
		}
		if userUsage >= promotion.MaxUsesPerUser {
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
			if applicableRideType == rideType {
				rideTypeValid = true
				break
			}
		}
		if !rideTypeValid {
			return nil, fmt.Errorf("promotion not applicable to ride type")
		}
	}

	// Check minimum order amount
	// Note: This would typically be checked against the actual ride fare
	// For now, we just validate the promotion structure

	return promotion, nil
}

// Status operations
func (r *promotionRepository) GetActivePromotions(ctx context.Context, params *utils.PaginationParams) ([]*models.Promotion, int64, error) {
	filter := bson.M{
		"status":     models.PromotionStatusActive,
		"start_date": bson.M{"$lte": time.Now()},
		"end_date":   bson.M{"$gte": time.Now()},
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

	if status == models.PromotionStatusInactive {
		updates["deactivated_at"] = time.Now()
	}

	return r.Update(ctx, id, updates)
}

// Usage tracking
func (r *promotionRepository) IncrementUsage(ctx context.Context, id primitive.ObjectID) error {
	_, err := r.collection.UpdateOne(
		ctx,
		bson.M{"_id": id},
		bson.M{
			"$inc": bson.M{"current_uses": 1},
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

	// Get usage by day for the last 30 days
	startDate := time.Now().AddDate(0, 0, -30)

	// Query ride collection for promotion usage
	ridesCollection := r.collection.Database().Collection("rides")

	pipeline := mongo.Pipeline{
		{{"$match", bson.M{
			"promotion_id": id,
			"created_at":   bson.M{"$gte": startDate},
		}}},
		{{"$group", bson.M{
			"_id": bson.M{
				"date": bson.M{"$dateToString": bson.M{
					"format": "%Y-%m-%d",
					"date":   "$created_at",
				}},
			},
			"usage_count":    bson.M{"$sum": 1},
			"total_discount": bson.M{"$sum": "$promotion_discount"},
		}}},
		{{"$sort", bson.M{"_id.date": 1}}},
	}

	cursor, err := ridesCollection.Aggregate(ctx, pipeline)
	if err != nil {
		return nil, fmt.Errorf("failed to get promotion usage stats: %w", err)
	}
	defer cursor.Close(ctx)

	var dailyUsage []map[string]interface{}
	var totalDiscountGiven float64

	for cursor.Next(ctx) {
		var result struct {
			ID struct {
				Date string `bson:"date"`
			} `bson:"_id"`
			UsageCount    int64   `bson:"usage_count"`
			TotalDiscount float64 `bson:"total_discount"`
		}

		if err := cursor.Decode(&result); err != nil {
			return nil, fmt.Errorf("failed to decode usage stats: %w", err)
		}

		dailyUsage = append(dailyUsage, map[string]interface{}{
			"date":           result.ID.Date,
			"usage_count":    result.UsageCount,
			"total_discount": result.TotalDiscount,
		})

		totalDiscountGiven += result.TotalDiscount
	}

	usageRate := float64(0)
	if promotion.MaxUses > 0 {
		usageRate = float64(promotion.CurrentUses) / float64(promotion.MaxUses) * 100
	}

	return map[string]interface{}{
		"promotion_id":         id,
		"current_uses":         promotion.CurrentUses,
		"max_uses":             promotion.MaxUses,
		"usage_rate":           usageRate,
		"total_discount_given": totalDiscountGiven,
		"daily_usage":          dailyUsage,
		"is_active":            promotion.Status == models.PromotionStatusActive,
		"start_date":           promotion.StartDate,
		"end_date":             promotion.EndDate,
	}, nil
}

// Type and applicability
func (r *promotionRepository) GetByType(ctx context.Context, promotionType models.PromotionType, params *utils.PaginationParams) ([]*models.Promotion, int64, error) {
	filter := bson.M{"type": promotionType}
	return r.findPromotionsWithFilter(ctx, filter, params)
}

func (r *promotionRepository) GetApplicablePromotions(ctx context.Context, userType models.UserType, rideType string, amount float64) ([]*models.Promotion, error) {
	filter := bson.M{
		"status":     models.PromotionStatusActive,
		"start_date": bson.M{"$lte": time.Now()},
		"end_date":   bson.M{"$gte": time.Now()},
		"$or": []bson.M{
			{"max_uses": 0}, // Unlimited
			{"$expr": bson.M{"$lt": []interface{}{"$current_uses", "$max_uses"}}}, // Usage available
		},
		"$or": []bson.M{
			{"applicable_user_types": bson.M{"$size": 0}}, // No restrictions
			{"applicable_user_types": bson.M{"$in": []models.UserType{userType}}},
		},
		"$or": []bson.M{
			{"minimum_order_amount": bson.M{"$lte": amount}},
			{"minimum_order_amount": 0}, // No minimum
		},
	}

	// Add ride type filter if specified
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
		"status":     models.PromotionStatusActive,
		"start_date": bson.M{"$lte": checkTime},
		"end_date":   bson.M{"$gte": checkTime},
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
		"status":   models.PromotionStatusActive,
		"end_date": bson.M{"$lt": time.Now()},
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
		"$or": []bson.M{
			{
				"start_date": bson.M{
					"$gte": startDate,
					"$lte": endDate,
				},
			},
			{
				"end_date": bson.M{
					"$gte": startDate,
					"$lte": endDate,
				},
			},
			{
				"start_date": bson.M{"$lte": startDate},
				"end_date":   bson.M{"$gte": endDate},
			},
		},
	}
	return r.findPromotionsWithFilter(ctx, filter, params)
}

// Search and filtering
func (r *promotionRepository) SearchPromotions(ctx context.Context, query string, params *utils.PaginationParams) ([]*models.Promotion, int64, error) {
	filter := bson.M{
		"$or": []bson.M{
			{"name": bson.M{"$regex": query, "$options": "i"}},
			{"code": bson.M{"$regex": strings.ToUpper(query), "$options": "i"}},
			{"description": bson.M{"$regex": query, "$options": "i"}},
		},
	}
	return r.findPromotionsWithFilter(ctx, filter, params)
}

func (r *promotionRepository) GetPromotionsForCity(ctx context.Context, city string, params *utils.PaginationParams) ([]*models.Promotion, int64, error) {
	filter := bson.M{
		"$or": []bson.M{
			{"applicable_cities": bson.M{"$size": 0}}, // No city restrictions
			{"applicable_cities": bson.M{"$in": []string{city}}},
		},
		"status": models.PromotionStatusActive,
	}
	return r.findPromotionsWithFilter(ctx, filter, params)
}

// Analytics
func (r *promotionRepository) GetPromotionStats(ctx context.Context, days int) (map[string]interface{}, error) {
	startDate := time.Now().AddDate(0, 0, -days)

	// Total promotions
	totalPromotions, err := r.collection.CountDocuments(ctx, bson.M{
		"created_at": bson.M{"$gte": startDate},
	})
	if err != nil {
		return nil, fmt.Errorf("failed to count total promotions: %w", err)
	}

	// Active promotions
	activePromotions, err := r.collection.CountDocuments(ctx, bson.M{
		"status":     models.PromotionStatusActive,
		"start_date": bson.M{"$lte": time.Now()},
		"end_date":   bson.M{"$gte": time.Now()},
	})
	if err != nil {
		return nil, fmt.Errorf("failed to count active promotions: %w", err)
	}

	// Promotions by type
	typePipeline := mongo.Pipeline{
		{{"$match", bson.M{
			"created_at": bson.M{"$gte": startDate},
		}}},
		{{"$group", bson.M{
			"_id":   "$type",
			"count": bson.M{"$sum": 1},
		}}},
	}

	cursor, err := r.collection.Aggregate(ctx, typePipeline)
	if err != nil {
		return nil, fmt.Errorf("failed to get promotions by type: %w", err)
	}
	defer cursor.Close(ctx)

	typeCounts := make(map[string]int64)
	for cursor.Next(ctx) {
		var result struct {
			Type  models.PromotionType `bson:"_id"`
			Count int64                `bson:"count"`
		}

		if err := cursor.Decode(&result); err != nil {
			return nil, fmt.Errorf("failed to decode type count: %w", err)
		}

		typeCounts[string(result.Type)] = result.Count
	}

	return map[string]interface{}{
		"total_promotions":  totalPromotions,
		"active_promotions": activePromotions,
		"type_counts":       typeCounts,
		"period_days":       days,
		"start_date":        startDate,
		"end_date":          time.Now(),
	}, nil
}

func (r *promotionRepository) GetTopPromotions(ctx context.Context, limit int, days int) ([]*models.Promotion, error) {
	startDate := time.Now().AddDate(0, 0, -days)

	// Get promotions used in rides during the period
	ridesCollection := r.collection.Database().Collection("rides")

	pipeline := mongo.Pipeline{
		{{"$match", bson.M{
			"promotion_id": bson.M{"$ne": nil},
			"created_at":   bson.M{"$gte": startDate},
		}}},
		{{"$group", bson.M{
			"_id":            "$promotion_id",
			"usage_count":    bson.M{"$sum": 1},
			"total_discount": bson.M{"$sum": "$promotion_discount"},
		}}},
		{{"$sort", bson.M{"usage_count": -1}}},
		{{"$limit", limit}},
	}

	cursor, err := ridesCollection.Aggregate(ctx, pipeline)
	if err != nil {
		return nil, fmt.Errorf("failed to get top promotions: %w", err)
	}
	defer cursor.Close(ctx)

	var promotionIDs []primitive.ObjectID
	usageMap := make(map[string]map[string]interface{})

	for cursor.Next(ctx) {
		var result struct {
			PromotionID   primitive.ObjectID `bson:"_id"`
			UsageCount    int64              `bson:"usage_count"`
			TotalDiscount float64            `bson:"total_discount"`
		}

		if err := cursor.Decode(&result); err != nil {
			return nil, fmt.Errorf("failed to decode top promotion: %w", err)
		}

		promotionIDs = append(promotionIDs, result.PromotionID)
		usageMap[result.PromotionID.Hex()] = map[string]interface{}{
			"usage_count":    result.UsageCount,
			"total_discount": result.TotalDiscount,
		}
	}

	// Get promotion details
	promotions := make([]*models.Promotion, 0)
	for _, id := range promotionIDs {
		promotion, err := r.GetByID(ctx, id)
		if err == nil {
			// Add usage stats to promotion
			if stats, exists := usageMap[id.Hex()]; exists {
				promotion.Metadata = stats
			}
			promotions = append(promotions, promotion)
		}
	}

	return promotions, nil
}

func (r *promotionRepository) GetPromotionEffectiveness(ctx context.Context, id primitive.ObjectID, days int) (map[string]interface{}, error) {
	startDate := time.Now().AddDate(0, 0, -days)

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
			"total_discount": bson.M{"$sum": "$promotion_discount"},
			"total_fare":     bson.M{"$sum": "$fare.total"},
			"avg_discount":   bson.M{"$avg": "$promotion_discount"},
			"unique_users":   bson.M{"$addToSet": "$rider_id"},
		}}},
	}

	cursor, err := ridesCollection.Aggregate(ctx, pipeline)
	if err != nil {
		return nil, fmt.Errorf("failed to get promotion effectiveness: %w", err)
	}
	defer cursor.Close(ctx)

	var result struct {
		TotalRides    int64                `bson:"total_rides"`
		TotalDiscount float64              `bson:"total_discount"`
		TotalFare     float64              `bson:"total_fare"`
		AvgDiscount   float64              `bson:"avg_discount"`
		UniqueUsers   []primitive.ObjectID `bson:"unique_users"`
	}

	if cursor.Next(ctx) {
		if err := cursor.Decode(&result); err != nil {
			return nil, fmt.Errorf("failed to decode effectiveness stats: %w", err)
		}
	}

	// Calculate effectiveness metrics
	discountRate := float64(0)
	if result.TotalFare > 0 {
		discountRate = (result.TotalDiscount / result.TotalFare) * 100
	}

	return map[string]interface{}{
		"promotion_id":   id,
		"promotion_name": promotion.Name,
		"promotion_code": promotion.Code,
		"total_rides":    result.TotalRides,
		"unique_users":   len(result.UniqueUsers),
		"total_discount": result.TotalDiscount,
		"avg_discount":   result.AvgDiscount,
		"discount_rate":  discountRate,
		"cost_per_ride":  result.TotalDiscount / float64(utils.Max(int(result.TotalRides), 1)),
		"period_days":    days,
		"start_date":     startDate,
		"end_date":       time.Now(),
	}, nil
}

// Helper methods
func (r *promotionRepository) findPromotionsWithFilter(ctx context.Context, filter bson.M, params *utils.PaginationParams) ([]*models.Promotion, int64, error) {
	// Add search filter if provided
	if params.Search != "" {
		searchFields := []string{"name", "code", "description"}
		filter = bson.M{
			"$and": []bson.M{
				filter,
				params.GetSearchFilter(searchFields),
			},
		}
	}

	// Get total count
	total, err := r.collection.CountDocuments(ctx, filter)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to count promotions: %w", err)
	}

	// Get paginated results
	opts := params.GetSortOptions()
	// Default sort by created_at descending for promotions
	if params.Sort == "created_at" || params.Sort == "" {
		opts.SetSort(bson.D{{Key: "created_at", Value: -1}})
	}

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
