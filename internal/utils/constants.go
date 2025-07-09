package utils

import "time"

// Application Constants
const (
	AppName    = "UberClone"
	AppVersion = "1.0.0"

	// Default values
	DefaultLanguage    = "en"
	DefaultCurrency    = "USD"
	DefaultCountryCode = "+1"
	DefaultTimeZone    = "UTC"

	// Pagination
	DefaultPageSize = 20
	MaxPageSize     = 100
	MinPageSize     = 1

	// Authentication
	JWTSecretKey       = "jwt_secret_key"
	JWTAccessTokenTTL  = 24 * time.Hour
	JWTRefreshTokenTTL = 7 * 24 * time.Hour
	PasswordMinLength  = 8
	PasswordMaxLength  = 128
	OTPLength          = 6
	OTPExpiry          = 10 * time.Minute

	// Ride Constants
	MaxWaitTime         = 15 * time.Minute
	DefaultSearchRadius = 10.0 // kilometers
	MaxSearchRadius     = 50.0 // kilometers
	RideRequestTimeout  = 5 * time.Minute
	MaxWaypoints        = 5
	MaxRideDistance     = 500.0 // kilometers

	// Driver Constants
	MinDriverRating              = 1.0
	MaxDriverRating              = 5.0
	MinAcceptanceRate            = 0.8
	MaxCancellationRate          = 0.1
	DriverLocationUpdateInterval = 30 * time.Second

	// Payment Constants
	MinFare              = 2.0
	MaxFare              = 1000.0
	DefaultTipPercentage = 0.15
	MaxTipAmount         = 100.0
	RefundProcessingTime = 3 * 24 * time.Hour

	// File Upload
	MaxImageSize    = 5 * 1024 * 1024   // 5MB
	MaxDocumentSize = 10 * 1024 * 1024  // 10MB
	MaxAudioSize    = 50 * 1024 * 1024  // 50MB
	MaxVideoSize    = 100 * 1024 * 1024 // 100MB

	// Rate Limiting
	DefaultRateLimit = 100
	LoginRateLimit   = 5
	OTPRateLimit     = 3

	// Emergency
	EmergencyResponseTime = 5 * time.Minute
	SOSAutoCancel         = 30 * time.Minute

	// Surge Pricing
	MinSurgeMultiplier  = 1.0
	MaxSurgeMultiplier  = 5.0
	SurgeUpdateInterval = 5 * time.Minute

	// Notification
	NotificationRetryAttempts = 3
	NotificationTimeout       = 30 * time.Second

	// Chat
	MaxMessageLength   = 1000
	ChatSessionTimeout = 24 * time.Hour

	// Loyalty
	DefaultPointsPerRide   = 10
	DefaultPointsPerDollar = 1
	MinRedemptionPoints    = 100

	// Referral
	DefaultReferralReward = 10.0
	ReferralCodeLength    = 8
	ReferralExpiry        = 90 * 24 * time.Hour
)

// HTTP Status Messages
const (
	StatusSuccess = "success"
	StatusError   = "error"
	StatusFailed  = "failed"
)

// Error Messages
const (
	ErrInvalidCredentials = "invalid credentials"
	ErrUserNotFound       = "user not found"
	ErrUserExists         = "user already exists"
	ErrInvalidToken       = "invalid token"
	ErrTokenExpired       = "token expired"
	ErrInvalidInput       = "invalid input"
	ErrInternalServer     = "internal server error"
	ErrUnauthorized       = "unauthorized"
	ErrForbidden          = "forbidden"
	ErrNotFound           = "not found"
	ErrConflict           = "conflict"
	ErrValidationFailed   = "validation failed"
	ErrFileUploadFailed   = "file upload failed"
	ErrPaymentFailed      = "payment failed"
	ErrRideNotFound       = "ride not found"
	ErrDriverNotFound     = "driver not found"
	ErrNoDriversAvailable = "no drivers available"
)

// Cache Keys
const (
	CacheUserPrefix         = "user:"
	CacheDriverPrefix       = "driver:"
	CacheRidePrefix         = "ride:"
	CacheLocationPrefix     = "location:"
	CacheSurgePricingPrefix = "surge:"
	CachePromotionPrefix    = "promotion:"
	CacheRateLimitPrefix    = "rate_limit:"
	CacheOTPPrefix          = "otp:"
	CacheSessionPrefix      = "session:"
)

// Event Types
const (
	EventUserRegistered     = "user_registered"
	EventUserLogin          = "user_login"
	EventRideRequested      = "ride_requested"
	EventRideAccepted       = "ride_accepted"
	EventRideStarted        = "ride_started"
	EventRideCompleted      = "ride_completed"
	EventRideCancelled      = "ride_cancelled"
	EventPaymentProcessed   = "payment_processed"
	EventEmergencyTriggered = "emergency_triggered"
)

// Notification Types
const (
	NotificationPush  = "push"
	NotificationSMS   = "sms"
	NotificationEmail = "email"
	NotificationInApp = "in_app"
)

// File Types
var (
	AllowedImageTypes    = []string{"jpg", "jpeg", "png", "gif", "webp"}
	AllowedDocumentTypes = []string{"pdf", "doc", "docx", "txt"}
	AllowedAudioTypes    = []string{"mp3", "wav", "aac", "m4a"}
	AllowedVideoTypes    = []string{"mp4", "avi", "mov", "wmv"}
)

// Geographic Constants
const (
	EarthRadiusKM    = 6371.0
	EarthRadiusMiles = 3959.0
)
