package services

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"goride/internal/models"
	"goride/internal/repositories/interfaces"
	"goride/internal/utils"
	"goride/pkg/logger"

	"go.mongodb.org/mongo-driver/bson/primitive"
)

type CacheService interface {
	// Basic cache operations
	Get(ctx context.Context, key string, dest interface{}) error
	Set(ctx context.Context, key string, value interface{}, expiration time.Duration) error
	Delete(ctx context.Context, keys ...string) error
	Exists(ctx context.Context, key string) (bool, error)

	// Advanced operations
	SetNX(ctx context.Context, key string, value interface{}, expiration time.Duration) (bool, error)
	Increment(ctx context.Context, key string, delta int64, expiration time.Duration) (int64, error)
	Decrement(ctx context.Context, key string, delta int64, expiration time.Duration) (int64, error)
	SetExpire(ctx context.Context, key string, expiration time.Duration) error
	GetTTL(ctx context.Context, key string) (time.Duration, error)

	// Set operations
	SAdd(ctx context.Context, key string, members ...interface{}) error
	SMembers(ctx context.Context, key string) ([]string, error)
	SRem(ctx context.Context, key string, members ...interface{}) error
	SIsMember(ctx context.Context, key string, member interface{}) (bool, error)
	SCard(ctx context.Context, key string) (int64, error)

	// Hash operations
	HSet(ctx context.Context, key string, field string, value interface{}) error
	HGet(ctx context.Context, key string, field string, dest interface{}) error
	HGetAll(ctx context.Context, key string) (map[string]string, error)
	HDel(ctx context.Context, key string, fields ...string) error
	HExists(ctx context.Context, key string, field string) (bool, error)

	// List operations
	LPush(ctx context.Context, key string, values ...interface{}) error
	RPush(ctx context.Context, key string, values ...interface{}) error
	LPop(ctx context.Context, key string, dest interface{}) error
	RPop(ctx context.Context, key string, dest interface{}) error
	LLen(ctx context.Context, key string) (int64, error)
	LRange(ctx context.Context, key string, start, stop int64) ([]string, error)

	// Geospatial operations
	GeoAdd(ctx context.Context, key string, locations ...*GeoLocation) error
	GeoRadius(ctx context.Context, key string, longitude, latitude, radius float64, unit string) ([]*GeoLocation, error)
	GeoPos(ctx context.Context, key string, members ...string) ([]*GeoPosition, error)
	GeoDist(ctx context.Context, key string, member1, member2, unit string) (float64, error)

	// Lock operations
	Lock(ctx context.Context, key string, expiration time.Duration) (*DistributedLock, error)
	Unlock(ctx context.Context, lock *DistributedLock) error

	// Pub/Sub operations
	Publish(ctx context.Context, channel string, message interface{}) error
	Subscribe(ctx context.Context, channels ...string) (*Subscription, error)
	Unsubscribe(ctx context.Context, subscription *Subscription, channels ...string) error

	// Pattern operations
	Keys(ctx context.Context, pattern string) ([]string, error)
	DeletePattern(ctx context.Context, pattern string) (int64, error)

	// Cache warming and invalidation
	WarmCache(ctx context.Context, keys []CacheWarmupItem) error
	InvalidateByTags(ctx context.Context, tags ...string) error
	InvalidateByPattern(ctx context.Context, pattern string) error

	// Health and monitoring
	Ping(ctx context.Context) error
	Stats(ctx context.Context) (*CacheStats, error)
	FlushAll(ctx context.Context) error
	FlushDB(ctx context.Context, db int) error

	// Session cache specific operations
	SetSession(ctx context.Context, sessionID string, session *UserSession, expiration time.Duration) error
	GetSession(ctx context.Context, sessionID string) (*UserSession, error)
	DeleteSession(ctx context.Context, sessionID string) error
	GetUserSessions(ctx context.Context, userID primitive.ObjectID) ([]*UserSession, error)

	// Application-specific cache operations
	CacheUser(ctx context.Context, user *models.User, expiration time.Duration) error
	GetCachedUser(ctx context.Context, userID primitive.ObjectID) (*models.User, error)
	InvalidateUser(ctx context.Context, userID primitive.ObjectID) error

	CacheDriver(ctx context.Context, driver *models.Driver, expiration time.Duration) error
	GetCachedDriver(ctx context.Context, driverID primitive.ObjectID) (*models.Driver, error)
	InvalidateDriver(ctx context.Context, driverID primitive.ObjectID) error

	CacheRide(ctx context.Context, ride *models.Ride, expiration time.Duration) error
	GetCachedRide(ctx context.Context, rideID primitive.ObjectID) (*models.Ride, error)
	InvalidateRide(ctx context.Context, rideID primitive.ObjectID) error

	// Real-time data caching
	SetDriverLocation(ctx context.Context, driverID primitive.ObjectID, location *models.Location, expiration time.Duration) error
	GetDriverLocation(ctx context.Context, driverID primitive.ObjectID) (*models.Location, error)
	GetNearbyDrivers(ctx context.Context, location *models.Location, radius float64, unit string) ([]*DriverLocationData, error)

	// Rate limiting
	CheckRateLimit(ctx context.Context, key string, limit int64, window time.Duration) (*RateLimitResult, error)
	IncrementRateLimit(ctx context.Context, key string, window time.Duration) (int64, error)

	// Performance optimization
	Pipeline(ctx context.Context) Pipeline
	Multi(ctx context.Context) Multi
}

