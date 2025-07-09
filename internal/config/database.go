package config

import (
	"time"
)

type DatabaseConfig struct {
	URI            string        `yaml:"uri"`
	Host           string        `yaml:"host"`
	Port           int           `yaml:"port"`
	Database       string        `yaml:"database"`
	Username       string        `yaml:"username"`
	Password       string        `yaml:"password"`
	MaxPoolSize    int           `yaml:"max_pool_size"`
	MinPoolSize    int           `yaml:"min_pool_size"`
	ConnectTimeout time.Duration `yaml:"connect_timeout"`
	SocketTimeout  time.Duration `yaml:"socket_timeout"`
	SSL            bool          `yaml:"ssl"`
	AuthSource     string        `yaml:"auth_source"`
}

func loadDatabaseConfig() *DatabaseConfig {
	return &DatabaseConfig{
		URI:            getEnv("MONGODB_URI", "mongodb://localhost:27017/uber_clone"),
		Host:           getEnv("MONGODB_HOST", "localhost"),
		Port:           getEnvAsInt("MONGODB_PORT", 27017),
		Database:       getEnv("MONGODB_DATABASE", "uber_clone"),
		Username:       getEnv("MONGODB_USERNAME", ""),
		Password:       getEnv("MONGODB_PASSWORD", ""),
		MaxPoolSize:    getEnvAsInt("MONGODB_MAX_POOL_SIZE", 100),
		MinPoolSize:    getEnvAsInt("MONGODB_MIN_POOL_SIZE", 5),
		ConnectTimeout: getEnvAsDuration("MONGODB_CONNECT_TIMEOUT", 10*time.Second),
		SocketTimeout:  getEnvAsDuration("MONGODB_SOCKET_TIMEOUT", 30*time.Second),
		SSL:            getEnvAsBool("MONGODB_SSL", false),
		AuthSource:     getEnv("MONGODB_AUTH_SOURCE", "admin"),
	}
}
