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

type riderRepository struct {
	collection *mongo.Collection
	cache      CacheService
}

func NewRiderRepository(db *mongo.Database, cache CacheService) interfaces.RiderRepository {
	return &riderRepository{
		collection: db.Collection("riders"),
		cache:      cache,
	}
}

// Basic CRUD operations
func (r *riderRepository) Create(ctx context.Context, rider *models.Rider) error {
	rider.ID = primitive.NewObjectID()
	rider.CreatedAt = time.Now()
	rider.UpdatedAt = time.Now()

	_, err := r.collection.InsertOne(ctx, rider)
	if err != nil {
		return fmt.Errorf("failed to create rider: %w", err)
	}

	// Cache the rider
	r.cacheRider(ctx, rider)

	return nil
}

func (r *riderRepository) GetByID(ctx context.Context, id primitive.ObjectID) (*models.Rider, error) {
	// Try cache first
	if rider := r.getRiderFromCache(ctx, id.Hex()); rider != nil {
		return rider, nil
	}

	var rider models.Rider
	err := r.collection.FindOne(ctx, bson.M{"_id": id}).Decode(&rider)
	if err != nil {
		if err == mongo.ErrNoDocuments {
			return nil, fmt.Errorf("rider not found")
		}
		return nil, fmt.Errorf("failed to get rider: %w", err)
	}

	// Cache the result
	r.cacheRider(ctx, &rider)

	return &rider, nil
}

func (r *riderRepository) GetByUserID(ctx context.Context, userID primitive.ObjectID) (*models.Rider, error) {
	// Try cache first
	cacheKey := fmt.Sprintf("rider_user_%s", userID.Hex())
	if r.cache != nil {
		var rider models.Rider
		if err := r.cache.Get(ctx, cacheKey, &rider); err == nil {
			return &rider, nil
		}
	}

	var rider models.Rider
	err := r.collection.FindOne(ctx, bson.M{"user_id": userID}).Decode(&rider)
	if err != nil {
		if err == mongo.ErrNoDocuments {
			return nil, fmt.Errorf("rider not found for user")
		}
		return nil, fmt.Errorf("failed to get rider by user ID: %w", err)
	}

	// Cache the result
	if r.cache != nil {
		r.cache.Set(ctx, cacheKey, rider, 15*time.Minute)
	}
	r.cacheRider(ctx, &rider)

	return &rider, nil
}

func (r *riderRepository) Update(ctx context.Context, id primitive.ObjectID, updates map[string]interface{}) error {
	updates["updated_at"] = time.Now()

	_, err := r.collection.UpdateOne(
		ctx,
		bson.M{"_id": id},
		bson.M{"$set": updates},
	)
	if err != nil {
		return fmt.Errorf("failed to update rider: %w", err)
	}

	// Invalidate cache
	r.invalidateRiderCache(ctx, id.Hex())

	return nil
}

func (r *riderRepository) Delete(ctx context.Context, id primitive.ObjectID) error {
	_, err := r.collection.DeleteOne(ctx, bson.M{"_id": id})
	if err != nil {
		return fmt.Errorf("failed to delete rider: %w", err)
	}

	// Invalidate cache
	r.invalidateRiderCache(ctx, id.Hex())

	return nil
}

// Payment methods
func (r *riderRepository) AddPaymentMethod(ctx context.Context, id primitive.ObjectID, paymentMethodID primitive.ObjectID) error {
	_, err := r.collection.UpdateOne(
		ctx,
		bson.M{"_id": id},
		bson.M{
			"$addToSet": bson.M{"payment_methods": paymentMethodID},
			"$set":      bson.M{"updated_at": time.Now()},
		},
	)
	if err != nil {
		return fmt.Errorf("failed to add payment method: %w", err)
	}

	// Invalidate cache
	r.invalidateRiderCache(ctx, id.Hex())

	return nil
}

