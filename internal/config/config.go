package config

import (
	"os"
	"strconv"
	"strings"
	"time"
)

type Config struct {
	App       *AppConfig       `yaml:"app"`
	Database  *DatabaseConfig  `yaml:"database"`
	Redis     *RedisConfig     `yaml:"redis"`
	SMTP      *SMTPConfig      `yaml:"smtp"`
	SMS       *SMSConfig       `yaml:"sms"`
	Push      *PushConfig      `yaml:"push"`
	Payment   *PaymentConfig   `yaml:"payment"`
	OAuth     *OAuthConfig     `yaml:"oauth"`
	Maps      *MapsConfig      `yaml:"maps"`
	ML        *MLConfig        `yaml:"ml"`
	Storage   *StorageConfig   `yaml:"storage"`
	WebSocket *WebSocketConfig `yaml:"websocket"`
	Security  *SecurityConfig  `yaml:"security"`
}

type AppConfig struct {
	Name        string `yaml:"name"`
	Version     string `yaml:"version"`
	Environment string `yaml:"environment"`
	Port        int    `yaml:"port"`
	Host        string `yaml:"host"`
	BaseURL     string `yaml:"base_url"`
	Debug       bool   `yaml:"debug"`
	LogLevel    string `yaml:"log_level"`
	Timezone    string `yaml:"timezone"`
	Language    string `yaml:"language"`
	Currency    string `yaml:"currency"`
}

type SecurityConfig struct {
	JWTSecret          string        `yaml:"jwt_secret"`
	JWTAccessTokenTTL  time.Duration `yaml:"jwt_access_token_ttl"`
	JWTRefreshTokenTTL time.Duration `yaml:"jwt_refresh_token_ttl"`
	EncryptionKey      string        `yaml:"encryption_key"`
	PasswordMinLength  int           `yaml:"password_min_length"`
	OTPLength          int           `yaml:"otp_length"`
	OTPExpiry          time.Duration `yaml:"otp_expiry"`
	RateLimitPerMinute int           `yaml:"rate_limit_per_minute"`
	MaxLoginAttempts   int           `yaml:"max_login_attempts"`
	LoginLockoutTime   time.Duration `yaml:"login_lockout_time"`
	CORSAllowedOrigins []string      `yaml:"cors_allowed_origins"`
	TrustedProxies     []string      `yaml:"trusted_proxies"`
}

func Load() (*Config, error) {
	config := &Config{
		App:       loadAppConfig(),
		Database:  loadDatabaseConfig(),
		Redis:     loadRedisConfig(),
		SMTP:      loadSMTPConfig(),
		SMS:       loadSMSConfig(),
		Push:      loadPushConfig(),
		Payment:   loadPaymentConfig(),
		OAuth:     loadOAuthConfig(),
		Maps:      loadMapsConfig(),
		ML:        loadMLConfig(),
		Storage:   loadStorageConfig(),
		WebSocket: loadWebSocketConfig(),
		Security:  loadSecurityConfig(),
	}

	return config, nil
}

func loadAppConfig() *AppConfig {
	return &AppConfig{
		Name:        getEnv("APP_NAME", "UberClone"),
		Version:     getEnv("APP_VERSION", "1.0.0"),
		Environment: getEnv("APP_ENV", "development"),
		Port:        getEnvAsInt("APP_PORT", 8080),
		Host:        getEnv("APP_HOST", "localhost"),
		BaseURL:     getEnv("APP_BASE_URL", "http://localhost:8080"),
		Debug:       getEnvAsBool("APP_DEBUG", true),
		LogLevel:    getEnv("LOG_LEVEL", "info"),
		Timezone:    getEnv("APP_TIMEZONE", "UTC"),
		Language:    getEnv("APP_LANGUAGE", "en"),
		Currency:    getEnv("APP_CURRENCY", "USD"),
	}
}

func loadSecurityConfig() *SecurityConfig {
	return &SecurityConfig{
		JWTSecret:          getEnv("JWT_SECRET", "your-super-secret-jwt-key"),
		JWTAccessTokenTTL:  getEnvAsDuration("JWT_ACCESS_TOKEN_TTL", 24*time.Hour),
		JWTRefreshTokenTTL: getEnvAsDuration("JWT_REFRESH_TOKEN_TTL", 7*24*time.Hour),
		EncryptionKey:      getEnv("ENCRYPTION_KEY", "your-super-secret-encryption-key"),
		PasswordMinLength:  getEnvAsInt("PASSWORD_MIN_LENGTH", 8),
		OTPLength:          getEnvAsInt("OTP_LENGTH", 6),
		OTPExpiry:          getEnvAsDuration("OTP_EXPIRY", 10*time.Minute),
		RateLimitPerMinute: getEnvAsInt("RATE_LIMIT_PER_MINUTE", 100),
		MaxLoginAttempts:   getEnvAsInt("MAX_LOGIN_ATTEMPTS", 5),
		LoginLockoutTime:   getEnvAsDuration("LOGIN_LOCKOUT_TIME", 15*time.Minute),
		CORSAllowedOrigins: getEnvAsSlice("CORS_ALLOWED_ORIGINS", []string{"*"}),
		TrustedProxies:     getEnvAsSlice("TRUSTED_PROXIES", []string{}),
	}
}

func getEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}

func getEnvAsInt(key string, defaultValue int) int {
	if value := os.Getenv(key); value != "" {
		if intValue, err := strconv.Atoi(value); err == nil {
			return intValue
		}
	}
	return defaultValue
}

func getEnvAsBool(key string, defaultValue bool) bool {
	if value := os.Getenv(key); value != "" {
		if boolValue, err := strconv.ParseBool(value); err == nil {
			return boolValue
		}
	}
	return defaultValue
}

func getEnvAsDuration(key string, defaultValue time.Duration) time.Duration {
	if value := os.Getenv(key); value != "" {
		if duration, err := time.ParseDuration(value); err == nil {
			return duration
		}
	}
	return defaultValue
}

func getEnvAsSlice(key string, defaultValue []string) []string {
	if value := os.Getenv(key); value != "" {
		return strings.Split(value, ",")
	}
	return defaultValue
}

func getEnvAsFloat64(key string, defaultValue float64) float64 {
	if value := os.Getenv(key); value != "" {
		if floatValue, err := strconv.ParseFloat(value, 64); err == nil {
			return floatValue
		}
	}
	return defaultValue
}

func IsProduction() bool {
	return getEnv("APP_ENV", "development") == "production"
}

func IsDevelopment() bool {
	return getEnv("APP_ENV", "development") == "development"
}

func IsTest() bool {
	return getEnv("APP_ENV", "development") == "test"
}
