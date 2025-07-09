package push

import (
	"context"
	"fmt"

	"firebase.google.com/go/v4"
	"firebase.google.com/go/v4/messaging"
	"google.golang.org/api/option"
)

type FCMProvider struct {
	client *messaging.Client
}

func NewFCMProvider(credentialsFile string) (*FCMProvider, error) {
	ctx := context.Background()

	opt := option.WithCredentialsFile(credentialsFile)
	app, err := firebase.NewApp(ctx, nil, opt)
	if err != nil {
		return nil, fmt.Errorf("failed to initialize Firebase app: %w", err)
	}

	client, err := app.Messaging(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get messaging client: %w", err)
	}

	return &FCMProvider{
		client: client,
	}, nil
}

func (f *FCMProvider) SendNotification(ctx context.Context, request *NotificationRequest) (*NotificationResponse, error) {
	message := f.buildMessage(request)

	response, err := f.client.Send(ctx, message)
	if err != nil {
		return &NotificationResponse{
			Success: false,
			Error:   err.Error(),
			Token:   request.Token,
		}, err
	}

	return &NotificationResponse{
		MessageID: response,
		Success:   true,
		Token:     request.Token,
	}, nil
}

func (f *FCMProvider) SendBulkNotifications(ctx context.Context, requests []*NotificationRequest) ([]*NotificationResponse, error) {
	messages := make([]*messaging.Message, len(requests))
	for i, req := range requests {
		messages[i] = f.buildMessage(req)
	}

	batchResponse, err := f.client.SendAll(ctx, messages)
	if err != nil {
		return nil, fmt.Errorf("failed to send bulk notifications: %w", err)
	}

	responses := make([]*NotificationResponse, len(requests))
	for i, response := range batchResponse.Responses {
		if response.Success {
			responses[i] = &NotificationResponse{
				MessageID: response.MessageID,
				Success:   true,
				Token:     requests[i].Token,
			}
		} else {
			responses[i] = &NotificationResponse{
				Success: false,
				Error:   response.Error.Error(),
				Token:   requests[i].Token,
			}
		}
	}

	return responses, nil
}

func (f *FCMProvider) SubscribeToTopic(ctx context.Context, tokens []string, topic string) error {
	_, err := f.client.SubscribeToTopic(ctx, tokens, topic)
	return err
}

func (f *FCMProvider) UnsubscribeFromTopic(ctx context.Context, tokens []string, topic string) error {
	_, err := f.client.UnsubscribeFromTopic(ctx, tokens, topic)
	return err
}

func (f *FCMProvider) ValidateToken(ctx context.Context, token string) (bool, error) {
	// FCM doesn't have a direct token validation API
	// We can try sending a dry run message
	message := &messaging.Message{
		Token: token,
		Data: map[string]string{
			"test": "validation",
		},
	}

	_, err := f.client.Send(ctx, message)
	return err == nil, err
}

func (f *FCMProvider) buildMessage(request *NotificationRequest) *messaging.Message {
	message := &messaging.Message{
		Data: request.Data,
	}

	// Set target
	if request.Token != "" {
		message.Token = request.Token
	} else if request.Topic != "" {
		message.Topic = request.Topic
	}

	// Set notification
	if request.Title != "" || request.Body != "" {
		message.Notification = &messaging.Notification{
			Title:    request.Title,
			Body:     request.Body,
			ImageURL: request.ImageURL,
		}
	}

	// Set Android config
	if request.Android != nil {
		message.Android = &messaging.AndroidConfig{
			Priority: request.Android.Priority,
			Data:     request.Android.CustomData,
			Notification: &messaging.AndroidNotification{
				Title:        request.Title,
				Body:         request.Body,
				Sound:        request.Android.Sound,
				Color:        request.Android.Color,
				Tag:          request.Android.Tag,
				ClickAction:  request.Android.ClickAction,
				BodyLocKey:   request.Android.BodyLocKey,
				BodyLocArgs:  request.Android.BodyLocArgs,
				TitleLocKey:  request.Android.TitleLocKey,
				TitleLocArgs: request.Android.TitleLocArgs,
				ChannelID:    request.Android.ChannelID,
			},
		}
	}

	// Set iOS config
	if request.iOS != nil {
		message.APNS = &messaging.APNSConfig{
			Payload: &messaging.APNSPayload{
				Aps: &messaging.Aps{
					Alert: &messaging.ApsAlert{
						Title: request.Title,
						Body:  request.Body,
					},
					Sound:            request.iOS.Sound,
					Badge:            &request.iOS.Badge,
					ContentAvailable: request.iOS.ContentAvailable,
					MutableContent:   request.iOS.MutableContent,
					Category:         request.iOS.Category,
				},
				CustomData: stringMapToInterfaceMap(request.iOS.CustomData),
			},
		}
	}

	return message
}

// stringMapToInterfaceMap converts a map[string]string to map[string]interface{}
func stringMapToInterfaceMap(m map[string]string) map[string]interface{} {
	if m == nil {
		return nil
	}
	result := make(map[string]interface{}, len(m))
	for k, v := range m {
		result[k] = v
	}
	return result
}
