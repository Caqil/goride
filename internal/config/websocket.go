package config

import (
	"time"
)

type WebSocketConfig struct {
	Port              int           `yaml:"port"`
	Path              string        `yaml:"path"`
	ReadBufferSize    int           `yaml:"read_buffer_size"`
	WriteBufferSize   int           `yaml:"write_buffer_size"`
	HandshakeTimeout  time.Duration `yaml:"handshake_timeout"`
	PingInterval      time.Duration `yaml:"ping_interval"`
	PongTimeout       time.Duration `yaml:"pong_timeout"`
	MaxConnections    int           `yaml:"max_connections"`
	EnableCompression bool          `yaml:"enable_compression"`
	AllowedOrigins    []string      `yaml:"allowed_origins"`
}

func loadWebSocketConfig() *WebSocketConfig {
	return &WebSocketConfig{
		Port:              getEnvAsInt("WEBSOCKET_PORT", 8081),
		Path:              getEnv("WEBSOCKET_PATH", "/ws"),
		ReadBufferSize:    getEnvAsInt("WEBSOCKET_READ_BUFFER_SIZE", 1024),
		WriteBufferSize:   getEnvAsInt("WEBSOCKET_WRITE_BUFFER_SIZE", 1024),
		HandshakeTimeout:  getEnvAsDuration("WEBSOCKET_HANDSHAKE_TIMEOUT", 10*time.Second),
		PingInterval:      getEnvAsDuration("WEBSOCKET_PING_INTERVAL", 54*time.Second),
		PongTimeout:       getEnvAsDuration("WEBSOCKET_PONG_TIMEOUT", 60*time.Second),
		MaxConnections:    getEnvAsInt("WEBSOCKET_MAX_CONNECTIONS", 10000),
		EnableCompression: getEnvAsBool("WEBSOCKET_ENABLE_COMPRESSION", true),
		AllowedOrigins:    getEnvAsSlice("WEBSOCKET_ALLOWED_ORIGINS", []string{"*"}),
	}
}
