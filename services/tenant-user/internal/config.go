package internal

import (
	"github.com/caarlos0/env/v11"
	"github.com/kou-etal/etalbaas/pkg/config"
)

type Config struct {
	config.BaseConfig
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
