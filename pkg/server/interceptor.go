package server

import (
	"log/slog"

	"connectrpc.com/connect"

	"github.com/kou-etal/etalbaas/pkg/auth"
	"github.com/kou-etal/etalbaas/pkg/middleware"
	"github.com/kou-etal/etalbaas/pkg/observability"
)

func DefaultInterceptors(logger *slog.Logger) connect.Option {
	otelInt, err := observability.ConnectInterceptor()
	if err != nil {
		logger.Warn("failed to create otel interceptor", "error", err)
	}
	return connect.WithInterceptors(
		middleware.NewRecoveryInterceptor(),
		middleware.NewRequestIDInterceptor(),
		middleware.NewLoggingInterceptor(logger),
		auth.NewDownstreamInterceptor(),
		otelInt,
	)
}
