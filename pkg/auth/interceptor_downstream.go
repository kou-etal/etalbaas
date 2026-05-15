package auth

import (
	"context"

	"connectrpc.com/connect"

	"github.com/kou-etal/etalbaas/pkg/requestctx"
)

func NewDownstreamInterceptor() connect.UnaryInterceptorFunc {
	return func(next connect.UnaryFunc) connect.UnaryFunc {
		return func(ctx context.Context, req connect.AnyRequest) (connect.AnyResponse, error) {
			userID := req.Header().Get(requestctx.HeaderUserID)
			if userID == "" {
				return nil, connect.NewError(connect.CodeUnauthenticated, nil)
			}
			ctx = requestctx.WithUserID(ctx, userID)

			projectID := req.Header().Get(requestctx.HeaderProjectID)
			if projectID != "" {
				ctx = requestctx.WithProjectID(ctx, projectID)
			}

			return next(ctx, req)
		}
	}
}
