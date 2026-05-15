package auth

import (
	"context"
	"errors"

	"connectrpc.com/connect"
	"github.com/google/uuid"

	"github.com/kou-etal/etalbaas/pkg/requestctx"
)

func NewDownstreamInterceptor() connect.UnaryInterceptorFunc {
	return func(next connect.UnaryFunc) connect.UnaryFunc {
		return func(ctx context.Context, req connect.AnyRequest) (connect.AnyResponse, error) {
			userIDStr := req.Header().Get(requestctx.HeaderUserID)
			if userIDStr == "" {
				return nil, connect.NewError(connect.CodeUnauthenticated, errors.New("missing user id"))
			}
			userID, err := uuid.Parse(userIDStr)
			if err != nil {
				return nil, connect.NewError(connect.CodeUnauthenticated, errors.New("invalid user id format"))
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
