package payment

import (
	"context"
)

type PaymentProvider interface {
	ProcessPayment(ctx context.Context, request *PaymentRequest) (*PaymentResponse, error)
	RefundPayment(ctx context.Context, request *RefundRequest) (*RefundResponse, error)
	CreatePaymentMethod(ctx context.Context, request *PaymentMethodRequest) (*PaymentMethodResponse, error)
	DeletePaymentMethod(ctx context.Context, paymentMethodID string) error
	GetPaymentMethod(ctx context.Context, paymentMethodID string) (*PaymentMethodResponse, error)
	ValidateWebhook(ctx context.Context, payload []byte, signature string) (*WebhookEvent, error)
}

type PaymentRequest struct {
	PaymentMethodID string                 `json:"payment_method_id"`
	Amount          float64                `json:"amount"`
	Currency        string                 `json:"currency"`
	Description     string                 `json:"description"`
	CustomerID      string                 `json:"customer_id"`
	Metadata        map[string]interface{} `json:"metadata"`
}

type PaymentResponse struct {
	TransactionID string                 `json:"transaction_id"`
	Status        string                 `json:"status"`
	Amount        float64                `json:"amount"`
	Currency      string                 `json:"currency"`
	Fees          float64                `json:"fees"`
	CreatedAt     int64                  `json:"created_at"`
	Metadata      map[string]interface{} `json:"metadata"`
}

type RefundRequest struct {
	TransactionID string  `json:"transaction_id"`
	Amount        float64 `json:"amount"`
	Reason        string  `json:"reason"`
}

type RefundResponse struct {
	RefundID  string  `json:"refund_id"`
	Status    string  `json:"status"`
	Amount    float64 `json:"amount"`
	Currency  string  `json:"currency"`
	CreatedAt int64   `json:"created_at"`
}

type PaymentMethodRequest struct {
	CustomerID     string                 `json:"customer_id"`
	Type           string                 `json:"type"`
	Token          string                 `json:"token"`
	BillingAddress *BillingAddress        `json:"billing_address"`
	Metadata       map[string]interface{} `json:"metadata"`
}

type PaymentMethodResponse struct {
	PaymentMethodID string          `json:"payment_method_id"`
	Type            string          `json:"type"`
	LastFourDigits  string          `json:"last_four_digits"`
	ExpiryMonth     int             `json:"expiry_month"`
	ExpiryYear      int             `json:"expiry_year"`
	BillingAddress  *BillingAddress `json:"billing_address"`
	CreatedAt       int64           `json:"created_at"`
}

type BillingAddress struct {
	Street     string `json:"street"`
	City       string `json:"city"`
	State      string `json:"state"`
	PostalCode string `json:"postal_code"`
	Country    string `json:"country"`
}

type WebhookEvent struct {
	EventID   string                 `json:"event_id"`
	EventType string                 `json:"event_type"`
	Data      map[string]interface{} `json:"data"`
	CreatedAt int64                  `json:"created_at"`
}
