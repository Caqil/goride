package services

import (
	"context"
	"fmt"
	"log"
	"strings"
	"time"

	"goride/internal/config"
	"goride/internal/models"
	"goride/internal/repositories/interfaces"
	"goride/internal/utils"
	"goride/pkg/cache"
	"goride/pkg/push"
	"goride/pkg/websocket"

	"github.com/twilio/twilio-go"
	api "github.com/twilio/twilio-go/rest/api/v2010"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

type CallService interface {
	// Call Management
	InitiateCall(ctx context.Context, request *models.CallRequest) (*models.CallResponse, error)
	EndCall(ctx context.Context, callID primitive.ObjectID, userID primitive.ObjectID) error
	GetCall(ctx context.Context, callID primitive.ObjectID) (*models.Call, error)
	GetCallHistory(ctx context.Context, userID primitive.ObjectID, params *utils.PaginationParams) ([]*models.Call, int64, error)
	GetRideCalls(ctx context.Context, rideID primitive.ObjectID) ([]*models.Call, error)

	// Call Status Management
	UpdateCallStatus(ctx context.Context, callSID string, status models.CallStatus, duration int) error
	HandleCallStatusWebhook(ctx context.Context, webhookData map[string]interface{}) error

	// Emergency Calls
	InitiateEmergencyCall(ctx context.Context, userID primitive.ObjectID, rideID *primitive.ObjectID) (*models.CallResponse, error)
	NotifyEmergencyContacts(ctx context.Context, userID primitive.ObjectID, emergencyID primitive.ObjectID) error

	// Call Analytics
	GetCallAnalytics(ctx context.Context, params *models.CallAnalyticsParams) (*models.CallAnalytics, error)
	RateCallQuality(ctx context.Context, callID primitive.ObjectID, userID primitive.ObjectID, rating int) error

	// Proxy Number Management
	GetProxyNumber(ctx context.Context, rideID primitive.ObjectID) (string, error)
	ReleaseProxyNumber(ctx context.Context, proxyNumber string) error
}

type callService struct {
	twilioClient       *twilio.RestClient
	twilioFromNumber   string
	twilioProxyService string
	cache              *cache.RedisCache
	wsHandler          *websocket.Handler
	pushService        *push.FCMProvider
	userRepo           interfaces.UserRepository
	rideRepo           interfaces.RideRepository
	emergencyRepo      interfaces.EmergencyRepository
	callRepo           interfaces.CallRepository
	config             *config.Config
}

func NewCallService(
	config *config.Config,
	cache *cache.RedisCache,
	wsHandler *websocket.Handler,
	pushService *push.FCMProvider,
	userRepo interfaces.UserRepository,
	rideRepo interfaces.RideRepository,
	emergencyRepo interfaces.EmergencyRepository,
	callRepo interfaces.CallRepository,
) CallService {
	// Initialize Twilio client
	twilioClient := twilio.NewRestClientWithParams(twilio.ClientParams{
		Username: config.SMS.Twilio.AccountSID,
		Password: config.SMS.Twilio.AuthToken,
	})

	return &callService{
		twilioClient:       twilioClient,
		twilioFromNumber:   config.SMS.Twilio.FromNumber,
		twilioProxyService: getEnv("TWILIO_PROXY_SERVICE_SID", ""),
		cache:              cache,
		wsHandler:          wsHandler,
		pushService:        pushService,
		userRepo:           userRepo,
		rideRepo:           rideRepo,
		emergencyRepo:      emergencyRepo,
		callRepo:           callRepo,
		config:             config,
	}
}

func (s *callService) InitiateCall(ctx context.Context, request *models.CallRequest) (*models.CallResponse, error) {
	// Validate users exist and get their phone numbers
	caller, err := s.userRepo.GetByID(ctx, request.CallerID)
	if err != nil {
		return nil, fmt.Errorf("failed to get caller: %w", err)
	}

	callee, err := s.userRepo.GetByID(ctx, request.CalleeID)
	if err != nil {
		return nil, fmt.Errorf("failed to get callee: %w", err)
	}

	// Check if callee is available for calls
	if !s.isUserAvailableForCall(ctx, request.CalleeID) {
		return &models.CallResponse{
			Status:  models.CallStatusBusy,
			Message: "User is currently unavailable for calls",
		}, nil
	}

	var proxyNumber string
	var callSID string

	// Use Twilio Proxy if available for ride-related calls
	if request.RideID != nil && s.twilioProxyService != "" {
		proxyNumber, callSID, err = s.initiateProxyCall(ctx, caller.Phone, callee.Phone, request)
	} else {
		callSID, err = s.initiateDirectCall(ctx, caller.Phone, callee.Phone, request)
	}

	if err != nil {
		return nil, fmt.Errorf("failed to initiate call: %w", err)
	}

	// Create call record
	call := &models.Call{
		ID:          primitive.NewObjectID(),
		CallSID:     callSID,
		RideID:      request.RideID,
		CallerID:    request.CallerID,
		CalleeID:    request.CalleeID,
		Type:        request.Type,
		Status:      models.CallStatusInitiated,
		Direction:   models.CallDirectionOutbound,
		FromNumber:  caller.Phone,
		ToNumber:    callee.Phone,
		ProxyNumber: proxyNumber,
		IsRecorded:  request.IsRecorded,
		IsEmergency: request.Type == models.CallTypeEmergency,
		CreatedAt:   time.Now(),
		UpdatedAt:   time.Now(),
	}

	// Store call in cache and database
	if err := s.storeCall(ctx, call); err != nil {
		log.Printf("Failed to store call: %v", err)
	}

	// Send real-time notification to callee
	s.notifyIncomingCall(ctx, call)

	// Send push notification
	s.sendCallPushNotification(ctx, call, "incoming_call")

	return &models.CallResponse{
		CallID:      call.ID,
		CallSID:     callSID,
		ProxyNumber: proxyNumber,
		Status:      models.CallStatusInitiated,
		Message:     "Call initiated successfully",
	}, nil
}

func (s *callService) initiateDirectCall(ctx context.Context, fromNumber, toNumber string, request *models.CallRequest) (string, error) {
	// Construct webhook URL for status callbacks
	webhookURL := fmt.Sprintf("%s/api/webhooks/twilio/call-status", s.config.App.BaseURL)

	params := &api.CreateCallParams{}
	params.SetTo(toNumber)
	params.SetFrom(s.twilioFromNumber)
	params.SetUrl(webhookURL)
	params.SetMethod("POST")

	// Add recording if requested
	if request.IsRecorded {
		params.SetRecord(true)
		params.SetRecordingStatusCallback(fmt.Sprintf("%s/api/webhooks/twilio/recording-status", s.config.App.BaseURL))
	}

	// Set timeout and other parameters
	params.SetTimeout(30)
	params.SetStatusCallback(webhookURL)
	params.SetStatusCallbackMethod("POST")

	resp, err := s.twilioClient.Api.CreateCall(params)
	if err != nil {
		return "", fmt.Errorf("failed to create Twilio call: %w", err)
	}

	return *resp.Sid, nil
}

func (s *callService) initiateProxyCall(ctx context.Context, fromNumber, toNumber string, request *models.CallRequest) (string, string, error) {
	// This would use Twilio Proxy Service for privacy
	// Implementation depends on your Twilio Proxy setup
	// For now, falling back to direct call
	callSID, err := s.initiateDirectCall(ctx, fromNumber, toNumber, request)
	return "", callSID, err
}

func (s *callService) EndCall(ctx context.Context, callID primitive.ObjectID, userID primitive.ObjectID) error {
	call, err := s.GetCall(ctx, callID)
	if err != nil {
		return fmt.Errorf("failed to get call: %w", err)
	}

	// Verify user has permission to end this call
	if call.CallerID != userID && call.CalleeID != userID {
		return fmt.Errorf("user not authorized to end this call")
	}

	// End the call via Twilio
	params := &api.UpdateCallParams{}
	params.SetStatus("completed")

	_, err = s.twilioClient.Api.UpdateCall(call.CallSID, params)
	if err != nil {
		return fmt.Errorf("failed to end call via Twilio: %w", err)
	}

	// Update call status
	return s.UpdateCallStatus(ctx, call.CallSID, models.CallStatusCompleted, 0)
}

func (s *callService) GetCall(ctx context.Context, callID primitive.ObjectID) (*models.Call, error) {
	// Try repository first
	call, err := s.callRepo.GetCallByID(ctx, callID)
	if err == nil {
		return call, nil
	}

	// Try cache as fallback
	cacheKey := fmt.Sprintf("call:%s", callID.Hex())
	var cachedCall models.Call
	if err := s.cache.Get(ctx, cacheKey, &cachedCall); err == nil {
		return &cachedCall, nil
	}

	return nil, fmt.Errorf("call not found")
}

func (s *callService) GetCallHistory(ctx context.Context, userID primitive.ObjectID, params *utils.PaginationParams) ([]*models.Call, int64, error) {
	return s.callRepo.GetCallsByUser(ctx, userID, params)
}

func (s *callService) GetRideCalls(ctx context.Context, rideID primitive.ObjectID) ([]*models.Call, error) {
	return s.callRepo.GetCallsByRide(ctx, rideID)
}

func (s *callService) UpdateCallStatus(ctx context.Context, callSID string, status models.CallStatus, duration int) error {
	// Find call by SID
	cacheKey := fmt.Sprintf("call_sid:%s", callSID)
	var call Call
	if err := s.cache.Get(ctx, cacheKey, &call); err != nil {
		// Could not find in cache, would need to query database
		log.Printf("Call not found in cache: %s", callSID)
		return nil
	}

	// Update call status
	call.Status = status
	call.UpdatedAt = time.Now()

	if duration > 0 {
		call.Duration = duration
	}

	if status == models.CallStatusAnswered && call.StartTime == nil {
		now := time.Now()
		call.StartTime = &now
	}

	if status == models.CallStatusCompleted || status == models.CallStatusFailed || status == models.CallStatusCancelled {
		now := time.Now()
		call.EndTime = &now

		// Calculate duration if not provided
		if call.StartTime != nil && call.Duration == 0 {
			call.Duration = int(now.Sub(*call.StartTime).Seconds())
		}
	}

	// Update cache
	s.cache.Set(ctx, cacheKey, call, time.Hour)
	s.cache.Set(ctx, fmt.Sprintf("call:%s", call.ID.Hex()), call, time.Hour)

	// Send real-time update
	s.notifyCallStatusUpdate(ctx, &call)

	// Release proxy number if call ended
	if call.ProxyNumber != "" && (status == models.CallStatusCompleted || status == models.CallStatusFailed || status == models.CallStatusCancelled) {
		s.ReleaseProxyNumber(ctx, call.ProxyNumber)
	}

	return nil
}

func (s *callService) HandleCallStatusWebhook(ctx context.Context, webhookData map[string]interface{}) error {
	callSID, ok := webhookData["CallSid"].(string)
	if !ok {
		return fmt.Errorf("missing CallSid in webhook data")
	}

	statusStr, ok := webhookData["CallStatus"].(string)
	if !ok {
		return fmt.Errorf("missing CallStatus in webhook data")
	}

	status := s.mapTwilioStatusToCallStatus(statusStr)

	duration := 0
	if durationStr, ok := webhookData["CallDuration"].(string); ok {
		if d, err := time.ParseDuration(durationStr + "s"); err == nil {
			duration = int(d.Seconds())
		}
	}

	return s.UpdateCallStatus(ctx, callSID, status, duration)
}

func (s *callService) InitiateEmergencyCall(ctx context.Context, userID primitive.ObjectID, rideID *primitive.ObjectID) (*models.CallResponse, error) {
	// Get emergency number from config (e.g., 911, local emergency services)
	emergencyNumber := getEnv("EMERGENCY_NUMBER", "911")

	user, err := s.userRepo.GetByID(ctx, userID)
	if err != nil {
		return nil, fmt.Errorf("failed to get user: %w", err)
	}

	// Create emergency call request
	request := &models.CallRequest{
		CallerID:   userID,
		CalleeID:   primitive.NewObjectID(), // Placeholder for emergency services
		RideID:     rideID,
		Type:       models.CallTypeEmergency,
		IsRecorded: true, // Always record emergency calls
	}

	// Initiate call to emergency services
	webhookURL := fmt.Sprintf("%s/api/webhooks/twilio/emergency-call-status", s.config.App.BaseURL)

	params := &api.CreateCallParams{}
	params.SetTo(emergencyNumber)
	params.SetFrom(user.Phone)
	params.SetUrl(webhookURL)
	params.SetRecord(true)
	params.SetStatusCallback(webhookURL)

	resp, err := s.twilioClient.Api.CreateCall(params)
	if err != nil {
		return nil, fmt.Errorf("failed to create emergency call: %w", err)
	}

	// Create emergency call record
	call := &models.Call{
		ID:          primitive.NewObjectID(),
		CallSID:     *resp.Sid,
		RideID:      rideID,
		CallerID:    userID,
		Type:        models.CallTypeEmergency,
		Status:      models.CallStatusInitiated,
		Direction:   models.CallDirectionOutbound,
		FromNumber:  user.Phone,
		ToNumber:    emergencyNumber,
		IsRecorded:  true,
		IsEmergency: true,
		CreatedAt:   time.Now(),
		UpdatedAt:   time.Now(),
	}

	// Store emergency call
	s.storeCall(ctx, call)

	// Create SOS record and notify emergency contacts
	if rideID != nil {
		s.createSOSRecord(ctx, userID, *rideID, call.ID)
	}

	return &models.CallResponse{
		CallID:  call.ID,
		CallSID: *resp.Sid,
		Status:  models.CallStatusInitiated,
		Message: "Emergency call initiated",
	}, nil
}

func (s *callService) NotifyEmergencyContacts(ctx context.Context, userID primitive.ObjectID, emergencyID primitive.ObjectID) error {
	// Implementation would get emergency contacts and notify them
	// This would involve SMS, push notifications, and potentially calls
	return nil
}

func (s *callService) GetCallAnalytics(ctx context.Context, params *models.CallAnalyticsParams) (*models.CallAnalytics, error) {
	// Implementation would query database for analytics
	return &models.CallAnalytics{
		TotalCalls:      0,
		SuccessfulCalls: 0,
		FailedCalls:     0,
		CallsByType:     make(map[models.CallType]int64),
		CallsByStatus:   make(map[models.CallStatus]int64),
	}, nil
}

func (s *callService) RateCallQuality(ctx context.Context, callID primitive.ObjectID, userID primitive.ObjectID, rating int) error {
	if rating < 1 || rating > 5 {
		return fmt.Errorf("rating must be between 1 and 5")
	}

	call, err := s.GetCall(ctx, callID)
	if err != nil {
		return err
	}

	if call.CallerID != userID && call.CalleeID != userID {
		return fmt.Errorf("user not authorized to rate this call")
	}

	call.QualityRating = rating
	call.UpdatedAt = time.Now()

	// Update in cache
	cacheKey := fmt.Sprintf("call:%s", callID.Hex())
	s.cache.Set(ctx, cacheKey, call, time.Hour)

	return nil
}

func (s *callService) GetProxyNumber(ctx context.Context, rideID primitive.ObjectID) (string, error) {
	// Implementation would use Twilio Proxy Service
	// For now, return the main Twilio number
	return s.twilioFromNumber, nil
}

func (s *callService) ReleaseProxyNumber(ctx context.Context, proxyNumber string) error {
	// Implementation would release the proxy number back to the pool
	return nil
}

// Helper methods

func (s *callService) isUserAvailableForCall(ctx context.Context, userID primitive.ObjectID) bool {
	// Check if user is currently on another call
	pattern := fmt.Sprintf("active_call:*:%s", userID.Hex())
	keys, err := s.cache.Keys(ctx, pattern)
	if err != nil {
		return true // Assume available if we can't check
	}

	return len(keys) == 0
}

func (s *callService) storeCall(ctx context.Context, call *models.Call) error {
	// Store in repository
	if err := s.callRepo.CreateCall(ctx, call); err != nil {
		log.Printf("Failed to store call in repository: %v", err)
	}

	// Store in cache
	cacheKey := fmt.Sprintf("call:%s", call.ID.Hex())
	s.cache.Set(ctx, cacheKey, call, time.Hour*24)

	// Store call SID mapping
	s.cache.Set(ctx, fmt.Sprintf("call_sid:%s", call.CallSID), *call, time.Hour*24)

	// Mark users as in active call
	activeCallKey := fmt.Sprintf("active_call:%s:%s", call.ID.Hex(), call.CallerID.Hex())
	s.cache.Set(ctx, activeCallKey, true, time.Hour)

	activeCallKey = fmt.Sprintf("active_call:%s:%s", call.ID.Hex(), call.CalleeID.Hex())
	s.cache.Set(ctx, activeCallKey, true, time.Hour)

	return nil
}

func (s *callService) notifyIncomingCall(ctx context.Context, call *models.Call) {
	// Send WebSocket notification to callee
	notification := map[string]interface{}{
		"call_id":      call.ID.Hex(),
		"caller_id":    call.CallerID.Hex(),
		"type":         call.Type,
		"proxy_number": call.ProxyNumber,
	}

	s.wsHandler.SendUserNotification(call.CalleeID, "incoming_call", notification)
}

func (s *callService) notifyCallStatusUpdate(ctx context.Context, call *models.Call) {
	statusUpdate := map[string]interface{}{
		"call_id":  call.ID.Hex(),
		"status":   call.Status,
		"duration": call.Duration,
	}

	// Notify both participants
	s.wsHandler.SendUserNotification(call.CallerID, "call_status_update", statusUpdate)
	s.wsHandler.SendUserNotification(call.CalleeID, "call_status_update", statusUpdate)
}

func (s *callService) sendCallPushNotification(ctx context.Context, call *models.Call, notificationType string) {
	// Implementation would send push notification
	// This would use the existing push service
}

func (s *callService) createSOSRecord(ctx context.Context, userID primitive.ObjectID, rideID primitive.ObjectID, callID primitive.ObjectID) {
	// Implementation would create an SOS record in the database
	// This would link the emergency call to an SOS incident
}

func (s *callService) mapTwilioStatusToCallStatus(twilioStatus string) models.CallStatus {
	switch strings.ToLower(twilioStatus) {
	case "queued", "initiated":
		return models.CallStatusInitiated
	case "ringing":
		return models.CallStatusRinging
	case "in-progress":
		return models.CallStatusAnswered
	case "completed":
		return models.CallStatusCompleted
	case "failed":
		return models.CallStatusFailed
	case "cancelled":
		return models.CallStatusCancelled
	case "busy":
		return models.CallStatusBusy
	case "no-answer":
		return models.CallStatusNoAnswer
	default:
		return models.CallStatusFailed
	}
}

func getEnv(key, defaultValue string) string {
	// This would use the same environment variable helper from config
	return defaultValue
}