type cacheService struct {
	redisClient  RedisClient
	userRepo     interfaces.UserRepository
	driverRepo   interfaces.DriverRepository
	rideRepo     interfaces.RideRepository
	auditLogRepo interfaces.AuditLogRepository
	logger       *logger.Logger
	defaultTTL   time.Duration
	keyPrefix    string
}

type RedisClient interface {
	Get(ctx context.Context, key string) (string, error)
	Set(ctx context.Context, key string, value interface{}, expiration time.Duration) error
	Del(ctx context.Context, keys ...string) error
	Exists(ctx context.Context, keys ...string) (int64, error)
	SetNX(ctx context.Context, key string, value interface{}, expiration time.Duration) (bool, error)
	Incr(ctx context.Context, key string) (int64, error)
	IncrBy(ctx context.Context, key string, value int64) (int64, error)
	Expire(ctx context.Context, key string, expiration time.Duration) (bool, error)
	TTL(ctx context.Context, key string) (time.Duration, error)
	SAdd(ctx context.Context, key string, members ...interface{}) (int64, error)
	SMembers(ctx context.Context, key string) ([]string, error)
	SRem(ctx context.Context, key string, members ...interface{}) (int64, error)
	SIsMember(ctx context.Context, key string, member interface{}) (bool, error)
	SCard(ctx context.Context, key string) (int64, error)
	HSet(ctx context.Context, key string, values ...interface{}) (int64, error)
	HGet(ctx context.Context, key string, field string) (string, error)
	HGetAll(ctx context.Context, key string) (map[string]string, error)
	HDel(ctx context.Context, key string, fields ...string) (int64, error)
	HExists(ctx context.Context, key string, field string) (bool, error)
	LPush(ctx context.Context, key string, values ...interface{}) (int64, error)
	RPush(ctx context.Context, key string, values ...interface{}) (int64, error)
	LPop(ctx context.Context, key string) (string, error)
	RPop(ctx context.Context, key string) (string, error)
	LLen(ctx context.Context, key string) (int64, error)
	LRange(ctx context.Context, key string, start, stop int64) ([]string, error)
	GeoAdd(ctx context.Context, key string, geoLocation ...*GeoLocation) (int64, error)
	GeoRadius(ctx context.Context, key string, query *GeoRadiusQuery) ([]GeoLocation, error)
	GeoPos(ctx context.Context, key string, members ...string) ([]*GeoPosition, error)
	GeoDist(ctx context.Context, key string, member1, member2, unit string) (float64, error)
	Publish(ctx context.Context, channel string, message interface{}) (int64, error)
	Subscribe(ctx context.Context, channels ...string) *PubSub
	Keys(ctx context.Context, pattern string) ([]string, error)
	Ping(ctx context.Context) (string, error)
	FlushAll(ctx context.Context) (string, error)
	FlushDB(ctx context.Context) (string, error)
	Pipeline() Pipeliner
	TxPipeline() Pipeliner
}

type GeoLocation struct {
	Name      string  `json:"name"`
	Longitude float64 `json:"longitude"`
	Latitude  float64 `json:"latitude"`
	Member    string  `json:"member"`
	Dist      float64 `json:"dist"`
}

type GeoPosition struct {
	Longitude float64 `json:"longitude"`
	Latitude  float64 `json:"latitude"`
}

