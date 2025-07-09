package main

import (
	"fmt"
	"log"
	"net/http"

	"goride/internal/config"
	handlers "goride/internal/handlers/shared"
	"goride/internal/middleware"
	"goride/internal/services"
	"goride/routes"

	"github.com/gin-gonic/gin"
)

func main() {
	// Load configuration
	cfg, err := config.Load()
	if err != nil {
		log.Fatalf("Failed to load config: %v", err)
	}

	// For now, create a simple call service with nil dependencies
	// In a real implementation, you would initialize all dependencies properly
	callService := services.NewCallService(nil, nil, nil, nil, nil, nil, nil, nil)

	// Initialize handlers
	callHandler := handlers.NewCallHandler(callService)

	// Initialize Gin router
	router := gin.New()

	// Global middleware
	router.Use(gin.Logger())
	router.Use(gin.Recovery())
	router.Use(middleware.CORSMiddleware())
	router.Use(middleware.RequestIDMiddleware())

	// API routes
	v1 := router.Group("/api/v1")
	{
		// Call routes
		routes.SetupCallRoutes(v1, callHandler)
	}

	// Health check
	router.GET("/health", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{
			"status":  "healthy",
			"version": "1.0.0",
		})
	})

	// Start server
	port := "8080"
	if cfg.App != nil && cfg.App.Port != 0 {
		port = fmt.Sprintf("%d", cfg.App.Port)
	}

	log.Printf("Starting server on port %s", port)
	log.Fatal(http.ListenAndServe(":"+port, router))
}