func (r *riderRepository) RemovePaymentMethod(ctx context.Context, id primitive.ObjectID, paymentMethodID primitive.ObjectID) error {
	_, err := r.collection.UpdateOne(
		ctx,
		bson.M{"_id": id},
		bson.M{
			"$pull": bson.M{"payment_methods": paymentMethodID},
			"$set":  bson.M{"updated_at": time.Now()},
		},
	)
	if err != nil {
		return fmt.Errorf("failed to remove payment method: %w", err)
	}

	// Also unset as default if it was the default
	_, err = r.collection.UpdateOne(
		ctx,
		bson.M{
			"_id":                id,
			"default_payment_id": paymentMethodID,
		},
		bson.M{
			"$unset": bson.M{"default_payment_id": ""},
			"$set":   bson.M{"updated_at": time.Now()},
		},
	)
	if err != nil {
		return fmt.Errorf("failed to unset default payment method: %w", err)
	}

	// Invalidate cache
	r.invalidateRiderCache(ctx, id.Hex())

	return nil
}

func (r *riderRepository) SetDefaultPaymentMethod(ctx context.Context, id primitive.ObjectID, paymentMethodID primitive.ObjectID) error {
	// First verify the payment method exists in the rider's methods
	rider, err := r.GetByID(ctx, id)
	if err != nil {
		return err
	}

	found := false
	for _, pmID := range rider.PaymentMethods {
		if pmID == paymentMethodID {
			found = true
			break
		}
	}

	if !found {
		return fmt.Errorf("payment method not associated with rider")
	}

	updates := map[string]interface{}{
		"default_payment_id": paymentMethodID,
	}

	return r.Update(ctx, id, updates)
}

// Favorite locations
func (r *riderRepository) AddFavoriteLocation(ctx context.Context, id primitive.ObjectID, location *models.FavoriteLocation) error {
	location.CreatedAt = time.Now()

	_, err := r.collection.UpdateOne(
		ctx,
		bson.M{"_id": id},
		bson.M{
			"$push": bson.M{"favorite_locations": location},
			"$set":  bson.M{"updated_at": time.Now()},
		},
	)
	if err != nil {
		return fmt.Errorf("failed to add favorite location: %w", err)
	}

	// Invalidate cache
	r.invalidateRiderCache(ctx, id.Hex())

	return nil
}

func (r *riderRepository) RemoveFavoriteLocation(ctx context.Context, id primitive.ObjectID, locationName string) error {
	_, err := r.collection.UpdateOne(
		ctx,
		bson.M{"_id": id},
		bson.M{
			"$pull": bson.M{"favorite_locations": bson.M{"name": locationName}},
			"$set":  bson.M{"updated_at": time.Now()},
		},
	)
	if err != nil {
		return fmt.Errorf("failed to remove favorite location: %w", err)
	}

	// Invalidate cache
	r.invalidateRiderCache(ctx, id.Hex())

	return nil
}

func (r *riderRepository) UpdateFavoriteLocation(ctx context.Context, id primitive.ObjectID, locationName string, updates map[string]interface{}) error {
	// MongoDB doesn't support direct update of array elements by field value easily
	// We need to get the rider, update the location, and save back
	rider, err := r.GetByID(ctx, id)
	if err != nil {
		return err
	}

	found := false
	for i, location := range rider.FavoriteLocations {
		if location.Name == locationName {
			// Update fields that are provided
			if name, exists := updates["name"]; exists {
				if nameStr, ok := name.(string); ok {
					rider.FavoriteLocations[i].Name = nameStr
				}
			}
			if address, exists := updates["address"]; exists {
				if addressStr, ok := address.(string); ok {
					rider.FavoriteLocations[i].Address = addressStr
				}
			}
			if locType, exists := updates["type"]; exists {
				if typeStr, ok := locType.(string); ok {
					rider.FavoriteLocations[i].Type = typeStr
				}
			}
			if location, exists := updates["location"]; exists {
				if loc, ok := location.(models.Location); ok {
					rider.FavoriteLocations[i].Location = loc
				}
			}
			found = true
			break
		}
	}

	if !found {
		return fmt.Errorf("favorite location not found")
	}

	// Update the entire favorite_locations array
	updateData := map[string]interface{}{
		"favorite_locations": rider.FavoriteLocations,
	}

	return r.Update(ctx, id, updateData)
}