type GeoRadiusQuery struct {
	Longitude   float64
	Latitude    float64
	Radius      float64
	Unit        string
	WithCoord   bool
	WithDist    bool
	WithGeoHash bool
	Count       int
	Sort        string
}

type DistributedLock struct {
	Key        string        `json:"key"`
	Value      string        `json:"value"`
	Expiration time.Duration `json:"expiration"`
	CreatedAt  time.Time     `json:"created_at"`
}

type Subscription struct {
	Channels []string      `json:"channels"`
	Messages chan *Message `json:"-"`
	PubSub   *PubSub       `json:"-"`
}

type PubSub interface {
	Subscribe(ctx context.Context, channels ...string) error
	Unsubscribe(ctx context.Context, channels ...string) error
	Receive(ctx context.Context) (interface{}, error)
	Close() error
}

type Message struct {
	Channel string `json:"channel"`
	Pattern string `json:"pattern"`
	Payload string `json:"payload"`
}

type Pipeline interface {
	Exec(ctx context.Context) error
	Set(ctx context.Context, key string, value interface{}, expiration time.Duration) error
	Get(ctx context.Context, key string) (string, error)
	Del(ctx context.Context, keys ...string) error
}

type Multi interface {
	Exec(ctx context.Context) ([]interface{}, error)
	Watch(ctx context.Context, keys ...string) error
	Unwatch(ctx context.Context) error
}

type Pipeliner interface {
	Exec(ctx context.Context) ([]interface{}, error)
	Set(ctx context.Context, key string, value interface{}, expiration time.Duration) *StringCmd
	Get(ctx context.Context, key string) *StringCmd
	Del(ctx context.Context, keys ...string) *IntCmd
}

type StringCmd interface {
	Result() (string, error)
	Val() string
	Err() error
}

type IntCmd interface {
	Result() (int64, error)
	Val() int64
	Err() error
}

type CacheWarmupItem struct {
	Key        string        `json:"key"`
	Value      interface{}   `json:"value"`
	Expiration time.Duration `json:"expiration"`
	Tags       []string      `json:"tags"`
}

type CacheStats struct {
	HitRate        float64                   `json:"hit_rate"`
	MissRate       float64                   `json:"miss_rate"`
	TotalHits      int64                     `json:"total_hits"`
	TotalMisses    int64                     `json:"total_misses"`
	TotalKeys      int64                     `json:"total_keys"`
	Memory         *MemoryStats              `json:"memory"`
	Connections    *ConnectionStats          `json:"connections"`
	KeyspaceStats  map[string]*KeyspaceStats `json:"keyspace_stats"`
	LastUpdateTime time.Time                 `json:"last_update_time"`
}

type MemoryStats struct {
	Used          int64   `json:"used"`
	Peak          int64   `json:"peak"`
	Total         int64   `json:"total"`
	Available     int64   `json:"available"`
	Fragmentation float64 `json:"fragmentation"`
}

type ConnectionStats struct {
	Active   int64 `json:"active"`
	Idle     int64 `json:"idle"`
	Total    int64 `json:"total"`
	Rejected int64 `json:"rejected"`
}

type KeyspaceStats struct {
	Keys    int64 `json:"keys"`
	Expires int64 `json:"expires"`
	AvgTTL  int64 `json:"avg_ttl"`
}

type DriverLocationData struct {
	DriverID  primitive.ObjectID `json:"driver_id"`
	Location  *models.Location   `json:"location"`
	Timestamp time.Time          `json:"timestamp"`
	Status    string             `json:"status"`
	Distance  float64            `json:"distance"`
}

type RateLimitResult struct {
	Allowed    bool          `json:"allowed"`
	Count      int64         `json:"count"`
	Remaining  int64         `json:"remaining"`
	ResetTime  time.Time     `json:"reset_time"`
	RetryAfter time.Duration `json:"retry_after"`
}

func NewCacheService(
	redisClient RedisClient,
	userRepo interfaces.UserRepository,
	driverRepo interfaces.DriverRepository,
	rideRepo interfaces.RideRepository,
	auditLogRepo interfaces.AuditLogRepository,
	logger *logger.Logger,
	keyPrefix string,
	defaultTTL time.Duration,
) CacheService {
	return &cacheService{
		redisClient:  redisClient,
		userRepo:     userRepo,
		driverRepo:   driverRepo,
		rideRepo:     rideRepo,
		auditLogRepo: auditLogRepo,
		logger:       logger,
		keyPrefix:    keyPrefix,
		defaultTTL:   defaultTTL,
	}
}

