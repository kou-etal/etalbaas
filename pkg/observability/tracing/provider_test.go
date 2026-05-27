package tracing_test

import (
	"context"
	"testing"

	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	"go.opentelemetry.io/otel/sdk/trace/tracetest"

	"github.com/kou-etal/etalbaas/pkg/observability"
	"github.com/kou-etal/etalbaas/pkg/observability/tracing"
)

func TestNewProvider(t *testing.T) {
	res, err := observability.NewResource(context.Background(), "test-service", "0.1.0", "test")
	if err != nil {
		t.Fatalf("create resource: %v", err)
	}

	exporter := tracetest.NewInMemoryExporter()
	tp, err := tracing.NewProvider(res, exporter)
	if err != nil {
		t.Fatalf("create provider: %v", err)
	}

	tracer := tp.Tracer("test")
	_, span := tracer.Start(context.Background(), "test-span")
	span.End()

	if err := tp.ForceFlush(context.Background()); err != nil {
		t.Fatalf("force flush: %v", err)
	}

	spans := exporter.GetSpans()
	if len(spans) != 1 {
		t.Fatalf("got %d spans, want 1", len(spans))
	}
	if spans[0].Name != "test-span" {
		t.Fatalf("got span name %q, want %q", spans[0].Name, "test-span")
	}

	if err := tp.Shutdown(context.Background()); err != nil {
		t.Fatalf("shutdown: %v", err)
	}
}

func TestNewProvider_WithBatchProcessor(t *testing.T) {
	res, err := observability.NewResource(context.Background(), "svc", "1.0.0", "development")
	if err != nil {
		t.Fatalf("create resource: %v", err)
	}

	exporter := tracetest.NewInMemoryExporter()
	tp, err := tracing.NewProvider(res, exporter)
	if err != nil {
		t.Fatalf("create provider: %v", err)
	}

	// Verify type
	var _ *sdktrace.TracerProvider = tp

	if err := tp.Shutdown(context.Background()); err != nil {
		t.Fatalf("shutdown: %v", err)
	}
}
