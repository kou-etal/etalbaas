package observability_test

import (
	"context"
	"testing"

	"github.com/kou-etal/etalbaas/pkg/observability"
)

func TestNewResource(t *testing.T) {
	res, err := observability.NewResource(context.Background(), "test-service", "1.0.0", "production")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if res == nil {
		t.Fatal("resource should not be nil")
	}

	attrs := res.Attributes()
	found := map[string]string{}
	for _, attr := range attrs {
		found[string(attr.Key)] = attr.Value.AsString()
	}

	if found["service.name"] != "test-service" {
		t.Fatalf("service.name = %q, want %q", found["service.name"], "test-service")
	}
	if found["service.version"] != "1.0.0" {
		t.Fatalf("service.version = %q, want %q", found["service.version"], "1.0.0")
	}
}
