package metrics

import (
	"fmt"
	"net/http"

	promclient "github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"go.opentelemetry.io/otel/exporters/prometheus"
	sdkmetric "go.opentelemetry.io/otel/sdk/metric"
)

func NewPrometheusExporter() (sdkmetric.Reader, http.Handler, error) {
	// Use an explicit registry so the exporter and handler share the same
	// Prometheus collector registry — avoids subtle default-registry mismatches.
	registry := promclient.NewRegistry()
	// Also collect Go runtime and process metrics.
	registry.MustRegister(promclient.NewGoCollector())
	registry.MustRegister(promclient.NewProcessCollector(promclient.ProcessCollectorOpts{}))

	exporter, err := prometheus.New(prometheus.WithRegisterer(registry))
	if err != nil {
		return nil, nil, fmt.Errorf("create prometheus exporter: %w", err)
	}
	handler := promhttp.HandlerFor(registry, promhttp.HandlerOpts{})
	return exporter, handler, nil
}