// Basic cache operations
func (s *cacheService) Get(ctx context.Context, key string, dest interface{}) error {
	fullKey := s.buildKey(key)

	data, err := s.redisClient.Get(ctx, fullKey)
	if err != nil {
		return fmt.Errorf("failed to get cache key %s: %w", key, err)
	}

	if err := json.Unmarshal([]byte(data), dest); err != nil {
		return fmt.Errorf("failed to unmarshal cache data: %w", err)
	}

	s.logger.WithField("cache_key", key).Debug("Cache hit")
	return nil
}

func (s *cacheService) Set(ctx context.Context, key string, value interface{}, expiration time.Duration) error {
	fullKey := s.buildKey(key)

	data, err := json.Marshal(value)
	if err != nil {
		return fmt.Errorf("failed to marshal cache data: %w", err)
	}

	if expiration == 0 {
		expiration = s.defaultTTL
	}

	if err := s.redisClient.Set(ctx, fullKey, data, expiration); err != nil {
		return fmt.Errorf("failed to set cache key %s: %w", key, err)
	}

	s.logger.WithField("cache_key", key).
		WithField("expiration", expiration).
		Debug("Cache set")

	return nil
}

func (s *cacheService) Delete(ctx context.Context, keys ...string) error {
	fullKeys := make([]string, len(keys))
	for i, key := range keys {
		fullKeys[i] = s.buildKey(key)
	}

	if err := s.redisClient.Del(ctx, fullKeys...); err != nil {
		return fmt.Errorf("failed to delete cache keys: %w", err)
	}

	s.logger.WithField("cache_keys", keys).Debug("Cache keys deleted")
	return nil
}

func (s *cacheService) Exists(ctx context.Context, key string) (bool, error) {
	fullKey := s.buildKey(key)

	count, err := s.redisClient.Exists(ctx, fullKey)
	if err != nil {
		return false, fmt.Errorf("failed to check cache key existence: %w", err)
	}

	return count > 0, nil
}

// Advanced operations
func (s *cacheService) SetNX(ctx context.Context, key string, value interface{}, expiration time.Duration) (bool, error) {
	fullKey := s.buildKey(key)

	data, err := json.Marshal(value)
	if err != nil {
		return false, fmt.Errorf("failed to marshal cache data: %w", err)
	}

	if expiration == 0 {
		expiration = s.defaultTTL
	}

	result, err := s.redisClient.SetNX(ctx, fullKey, data, expiration)
	if err != nil {
		return false, fmt.Errorf("failed to set cache key if not exists: %w", err)
	}

	return result, nil
}

func (s *cacheService) Increment(ctx context.Context, key string, delta int64, expiration time.Duration) (int64, error) {
	fullKey := s.buildKey(key)

	result, err := s.redisClient.IncrBy(ctx, fullKey, delta)
	if err != nil {
		return 0, fmt.Errorf("failed to increment cache key: %w", err)
	}

	if expiration > 0 {
		s.redisClient.Expire(ctx, fullKey, expiration)
	}

	return result, nil
}

func (s *cacheService) Decrement(ctx context.Context, key string, delta int64, expiration time.Duration) (int64, error) {
	return s.Increment(ctx, key, -delta, expiration)
}

func (s *cacheService) SetExpire(ctx context.Context, key string, expiration time.Duration) error {
	fullKey := s.buildKey(key)

	_, err := s.redisClient.Expire(ctx, fullKey, expiration)
	if err != nil {
		return fmt.Errorf("failed to set expiration: %w", err)
	}

	return nil
}

func (s *cacheService) GetTTL(ctx context.Context, key string) (time.Duration, error) {
	fullKey := s.buildKey(key)

	ttl, err := s.redisClient.TTL(ctx, fullKey)
	if err != nil {
		return 0, fmt.Errorf("failed to get TTL: %w", err)
	}

	return ttl, nil
}

// Session cache operations
func (s *cacheService) SetSession(ctx context.Context, sessionID string, session *UserSession, expiration time.Duration) error {
	key := fmt.Sprintf("session:%s", sessionID)
	return s.Set(ctx, key, session, expiration)
}

func (s *cacheService) GetSession(ctx context.Context, sessionID string) (*UserSession, error) {
	key := fmt.Sprintf("session:%s", sessionID)
	var session UserSession

	if err := s.Get(ctx, key, &session); err != nil {
		return nil, err
	}

	return &session, nil
}

