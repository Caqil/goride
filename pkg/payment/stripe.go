package payment

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/stripe/stripe-go/v76"
	"github.com/stripe/stripe-go/v76/client"
	"github.com/stripe/stripe-go/v76/webhook"
)

type StripeProvider struct {
	client        *client.API
	webhookSecret string
}

func NewStripeProvider(secretKey, webhookSecret string) *StripeProvider {
	sc := &client.API{}
	sc.Init(secretKey, nil)

	return &StripeProvider{
		client:        sc,
		webhookSecret: webhookSecret,
	}
}

func (s *StripeProvider) ProcessPayment(ctx context.Context, request *PaymentRequest) (*PaymentResponse, error) {
	params := &stripe.PaymentIntentParams{
		Amount:             stripe.Int64(int64(request.Amount * 100)), // Convert to cents
		Currency:           stripe.String(request.Currency),
		PaymentMethod:      stripe.String(request.PaymentMethodID),
		Customer:           stripe.String(request.CustomerID),
		Description:        stripe.String(request.Description),
		ConfirmationMethod: stripe.String("manual"),
		Confirm:            stripe.Bool(true),
	}

	// Add metadata
	if request.Metadata != nil {
		for key, value := range request.Metadata {
			params.AddMetadata(key, fmt.Sprintf("%v", value))
		}
	}

	pi, err := s.client.PaymentIntents.New(params)
	if err != nil {
		return nil, fmt.Errorf("failed to create payment intent: %w", err)
	}

	return &PaymentResponse{
		TransactionID: pi.ID,
		Status:        string(pi.Status),
		Amount:        float64(pi.Amount) / 100,
		Currency:      string(pi.Currency),
		Fees:          float64(pi.ApplicationFeeAmount) / 100,
		CreatedAt:     pi.Created,
		Metadata:      convertStripeMetadata(pi.Metadata),
	}, nil
}

func (s *StripeProvider) RefundPayment(ctx context.Context, request *RefundRequest) (*RefundResponse, error) {
	params := &stripe.RefundParams{
		PaymentIntent: stripe.String(request.TransactionID),
		Reason:        stripe.String(request.Reason),
	}

	if request.Amount > 0 {
		params.Amount = stripe.Int64(int64(request.Amount * 100))
	}

	refund, err := s.client.Refunds.New(params)
	if err != nil {
		return nil, fmt.Errorf("failed to create refund: %w", err)
	}

	return &RefundResponse{
		RefundID:  refund.ID,
		Status:    string(refund.Status),
		Amount:    float64(refund.Amount) / 100,
		Currency:  string(refund.Currency),
		CreatedAt: refund.Created,
	}, nil
}

func (s *StripeProvider) CreatePaymentMethod(ctx context.Context, request *PaymentMethodRequest) (*PaymentMethodResponse, error) {
	params := &stripe.PaymentMethodParams{
		Type: stripe.String(request.Type),
		Card: &stripe.PaymentMethodCardParams{
			Token: stripe.String(request.Token),
		},
	}

	if request.BillingAddress != nil {
		params.BillingDetails = &stripe.PaymentMethodBillingDetailsParams{
			Address: &stripe.AddressParams{
				Line1:      stripe.String(request.BillingAddress.Street),
				City:       stripe.String(request.BillingAddress.City),
				State:      stripe.String(request.BillingAddress.State),
				PostalCode: stripe.String(request.BillingAddress.PostalCode),
				Country:    stripe.String(request.BillingAddress.Country),
			},
		}
	}

	pm, err := s.client.PaymentMethods.New(params)
	if err != nil {
		return nil, fmt.Errorf("failed to create payment method: %w", err)
	}

	// Attach to customer
	attachParams := &stripe.PaymentMethodAttachParams{
		Customer: stripe.String(request.CustomerID),
	}
	_, err = s.client.PaymentMethods.Attach(pm.ID, attachParams)
	if err != nil {
		return nil, fmt.Errorf("failed to attach payment method to customer: %w", err)
	}

	return &PaymentMethodResponse{
		PaymentMethodID: pm.ID,
		Type:            string(pm.Type),
		LastFourDigits:  pm.Card.Last4,
		ExpiryMonth:     int(pm.Card.ExpMonth),
		ExpiryYear:      int(pm.Card.ExpYear),
		BillingAddress:  convertStripeBillingAddress(pm.BillingDetails),
		CreatedAt:       pm.Created,
	}, nil
}

func (s *StripeProvider) DeletePaymentMethod(ctx context.Context, paymentMethodID string) error {
	_, err := s.client.PaymentMethods.Detach(paymentMethodID, nil)
	return err
}

func (s *StripeProvider) GetPaymentMethod(ctx context.Context, paymentMethodID string) (*PaymentMethodResponse, error) {
	pm, err := s.client.PaymentMethods.Get(paymentMethodID, nil)
	if err != nil {
		return nil, err
	}

	return &PaymentMethodResponse{
		PaymentMethodID: pm.ID,
		Type:            string(pm.Type),
		LastFourDigits:  pm.Card.Last4,
		ExpiryMonth:     int(pm.Card.ExpMonth),
		ExpiryYear:      int(pm.Card.ExpYear),
		BillingAddress:  convertStripeBillingAddress(pm.BillingDetails),
		CreatedAt:       pm.Created,
	}, nil
}

func (s *StripeProvider) ValidateWebhook(ctx context.Context, payload []byte, signature string) (*WebhookEvent, error) {
	event, err := webhook.ConstructEvent(payload, signature, s.webhookSecret)
	if err != nil {
		return nil, fmt.Errorf("failed to verify webhook signature: %w", err)
	}

	data := make(map[string]interface{})
	if err := json.Unmarshal(event.Data.Raw, &data); err != nil {
		return nil, fmt.Errorf("failed to unmarshal event data: %w", err)
	}

	return &WebhookEvent{
		EventID:   event.ID,
		EventType: string(event.Type),
		Data:      data,
		CreatedAt: event.Created,
	}, nil
}

// Helper functions
func convertStripeMetadata(metadata map[string]string) map[string]interface{} {
	result := make(map[string]interface{})
	for key, value := range metadata {
		result[key] = value
	}
	return result
}

func convertStripeBillingAddress(details *stripe.PaymentMethodBillingDetails) *BillingAddress {
	if details == nil || details.Address == nil {
		return nil
	}

	return &BillingAddress{
		Street:     details.Address.Line1,
		City:       details.Address.City,
		State:      details.Address.State,
		PostalCode: details.Address.PostalCode,
		Country:    details.Address.Country,
	}
}
