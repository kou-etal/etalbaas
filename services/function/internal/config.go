package internal

import (
	"github.com/caarlos0/env/v11"
	"github.com/kou-etal/etalbaas/pkg/config"
)

type Config struct {
	config.BaseConfig
	K8sEnabled bool `env:"K8S_ENABLED" envDefault:"false"`
}

func NewConfig() (*Config, error) {
	cfg := &Config{}
	if err := env.Parse(cfg); err != nil {
		return nil, err
	}
	if err := cfg.Validate(); err != nil {
		return nil, err
	}
	return cfg, nil
}
