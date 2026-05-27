package internal

import (
	"github.com/caarlos0/env/v11"
	"github.com/kou-etal/etalbaas/pkg/config"
)

type Config struct {
	config.BaseConfig
	K8sEnabled          bool   `env:"K8S_ENABLED" envDefault:"false"`
	GPUEnabled          bool   `env:"GPU_ENABLED" envDefault:"false"`
	PlatformNamespace   string `env:"PLATFORM_NAMESPACE" envDefault:"platform-system"`
	DispatcherImage     string `env:"DISPATCHER_IMAGE" envDefault:"etalbaas/gpu-dispatcher:latest"`
	SandboxRuntimeClass string `env:"SANDBOX_RUNTIME_CLASS" envDefault:""`
	GPUAPIKeySecret     string `env:"GPU_API_KEY_SECRET" envDefault:"runpod-creds"`
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
