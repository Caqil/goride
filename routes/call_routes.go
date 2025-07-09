package routes

import (
	"goride/internal/middleware"

	"github.com/gin-gonic/gin"
)

// SetupCallRoutes sets up routes for call functionality
func SetupCallRoutes(r *gin.RouterGroup, callHandler *shared.CallHandler) {
	// Public webhook routes (no auth required)
	webhooks := r.Group("/webhooks/twilio")
	{
		webhooks.POST("/call-status", callHandler.HandleCallStatusWebhook)
		webhooks.POST("/recording-status", callHandler.HandleRecordingStatusWebhook)
		webhooks.POST("/emergency-call-status", callHandler.HandleCallStatusWebhook)
	}

	// Protected call routes (require authentication)
	calls := r.Group("/calls")
	calls.Use(middleware.AuthRequired())
	{
		// Basic call operations
		calls.POST("/", callHandler.InitiateCall)
		calls.GET("/:id", callHandler.GetCall)
		calls.PUT("/:id/end", callHandler.EndCall)
		calls.POST("/:id/rate", callHandler.RateCallQuality)

		// Call history
		calls.GET("/history", callHandler.GetCallHistory)

		// Emergency calls
		calls.POST("/emergency", callHandler.InitiateEmergencyCall)

		// Ride-specific calls
		calls.GET("/rides/:ride_id", callHandler.GetRideCalls)
	}

	// Admin routes for call analytics
	admin := r.Group("/admin/calls")
	admin.Use(middleware.AuthRequired(), middleware.AdminRequired())
	{
		admin.GET("/analytics", callHandler.GetCallAnalytics)
	}
}

// Alternative route setup for different user types
func SetupDriverCallRoutes(r *gin.RouterGroup, callHandler *shared.CallHandler) {
	calls := r.Group("/calls")
	calls.Use(middleware.AuthRequired(), middleware.DriverRequired())
	{
		calls.POST("/", callHandler.InitiateCall)
		calls.GET("/:id", callHandler.GetCall)
		calls.PUT("/:id/end", callHandler.EndCall)
		calls.POST("/:id/rate", callHandler.RateCallQuality)
		calls.GET("/history", callHandler.GetCallHistory)
		calls.POST("/emergency", callHandler.InitiateEmergencyCall)
	}
}

func SetupRiderCallRoutes(r *gin.RouterGroup, callHandler *shared.CallHandler) {
	calls := r.Group("/calls")
	calls.Use(middleware.AuthRequired(), middleware.RiderRequired())
	{
		calls.POST("/", callHandler.InitiateCall)
		calls.GET("/:id", callHandler.GetCall)
		calls.PUT("/:id/end", callHandler.EndCall)
		calls.POST("/:id/rate", callHandler.RateCallQuality)
		calls.GET("/history", callHandler.GetCallHistory)
		calls.POST("/emergency", callHandler.InitiateEmergencyCall)
	}
}
