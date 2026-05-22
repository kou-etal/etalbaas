package internal

import (
	"errors"

	"github.com/caarlos0/env/v11"

	"github.com/kou-etal/etalbaas/pkg/config"
)

type Config struct {
	config.BaseConfig

	// S3-compatible provider settings.
	S3Endpoint  string `env:"S3_ENDPOINT"   envDefault:""`
	S3Bucket    string `env:"S3_BUCKET"     envDefault:"etalbaas-storage"`
	S3AccessKey string `env:"S3_ACCESS_KEY" envDefault:""`
	S3SecretKey string `env:"S3_SECRET_KEY" envDefault:""`
	S3Region    string `env:"S3_REGION"     envDefault:"auto"`
	S3UseSSL    bool   `env:"S3_USE_SSL"    envDefault:"true"`

	// K8s toggle — when false, uses static pool (no per-project routing).
	K8sEnabled bool `env:"K8S_ENABLED" envDefault:"true"`
}

func NewConfig() (*Config, error) {
	cfg := &Config{}
	if err := env.Parse(cfg); err != nil {
		return nil, err
	}
	if err := cfg.Validate(); err != nil {
		return nil, err
	}
	if cfg.S3Endpoint == "" {
		return nil, errors.New("S3_ENDPOINT is required")
	}
	if cfg.S3AccessKey == "" || cfg.S3SecretKey == "" {
		return nil, errors.New("S3_ACCESS_KEY and S3_SECRET_KEY are required")
	}
	return cfg, nil
}
