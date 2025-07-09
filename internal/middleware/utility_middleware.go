package middleware

import (
	"fmt"
	"math/rand"
	"time"

	"github.com/gin-gonic/gin"
)

// CORSMiddleware configures CORS headers
func CORSMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		c.Header("Access-Control-Allow-Origin", "*")
		c.Header("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
		c.Header("Access-Control-Allow-Headers", "Content-Type, Authorization, X-Request-ID")
		c.Header("Access-Control-Expose-Headers", "Content-Length")
		c.Header("Access-Control-Allow-Credentials", "true")

		if c.Request.Method == "OPTIONS" {
			c.AbortWithStatus(204)
			return
		}

		c.Next()
	}
}

// RequestIDMiddleware adds a request ID to each request
func RequestIDMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		requestID := c.GetHeader("X-Request-ID")
		if requestID == "" {
			// Generate a simple request ID using timestamp and random number
			requestID = fmt.Sprintf("%d-%d", time.Now().UnixNano(), rand.Intn(10000))
		}
		c.Set("request_id", requestID)
		c.Header("X-Request-ID", requestID)
		c.Next()
	}
}

// LoggingMiddleware provides structured logging
func LoggingMiddleware(logger interface{}) gin.HandlerFunc {
	return gin.LoggerWithFormatter(func(param gin.LogFormatterParams) string {
		// You can customize this based on your logger interface
		return ""
	})
}
