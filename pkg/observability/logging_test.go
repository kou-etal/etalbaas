package observability_test

import (
	"testing"

	"github.com/kou-etal/etalbaas/pkg/observability"
)

func TestNewLogger_Development(t *testing.T) {
	logger := observability.NewLogger("test-service", "development")
	if logger == nil {
		t.Fatal("logger should not be nil")
	}
}

func TestNewLogger_Production(t *testing.T) {
	logger := observability.NewLogger("test-service", "production")
	if logger == nil {
		t.Fatal("logger should not be nil")
	}
}