// Ride statistics
func (r *riderRepository) UpdateRating(ctx context.Context, id primitive.ObjectID, rating float64, totalRatings int64) error {
	updates := map[string]interface{}{
		"rating":        rating,
		"total_ratings": totalRatings,
	}
	return r.Update(ctx, id, updates)
}

func (r *riderRepository) UpdateRideStats(ctx context.Context, id primitive.ObjectID, totalRides int64, totalSpent float64) error {
	updates := map[string]interface{}{
		"total_rides": totalRides,
		"total_spent": totalSpent,
	}
	return r.Update(ctx, id, updates)
}

func (r *riderRepository) UpdateLoyaltyPoints(ctx context.Context, id primitive.ObjectID, points int64) error {
	_, err := r.collection.UpdateOne(
		ctx,
		bson.M{"_id": id},
		bson.M{
			"$inc": bson.M{"loyalty_points": points},
			"$set": bson.M{"updated_at": time.Now()},
		},
	)
	if err != nil {
		return fmt.Errorf("failed to update loyalty points: %w", err)
	}

	// Invalidate cache
	r.invalidateRiderCache(ctx, id.Hex())

	return nil
}

// Referral system
func (r *riderRepository) GetByReferralCode(ctx context.Context, referralCode string) (*models.Rider, error) {
	// Try cache first
	cacheKey := fmt.Sprintf("rider_referral_%s", referralCode)
	if r.cache != nil {
		var rider models.Rider
		if err := r.cache.Get(ctx, cacheKey, &rider); err == nil {
			return &rider, nil
		}
	}

	var rider models.Rider
	err := r.collection.FindOne(ctx, bson.M{"referral_code": referralCode}).Decode(&rider)
	if err != nil {
		if err == mongo.ErrNoDocuments {
			return nil, fmt.Errorf("rider not found with referral code")
		}
		return nil, fmt.Errorf("failed to get rider by referral code: %w", err)
	}

	// Cache the result
	if r.cache != nil {
		r.cache.Set(ctx, cacheKey, rider, 30*time.Minute)
	}

	return &rider, nil
}

func (r *riderRepository) UpdateReferralStats(ctx context.Context, id primitive.ObjectID, referredBy primitive.ObjectID) error {
	updates := map[string]interface{}{
		"referred_by": referredBy,
	}
	return r.Update(ctx, id, updates)
}

// Search and listing
func (r *riderRepository) List(ctx context.Context, params *utils.PaginationParams) ([]*models.Rider, int64, error) {
	filter := bson.M{}
	return r.findRidersWithFilter(ctx, filter, params)
}

func (r *riderRepository) GetByRatingRange(ctx context.Context, minRating, maxRating float64, params *utils.PaginationParams) ([]*models.Rider, int64, error) {
	filter := bson.M{
		"rating": bson.M{
			"$gte": minRating,
			"$lte": maxRating,
		},
		"total_ratings": bson.M{"$gt": 0}, // Only riders with ratings
	}
	return r.findRidersWithFilter(ctx, filter, params)
}

func (r *riderRepository) GetHighValueRiders(ctx context.Context, minSpent float64, params *utils.PaginationParams) ([]*models.Rider, int64, error) {
	filter := bson.M{
		"total_spent": bson.M{"$gte": minSpent},
	}
	return r.findRidersWithFilter(ctx, filter, params)
}

// Analytics
func (r *riderRepository) GetTotalCount(ctx context.Context) (int64, error) {
	return r.collection.CountDocuments(ctx, bson.M{})
}

func (r *riderRepository) GetAverageRating(ctx context.Context) (float64, error) {
	pipeline := mongo.Pipeline{
		{{"$match", bson.M{"total_ratings": bson.M{"$gt": 0}}}},
		{{"$group", bson.M{
			"_id":        nil,
			"avg_rating": bson.M{"$avg": "$rating"},
		}}},
	}

	cursor, err := r.collection.Aggregate(ctx, pipeline)
	if err != nil {
		return 0, fmt.Errorf("failed to calculate average rating: %w", err)
	}
	defer cursor.Close(ctx)

	var result struct {
		AvgRating float64 `bson:"avg_rating"`
	}

	if cursor.Next(ctx) {
		if err := cursor.Decode(&result); err != nil {
			return 0, fmt.Errorf("failed to decode average rating: %w", err)
		}
	}

	return result.AvgRating, nil
}

