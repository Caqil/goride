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

// Basic CRUD operations
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
	// Soft delete - set deleted_at timestamp
	updates := map[string]interface{}{
		"deleted_at": time.Now(),
	}

	return r.Update(ctx, id, updates)
}

// Authentication operations
func (r *userRepository) GetByEmail(ctx context.Context, email string) (*models.User, error) {
	// Try cache first
	cacheKey := fmt.Sprintf("user_email_%s", email)
	if r.cache != nil {
		var user models.User
		if err := r.cache.Get(ctx, cacheKey, &user); err == nil {
			return &user, nil
		}
	}

	var user models.User
	err := r.collection.FindOne(ctx, bson.M{
		"email":      email,
		"deleted_at": nil,
	}).Decode(&user)
	if err != nil {
		if err == mongo.ErrNoDocuments {
			return nil, fmt.Errorf("user not found with email")
		}
		return nil, fmt.Errorf("failed to get user by email: %w", err)
	}

	// Cache the result
	if r.cache != nil {
		r.cache.Set(ctx, cacheKey, user, 15*time.Minute)
	}
	r.cacheUser(ctx, &user)

	return &user, nil
}

func (r *userRepository) GetByPhone(ctx context.Context, phone string) (*models.User, error) {
	// Try cache first
	cacheKey := fmt.Sprintf("user_phone_%s", phone)
	if r.cache != nil {
		var user models.User
		if err := r.cache.Get(ctx, cacheKey, &user); err == nil {
			return &user, nil
		}
	}

	var user models.User
	err := r.collection.FindOne(ctx, bson.M{
		"phone":      phone,
		"deleted_at": nil,
	}).Decode(&user)
	if err != nil {
		if err == mongo.ErrNoDocuments {
			return nil, fmt.Errorf("user not found with phone")
		}
		return nil, fmt.Errorf("failed to get user by phone: %w", err)
	}

	// Cache the result
	if r.cache != nil {
		r.cache.Set(ctx, cacheKey, user, 15*time.Minute)
	}
	r.cacheUser(ctx, &user)

	return &user, nil
}

func (r *userRepository) GetBySocialID(ctx context.Context, provider string, socialID string) (*models.User, error) {
	// Try cache first
	cacheKey := fmt.Sprintf("user_social_%s_%s", provider, socialID)
	if r.cache != nil {
		var user models.User
		if err := r.cache.Get(ctx, cacheKey, &user); err == nil {
			return &user, nil
		}
	}

	var user models.User
	err := r.collection.FindOne(ctx, bson.M{
		"auth_provider": provider,
		"social_id":     socialID,
		"deleted_at":    nil,
	}).Decode(&user)
	if err != nil {
		if err == mongo.ErrNoDocuments {
			return nil, fmt.Errorf("user not found with social ID")
		}
		return nil, fmt.Errorf("failed to get user by social ID: %w", err)
	}

	// Cache the result
	if r.cache != nil {
		r.cache.Set(ctx, cacheKey, user, 15*time.Minute)
	}
	r.cacheUser(ctx, &user)

	return &user, nil
}

// Verification operations
func (r *userRepository) UpdateEmailVerification(ctx context.Context, id primitive.ObjectID, verified bool) error {
	updates := map[string]interface{}{
		"is_email_verified": verified,
	}
	return r.Update(ctx, id, updates)
}

func (r *userRepository) UpdatePhoneVerification(ctx context.Context, id primitive.ObjectID, verified bool) error {
	updates := map[string]interface{}{
		"is_phone_verified": verified,
	}
	return r.Update(ctx, id, updates)
}

func (r *userRepository) UpdateLastLogin(ctx context.Context, id primitive.ObjectID) error {
	now := time.Now()
	updates := map[string]interface{}{
		"last_login_at":  &now,
		"last_active_at": &now,
	}
	return r.Update(ctx, id, updates)
}

func (r *userRepository) UpdateLastActive(ctx context.Context, id primitive.ObjectID) error {
	now := time.Now()
	updates := map[string]interface{}{
		"last_active_at": &now,
	}
	return r.Update(ctx, id, updates)
}

// Search and listing
func (r *userRepository) List(ctx context.Context, params *utils.PaginationParams) ([]*models.User, int64, error) {
	filter := bson.M{"deleted_at": nil}
	return r.findUsersWithFilter(ctx, filter, params)
}

func (r *userRepository) SearchByName(ctx context.Context, name string, params *utils.PaginationParams) ([]*models.User, int64, error) {
	filter := bson.M{
		"$or": []bson.M{
			{"first_name": bson.M{"$regex": name, "$options": "i"}},
			{"last_name": bson.M{"$regex": name, "$options": "i"}},
		},
		"deleted_at": nil,
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

// Statistics
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
	// Add search filter if provided
	if params.Search != "" {
		searchFields := []string{"first_name", "last_name", "email", "phone"}
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

		// Also cache by email and phone for auth lookups
		if user.Email != "" {
			emailKey := fmt.Sprintf("user_email_%s", user.Email)
			r.cache.Set(ctx, emailKey, user, 15*time.Minute)
		}

		if user.Phone != "" {
			phoneKey := fmt.Sprintf("user_phone_%s", user.Phone)
			r.cache.Set(ctx, phoneKey, user, 15*time.Minute)
		}

		// Also cache by social ID if exists
		if user.SocialID != "" && user.AuthProvider != "" {
			socialKey := fmt.Sprintf("user_social_%s_%s", user.AuthProvider, user.SocialID)
			r.cache.Set(ctx, socialKey, user, 15*time.Minute)
		}
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

		// Note: We can't easily invalidate the email, phone, and social caches
		// without additional lookups. This is a trade-off for performance vs cache consistency
		// In a production system, you might want to implement a more sophisticated
		// cache invalidation strategy
	}
}
