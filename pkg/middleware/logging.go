package middleware

import (
	"context"
	"errors"
	"log/slog"
	"time"

	"connectrpc.com/connect"
)

func NewLoggingInterceptor(logger *slog.Logger) connect.UnaryInterceptorFunc {
	return func(next connect.UnaryFunc) connect.UnaryFunc {
		return func(ctx context.Context, req connect.AnyRequest) (connect.AnyResponse, error) {
			start := time.Now()
			procedure := req.Spec().Procedure

			resp, err := next(ctx, req)

			duration := time.Since(start)
			attrs := []slog.Attr{
				slog.String("procedure", procedure),
				slog.Duration("duration", duration),
			}

			if err != nil {
				attrs = append(attrs, slog.String("error", err.Error()))
				var connectErr *connect.Error
				if errors.As(err, &connectErr) {
					attrs = append(attrs, slog.String("code", connectErr.Code().String()))
				}
				logger.LogAttrs(ctx, slog.LevelError, "rpc failed", attrs...)
			} else {
				logger.LogAttrs(ctx, slog.LevelInfo, "rpc completed", attrs...)
			}

			return resp, err
		}
	}
}