func (s *cacheService) DeleteSession(ctx context.Context, sessionID string) error {
	key := fmt.Sprintf("session:%s", sessionID)
	return s.Delete(ctx, key)
}

func (s *cacheService) GetUserSessions(ctx context.Context, userID primitive.ObjectID) ([]*UserSession, error) {
	pattern := "session:*"
	keys, err := s.Keys(ctx, pattern)
	if err != nil {
		return nil, err
	}

	var sessions []*UserSession
	for _, key := range keys {
		var session UserSession
		if err := s.Get(ctx, key, &session); err != nil {
			continue // Skip invalid sessions
		}

		if session.UserID == userID {
			sessions = append(sessions, &session)
		}
	}

	return sessions, nil
}

// Application-specific cache operations
func (s *cacheService) CacheUser(ctx context.Context, user *models.User, expiration time.Duration) error {
	key := fmt.Sprintf("user:%s", user.ID.Hex())
	return s.Set(ctx, key, user, expiration)
}

func (s *cacheService) GetCachedUser(ctx context.Context, userID primitive.ObjectID) (*models.User, error) {
	key := fmt.Sprintf("user:%s", userID.Hex())
	var user models.User

	if err := s.Get(ctx, key, &user); err != nil {
		return nil, err
	}

	return &user, nil
}

func (s *cacheService) InvalidateUser(ctx context.Context, userID primitive.ObjectID) error {
	keys := []string{
		fmt.Sprintf("user:%s", userID.Hex()),
		fmt.Sprintf("user_profile:%s", userID.Hex()),
		fmt.Sprintf("user_permissions:%s", userID.Hex()),
	}

	return s.Delete(ctx, keys...)
}

func (s *cacheService) CacheDriver(ctx context.Context, driver *models.Driver, expiration time.Duration) error {
	key := fmt.Sprintf("driver:%s", driver.ID.Hex())
	return s.Set(ctx, key, driver, expiration)
}

func (s *cacheService) GetCachedDriver(ctx context.Context, driverID primitive.ObjectID) (*models.Driver, error) {
	key := fmt.Sprintf("driver:%s", driverID.Hex())
	var driver models.Driver

	if err := s.Get(ctx, key, &driver); err != nil {
		return nil, err
	}

	return &driver, nil
}

func (s *cacheService) InvalidateDriver(ctx context.Context, driverID primitive.ObjectID) error {
	keys := []string{
		fmt.Sprintf("driver:%s", driverID.Hex()),
		fmt.Sprintf("driver_location:%s", driverID.Hex()),
		fmt.Sprintf("driver_status:%s", driverID.Hex()),
	}

	return s.Delete(ctx, keys...)
}

func (s *cacheService) CacheRide(ctx context.Context, ride *models.Ride, expiration time.Duration) error {
	key := fmt.Sprintf("ride:%s", ride.ID.Hex())
	return s.Set(ctx, key, ride, expiration)
}

func (s *cacheService) GetCachedRide(ctx context.Context, rideID primitive.ObjectID) (*models.Ride, error) {
	key := fmt.Sprintf("ride:%s", rideID.Hex())
	var ride models.Ride

	if err := s.Get(ctx, key, &ride); err != nil {
		return nil, err
	}

	return &ride, nil
}

func (s *cacheService) InvalidateRide(ctx context.Context, rideID primitive.ObjectID) error {
	key := fmt.Sprintf("ride:%s", rideID.Hex())
	return s.Delete(ctx, key)
}

// Real-time data caching
func (s *cacheService) SetDriverLocation(ctx context.Context, driverID primitive.ObjectID, location *models.Location, expiration time.Duration) error {
	// Store in both regular cache and geospatial index
	key := fmt.Sprintf("driver_location:%s", driverID.Hex())
	if err := s.Set(ctx, key, location, expiration); err != nil {
		return err
	}

	// Add to geospatial index
	geoKey := "drivers_geo"
	geoLocation := &GeoLocation{
		Name:      driverID.Hex(),
		Longitude: location.Longitude(),
		Latitude:  location.Latitude(),
		Member:    driverID.Hex(),
	}

	return s.GeoAdd(ctx, geoKey, geoLocation)
}

