package payment

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"
)

type PayPalProvider struct {
	clientID     string
	clientSecret string
	baseURL      string
	httpClient   *http.Client
}

type PayPalTokenResponse struct {
	AccessToken string `json:"access_token"`
	TokenType   string `json:"token_type"`
	ExpiresIn   int    `json:"expires_in"`
}

type PayPalPaymentRequest struct {
	Intent        string                   `json:"intent"`
	PurchaseUnits []PayPalPurchaseUnit     `json:"purchase_units"`
	PaymentSource PayPalPaymentSource      `json:"payment_source"`
	AppContext    PayPalApplicationContext `json:"application_context"`
}

type PayPalPurchaseUnit struct {
	Amount      PayPalAmount `json:"amount"`
	Description string       `json:"description"`
	ReferenceID string       `json:"reference_id"`
}

type PayPalAmount struct {
	CurrencyCode string `json:"currency_code"`
	Value        string `json:"value"`
}

type PayPalPaymentSource struct {
	Card PayPalCard `json:"card"`
}

type PayPalCard struct {
	Number      string `json:"number"`
	ExpiryMonth string `json:"expiry_month"`
	ExpiryYear  string `json:"expiry_year"`
	SecurityCode string `json:"security_code"`
}

type PayPalApplicationContext struct {
	ReturnURL string `json:"return_url"`
	CancelURL string `json:"cancel_url"`
}

func NewPayPalProvider(clientID, clientSecret, mode string) *PayPalProvider {
	baseURL := "https://api.sandbox.paypal.com"
	if mode == "live" {
		baseURL = "https://api.paypal.com"
	}

	return &PayPalProvider{
		clientID:     clientID,
		clientSecret: clientSecret,
		baseURL:      baseURL,
		httpClient:   &http.Client{Timeout: 30 * time.Second},
	}
}

func (p *PayPalProvider) ProcessPayment(ctx context.Context, request *PaymentRequest) (*PaymentResponse, error) {
	token, err := p.getAccessToken(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get access token: %w", err)
	}

	paypalRequest := PayPalPaymentRequest{
		Intent: "CAPTURE",
		PurchaseUnits: []PayPalPurchaseUnit{
			{
				Amount: PayPalAmount{
					CurrencyCode: strings.ToUpper(request.Currency),
					Value:        fmt.Sprintf("%.2f", request.Amount),
				},
				Description: request.Description,
				ReferenceID: request.CustomerID,
			},
		},
		PaymentSource: PayPalPaymentSource{
			Card: PayPalCard{
				Number: request.PaymentMethodID, // This would be tokenized in real implementation
			},
		},
	}

	reqBody, err := json.Marshal(paypalRequest)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal request: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, "POST", p.baseURL+"/v2/checkout/orders", bytes.NewBuffer(reqBody))
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+token)

	resp, err := p.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to execute request: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response: %w", err)
	}

	if resp.StatusCode != http.StatusCreated {
		return nil, fmt.Errorf("PayPal API error: %s", string(body))
	}

	var result map[string]interface{}
	err = json.Unmarshal(body, &result)
	if err != nil {
		return nil, fmt.Errorf("failed to unmarshal response: %w", err)
	}

	return &PaymentResponse{
		TransactionID: result["id"].(string),
		Status:        result["status"].(string),
		Amount:        request.Amount,
		Currency:      request.Currency,
		CreatedAt:     time.Now().Unix(),
	}, nil
}

func (p *PayPalProvider) RefundPayment(ctx context.Context, request *RefundRequest) (*RefundResponse, error) {
	token, err := p.getAccessToken(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get access token: %w", err)
	}

	refundRequest := map[string]interface{}{
		"amount": map[string]string{
			"value":         fmt.Sprintf("%.2f", request.Amount),
			"currency_code": "USD", // Should be dynamic
		},
		"note_to_payer": request.Reason,
	}

	reqBody, err := json.Marshal(refundRequest)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal request: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, "POST", 
		fmt.Sprintf("%s/v2/payments/captures/%s/refund", p.baseURL, request.TransactionID), 
		bytes.NewBuffer(reqBody))
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+token)

	resp, err := p.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to execute request: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response: %w", err)
	}

	if resp.StatusCode != http.StatusCreated {
		return nil, fmt.Errorf("PayPal API error: %s", string(body))
	}

	var result map[string]interface{}
	err = json.Unmarshal(body, &result)
	if err != nil {
		return nil, fmt.Errorf("failed to unmarshal response: %w", err)
	}

	return &RefundResponse{
		RefundID:  result["id"].(string),
		Status:    result["status"].(string),
		Amount:    request.Amount,
		Currency:  "USD", // Should be dynamic
		CreatedAt: time.Now().Unix(),
	}, nil
}

func (p *PayPalProvider) CreatePaymentMethod(ctx context.Context, request *PaymentMethodRequest) (*PaymentMethodResponse, error) {
	// PayPal doesn't have a direct equivalent to Stripe's payment methods
	// This would typically involve vault API for storing payment information
	return nil, fmt.Errorf("CreatePaymentMethod not implemented for PayPal")
}

func (p *PayPalProvider) DeletePaymentMethod(ctx context.Context, paymentMethodID string) error {
	return fmt.Errorf("DeletePaymentMethod not implemented for PayPal")
}

func (p *PayPalProvider) GetPaymentMethod(ctx context.Context, paymentMethodID string) (*PaymentMethodResponse, error) {
	return nil, fmt.Errorf("GetPaymentMethod not implemented for PayPal")
}

func (p *PayPalProvider) ValidateWebhook(ctx context.Context, payload []byte, signature string) (*WebhookEvent, error) {
	// PayPal webhook validation implementation
	var event map[string]interface{}
	err := json.Unmarshal(payload, &event)
	if err != nil {
		return nil, fmt.Errorf("failed to unmarshal webhook payload: %w", err)
	}

	return &WebhookEvent{
		EventID:   event["id"].(string),
		EventType: event["event_type"].(string),
		Data:      event,
		CreatedAt: time.Now().Unix(),
	}, nil
}

func (p *PayPalProvider) getAccessToken(ctx context.Context) (string, error) {
	data := "grant_type=client_credentials"
	req, err := http.NewRequestWithContext(ctx, "POST", p.baseURL+"/v1/oauth2/token", strings.NewReader(data))
	if err != nil {
		return "", err
	}

	req.SetBasicAuth(p.clientID, p.clientSecret)
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	resp, err := p.httpClient.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	var tokenResp PayPalTokenResponse
	err = json.NewDecoder(resp.Body).Decode(&tokenResp)
	if err != nil {
		return "", err
	}

	return tokenResp.AccessToken, nil
}
