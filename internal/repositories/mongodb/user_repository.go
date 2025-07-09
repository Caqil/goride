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
)

type userRepository struct {
	collection *mongo.Collection
	cache      CacheService
}

func NewUserRepository(db *mongo.Database, cache CacheService) interfaces.UserRepository {
	return &userRepository{
		collection: db.Collection("users"),
		cache:      cache,
	}
}

func (r *userRepository) Create(ctx context.Context, user *models.User) error {
	user.ID = primitive.NewObjectID()
	user.CreatedAt = time.Now()
	user.UpdatedAt = time.Now()

	_, err := r.collection.InsertOne(ctx, user)
	if err != nil {
		return fmt.Errorf("failed to create user: %w", err)
	}

	// Cache the user
	r.cacheUser(ctx, user)

	return nil
}

func (r *userRepository) GetByID(ctx context.Context, id primitive.ObjectID) (*models.User, error) {
	// Try cache first
	if user := r.getUserFromCache(ctx, id.Hex()); user != nil {
		return user, nil
	}

	var user models.User
	err := r.collection.FindOne(ctx, bson.M{"_id": id, "deleted_at": nil}).Decode(&user)
	if err != nil {
		if err == mongo.ErrNoDocuments {
			return nil, fmt.Errorf("user not found")
		}
		return nil, fmt.Errorf("failed to get user: %w", err)
	}

	// Cache the result
	r.cacheUser(ctx, &user)

	return &user, nil
}

func (r *userRepository) Update(ctx context.Context, id primitive.ObjectID, updates map[string]interface{}) error {
	updates["updated_at"] = time.Now()

	_, err := r.collection.UpdateOne(
		ctx,
		bson.M{"_id": id, "deleted_at": nil},
		bson.M{"$set": updates},
	)
	if err != nil {
		return fmt.Errorf("failed to update user: %w", err)
	}

	// Invalidate cache
	r.invalidateUserCache(ctx, id.Hex())

	return nil
}

func (r *userRepository) Delete(ctx context.Context, id primitive.ObjectID) error {
	_, err := r.collection.UpdateOne(
		ctx,
		bson.M{"_id": id},
		bson.M{"$set": bson.M{"deleted_at": time.Now()}},
	)
	if err != nil {
		return fmt.Errorf("failed to delete user: %w", err)
	}

	// Invalidate cache
	r.invalidateUserCache(ctx, id.Hex())

	return nil
}

func (r *userRepository) GetByEmail(ctx context.Context, email string) (*models.User, error) {
	var user models.User
	err := r.collection.FindOne(ctx, bson.M{
		"email":      email,
		"deleted_at": nil,
	}).Decode(&user)

	if err != nil {
		if err == mongo.ErrNoDocuments {
			return nil, fmt.Errorf("user not found")
		}
		return nil, fmt.Errorf("failed to get user by email: %w", err)
	}

	return &user, nil
}

func (r *userRepository) GetByPhone(ctx context.Context, phone string) (*models.User, error) {
	var user models.User
	err := r.collection.FindOne(ctx, bson.M{
		"phone":      phone,
		"deleted_at": nil,
	}).Decode(&user)

	if err != nil {
		if err == mongo.ErrNoDocuments {
			return nil, fmt.Errorf("user not found")
		}
		return nil, fmt.Errorf("failed to get user by phone: %w", err)
	}

	return &user, nil
}

func (r *userRepository) GetBySocialID(ctx context.Context, provider string, socialID string) (*models.User, error) {
	var user models.User
	err := r.collection.FindOne(ctx, bson.M{
		"auth_provider": provider,
		"social_id":     socialID,
		"deleted_at":    nil,
	}).Decode(&user)

	if err != nil {
		if err == mongo.ErrNoDocuments {
			return nil, fmt.Errorf("user not found")
		}
		return nil, fmt.Errorf("failed to get user by social ID: %w", err)
	}

	return &user, nil
}

func (r *userRepository) UpdateEmailVerification(ctx context.Context, id primitive.ObjectID, verified bool) error {
	return r.Update(ctx, id, map[string]interface{}{
		"is_email_verified": verified,
	})
}

func (r *userRepository) UpdatePhoneVerification(ctx context.Context, id primitive.ObjectID, verified bool) error {
	return r.Update(ctx, id, map[string]interface{}{
		"is_phone_verified": verified,
	})
}

func (r *userRepository) UpdateLastLogin(ctx context.Context, id primitive.ObjectID) error {
	now := time.Now()
	return r.Update(ctx, id, map[string]interface{}{
		"last_login_at":  now,
		"last_active_at": now,
	})
}

func (r *userRepository) UpdateLastActive(ctx context.Context, id primitive.ObjectID) error {
	return r.Update(ctx, id, map[string]interface{}{
		"last_active_at": time.Now(),
	})
}

func (r *userRepository) List(ctx context.Context, params *utils.PaginationParams) ([]*models.User, int64, error) {
	filter := bson.M{"deleted_at": nil}

	// Add search filter if provided
	if params.Search != "" {
		searchFields := []string{"first_name", "last_name", "email", "phone"}
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
		return nil, 0, fmt.Errorf("failed to count users: %w", err)
	}

	// Get paginated results
	opts := params.GetSortOptions()
	cursor, err := r.collection.Find(ctx, filter, opts)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to find users: %w", err)
	}
	defer cursor.Close(ctx)

	var users []*models.User
	for cursor.Next(ctx) {
		var user models.User
		if err := cursor.Decode(&user); err != nil {
			return nil, 0, fmt.Errorf("failed to decode user: %w", err)
		}
		users = append(users, &user)
	}

	return users, total, nil
}

