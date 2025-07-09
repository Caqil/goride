package push

import (
	"context"
	"fmt"
	"time"

	"github.com/sideshow/apns2"
	"github.com/sideshow/apns2/token"
)

type APNSProvider struct {
	client *apns2.Client
	topic  string
}

func NewAPNSProvider(keyFile, keyID, teamID, topic string, production bool) (*APNSProvider, error) {
	authKey, err := token.AuthKeyFromFile(keyFile)
	if err != nil {
		return nil, fmt.Errorf("failed to load auth key: %w", err)
	}

	tokenProvider := &token.Token{
		AuthKey: authKey,
		KeyID:   keyID,
		TeamID:  teamID,
	}

	client := apns2.NewTokenClient(tokenProvider)
	if production {
		client = client.Production()
	} else {
		client = client.Development()
	}

	return &APNSProvider{
		client: client,
		topic:  topic,
	}, nil
}

func (a *APNSProvider) SendNotification(ctx context.Context, request *NotificationRequest) (*NotificationResponse, error) {
	notification := a.buildNotification(request)

	response, err := a.client.PushWithContext(ctx, notification)
	if err != nil {
		return &NotificationResponse{
			Success: false,
			Error:   err.Error(),
			Token:   request.Token,
		}, err
	}

	if response.Sent() {
		return &NotificationResponse{
			MessageID: response.ApnsID,
			Success:   true,
			Token:     request.Token,
		}, nil
	}

	return &NotificationResponse{
		Success: false,
		Error:   response.Reason,
		Token:   request.Token,
	}, fmt.Errorf("APNS error: %s", response.Reason)
}

func (a *APNSProvider) SendBulkNotifications(ctx context.Context, requests []*NotificationRequest) ([]*NotificationResponse, error) {
	responses := make([]*NotificationResponse, len(requests))

	for i, req := range requests {
		response, err := a.SendNotification(ctx, req)
		if err != nil {
			response = &NotificationResponse{
				Success: false,
				Error:   err.Error(),
				Token:   req.Token,
			}
		}
		responses[i] = response
	}

	return responses, nil
}

func (a *APNSProvider) SubscribeToTopic(ctx context.Context, tokens []string, topic string) error {
	// APNS doesn't have topic subscription like FCM
	return fmt.Errorf("topic subscription not supported by APNS")
}

func (a *APNSProvider) UnsubscribeFromTopic(ctx context.Context, tokens []string, topic string) error {
	// APNS doesn't have topic subscription like FCM
	return fmt.Errorf("topic unsubscription not supported by APNS")
}

func (a *APNSProvider) ValidateToken(ctx context.Context, token string) (bool, error) {
	// Create a test notification
	notification := &apns2.Notification{
		DeviceToken: token,
		Topic:       a.topic,
		Payload: map[string]interface{}{
			"aps": map[string]interface{}{
				"content-available": 1,
			},
		},
	}

	response, err := a.client.PushWithContext(ctx, notification)
	if err != nil {
		return false, err
	}

	return response.Sent(), nil
}

func (a *APNSProvider) buildNotification(request *NotificationRequest) *apns2.Notification {
	payload := map[string]interface{}{}

	aps := map[string]interface{}{}

	// Set alert
	if request.Title != "" || request.Body != "" {
		alert := map[string]interface{}{}
		if request.Title != "" {
			alert["title"] = request.Title
		}
		if request.Body != "" {
			alert["body"] = request.Body
		}
		aps["alert"] = alert
	}

	// Set sound
	if request.Sound != "" {
		aps["sound"] = request.Sound
	} else if request.iOS != nil && request.iOS.Sound != "" {
		aps["sound"] = request.iOS.Sound
	}

	// Set badge
	if request.Badge > 0 {
		aps["badge"] = request.Badge
	} else if request.iOS != nil && request.iOS.Badge > 0 {
		aps["badge"] = request.iOS.Badge
	}

	// Set content-available
	if request.iOS != nil && request.iOS.ContentAvailable {
		aps["content-available"] = 1
	}

	// Set mutable-content
	if request.iOS != nil && request.iOS.MutableContent {
		aps["mutable-content"] = 1
	}

	// Set category
	if request.iOS != nil && request.iOS.Category != "" {
		aps["category"] = request.iOS.Category
	}

	payload["aps"] = aps

	// Add custom data
	if request.Data != nil {
		for key, value := range request.Data {
			payload[key] = value
		}
	}

	if request.iOS != nil && request.iOS.CustomData != nil {
		for key, value := range request.iOS.CustomData {
			payload[key] = value
		}
	}

	notification := &apns2.Notification{
		DeviceToken: request.Token,
		Topic:       a.topic,
		Payload:     payload,
	}

	// Set priority
	if request.Priority == "high" {
		notification.Priority = apns2.PriorityHigh
	} else {
		notification.Priority = apns2.PriorityLow
	}

	// Set expiration
	if request.TTL > 0 {
		notification.Expiration = time.Now().Add(time.Duration(request.TTL) * time.Second)
	}

	// Set collapse ID
	if request.CollapseKey != "" {
		notification.CollapseID = request.CollapseKey
	}

	return notification
}