func (s *cacheService) GetDriverLocation(ctx context.Context, driverID primitive.ObjectID) (*models.Location, error) {
	key := fmt.Sprintf("driver_location:%s", driverID.Hex())
	var location models.Location

	if err := s.Get(ctx, key, &location); err != nil {
		return nil, err
	}

	return &location, nil
}

func (s *cacheService) GetNearbyDrivers(ctx context.Context, location *models.Location, radius float64, unit string) ([]*DriverLocationData, error) {
	geoKey := "drivers_geo"

	locations, err := s.GeoRadius(ctx, geoKey, location.Longitude(), location.Latitude(), radius, unit)
	if err != nil {
		return nil, err
	}

	var drivers []*DriverLocationData
	for _, loc := range locations {
		driverID, err := primitive.ObjectIDFromHex(loc.Member)
		if err != nil {
			continue
		}

		driverLocation := &models.Location{
			Type:        "Point",
			Coordinates: []float64{loc.Longitude, loc.Latitude},
		}

		drivers = append(drivers, &DriverLocationData{
			DriverID:  driverID,
			Location:  driverLocation,
			Timestamp: time.Now(),
			Distance:  loc.Dist,
		})
	}

	return drivers, nil
}

// Rate limiting
func (s *cacheService) CheckRateLimit(ctx context.Context, key string, limit int64, window time.Duration) (*RateLimitResult, error) {
	rateLimitKey := fmt.Sprintf("rate_limit:%s", key)

	count, err := s.redisClient.IncrBy(ctx, rateLimitKey, 1)
	if err != nil {
		return nil, fmt.Errorf("failed to increment rate limit counter: %w", err)
	}

	if count == 1 {
		s.redisClient.Expire(ctx, rateLimitKey, window)
	}

	remaining := limit - count
	if remaining < 0 {
		remaining = 0
	}

	resetTime := time.Now().Add(window)
	retryAfter := time.Duration(0)
	if count > limit {
		ttl, _ := s.redisClient.TTL(ctx, rateLimitKey)
		retryAfter = ttl
	}

	return &RateLimitResult{
		Allowed:    count <= limit,
		Count:      count,
		Remaining:  remaining,
		ResetTime:  resetTime,
		RetryAfter: retryAfter,
	}, nil
}

func (s *cacheService) IncrementRateLimit(ctx context.Context, key string, window time.Duration) (int64, error) {
	rateLimitKey := fmt.Sprintf("rate_limit:%s", key)

	count, err := s.redisClient.IncrBy(ctx, rateLimitKey, 1)
	if err != nil {
		return 0, fmt.Errorf("failed to increment rate limit: %w", err)
	}

	if count == 1 {
		s.redisClient.Expire(ctx, rateLimitKey, window)
	}

	return count, nil
}

// Helper methods
func (s *cacheService) buildKey(key string) string {
	if s.keyPrefix != "" {
		return fmt.Sprintf("%s:%s", s.keyPrefix, key)
	}
	return key
}

func (s *cacheService) Ping(ctx context.Context) error {
	_, err := s.redisClient.Ping(ctx)
	return err
}

func (s *cacheService) Keys(ctx context.Context, pattern string) ([]string, error) {
	fullPattern := s.buildKey(pattern)
	return s.redisClient.Keys(ctx, fullPattern)
}

// Implement remaining interface methods with placeholder implementations or delegations to Redis client
func (s *cacheService) SAdd(ctx context.Context, key string, members ...interface{}) error {
	fullKey := s.buildKey(key)
	_, err := s.redisClient.SAdd(ctx, fullKey, members...)
	return err
}

func (s *cacheService) SMembers(ctx context.Context, key string) ([]string, error) {
	fullKey := s.buildKey(key)
	return s.redisClient.SMembers(ctx, fullKey)
}

func (s *cacheService) SRem(ctx context.Context, key string, members ...interface{}) error {
	fullKey := s.buildKey(key)
	_, err := s.redisClient.SRem(ctx, fullKey, members...)
	return err
}

func (s *cacheService) SIsMember(ctx context.Context, key string, member interface{}) (bool, error) {
	fullKey := s.buildKey(key)
	return s.redisClient.SIsMember(ctx, fullKey, member)
}

func (s *cacheService) SCard(ctx context.Context, key string) (int64, error) {
	fullKey := s.buildKey(key)
	return s.redisClient.SCard(ctx, fullKey)
}

