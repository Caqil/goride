package mongodb

import (
	"context"
	"fmt"
	"math"
	"time"

	"goride/internal/models"
	"goride/internal/repositories/interfaces"
	"goride/internal/utils"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

type ratingRepository struct {
	collection *mongo.Collection
	cache      CacheService
}

func NewRatingRepository(db *mongo.Database, cache CacheService) interfaces.RatingRepository {
	return &ratingRepository{
		collection: db.Collection("ratings"),
		cache:      cache,
	}
}

// Basic CRUD operations
func (r *ratingRepository) Create(ctx context.Context, rating *models.Rating) error {
	rating.ID = primitive.NewObjectID()
	rating.CreatedAt = time.Now()
	rating.UpdatedAt = time.Now()

	_, err := r.collection.InsertOne(ctx, rating)
	if err != nil {
		return fmt.Errorf("failed to create rating: %w", err)
	}

	// Invalidate average rating cache for the rated user
	r.invalidateAverageRatingCache(ctx, rating.RatedID)

	return nil
}

func (r *ratingRepository) GetByID(ctx context.Context, id primitive.ObjectID) (*models.Rating, error) {
	var rating models.Rating
	err := r.collection.FindOne(ctx, bson.M{"_id": id}).Decode(&rating)
	if err != nil {
		if err == mongo.ErrNoDocuments {
			return nil, fmt.Errorf("rating not found")
		}
		return nil, fmt.Errorf("failed to get rating: %w", err)
	}

	return &rating, nil
}

func (r *ratingRepository) Update(ctx context.Context, id primitive.ObjectID, updates map[string]interface{}) error {
	updates["updated_at"] = time.Now()

	result, err := r.collection.UpdateOne(
		ctx,
		bson.M{"_id": id},
		bson.M{"$set": updates},
	)
	if err != nil {
		return fmt.Errorf("failed to update rating: %w", err)
	}

	if result.ModifiedCount == 0 {
		return fmt.Errorf("rating not found or no changes made")
	}

	// If the rating value was updated, invalidate cache
	if _, exists := updates["rating"]; exists {
		// Get the rating to find rated_id for cache invalidation
		rating, err := r.GetByID(ctx, id)
		if err == nil {
			r.invalidateAverageRatingCache(ctx, rating.RatedID)
		}
	}

	return nil
}

func (r *ratingRepository) Delete(ctx context.Context, id primitive.ObjectID) error {
	// Get rating first to invalidate cache
	rating, err := r.GetByID(ctx, id)
	if err != nil {
		return err
	}

	_, err = r.collection.DeleteOne(ctx, bson.M{"_id": id})
	if err != nil {
		return fmt.Errorf("failed to delete rating: %w", err)
	}

	// Invalidate cache
	r.invalidateAverageRatingCache(ctx, rating.RatedID)

	return nil
}

// Ride ratings
func (r *ratingRepository) GetByRideID(ctx context.Context, rideID primitive.ObjectID) ([]*models.Rating, error) {
	filter := bson.M{"ride_id": rideID}

	cursor, err := r.collection.Find(ctx, filter, options.Find().SetSort(bson.D{{Key: "created_at", Value: -1}}))
	if err != nil {
		return nil, fmt.Errorf("failed to find ratings by ride ID: %w", err)
	}
	defer cursor.Close(ctx)

	var ratings []*models.Rating
	for cursor.Next(ctx) {
		var rating models.Rating
		if err := cursor.Decode(&rating); err != nil {
			return nil, fmt.Errorf("failed to decode rating: %w", err)
		}
		ratings = append(ratings, &rating)
	}

	return ratings, nil
}

func (r *ratingRepository) GetRideRating(ctx context.Context, rideID primitive.ObjectID, raterType models.UserType) (*models.Rating, error) {
	filter := bson.M{
		"ride_id":    rideID,
		"rater_type": raterType,
	}

	var rating models.Rating
	err := r.collection.FindOne(ctx, filter).Decode(&rating)
	if err != nil {
		if err == mongo.ErrNoDocuments {
			return nil, fmt.Errorf("rating not found for ride and rater type")
		}
		return nil, fmt.Errorf("failed to get ride rating: %w", err)
	}

	return &rating, nil
}

// User ratings
func (r *ratingRepository) GetByRaterID(ctx context.Context, raterID primitive.ObjectID, params *utils.PaginationParams) ([]*models.Rating, int64, error) {
	filter := bson.M{"rater_id": raterID}
	return r.findRatingsWithFilter(ctx, filter, params)
}

func (r *ratingRepository) GetByRatedID(ctx context.Context, ratedID primitive.ObjectID, params *utils.PaginationParams) ([]*models.Rating, int64, error) {
	filter := bson.M{"rated_id": ratedID}
	return r.findRatingsWithFilter(ctx, filter, params)
}

// Rating statistics
func (r *ratingRepository) GetAverageRating(ctx context.Context, ratedID primitive.ObjectID) (float64, error) {
	// Try cache first
	cacheKey := fmt.Sprintf("avg_rating_%s", ratedID.Hex())
	if r.cache != nil {
		var avgRating float64
		if err := r.cache.Get(ctx, cacheKey, &avgRating); err == nil {
			return avgRating, nil
		}
	}

	pipeline := mongo.Pipeline{
		{{"$match", bson.M{"rated_id": ratedID}}},
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

	avgRating := float64(0)
	if cursor.Next(ctx) {
		if err := cursor.Decode(&result); err == nil {
			avgRating = math.Round(result.AvgRating*100) / 100 // Round to 2 decimal places
		}
	}

	// Cache the result
	if r.cache != nil {
		r.cache.Set(ctx, cacheKey, avgRating, 15*time.Minute)
	}

	return avgRating, nil
}

func (r *ratingRepository) GetRatingDistribution(ctx context.Context, ratedID primitive.ObjectID) (map[int]int64, error) {
	pipeline := mongo.Pipeline{
		{{"$match", bson.M{"rated_id": ratedID}}},
		{{"$group", bson.M{
			"_id":   bson.M{"$floor": "$rating"},
			"count": bson.M{"$sum": 1},
		}}},
		{{"$sort", bson.D{{Key: "_id", Value: 1}}}},
	}

	cursor, err := r.collection.Aggregate(ctx, pipeline)
	if err != nil {
		return nil, fmt.Errorf("failed to get rating distribution: %w", err)
	}
	defer cursor.Close(ctx)

	distribution := make(map[int]int64)

	for cursor.Next(ctx) {
		var result struct {
			ID    int   `bson:"_id"`
			Count int64 `bson:"count"`
		}

		if err := cursor.Decode(&result); err != nil {
			return nil, fmt.Errorf("failed to decode rating distribution: %w", err)
		}

		distribution[result.ID] = result.Count
	}

	return distribution, nil
}

func (r *ratingRepository) GetRatingCount(ctx context.Context, ratedID primitive.ObjectID) (int64, error) {
	count, err := r.collection.CountDocuments(ctx, bson.M{"rated_id": ratedID})
	if err != nil {
		return 0, fmt.Errorf("failed to count ratings: %w", err)
	}

	return count, nil
}

// Rating analysis
func (r *ratingRepository) GetRatingsByRange(ctx context.Context, minRating, maxRating float64, params *utils.PaginationParams) ([]*models.Rating, int64, error) {
	filter := bson.M{
		"rating": bson.M{
			"$gte": minRating,
			"$lte": maxRating,
		},
	}
	return r.findRatingsWithFilter(ctx, filter, params)
}

func (r *ratingRepository) GetRecentRatings(ctx context.Context, ratedID primitive.ObjectID, days int) ([]*models.Rating, error) {
	startDate := time.Now().AddDate(0, 0, -days)
	filter := bson.M{
		"rated_id":   ratedID,
		"created_at": bson.M{"$gte": startDate},
	}

	cursor, err := r.collection.Find(ctx, filter, options.Find().SetSort(bson.D{{Key: "created_at", Value: -1}}))
	if err != nil {
		return nil, fmt.Errorf("failed to find recent ratings: %w", err)
	}
	defer cursor.Close(ctx)

	var ratings []*models.Rating
	for cursor.Next(ctx) {
		var rating models.Rating
		if err := cursor.Decode(&rating); err != nil {
			return nil, fmt.Errorf("failed to decode rating: %w", err)
		}
		ratings = append(ratings, &rating)
	}

	return ratings, nil
}

func (r *ratingRepository) GetRatingTrends(ctx context.Context, ratedID primitive.ObjectID, days int) ([]map[string]interface{}, error) {
	startDate := time.Now().AddDate(0, 0, -days)

	pipeline := mongo.Pipeline{
		{{"$match", bson.M{
			"rated_id":   ratedID,
			"created_at": bson.M{"$gte": startDate},
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
			"avg_rating":   bson.M{"$avg": "$rating"},
			"rating_count": bson.M{"$sum": 1},
		}}},
		{{"$sort", bson.D{{Key: "_id.date", Value: 1}}}},
	}

	cursor, err := r.collection.Aggregate(ctx, pipeline)
	if err != nil {
		return nil, fmt.Errorf("failed to get rating trends: %w", err)
	}
	defer cursor.Close(ctx)

	var trends []map[string]interface{}

	for cursor.Next(ctx) {
		var result struct {
			ID struct {
				Date string `bson:"date"`
			} `bson:"_id"`
			AvgRating   float64 `bson:"avg_rating"`
			RatingCount int64   `bson:"rating_count"`
		}

		if err := cursor.Decode(&result); err != nil {
			return nil, fmt.Errorf("failed to decode rating trends: %w", err)
		}

		trends = append(trends, map[string]interface{}{
			"date":         result.ID.Date,
			"avg_rating":   math.Round(result.AvgRating*100) / 100,
			"rating_count": result.RatingCount,
		})
	}

	return trends, nil
}

// Tag analysis
func (r *ratingRepository) GetRatingsByTag(ctx context.Context, tag string, params *utils.PaginationParams) ([]*models.Rating, int64, error) {
	filter := bson.M{"tags": bson.M{"$in": []string{tag}}}
	return r.findRatingsWithFilter(ctx, filter, params)
}

func (r *ratingRepository) GetPopularTags(ctx context.Context, ratedID primitive.ObjectID, limit int) ([]map[string]interface{}, error) {
	pipeline := mongo.Pipeline{
		{{"$match", bson.M{"rated_id": ratedID}}},
		{{"$unwind", "$tags"}},
		{{"$group", bson.M{
			"_id":        "$tags",
			"count":      bson.M{"$sum": 1},
			"avg_rating": bson.M{"$avg": "$rating"},
		}}},
		{{"$sort", bson.D{{Key: "count", Value: -1}}}},
		{{"$limit", limit}},
	}

	cursor, err := r.collection.Aggregate(ctx, pipeline)
	if err != nil {
		return nil, fmt.Errorf("failed to get popular tags: %w", err)
	}
	defer cursor.Close(ctx)

	var tags []map[string]interface{}

	for cursor.Next(ctx) {
		var result struct {
			ID        string  `bson:"_id"`
			Count     int64   `bson:"count"`
			AvgRating float64 `bson:"avg_rating"`
		}

		if err := cursor.Decode(&result); err != nil {
			return nil, fmt.Errorf("failed to decode popular tags: %w", err)
		}

		tags = append(tags, map[string]interface{}{
			"tag":        result.ID,
			"count":      result.Count,
			"avg_rating": math.Round(result.AvgRating*100) / 100,
		})
	}

	return tags, nil
}

// Search and filtering
func (r *ratingRepository) SearchRatings(ctx context.Context, query string, params *utils.PaginationParams) ([]*models.Rating, int64, error) {
	filter := bson.M{
		"$or": []bson.M{
			{"comment": bson.M{"$regex": query, "$options": "i"}},
			{"tags": bson.M{"$in": []string{query}}},
		},
	}
	return r.findRatingsWithFilter(ctx, filter, params)
}

func (r *ratingRepository) GetRatingsByType(ctx context.Context, raterType models.UserType, params *utils.PaginationParams) ([]*models.Rating, int64, error) {
	filter := bson.M{"rater_type": raterType}
	return r.findRatingsWithFilter(ctx, filter, params)
}

// Analytics
func (r *ratingRepository) GetRatingStats(ctx context.Context, days int) (map[string]interface{}, error) {
	startDate := time.Now().AddDate(0, 0, -days)

	pipeline := mongo.Pipeline{
		{{"$match", bson.M{"created_at": bson.M{"$gte": startDate}}}},
		{{"$group", bson.M{
			"_id":           "$rater_type",
			"total_ratings": bson.M{"$sum": 1},
			"avg_rating":    bson.M{"$avg": "$rating"},
			"min_rating":    bson.M{"$min": "$rating"},
			"max_rating":    bson.M{"$max": "$rating"},
		}}},
	}

	cursor, err := r.collection.Aggregate(ctx, pipeline)
	if err != nil {
		return nil, fmt.Errorf("failed to get rating stats: %w", err)
	}
	defer cursor.Close(ctx)

	stats := make(map[string]interface{})
	var totalRatings int64

	for cursor.Next(ctx) {
		var result struct {
			ID           models.UserType `bson:"_id"`
			TotalRatings int64           `bson:"total_ratings"`
			AvgRating    float64         `bson:"avg_rating"`
			MinRating    float64         `bson:"min_rating"`
			MaxRating    float64         `bson:"max_rating"`
		}

		if err := cursor.Decode(&result); err != nil {
			return nil, fmt.Errorf("failed to decode rating stats: %w", err)
		}

		stats[string(result.ID)] = map[string]interface{}{
			"total_ratings": result.TotalRatings,
			"avg_rating":    math.Round(result.AvgRating*100) / 100,
			"min_rating":    result.MinRating,
			"max_rating":    result.MaxRating,
		}

		totalRatings += result.TotalRatings
	}

	// Get overall stats
	overallPipeline := mongo.Pipeline{
		{{"$match", bson.M{"created_at": bson.M{"$gte": startDate}}}},
		{{"$group", bson.M{
			"_id":           nil,
			"total_ratings": bson.M{"$sum": 1},
			"avg_rating":    bson.M{"$avg": "$rating"},
			"with_comments": bson.M{"$sum": bson.M{
				"$cond": []interface{}{
					bson.M{"$ne": []interface{}{"$comment", ""}},
					1,
					0,
				},
			}},
			"anonymous_count": bson.M{"$sum": bson.M{
				"$cond": []interface{}{
					"$is_anonymous",
					1,
					0,
				},
			}},
		}}},
	}

	overallCursor, err := r.collection.Aggregate(ctx, overallPipeline)
	if err != nil {
		return nil, fmt.Errorf("failed to get overall rating stats: %w", err)
	}
	defer overallCursor.Close(ctx)

	var overallResult struct {
		TotalRatings   int64   `bson:"total_ratings"`
		AvgRating      float64 `bson:"avg_rating"`
		WithComments   int64   `bson:"with_comments"`
		AnonymousCount int64   `bson:"anonymous_count"`
	}

	if overallCursor.Next(ctx) {
		if err := overallCursor.Decode(&overallResult); err == nil {
			commentRate := float64(0)
			anonymousRate := float64(0)
			if overallResult.TotalRatings > 0 {
				commentRate = float64(overallResult.WithComments) / float64(overallResult.TotalRatings) * 100
				anonymousRate = float64(overallResult.AnonymousCount) / float64(overallResult.TotalRatings) * 100
			}

			stats["overall"] = map[string]interface{}{
				"total_ratings":  overallResult.TotalRatings,
				"avg_rating":     math.Round(overallResult.AvgRating*100) / 100,
				"comment_rate":   math.Round(commentRate*100) / 100,
				"anonymous_rate": math.Round(anonymousRate*100) / 100,
			}
		}
	}

	stats["summary"] = map[string]interface{}{
		"period_days": days,
		"start_date":  startDate,
		"end_date":    time.Now(),
	}

	return stats, nil
}

func (r *ratingRepository) GetLowRatedUsers(ctx context.Context, threshold float64, userType models.UserType) ([]*models.Rating, error) {
	pipeline := mongo.Pipeline{
		{{"$match", bson.M{"rater_type": userType}}},
		{{"$group", bson.M{
			"_id":          "$rated_id",
			"avg_rating":   bson.M{"$avg": "$rating"},
			"rating_count": bson.M{"$sum": 1},
		}}},
		{{"$match", bson.M{
			"avg_rating":   bson.M{"$lt": threshold},
			"rating_count": bson.M{"$gte": 5}, // At least 5 ratings
		}}},
		{{"$sort", bson.D{{Key: "avg_rating", Value: 1}}}},
	}

	cursor, err := r.collection.Aggregate(ctx, pipeline)
	if err != nil {
		return nil, fmt.Errorf("failed to get low rated users: %w", err)
	}
	defer cursor.Close(ctx)

	var userIDs []primitive.ObjectID
	for cursor.Next(ctx) {
		var result struct {
			ID primitive.ObjectID `bson:"_id"`
		}

		if err := cursor.Decode(&result); err != nil {
			return nil, fmt.Errorf("failed to decode low rated user: %w", err)
		}

		userIDs = append(userIDs, result.ID)
	}

	// Get recent ratings for these users
	filter := bson.M{
		"rated_id": bson.M{"$in": userIDs},
		"rating":   bson.M{"$lt": threshold},
	}

	ratingssCursor, err := r.collection.Find(ctx, filter, options.Find().SetSort(bson.D{{Key: "created_at", Value: -1}}))
	if err != nil {
		return nil, fmt.Errorf("failed to find low ratings: %w", err)
	}
	defer ratingssCursor.Close(ctx)

	var ratings []*models.Rating
	for ratingssCursor.Next(ctx) {
		var rating models.Rating
		if err := ratingssCursor.Decode(&rating); err != nil {
			return nil, fmt.Errorf("failed to decode rating: %w", err)
		}
		ratings = append(ratings, &rating)
	}

	return ratings, nil
}

// Helper methods
func (r *ratingRepository) findRatingsWithFilter(ctx context.Context, filter bson.M, params *utils.PaginationParams) ([]*models.Rating, int64, error) {
	// Add search filter if provided
	if params.Search != "" {
		searchFields := []string{"comment"}
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
		return nil, 0, fmt.Errorf("failed to count ratings: %w", err)
	}

	// Get paginated results
	opts := params.GetSortOptions()
	cursor, err := r.collection.Find(ctx, filter, opts)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to find ratings: %w", err)
	}
	defer cursor.Close(ctx)

	var ratings []*models.Rating
	for cursor.Next(ctx) {
		var rating models.Rating
		if err := cursor.Decode(&rating); err != nil {
			return nil, 0, fmt.Errorf("failed to decode rating: %w", err)
		}
		ratings = append(ratings, &rating)
	}

	return ratings, total, nil
}

// Cache operations
func (r *ratingRepository) invalidateAverageRatingCache(ctx context.Context, ratedID primitive.ObjectID) {
	if r.cache != nil {
		cacheKey := fmt.Sprintf("avg_rating_%s", ratedID.Hex())
		r.cache.Delete(ctx, cacheKey)
	}
}
