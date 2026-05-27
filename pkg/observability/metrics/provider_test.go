package metrics_test

import (
	"context"
	"testing"

	sdkmetric "go.opentelemetry.io/otel/sdk/metric"
	"go.opentelemetry.io/otel/sdk/metric/metricdata"

	"github.com/kou-etal/etalbaas/pkg/observability"
	"github.com/kou-etal/etalbaas/pkg/observability/metrics"
)

func TestNewProvider(t *testing.T) {
	res, err := observability.NewResource(context.Background(), "test-service", "0.1.0", "test")
	if err != nil {
		t.Fatalf("create resource: %v", err)
	}

	reader := sdkmetric.NewManualReader()
	mp, err := metrics.NewProvider(res, reader)
	if err != nil {
		t.Fatalf("create provider: %v", err)
	}

	meter := mp.Meter("test")
	counter, err := meter.Int64Counter("test_counter")
	if err != nil {
		t.Fatalf("create counter: %v", err)
	}
	counter.Add(context.Background(), 1)

	rm := &metricdata.ResourceMetrics{}
	if err := reader.Collect(context.Background(), rm); err != nil {
		t.Fatalf("collect: %v", err)
	}

	if len(rm.ScopeMetrics) == 0 {
		t.Fatal("expected scope metrics")
	}

	if err := mp.Shutdown(context.Background()); err != nil {
		t.Fatalf("shutdown: %v", err)
	}
}