func (r *userRepository) SearchByName(ctx context.Context, name string, params *utils.PaginationParams) ([]*models.User, int64, error) {
	filter := bson.M{
		"deleted_at": nil,
		"$or": []bson.M{
			{"first_name": bson.M{"$regex": name, "$options": "i"}},
			{"last_name": bson.M{"$regex": name, "$options": "i"}},
			{"$expr": bson.M{
				"$regexMatch": bson.M{
					"input":   bson.M{"$concat": []string{"$first_name", " ", "$last_name"}},
					"regex":   name,
					"options": "i",
				},
			}},
		},
	}

	return r.findUsersWithFilter(ctx, filter, params)
}

func (r *userRepository) GetByStatus(ctx context.Context, status models.UserStatus, params *utils.PaginationParams) ([]*models.User, int64, error) {
	filter := bson.M{
		"status":     status,
		"deleted_at": nil,
	}

	return r.findUsersWithFilter(ctx, filter, params)
}

func (r *userRepository) GetByType(ctx context.Context, userType models.UserType, params *utils.PaginationParams) ([]*models.User, int64, error) {
	filter := bson.M{
		"user_type":  userType,
		"deleted_at": nil,
	}

	return r.findUsersWithFilter(ctx, filter, params)
}

func (r *userRepository) GetTotalCount(ctx context.Context) (int64, error) {
	return r.collection.CountDocuments(ctx, bson.M{"deleted_at": nil})
}

func (r *userRepository) GetCountByStatus(ctx context.Context, status models.UserStatus) (int64, error) {
	return r.collection.CountDocuments(ctx, bson.M{
		"status":     status,
		"deleted_at": nil,
	})
}

func (r *userRepository) GetCountByType(ctx context.Context, userType models.UserType) (int64, error) {
	return r.collection.CountDocuments(ctx, bson.M{
		"user_type":  userType,
		"deleted_at": nil,
	})
}

func (r *userRepository) GetRegistrationStats(ctx context.Context, days int) (map[string]int64, error) {
	startDate := time.Now().AddDate(0, 0, -days)

	pipeline := mongo.Pipeline{
		{{"$match", bson.M{
			"created_at": bson.M{"$gte": startDate},
			"deleted_at": nil,
		}}},
		{{"$group", bson.M{
			"_id": bson.M{
				"date": bson.M{"$dateToString": bson.M{
					"format": "%Y-%m-%d",
					"date":   "$created_at",
				}},
			},
			"count": bson.M{"$sum": 1},
		}}},
		{{"$sort", bson.M{"_id.date": 1}}},
	}

	cursor, err := r.collection.Aggregate(ctx, pipeline)
	if err != nil {
		return nil, fmt.Errorf("failed to get registration stats: %w", err)
	}
	defer cursor.Close(ctx)

	stats := make(map[string]int64)
	for cursor.Next(ctx) {
		var result struct {
			ID struct {
				Date string `bson:"date"`
			} `bson:"_id"`
			Count int64 `bson:"count"`
		}

		if err := cursor.Decode(&result); err != nil {
			return nil, fmt.Errorf("failed to decode stats: %w", err)
		}

		stats[result.ID.Date] = result.Count
	}

	return stats, nil
}

// Helper methods
func (r *userRepository) findUsersWithFilter(ctx context.Context, filter bson.M, params *utils.PaginationParams) ([]*models.User, int64, error) {
	// Get total count
	total, err := r.collection.CountDocuments(ctx, filter)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to count users: %w", err)
	}

	// Get paginated results
	opts := params.GetSortOptions()
	cursor, err := r.collection.Find(ctx, filter, opts)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to find users: %w", err)
	}
	defer cursor.Close(ctx)

	var users []*models.User
	for cursor.Next(ctx) {
		var user models.User
		if err := cursor.Decode(&user); err != nil {
			return nil, 0, fmt.Errorf("failed to decode user: %w", err)
		}
		users = append(users, &user)
	}

	return users, total, nil
}

// Cache operations
func (r *userRepository) cacheUser(ctx context.Context, user *models.User) {
	if r.cache != nil {
		cacheKey := fmt.Sprintf("user:%s", user.ID.Hex())
		r.cache.Set(ctx, cacheKey, user, 15*time.Minute)
	}
}

func (r *userRepository) getUserFromCache(ctx context.Context, userID string) *models.User {
	if r.cache == nil {
		return nil
	}

	cacheKey := fmt.Sprintf("user:%s", userID)
	var user models.User
	err := r.cache.Get(ctx, cacheKey, &user)
	if err != nil {
		return nil
	}

	return &user
}

func (r *userRepository) invalidateUserCache(ctx context.Context, userID string) {
	if r.cache != nil {
		cacheKey := fmt.Sprintf("user:%s", userID)
		r.cache.Delete(ctx, cacheKey)
	}
}

// Cache service interface
type CacheService interface {
	Set(ctx context.Context, key string, value interface{}, expiration time.Duration) error
	Get(ctx context.Context, key string, dest interface{}) error
	Delete(ctx context.Context, keys ...string) error
}
