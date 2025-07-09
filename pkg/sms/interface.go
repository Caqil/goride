package sms

import "context"

type SMSProvider interface {
	SendSMS(ctx context.Context, request *SMSRequest) (*SMSResponse, error)
	SendBulkSMS(ctx context.Context, requests []*SMSRequest) ([]*SMSResponse, error)
	GetDeliveryStatus(ctx context.Context, messageID string) (*DeliveryStatus, error)
}

type SMSRequest struct {
	To      string `json:"to"`
	From    string `json:"from"`
	Message string `json:"message"`
	Type    string `json:"type"` // transactional, promotional, otp
}

type SMSResponse struct {
	MessageID string `json:"message_id"`
	Status    string `json:"status"`
	Cost      string `json:"cost"`
	Error     string `json:"error,omitempty"`
}

type DeliveryStatus struct {
	MessageID    string `json:"message_id"`
	Status       string `json:"status"`
	DeliveredAt  int64  `json:"delivered_at,omitempty"`
	ErrorCode    string `json:"error_code,omitempty"`
	ErrorMessage string `json:"error_message,omitempty"`
}
