package config

import (
	"fmt"

	"github.com/caarlos0/env/v11"
)

// Config holds all NATS sidecar configuration, loaded from environment variables.
type Config struct {
	NATSURL     string `env:"SIDECAR_NATS_URL,required"`
	Stream      string `env:"SIDECAR_STREAM,required"`
	Consumer    string `env:"SIDECAR_CONSUMER,required"`
	FunctionURL string `env:"SIDECAR_FUNCTION_URL" envDefault:"http://localhost:8080/invoke"`

	HealthPort int `env:"SIDECAR_HEALTH_PORT" envDefault:"8081"`

	// MaxRetries is the number of times to retry an HTTP call before Nak-ing the message.
	MaxRetries int `env:"SIDECAR_MAX_RETRIES" envDefault:"2"`
}

// Load parses environment variables into Config.
func Load() (*Config, error) {
	cfg := &Config{}
	if err := env.Parse(cfg); err != nil {
		return nil, fmt.Errorf("parse config: %w", err)
	}
	return cfg, nil
}
