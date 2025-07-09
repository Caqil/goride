package config

import (
	"time"
)

type MLConfig struct {
	ETAModel       *MLModelConfig `yaml:"eta_model"`
	DemandModel    *MLModelConfig `yaml:"demand_model"`
	FraudModel     *MLModelConfig `yaml:"fraud_model"`
	SurgeModel     *MLModelConfig `yaml:"surge_model"`
	UpdateInterval time.Duration  `yaml:"update_interval"`
}

type MLModelConfig struct {
	Enabled   bool    `yaml:"enabled"`
	ModelPath string  `yaml:"model_path"`
	Version   string  `yaml:"version"`
	Threshold float64 `yaml:"threshold"`
}

func loadMLConfig() *MLConfig {
	return &MLConfig{
		ETAModel: &MLModelConfig{
			Enabled:   getEnvAsBool("ML_ETA_ENABLED", false),
			ModelPath: getEnv("ML_ETA_MODEL_PATH", ""),
			Version:   getEnv("ML_ETA_VERSION", "1.0"),
			Threshold: getEnvAsFloat64("ML_ETA_THRESHOLD", 0.8),
		},
		DemandModel: &MLModelConfig{
			Enabled:   getEnvAsBool("ML_DEMAND_ENABLED", false),
			ModelPath: getEnv("ML_DEMAND_MODEL_PATH", ""),
			Version:   getEnv("ML_DEMAND_VERSION", "1.0"),
			Threshold: getEnvAsFloat64("ML_DEMAND_THRESHOLD", 0.7),
		},
		FraudModel: &MLModelConfig{
			Enabled:   getEnvAsBool("ML_FRAUD_ENABLED", false),
			ModelPath: getEnv("ML_FRAUD_MODEL_PATH", ""),
			Version:   getEnv("ML_FRAUD_VERSION", "1.0"),
			Threshold: getEnvAsFloat64("ML_FRAUD_THRESHOLD", 0.9),
		},
		SurgeModel: &MLModelConfig{
			Enabled:   getEnvAsBool("ML_SURGE_ENABLED", false),
			ModelPath: getEnv("ML_SURGE_MODEL_PATH", ""),
			Version:   getEnv("ML_SURGE_VERSION", "1.0"),
			Threshold: getEnvAsFloat64("ML_SURGE_THRESHOLD", 0.75),
		},
		UpdateInterval: getEnvAsDuration("ML_UPDATE_INTERVAL", 1*time.Hour),
	}
}
