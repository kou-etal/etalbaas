package config_test

import (
	"testing"

	"github.com/kou-etal/etalbaas/pkg/config"
)

func TestNew_Development(t *testing.T) {
	t.Setenv("DATABASE_URL", "postgres://localhost/test")
	t.Setenv("ENV", "development")

	cfg, err := config.New()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if cfg.Port != 8080 {
		t.Fatalf("got port %d, want 8080", cfg.Port)
	}
	if cfg.MetricsPort != 9090 {
		t.Fatalf("got metrics port %d, want 9090", cfg.MetricsPort)
	}
	if cfg.Environment != "development" {
		t.Fatalf("got env %q, want %q", cfg.Environment, "development")
	}
}

func TestNew_MissingDatabaseURL(t *testing.T) {
	t.Setenv("DATABASE_URL", "")
	t.Setenv("ENV", "development")

	_, err := config.New()
	if err == nil {
		t.Fatal("expected error for missing DATABASE_URL")
	}
}

func TestNew_ProductionWithDefaultJWTSecret(t *testing.T) {
	t.Setenv("DATABASE_URL", "postgres://localhost/test")
	t.Setenv("ENV", "production")
	t.Setenv("JWT_SECRET", "dev-secret-change-me")

	_, err := config.New()
	if err == nil {
		t.Fatal("expected error for default JWT secret in production")
	}
}

func TestNew_ProductionWithCustomJWTSecret(t *testing.T) {
	t.Setenv("DATABASE_URL", "postgres://localhost/test")
	t.Setenv("ENV", "production")
	t.Setenv("JWT_SECRET", "my-secure-secret-key-123")

	cfg, err := config.New()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if cfg.JWTSecret != "my-secure-secret-key-123" {
		t.Fatalf("got jwt secret %q", cfg.JWTSecret)
	}
}

func TestNew_CustomPorts(t *testing.T) {
	t.Setenv("DATABASE_URL", "postgres://localhost/test")
	t.Setenv("PORT", "3000")
	t.Setenv("METRICS_PORT", "9191")

	cfg, err := config.New()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if cfg.Port != 3000 {
		t.Fatalf("got port %d, want 3000", cfg.Port)
	}
	if cfg.MetricsPort != 9191 {
		t.Fatalf("got metrics port %d, want 9191", cfg.MetricsPort)
	}
}