func (s *cacheService) HSet(ctx context.Context, key string, field string, value interface{}) error {
	fullKey := s.buildKey(key)
	data, err := json.Marshal(value)
	if err != nil {
		return err
	}
	_, err = s.redisClient.HSet(ctx, fullKey, field, data)
	return err
}

func (s *cacheService) HGet(ctx context.Context, key string, field string, dest interface{}) error {
	fullKey := s.buildKey(key)
	data, err := s.redisClient.HGet(ctx, fullKey, field)
	if err != nil {
		return err
	}
	return json.Unmarshal([]byte(data), dest)
}

func (s *cacheService) HGetAll(ctx context.Context, key string) (map[string]string, error) {
	fullKey := s.buildKey(key)
	return s.redisClient.HGetAll(ctx, fullKey)
}

func (s *cacheService) HDel(ctx context.Context, key string, fields ...string) error {
	fullKey := s.buildKey(key)
	_, err := s.redisClient.HDel(ctx, fullKey, fields...)
	return err
}

func (s *cacheService) HExists(ctx context.Context, key string, field string) (bool, error) {
	fullKey := s.buildKey(key)
	return s.redisClient.HExists(ctx, fullKey, field)
}

func (s *cacheService) LPush(ctx context.Context, key string, values ...interface{}) error {
	fullKey := s.buildKey(key)
	_, err := s.redisClient.LPush(ctx, fullKey, values...)
	return err
}

func (s *cacheService) RPush(ctx context.Context, key string, values ...interface{}) error {
	fullKey := s.buildKey(key)
	_, err := s.redisClient.RPush(ctx, fullKey, values...)
	return err
}

func (s *cacheService) LPop(ctx context.Context, key string, dest interface{}) error {
	fullKey := s.buildKey(key)
	data, err := s.redisClient.LPop(ctx, fullKey)
	if err != nil {
		return err
	}
	return json.Unmarshal([]byte(data), dest)
}

func (s *cacheService) RPop(ctx context.Context, key string, dest interface{}) error {
	fullKey := s.buildKey(key)
	data, err := s.redisClient.RPop(ctx, fullKey)
	if err != nil {
		return err
	}
	return json.Unmarshal([]byte(data), dest)
}

func (s *cacheService) LLen(ctx context.Context, key string) (int64, error) {
	fullKey := s.buildKey(key)
	return s.redisClient.LLen(ctx, fullKey)
}

func (s *cacheService) LRange(ctx context.Context, key string, start, stop int64) ([]string, error) {
	fullKey := s.buildKey(key)
	return s.redisClient.LRange(ctx, fullKey, start, stop)
}

func (s *cacheService) GeoAdd(ctx context.Context, key string, locations ...*GeoLocation) error {
	fullKey := s.buildKey(key)
	_, err := s.redisClient.GeoAdd(ctx, fullKey, locations...)
	return err
}

func (s *cacheService) GeoRadius(ctx context.Context, key string, longitude, latitude, radius float64, unit string) ([]*GeoLocation, error) {
	fullKey := s.buildKey(key)
	query := &GeoRadiusQuery{
		Radius:    radius,
		Unit:      unit,
		WithCoord: true,
		WithDist:  true,
		Sort:      "ASC",
		Longitude: longitude,
		Latitude:  latitude,
	}

	locations, err := s.redisClient.GeoRadius(ctx, fullKey, query)
	if err != nil {
		return nil, err
	}

	result := make([]*GeoLocation, len(locations))
	for i, loc := range locations {
		result[i] = &loc
	}

	return result, nil
}

func (s *cacheService) GeoPos(ctx context.Context, key string, members ...string) ([]*GeoPosition, error) {
	fullKey := s.buildKey(key)
	return s.redisClient.GeoPos(ctx, fullKey, members...)
}

func (s *cacheService) GeoDist(ctx context.Context, key string, member1, member2, unit string) (float64, error) {
	fullKey := s.buildKey(key)
	return s.redisClient.GeoDist(ctx, fullKey, member1, member2, unit)
}

func (s *cacheService) Lock(ctx context.Context, key string, expiration time.Duration) (*DistributedLock, error) {
	lockKey := fmt.Sprintf("lock:%s", key)
	lockValue := utils.GenerateRandomString(32)

	success, err := s.SetNX(ctx, lockKey, lockValue, expiration)
	if err != nil {
		return nil, err
	}

	if !success {
		return nil, fmt.Errorf("failed to acquire lock")
	}

	return &DistributedLock{
		Key:        lockKey,
		Value:      lockValue,
		Expiration: expiration,
		CreatedAt:  time.Now(),
	}, nil
}

