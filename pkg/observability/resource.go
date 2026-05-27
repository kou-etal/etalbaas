package observability

import (
	"context"
	"fmt"

	"go.opentelemetry.io/otel/sdk/resource"
	semconv "go.opentelemetry.io/otel/semconv/v1.26.0"
)

func NewResource(ctx context.Context, serviceName, version, environment string) (*resource.Resource, error) {
	res, err := resource.New(ctx,
		resource.WithAttributes(
			semconv.ServiceName(serviceName),
			semconv.ServiceVersion(version),
			semconv.DeploymentEnvironment(environment),
		),
		resource.WithTelemetrySDK(),
		resource.WithProcessRuntimeDescription(),
	)
	if err != nil {
		return nil, fmt.Errorf("create otel resource: %w", err)
	}
	return res, nil
}
