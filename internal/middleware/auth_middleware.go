package middleware

import (
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

// JWTClaims represents the JWT token claims
type JWTClaims struct {
	UserID   string `json:"user_id"`
	UserType string `json:"user_type"`
	jwt.RegisteredClaims
}

// AuthRequired middleware validates JWT token and sets user context
func AuthRequired() gin.HandlerFunc {
	return func(c *gin.Context) {
		authHeader := c.GetHeader("Authorization")
		if authHeader == "" {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "Authorization header required"})
			c.Abort()
			return
		}

		tokenString := strings.TrimPrefix(authHeader, "Bearer ")
		if tokenString == authHeader {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "Bearer token required"})
			c.Abort()
			return
		}

		// Parse and validate token
		token, err := jwt.ParseWithClaims(tokenString, &JWTClaims{}, func(token *jwt.Token) (interface{}, error) {
			// In a real app, you'd get this from config
			return []byte("your-secret-key"), nil
		})

		if err != nil || !token.Valid {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "Invalid token"})
			c.Abort()
			return
		}

		claims, ok := token.Claims.(*JWTClaims)
		if !ok {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "Invalid token claims"})
			c.Abort()
			return
		}

		// Convert user ID to ObjectID
		userID, err := primitive.ObjectIDFromHex(claims.UserID)
		if err != nil {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "Invalid user ID in token"})
			c.Abort()
			return
		}

		// Set user context
		c.Set("user_id", userID)
		c.Set("user_type", claims.UserType)

		c.Next()
	}
}

// AdminRequired middleware ensures user is an admin
func AdminRequired() gin.HandlerFunc {
	return func(c *gin.Context) {
		userType, exists := c.Get("user_type")
		if !exists {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "User type not found"})
			c.Abort()
			return
		}

		userTypeStr, ok := userType.(string)
		if !ok || userTypeStr != "admin" {
			c.JSON(http.StatusForbidden, gin.H{"error": "Admin access required"})
			c.Abort()
			return
		}

		c.Next()
	}
}

// DriverRequired middleware ensures user is a driver
func DriverRequired() gin.HandlerFunc {
	return func(c *gin.Context) {
		userType, exists := c.Get("user_type")
		if !exists {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "User type not found"})
			c.Abort()
			return
		}

		userTypeStr, ok := userType.(string)
		if !ok || userTypeStr != "driver" {
			c.JSON(http.StatusForbidden, gin.H{"error": "Driver access required"})
			c.Abort()
			return
		}

		c.Next()
	}
}

// RiderRequired middleware ensures user is a rider
func RiderRequired() gin.HandlerFunc {
	return func(c *gin.Context) {
		userType, exists := c.Get("user_type")
		if !exists {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "User type not found"})
			c.Abort()
			return
		}

		userTypeStr, ok := userType.(string)
		if !ok || userTypeStr != "rider" {
			c.JSON(http.StatusForbidden, gin.H{"error": "Rider access required"})
			c.Abort()
			return
		}

		c.Next()
	}
}