func (r *riderRepository) GetTopRiders(ctx context.Context, limit int) ([]*models.Rider, error) {
	filter := bson.M{
		"total_rides": bson.M{"$gte": 5}, // At least 5 rides
	}

	opts := options.Find().
		SetSort(bson.D{{Key: "total_spent", Value: -1}}).
		SetLimit(int64(limit))

	cursor, err := r.collection.Find(ctx, filter, opts)
	if err != nil {
		return nil, fmt.Errorf("failed to find top riders: %w", err)
	}
	defer cursor.Close(ctx)

	var riders []*models.Rider
	for cursor.Next(ctx) {
		var rider models.Rider
		if err := cursor.Decode(&rider); err != nil {
			return nil, fmt.Errorf("failed to decode rider: %w", err)
		}
		riders = append(riders, &rider)
	}

	return riders, nil
}

func (r *riderRepository) GetRiderStats(ctx context.Context, riderID primitive.ObjectID, days int) (map[string]interface{}, error) {
	rider, err := r.GetByID(ctx, riderID)
	if err != nil {
		return nil, err
	}

	startDate := time.Now().AddDate(0, 0, -days)

	// Get ride stats from rides collection
	ridesCollection := r.collection.Database().Collection("rides")

	// Get ride counts by status for the period
	ridePipeline := mongo.Pipeline{
		{{"$match", bson.M{
			"rider_id":     riderID,
			"requested_at": bson.M{"$gte": startDate},
		}}},
		{{"$group", bson.M{
			"_id":            "$status",
			"count":          bson.M{"$sum": 1},
			"total_spending": bson.M{"$sum": "$actual_fare"},
		}}},
	}

	cursor, err := ridesCollection.Aggregate(ctx, ridePipeline)
	if err != nil {
		return nil, fmt.Errorf("failed to get ride stats: %w", err)
	}
	defer cursor.Close(ctx)

	rideCounts := make(map[string]int64)
	var totalSpending float64

	for cursor.Next(ctx) {
		var result struct {
			Status        models.RideStatus `bson:"_id"`
			Count         int64             `bson:"count"`
			TotalSpending float64           `bson:"total_spending"`
		}

		if err := cursor.Decode(&result); err != nil {
			return nil, fmt.Errorf("failed to decode ride stats: %w", err)
		}

		rideCounts[string(result.Status)] = result.Count
		if result.Status == models.RideStatusCompleted {
			totalSpending = result.TotalSpending
		}
	}

	// Calculate metrics
	totalRides := int64(0)
	completedRides := rideCounts["completed"]
	cancelledRides := rideCounts["cancelled"]

	for _, count := range rideCounts {
		totalRides += count
	}

	completionRate := float64(0)
	if totalRides > 0 {
		completionRate = float64(completedRides) / float64(totalRides) * 100
	}

	cancellationRate := float64(0)
	if totalRides > 0 {
		cancellationRate = float64(cancelledRides) / float64(totalRides) * 100
	}

	avgSpendingPerRide := float64(0)
	if completedRides > 0 {
		avgSpendingPerRide = totalSpending / float64(completedRides)
	}

	// Get recent average rating
	ratingPipeline := mongo.Pipeline{
		{{"$match", bson.M{
			"rated_id":   riderID,
			"created_at": bson.M{"$gte": startDate},
		}}},
		{{"$group", bson.M{
			"_id":          nil,
			"avg_rating":   bson.M{"$avg": "$rating"},
			"rating_count": bson.M{"$sum": 1},
		}}},
	}

	ratingsCollection := r.collection.Database().Collection("ratings")
	ratingCursor, err := ratingsCollection.Aggregate(ctx, ratingPipeline)
	if err != nil {
		return nil, fmt.Errorf("failed to get recent ratings: %w", err)
	}
	defer ratingCursor.Close(ctx)

	var recentRating float64
	var recentRatingCount int64

	if ratingCursor.Next(ctx) {
		var result struct {
			AvgRating   float64 `bson:"avg_rating"`
			RatingCount int64   `bson:"rating_count"`
		}

		if err := ratingCursor.Decode(&result); err == nil {
			recentRating = result.AvgRating
			recentRatingCount = result.RatingCount
		}
	}

	return map[string]interface{}{
		"rider_id":                 riderID,
		"period_days":              days,
		"total_rides":              totalRides,
		"completed_rides":          completedRides,
		"cancelled_rides":          cancelledRides,
		"completion_rate":          completionRate,
		"cancellation_rate":        cancellationRate,
		"total_spending":           totalSpending,
		"avg_spending_per_ride":    avgSpendingPerRide,
		"recent_avg_rating":        recentRating,
		"recent_rating_count":      recentRatingCount,
		"overall_rating":           rider.Rating,
		"overall_total_rides":      rider.TotalRides,
		"overall_total_spent":      rider.TotalSpent,
		"loyalty_points":           rider.LoyaltyPoints,
		"ride_counts_by_status":    rideCounts,
		"favorite_locations_count": len(rider.FavoriteLocations),
		"payment_methods_count":    len(rider.PaymentMethods),
		"start_date":               startDate,
		"end_date":                 time.Now(),
	}, nil
}

