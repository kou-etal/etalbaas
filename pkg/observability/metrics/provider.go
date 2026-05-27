package metrics

import (
	sdkmetric "go.opentelemetry.io/otel/sdk/metric"
	"go.opentelemetry.io/otel/sdk/resource"
)

func NewProvider(res *resource.Resource, reader sdkmetric.Reader) (*sdkmetric.MeterProvider, error) {
	mp := sdkmetric.NewMeterProvider(
		sdkmetric.WithReader(reader),
		sdkmetric.WithResource(res),
	)
	return mp, nil
}
