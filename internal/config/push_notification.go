package config

type PushConfig struct {
	Provider string      `yaml:"provider"`
	FCM      *FCMConfig  `yaml:"fcm"`
	APNS     *APNSConfig `yaml:"apns"`
}

type FCMConfig struct {
	ServerKey   string `yaml:"server_key"`
	ProjectID   string `yaml:"project_id"`
	Credentials string `yaml:"credentials_file"`
}

type APNSConfig struct {
	KeyID      string `yaml:"key_id"`
	TeamID     string `yaml:"team_id"`
	BundleID   string `yaml:"bundle_id"`
	KeyFile    string `yaml:"key_file"`
	Production bool   `yaml:"production"`
}

func loadPushConfig() *PushConfig {
	return &PushConfig{
		Provider: getEnv("PUSH_PROVIDER", "fcm"),
		FCM: &FCMConfig{
			ServerKey:   getEnv("FCM_SERVER_KEY", ""),
			ProjectID:   getEnv("FCM_PROJECT_ID", ""),
			Credentials: getEnv("FCM_CREDENTIALS_FILE", ""),
		},
		APNS: &APNSConfig{
			KeyID:      getEnv("APNS_KEY_ID", ""),
			TeamID:     getEnv("APNS_TEAM_ID", ""),
			BundleID:   getEnv("APNS_BUNDLE_ID", ""),
			KeyFile:    getEnv("APNS_KEY_FILE", ""),
			Production: getEnvAsBool("APNS_PRODUCTION", false),
		},
	}
}
