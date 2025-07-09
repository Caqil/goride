package sms

import (
	"context"
	"fmt"

	"github.com/twilio/twilio-go"
	api "github.com/twilio/twilio-go/rest/api/v2010"
)

type TwilioProvider struct {
	client     *twilio.RestClient
	fromNumber string
}

func NewTwilioProvider(accountSID, authToken, fromNumber string) *TwilioProvider {
	client := twilio.NewRestClientWithParams(twilio.ClientParams{
		Username: accountSID,
		Password: authToken,
	})

	return &TwilioProvider{
		client:     client,
		fromNumber: fromNumber,
	}
}

func (t *TwilioProvider) SendSMS(ctx context.Context, request *SMSRequest) (*SMSResponse, error) {
	params := &api.CreateMessageParams{}
	params.SetTo(request.To)
	params.SetFrom(t.getFromNumber(request.From))
	params.SetBody(request.Message)

	resp, err := t.client.Api.CreateMessage(params)
	if err != nil {
		return &SMSResponse{
			Status: "failed",
			Error:  err.Error(),
		}, err
	}

	return &SMSResponse{
		MessageID: *resp.Sid,
		Status:    string(*resp.Status),
	}, nil
}

func (t *TwilioProvider) SendBulkSMS(ctx context.Context, requests []*SMSRequest) ([]*SMSResponse, error) {
	responses := make([]*SMSResponse, len(requests))

	for i, req := range requests {
		resp, err := t.SendSMS(ctx, req)
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

func (t *TwilioProvider) GetDeliveryStatus(ctx context.Context, messageID string) (*DeliveryStatus, error) {
	params := &api.FetchMessageParams{}

	resp, err := t.client.Api.FetchMessage(messageID, params)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch message status: %w", err)
	}

	status := &DeliveryStatus{
		MessageID: messageID,
		Status:    string(*resp.Status),
	}

	if resp.ErrorCode != nil {
		status.ErrorCode = fmt.Sprintf("%d", *resp.ErrorCode)
	}

	if resp.ErrorMessage != nil {
		status.ErrorMessage = *resp.ErrorMessage
	}

	return status, nil
}

func (t *TwilioProvider) getFromNumber(from string) string {
	if from != "" {
		return from
	}
	return t.fromNumber
}
