package config

type SMTPConfig struct {
	Host       string `yaml:"host"`
	Port       int    `yaml:"port"`
	Username   string `yaml:"username"`
	Password   string `yaml:"password"`
	FromEmail  string `yaml:"from_email"`
	FromName   string `yaml:"from_name"`
	SSL        bool   `yaml:"ssl"`
	TLS        bool   `yaml:"tls"`
	AuthMethod string `yaml:"auth_method"`
}

func loadSMTPConfig() *SMTPConfig {
	return &SMTPConfig{
		Host:       getEnv("SMTP_HOST", "smtp.gmail.com"),
		Port:       getEnvAsInt("SMTP_PORT", 587),
		Username:   getEnv("SMTP_USERNAME", ""),
		Password:   getEnv("SMTP_PASSWORD", ""),
		FromEmail:  getEnv("SMTP_FROM_EMAIL", "noreply@uberclone.com"),
		FromName:   getEnv("SMTP_FROM_NAME", "UberClone"),
		SSL:        getEnvAsBool("SMTP_SSL", false),
		TLS:        getEnvAsBool("SMTP_TLS", true),
		AuthMethod: getEnv("SMTP_AUTH_METHOD", "plain"),
	}
}
