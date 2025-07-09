package config

type MapsConfig struct {
	Provider   string            `yaml:"provider"`
	GoogleMaps *GoogleMapsConfig `yaml:"google_maps"`
	Mapbox     *MapboxConfig     `yaml:"mapbox"`
}

type GoogleMapsConfig struct {
	APIKey    string `yaml:"api_key"`
	SecretKey string `yaml:"secret_key"`
}

type MapboxConfig struct {
	AccessToken string `yaml:"access_token"`
	SecretKey   string `yaml:"secret_key"`
}

func loadMapsConfig() *MapsConfig {
	return &MapsConfig{
		Provider: getEnv("MAPS_PROVIDER", "google"),
		GoogleMaps: &GoogleMapsConfig{
			APIKey:    getEnv("GOOGLE_MAPS_API_KEY", ""),
			SecretKey: getEnv("GOOGLE_MAPS_SECRET_KEY", ""),
		},
		Mapbox: &MapboxConfig{
			AccessToken: getEnv("MAPBOX_ACCESS_TOKEN", ""),
			SecretKey:   getEnv("MAPBOX_SECRET_KEY", ""),
		},
	}
}
