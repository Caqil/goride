package sms

import (
	"context"
	"fmt"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/sns"
	snsTypes "github.com/aws/aws-sdk-go-v2/service/sns/types"
)

type AWSSNSProvider struct {
	client *sns.Client
	region string
}

func NewAWSSNSProvider(region string) (*AWSSNSProvider, error) {
	cfg, err := config.LoadDefaultConfig(context.Background(), config.WithRegion(region))
	if err != nil {
		return nil, fmt.Errorf("failed to load AWS config: %w", err)
	}

	return &AWSSNSProvider{
		client: sns.NewFromConfig(cfg),
		region: region,
	}, nil
}

func (a *AWSSNSProvider) SendSMS(ctx context.Context, request *SMSRequest) (*SMSResponse, error) {
	input := &sns.PublishInput{
		PhoneNumber: aws.String(request.To),
		MessageAttributes: map[string]snsTypes.MessageAttributeValue{
			"AWS.SNS.SMS.SMSType": {
				DataType:    aws.String("String"),
				StringValue: aws.String(a.getSMSType(request.Type)),
			},
		},
	}

	resp, err := a.client.Publish(ctx, input)
	if err != nil {
		return &SMSResponse{
			Status: "failed",
			Error:  err.Error(),
		}, err
	}

	return &SMSResponse{
		MessageID: *resp.MessageId,
		Status:    "sent",
	}, nil
}

func (a *AWSSNSProvider) SendBulkSMS(ctx context.Context, requests []*SMSRequest) ([]*SMSResponse, error) {
	responses := make([]*SMSResponse, len(requests))

	for i, req := range requests {
		resp, err := a.SendSMS(ctx, req)
		if err != nil {
			resp = &SMSResponse{
				Status: "failed",
				Error:  err.Error(),
			}
		}
		responses[i] = resp
	}

	return responses, nil
}

func (a *AWSSNSProvider) GetDeliveryStatus(ctx context.Context, messageID string) (*DeliveryStatus, error) {
	// AWS SNS doesn't provide direct delivery status checking
	// You would need to set up delivery status logging to CloudWatch
	return &DeliveryStatus{
		MessageID: messageID,
		Status:    "unknown",
	}, nil
}

func (a *AWSSNSProvider) getSMSType(messageType string) string {
	switch messageType {
	case "promotional":
		return "Promotional"
	case "transactional", "otp":
		return "Transactional"
	default:
		return "Transactional"
	}
}
