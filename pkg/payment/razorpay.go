package payment

import (
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"time"

	"github.com/razorpay/razorpay-go"
)

type RazorpayProvider struct {
	client        *razorpay.Client
	webhookSecret string
}

func NewRazorpayProvider(keyID, keySecret, webhookSecret string) *RazorpayProvider {
	client := razorpay.NewClient(keyID, keySecret)

	return &RazorpayProvider{
		client:        client,
		webhookSecret: webhookSecret,
	}
}

func (r *RazorpayProvider) ProcessPayment(ctx context.Context, request *PaymentRequest) (*PaymentResponse, error) {
	// Create order first
	orderData := map[string]interface{}{
		"amount":   int(request.Amount * 100), // Amount in paise
		"currency": request.Currency,
		"receipt":  request.CustomerID,
		"notes":    request.Metadata,
	}

	order, err := r.client.Order.Create(orderData, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create order: %w", err)
	}

	// Return order details - actual payment will be processed on frontend
	// In Razorpay, payments are typically authorized on the frontend and then captured
	return &PaymentResponse{
		TransactionID: order["id"].(string),
		Status:        "created",
		Amount:        float64(order["amount"].(int)) / 100,
		Currency:      order["currency"].(string),
		CreatedAt:     int64(order["created_at"].(int)),
	}, nil
}

func (r *RazorpayProvider) RefundPayment(ctx context.Context, request *RefundRequest) (*RefundResponse, error) {
	refundData := map[string]interface{}{
		"amount": int(request.Amount * 100),
		"notes": map[string]interface{}{
			"reason": request.Reason,
		},
	}

	amount := int(request.Amount * 100)
	refund, err := r.client.Payment.Refund(request.TransactionID, amount, refundData, map[string]string{})
	if err != nil {
		return nil, fmt.Errorf("failed to create refund: %w", err)
	}

	return &RefundResponse{
		RefundID:  refund["id"].(string),
		Status:    refund["status"].(string),
		Amount:    float64(refund["amount"].(int)) / 100,
		Currency:  refund["currency"].(string),
		CreatedAt: int64(refund["created_at"].(int)),
	}, nil
}

func (r *RazorpayProvider) CreatePaymentMethod(ctx context.Context, request *PaymentMethodRequest) (*PaymentMethodResponse, error) {
	// Razorpay doesn't have direct payment method storage like Stripe
	// This would involve creating a customer and token
	return nil, fmt.Errorf("CreatePaymentMethod not fully implemented for Razorpay")
}

func (r *RazorpayProvider) DeletePaymentMethod(ctx context.Context, paymentMethodID string) error {
	return fmt.Errorf("DeletePaymentMethod not implemented for Razorpay")
}

func (r *RazorpayProvider) GetPaymentMethod(ctx context.Context, paymentMethodID string) (*PaymentMethodResponse, error) {
	return nil, fmt.Errorf("GetPaymentMethod not implemented for Razorpay")
}

func (r *RazorpayProvider) ValidateWebhook(ctx context.Context, payload []byte, signature string) (*WebhookEvent, error) {
	// Verify webhook signature
	expectedSignature := r.generateSignature(string(payload))
	if !hmac.Equal([]byte(signature), []byte(expectedSignature)) {
		return nil, fmt.Errorf("invalid webhook signature")
	}

	var event map[string]interface{}
	err := json.Unmarshal(payload, &event)
	if err != nil {
		return nil, fmt.Errorf("failed to unmarshal webhook payload: %w", err)
	}

	return &WebhookEvent{
		EventID:   event["id"].(string),
		EventType: event["event"].(string),
		Data:      event,
		CreatedAt: time.Now().Unix(),
	}, nil
}

func (r *RazorpayProvider) generateSignature(payload string) string {
	h := hmac.New(sha256.New, []byte(r.webhookSecret))
	h.Write([]byte(payload))
	return hex.EncodeToString(h.Sum(nil))
}
