package observability

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"net/http"

	"go.opentelemetry.io/otel"

	"github.com/kou-etal/etalbaas/pkg/observability/metrics"
	"github.com/kou-etal/etalbaas/pkg/observability/tracing"
)

// Result holds the observability resources returned by Init.
type Result struct {
	Shutdown       func(context.Context) error
	MetricsHandler http.Handler
}

type Config struct {
	ServiceName    string
	ServiceVersion string
	OTELEndpoint   string
	Environment    string
}

func Init(ctx context.Context, cfg Config) (*Result, error) {
	logger := NewLogger(cfg.ServiceName, cfg.Environment)
	slog.SetDefault(logger)

	res, err := NewResource(ctx, cfg.ServiceName, cfg.ServiceVersion, cfg.Environment)
	if err != nil {
		return nil, fmt.Errorf("create resource: %w", err)
	}

	var shutdownFuncs []func(context.Context) error

	// Tracing
	if cfg.OTELEndpoint != "" {
		exp, err := tracing.NewOTLPExporter(ctx, cfg.OTELEndpoint)
		if err != nil {
			return nil, fmt.Errorf("create trace exporter: %w", err)
		}
		tp, err := tracing.NewProvider(res, exp)
		if err != nil {
			return nil, fmt.Errorf("create trace provider: %w", err)
		}
		otel.SetTracerProvider(tp)
		shutdownFuncs = append(shutdownFuncs, tp.Shutdown)
	}

	// Metrics
	reader, metricsHandler, err := metrics.NewPrometheusExporter()
	if err != nil {
		return nil, fmt.Errorf("create metrics exporter: %w", err)
	}
	mp, err := metrics.NewProvider(res, reader)
	if err != nil {
		return nil, fmt.Errorf("create meter provider: %w", err)
	}
	otel.SetMeterProvider(mp)
	shutdownFuncs = append(shutdownFuncs, mp.Shutdown)

	shutdown := func(ctx context.Context) error {
		var errs []error
		for _, fn := range shutdownFuncs {
			if err := fn(ctx); err != nil {
				errs = append(errs, err)
			}
		}
		return errors.Join(errs...)
	}

	slog.Info("observability initialized",
		slog.String("service", cfg.ServiceName),
		slog.String("environment", cfg.Environment),
		slog.Bool("tracing", cfg.OTELEndpoint != ""),
	)

	return &Result{
		Shutdown:       shutdown,
		MetricsHandler: metricsHandler,
	}, nil
}
