package config

type StorageConfig struct {
	Provider string              `yaml:"provider"`
	Local    *LocalStorageConfig `yaml:"local"`
	AWS      *AWSStorageConfig   `yaml:"aws"`
	GCP      *GCPStorageConfig   `yaml:"gcp"`
}

type LocalStorageConfig struct {
	BasePath string `yaml:"base_path"`
	BaseURL  string `yaml:"base_url"`
}

type AWSStorageConfig struct {
	Region          string `yaml:"region"`
	Bucket          string `yaml:"bucket"`
	AccessKeyID     string `yaml:"access_key_id"`
	SecretAccessKey string `yaml:"secret_access_key"`
	CDNDomain       string `yaml:"cdn_domain"`
}

type GCPStorageConfig struct {
	ProjectID       string `yaml:"project_id"`
	Bucket          string `yaml:"bucket"`
	CredentialsFile string `yaml:"credentials_file"`
	CDNDomain       string `yaml:"cdn_domain"`
}

func loadStorageConfig() *StorageConfig {
	return &StorageConfig{
		Provider: getEnv("STORAGE_PROVIDER", "local"),
		Local: &LocalStorageConfig{
			BasePath: getEnv("STORAGE_LOCAL_PATH", "./uploads"),
			BaseURL:  getEnv("STORAGE_LOCAL_URL", "http://localhost:8080/uploads"),
		},
		AWS: &AWSStorageConfig{
			Region:          getEnv("AWS_S3_REGION", "us-east-1"),
			Bucket:          getEnv("AWS_S3_BUCKET", ""),
			AccessKeyID:     getEnv("AWS_ACCESS_KEY_ID", ""),
			SecretAccessKey: getEnv("AWS_SECRET_ACCESS_KEY", ""),
			CDNDomain:       getEnv("AWS_CLOUDFRONT_DOMAIN", ""),
		},
		GCP: &GCPStorageConfig{
			ProjectID:       getEnv("GCP_PROJECT_ID", ""),
			Bucket:          getEnv("GCP_STORAGE_BUCKET", ""),
			CredentialsFile: getEnv("GCP_CREDENTIALS_FILE", ""),
			CDNDomain:       getEnv("GCP_CDN_DOMAIN", ""),
		},
	}
}