func (s *cacheService) Unlock(ctx context.Context, lock *DistributedLock) error {
	// TODO: Implement Lua script for atomic unlock
	return s.Delete(ctx, lock.Key)
}

func (s *cacheService) Publish(ctx context.Context, channel string, message interface{}) error {
	data, err := json.Marshal(message)
	if err != nil {
		return err
	}

	_, err = s.redisClient.Publish(ctx, channel, data)
	return err
}

func (s *cacheService) Subscribe(ctx context.Context, channels ...string) (*Subscription, error) {
	pubsub := s.redisClient.Subscribe(ctx, channels...)

	return &Subscription{
		Channels: channels,
		Messages: make(chan *Message, 100),
		PubSub:   pubsub,
	}, nil
}

func (s *cacheService) Unsubscribe(ctx context.Context, subscription *Subscription, channels ...string) error {
	return (*subscription.PubSub).Unsubscribe(ctx, channels...)
}

func (s *cacheService) DeletePattern(ctx context.Context, pattern string) (int64, error) {
	keys, err := s.Keys(ctx, pattern)
	if err != nil {
		return 0, err
	}

	if len(keys) == 0 {
		return 0, nil
	}

	err = s.Delete(ctx, keys...)
	return int64(len(keys)), err
}

func (s *cacheService) WarmCache(ctx context.Context, keys []CacheWarmupItem) error {
	for _, item := range keys {
		if err := s.Set(ctx, item.Key, item.Value, item.Expiration); err != nil {
			s.logger.WithError(err).WithField("key", item.Key).Error("Failed to warm cache key")
			continue
		}
	}
	return nil
}

func (s *cacheService) InvalidateByTags(ctx context.Context, tags ...string) error {
	// TODO: Implement tag-based invalidation
	return fmt.Errorf("invalidate by tags not implemented")
}

func (s *cacheService) InvalidateByPattern(ctx context.Context, pattern string) error {
	_, err := s.DeletePattern(ctx, pattern)
	return err
}

func (s *cacheService) Stats(ctx context.Context) (*CacheStats, error) {
	// TODO: Implement cache statistics gathering
	return &CacheStats{
		LastUpdateTime: time.Now(),
		KeyspaceStats:  make(map[string]*KeyspaceStats),
	}, nil
}

func (s *cacheService) FlushAll(ctx context.Context) error {
	_, err := s.redisClient.FlushAll(ctx)
	return err
}

func (s *cacheService) FlushDB(ctx context.Context, db int) error {
	_, err := s.redisClient.FlushDB(ctx)
	return err
}

func (s *cacheService) Pipeline(ctx context.Context) Pipeline {
	return &pipelineWrapper{pipeliner: s.redisClient.Pipeline()}
}

func (s *cacheService) Multi(ctx context.Context) Multi {
	return &multiWrapper{pipeliner: s.redisClient.TxPipeline()}
}

// Wrapper types for Pipeline and Multi
type pipelineWrapper struct {
	pipeliner Pipeliner
}

func (p *pipelineWrapper) Exec(ctx context.Context) error {
	_, err := p.pipeliner.Exec(ctx)
	return err
}

func (p *pipelineWrapper) Set(ctx context.Context, key string, value interface{}, expiration time.Duration) error {
	data, err := json.Marshal(value)
	if err != nil {
		return err
	}
	p.pipeliner.Set(ctx, key, data, expiration)
	return nil
}

func (p *pipelineWrapper) Get(ctx context.Context, key string) (string, error) {
	cmd := p.pipeliner.Get(ctx, key)
	return (*cmd).Result()
}

func (p *pipelineWrapper) Del(ctx context.Context, keys ...string) error {
	p.pipeliner.Del(ctx, keys...)
	return nil
}

type multiWrapper struct {
	pipeliner Pipeliner
}

func (m *multiWrapper) Exec(ctx context.Context) ([]interface{}, error) {
	return m.pipeliner.Exec(ctx)
}

func (m *multiWrapper) Watch(ctx context.Context, keys ...string) error {
	// TODO: Implement WATCH command
	return fmt.Errorf("watch not implemented")
}

func (m *multiWrapper) Unwatch(ctx context.Context) error {
	// TODO: Implement UNWATCH command
	return fmt.Errorf("unwatch not implemented")
}
