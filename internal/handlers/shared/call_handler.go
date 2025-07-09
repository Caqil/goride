package handlers

import (
	"net/http"

	"goride/internal/models"
	"goride/internal/services"
	"goride/internal/utils"

	"github.com/gin-gonic/gin"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

type CallHandler struct {
	callService services.CallService
}

func NewCallHandler(callService services.CallService) *CallHandler {
	return &CallHandler{
		callService: callService,
	}
}

// InitiateCall initiates a new call between users
func (h *CallHandler) InitiateCall(c *gin.Context) {
	var request models.CallRequest
	if err := c.ShouldBindJSON(&request); err != nil {
		utils.BadRequestResponse(c, "Invalid request: "+err.Error())
		return
	}

	// Get caller ID from context (set by auth middleware)
	callerID, exists := c.Get("user_id")
	if !exists {
		utils.UnauthorizedResponse(c)
		return
	}

	callerObjectID, ok := callerID.(primitive.ObjectID)
	if !ok {
		utils.BadRequestResponse(c, "Invalid user ID")
		return
	}

	request.CallerID = callerObjectID

	response, err := h.callService.InitiateCall(c.Request.Context(), &request)
	if err != nil {
		utils.ErrorResponse(c, http.StatusInternalServerError, "CALL_INITIATION_FAILED", "Failed to initiate call: "+err.Error())
		return
	}

	utils.SuccessResponse(c, "Call initiated successfully", response)
}

// EndCall ends an active call
func (h *CallHandler) EndCall(c *gin.Context) {
	callIDStr := c.Param("id")
	callID, err := primitive.ObjectIDFromHex(callIDStr)
	if err != nil {
		utils.BadRequestResponse(c, "Invalid call ID")
		return
	}

	userID, exists := c.Get("user_id")
	if !exists {
		utils.UnauthorizedResponse(c)
		return
	}

	userObjectID, ok := userID.(primitive.ObjectID)
	if !ok {
		utils.BadRequestResponse(c, "Invalid user ID")
		return
	}

	err = h.callService.EndCall(c.Request.Context(), callID, userObjectID)
	if err != nil {
		utils.ErrorResponse(c, http.StatusInternalServerError, "CALL_END_FAILED", "Failed to end call: "+err.Error())
		return
	}

	utils.SuccessResponse(c, "Call ended successfully", nil)
}

// GetCall retrieves call details
func (h *CallHandler) GetCall(c *gin.Context) {
	callIDStr := c.Param("id")
	callID, err := primitive.ObjectIDFromHex(callIDStr)
	if err != nil {
		utils.BadRequestResponse(c, "Invalid call ID")
		return
	}

	call, err := h.callService.GetCall(c.Request.Context(), callID)
	if err != nil {
		utils.ErrorResponse(c, http.StatusInternalServerError, "CALL_FETCH_FAILED", "Failed to get call: "+err.Error())
		return
	}

	utils.SuccessResponse(c, "Call retrieved successfully", call)
}

// GetCallHistory retrieves user's call history
func (h *CallHandler) GetCallHistory(c *gin.Context) {
	userID, exists := c.Get("user_id")
	if !exists {
		utils.UnauthorizedResponse(c)
		return
	}

	userObjectID, ok := userID.(primitive.ObjectID)
	if !ok {
		utils.BadRequestResponse(c, "Invalid user ID")
		return
	}

	params := utils.GetPaginationParams(c)
	calls, total, err := h.callService.GetCallHistory(c.Request.Context(), userObjectID, params)
	if err != nil {
		utils.ErrorResponse(c, http.StatusInternalServerError, "CALL_HISTORY_FAILED", "Failed to get call history: "+err.Error())
		return
	}

	meta := &utils.Meta{
		Pagination: utils.CreatePaginationMeta(params, total),
	}

	response := map[string]interface{}{
		"calls": calls,
	}

	utils.SuccessResponseWithMeta(c, "Call history retrieved successfully", response, meta)
}

// GetRideCalls retrieves calls for a specific ride
func (h *CallHandler) GetRideCalls(c *gin.Context) {
	rideIDStr := c.Param("ride_id")
	rideID, err := primitive.ObjectIDFromHex(rideIDStr)
	if err != nil {
		utils.BadRequestResponse(c, "Invalid ride ID")
		return
	}

	calls, err := h.callService.GetRideCalls(c.Request.Context(), rideID)
	if err != nil {
		utils.ErrorResponse(c, http.StatusInternalServerError, "RIDE_CALLS_FAILED", "Failed to get ride calls: "+err.Error())
		return
	}

	utils.SuccessResponse(c, "Ride calls retrieved successfully", calls)
}

// InitiateEmergencyCall initiates an emergency call
func (h *CallHandler) InitiateEmergencyCall(c *gin.Context) {
	userID, exists := c.Get("user_id")
	if !exists {
		utils.UnauthorizedResponse(c)
		return
	}

	userObjectID, ok := userID.(primitive.ObjectID)
	if !ok {
		utils.BadRequestResponse(c, "Invalid user ID")
		return
	}

	var rideID *primitive.ObjectID
	if rideIDStr := c.Query("ride_id"); rideIDStr != "" {
		if rideObjID, err := primitive.ObjectIDFromHex(rideIDStr); err == nil {
			rideID = &rideObjID
		}
	}

	response, err := h.callService.InitiateEmergencyCall(c.Request.Context(), userObjectID, rideID)
	if err != nil {
		utils.ErrorResponse(c, http.StatusInternalServerError, "EMERGENCY_CALL_FAILED", "Failed to initiate emergency call: "+err.Error())
		return
	}

	utils.SuccessResponse(c, "Emergency call initiated successfully", response)
}

// RateCallQuality allows users to rate call quality
func (h *CallHandler) RateCallQuality(c *gin.Context) {
	callIDStr := c.Param("id")
	callID, err := primitive.ObjectIDFromHex(callIDStr)
	if err != nil {
		utils.BadRequestResponse(c, "Invalid call ID")
		return
	}

	userID, exists := c.Get("user_id")
	if !exists {
		utils.UnauthorizedResponse(c)
		return
	}

	userObjectID, ok := userID.(primitive.ObjectID)
	if !ok {
		utils.BadRequestResponse(c, "Invalid user ID")
		return
	}

	var request struct {
		Rating int `json:"rating" validate:"required,min=1,max=5"`
	}

	if err := c.ShouldBindJSON(&request); err != nil {
		utils.BadRequestResponse(c, "Invalid request: "+err.Error())
		return
	}

	err = h.callService.RateCallQuality(c.Request.Context(), callID, userObjectID, request.Rating)
	if err != nil {
		utils.ErrorResponse(c, http.StatusInternalServerError, "CALL_RATING_FAILED", "Failed to rate call quality: "+err.Error())
		return
	}

	utils.SuccessResponse(c, "Call quality rated successfully", nil)
}

// GetCallAnalytics retrieves call analytics (admin only)
func (h *CallHandler) GetCallAnalytics(c *gin.Context) {
	var params models.CallAnalyticsParams

	// Parse date parameters
	if startDateStr := c.Query("start_date"); startDateStr != "" {
		// Parse start date
	}
	if endDateStr := c.Query("end_date"); endDateStr != "" {
		// Parse end date
	}

	analytics, err := h.callService.GetCallAnalytics(c.Request.Context(), &params)
	if err != nil {
		utils.ErrorResponse(c, http.StatusInternalServerError, "ANALYTICS_FAILED", "Failed to get call analytics: "+err.Error())
		return
	}

	utils.SuccessResponse(c, "Call analytics retrieved successfully", analytics)
}

// HandleCallStatusWebhook handles Twilio call status webhooks
func (h *CallHandler) HandleCallStatusWebhook(c *gin.Context) {
	var webhookData map[string]interface{}
	if err := c.ShouldBindJSON(&webhookData); err != nil {
		utils.BadRequestResponse(c, "Invalid webhook data")
		return
	}

	err := h.callService.HandleCallStatusWebhook(c.Request.Context(), webhookData)
	if err != nil {
		utils.ErrorResponse(c, http.StatusInternalServerError, "WEBHOOK_FAILED", "Failed to handle webhook: "+err.Error())
		return
	}

	c.JSON(http.StatusOK, gin.H{"status": "success"})
}

// HandleRecordingStatusWebhook handles Twilio recording status webhooks
func (h *CallHandler) HandleRecordingStatusWebhook(c *gin.Context) {
	// Implementation for handling recording status updates
	c.JSON(http.StatusOK, gin.H{"status": "success"})
}
