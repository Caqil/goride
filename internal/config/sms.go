package config

type SMSConfig struct {
	Provider    string            `yaml:"provider"`
	Twilio      *TwilioConfig     `yaml:"twilio"`
	AWS         *AWSSNSConfig     `yaml:"aws"`
	DefaultFrom string            `yaml:"default_from"`
	Settings    map[string]string `yaml:"settings"`
}

type TwilioConfig struct {
	AccountSID string `yaml:"account_sid"`
	AuthToken  string `yaml:"auth_token"`
	FromNumber string `yaml:"from_number"`
}

type AWSSNSConfig struct {
	Region          string `yaml:"region"`
	AccessKeyID     string `yaml:"access_key_id"`
	SecretAccessKey string `yaml:"secret_access_key"`
}

func loadSMSConfig() *SMSConfig {
	return &SMSConfig{
		Provider: getEnv("SMS_PROVIDER", "twilio"),
		Twilio: &TwilioConfig{
			AccountSID: getEnv("TWILIO_ACCOUNT_SID", ""),
			AuthToken:  getEnv("TWILIO_AUTH_TOKEN", ""),
			FromNumber: getEnv("TWILIO_FROM_NUMBER", ""),
		},
		AWS: &AWSSNSConfig{
			Region:          getEnv("AWS_REGION", "us-east-1"),
			AccessKeyID:     getEnv("AWS_ACCESS_KEY_ID", ""),
			SecretAccessKey: getEnv("AWS_SECRET_ACCESS_KEY", ""),
		},
		DefaultFrom: getEnv("SMS_DEFAULT_FROM", "UberClone"),
		Settings:    make(map[string]string),
	}
}