// Helper methods
func (r *riderRepository) findRidersWithFilter(ctx context.Context, filter bson.M, params *utils.PaginationParams) ([]*models.Rider, int64, error) {
	// Add search filter if provided
	if params.Search != "" {
		// Since rider doesn't have name directly, we'd need to join with users
		// For now, we'll search in referral_code
		searchFields := []string{"referral_code"}
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
		return nil, 0, fmt.Errorf("failed to count riders: %w", err)
	}

	// Get paginated results
	opts := params.GetSortOptions()
	cursor, err := r.collection.Find(ctx, filter, opts)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to find riders: %w", err)
	}
	defer cursor.Close(ctx)

	var riders []*models.Rider
	for cursor.Next(ctx) {
		var rider models.Rider
		if err := cursor.Decode(&rider); err != nil {
			return nil, 0, fmt.Errorf("failed to decode rider: %w", err)
		}
		riders = append(riders, &rider)
	}

	return riders, total, nil
}

// Cache operations
func (r *riderRepository) cacheRider(ctx context.Context, rider *models.Rider) {
	if r.cache != nil {
		cacheKey := fmt.Sprintf("rider:%s", rider.ID.Hex())
		r.cache.Set(ctx, cacheKey, rider, 15*time.Minute)

		// Also cache by user_id
		if !rider.UserID.IsZero() {
			userKey := fmt.Sprintf("rider_user_%s", rider.UserID.Hex())
			r.cache.Set(ctx, userKey, rider, 15*time.Minute)
		}

		// Also cache by referral code if exists
		if rider.ReferralCode != "" {
			referralKey := fmt.Sprintf("rider_referral_%s", rider.ReferralCode)
			r.cache.Set(ctx, referralKey, rider, 30*time.Minute)
		}
	}
}

func (r *riderRepository) getRiderFromCache(ctx context.Context, riderID string) *models.Rider {
	if r.cache == nil {
		return nil
	}

	cacheKey := fmt.Sprintf("rider:%s", riderID)
	var rider models.Rider
	err := r.cache.Get(ctx, cacheKey, &rider)
	if err != nil {
		return nil
	}

	return &rider
}

func (r *riderRepository) invalidateRiderCache(ctx context.Context, riderID string) {
	if r.cache != nil {
		cacheKey := fmt.Sprintf("rider:%s", riderID)
		r.cache.Delete(ctx, cacheKey)

		// Note: We can't easily invalidate the user_id and referral_code caches
		// without additional lookups. This is a trade-off for performance vs cache consistency
	}
}
