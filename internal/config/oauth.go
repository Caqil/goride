package config

type OAuthConfig struct {
	Google   *GoogleOAuthConfig   `yaml:"google"`
	Facebook *FacebookOAuthConfig `yaml:"facebook"`
	Apple    *AppleOAuthConfig    `yaml:"apple"`
}

type GoogleOAuthConfig struct {
	ClientID     string   `yaml:"client_id"`
	ClientSecret string   `yaml:"client_secret"`
	RedirectURL  string   `yaml:"redirect_url"`
	Scopes       []string `yaml:"scopes"`
}

type FacebookOAuthConfig struct {
	AppID       string   `yaml:"app_id"`
	AppSecret   string   `yaml:"app_secret"`
	RedirectURL string   `yaml:"redirect_url"`
	Scopes      []string `yaml:"scopes"`
}

type AppleOAuthConfig struct {
	ClientID    string `yaml:"client_id"`
	TeamID      string `yaml:"team_id"`
	KeyID       string `yaml:"key_id"`
	KeyFile     string `yaml:"key_file"`
	RedirectURL string `yaml:"redirect_url"`
}

func loadOAuthConfig() *OAuthConfig {
	return &OAuthConfig{
		Google: &GoogleOAuthConfig{
			ClientID:     getEnv("GOOGLE_CLIENT_ID", ""),
			ClientSecret: getEnv("GOOGLE_CLIENT_SECRET", ""),
			RedirectURL:  getEnv("GOOGLE_REDIRECT_URL", ""),
			Scopes:       getEnvAsSlice("GOOGLE_SCOPES", []string{"email", "profile"}),
		},
		Facebook: &FacebookOAuthConfig{
			AppID:       getEnv("FACEBOOK_APP_ID", ""),
			AppSecret:   getEnv("FACEBOOK_APP_SECRET", ""),
			RedirectURL: getEnv("FACEBOOK_REDIRECT_URL", ""),
			Scopes:      getEnvAsSlice("FACEBOOK_SCOPES", []string{"email", "public_profile"}),
		},
		Apple: &AppleOAuthConfig{
			ClientID:    getEnv("APPLE_CLIENT_ID", ""),
			TeamID:      getEnv("APPLE_TEAM_ID", ""),
			KeyID:       getEnv("APPLE_KEY_ID", ""),
			KeyFile:     getEnv("APPLE_KEY_FILE", ""),
			RedirectURL: getEnv("APPLE_REDIRECT_URL", ""),
		},
	}
}
