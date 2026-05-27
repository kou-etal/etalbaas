package config

import (
	"errors"
	"fmt"

	"github.com/caarlos0/env/v11"
)

type BaseConfig struct {
	Port         int    `env:"PORT"                        envDefault:"8080"`
	MetricsPort  int    `env:"METRICS_PORT"                envDefault:"9090"`
	DatabaseURL  string `env:"DATABASE_URL"                envDefault:""`
	OTELEndpoint string `env:"OTEL_EXPORTER_OTLP_ENDPOINT" envDefault:""`
	Environment  string `env:"ENV"                         envDefault:"development"`
	JWTSecret    string `env:"JWT_SECRET"                  envDefault:"dev-secret-change-me"`
}

func New() (*BaseConfig, error) {
	cfg := &BaseConfig{}
	if err := env.Parse(cfg); err != nil {
		return nil, fmt.Errorf("parse env: %w", err)
	}
	if err := cfg.Validate(); err != nil {
		return nil, err
	}
	return cfg, nil
}

func (c *BaseConfig) Validate() error {
	if c.Environment != "development" && c.JWTSecret == "dev-secret-change-me" {
		return errors.New("JWT_SECRET must be set in non-development environment")
	}
	if c.DatabaseURL == "" {
		return errors.New("DATABASE_URL is required")
	}
	return nil
}
