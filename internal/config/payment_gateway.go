package config

type PaymentConfig struct {
	DefaultProvider string          `yaml:"default_provider"`
	Stripe          *StripeConfig   `yaml:"stripe"`
	PayPal          *PayPalConfig   `yaml:"paypal"`
	Razorpay        *RazorpayConfig `yaml:"razorpay"`
	Currency        string          `yaml:"currency"`
	CommissionRate  float64         `yaml:"commission_rate"`
}

type StripeConfig struct {
	PublishableKey string `yaml:"publishable_key"`
	SecretKey      string `yaml:"secret_key"`
	WebhookSecret  string `yaml:"webhook_secret"`
	ConnectAccount string `yaml:"connect_account"`
}

type PayPalConfig struct {
	ClientID     string `yaml:"client_id"`
	ClientSecret string `yaml:"client_secret"`
	Mode         string `yaml:"mode"` // sandbox or live
	WebhookID    string `yaml:"webhook_id"`
}

type RazorpayConfig struct {
	KeyID     string `yaml:"key_id"`
	KeySecret string `yaml:"key_secret"`
	Webhook   string `yaml:"webhook_secret"`
}

func loadPaymentConfig() *PaymentConfig {
	return &PaymentConfig{
		DefaultProvider: getEnv("PAYMENT_DEFAULT_PROVIDER", "stripe"),
		Stripe: &StripeConfig{
			PublishableKey: getEnv("STRIPE_PUBLISHABLE_KEY", ""),
			SecretKey:      getEnv("STRIPE_SECRET_KEY", ""),
			WebhookSecret:  getEnv("STRIPE_WEBHOOK_SECRET", ""),
			ConnectAccount: getEnv("STRIPE_CONNECT_ACCOUNT", ""),
		},
		PayPal: &PayPalConfig{
			ClientID:     getEnv("PAYPAL_CLIENT_ID", ""),
			ClientSecret: getEnv("PAYPAL_CLIENT_SECRET", ""),
			Mode:         getEnv("PAYPAL_MODE", "sandbox"),
			WebhookID:    getEnv("PAYPAL_WEBHOOK_ID", ""),
		},
		Razorpay: &RazorpayConfig{
			KeyID:     getEnv("RAZORPAY_KEY_ID", ""),
			KeySecret: getEnv("RAZORPAY_KEY_SECRET", ""),
			Webhook:   getEnv("RAZORPAY_WEBHOOK_SECRET", ""),
		},
		Currency:       getEnv("PAYMENT_CURRENCY", "USD"),
		CommissionRate: getEnvAsFloat64("PAYMENT_COMMISSION_RATE", 0.05), // 5%
	}
}
