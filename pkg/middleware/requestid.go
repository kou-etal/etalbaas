package middleware

import (
	"context"

	"connectrpc.com/connect"
	"github.com/google/uuid"
)

const HeaderRequestID = "X-Request-Id"

type requestIDKey struct{}

func NewRequestIDInterceptor() connect.UnaryInterceptorFunc {
	return func(next connect.UnaryFunc) connect.UnaryFunc {
		return func(ctx context.Context, req connect.AnyRequest) (connect.AnyResponse, error) {
			requestID := req.Header().Get(HeaderRequestID)
			if requestID == "" {
				requestID = uuid.NewString()
			}

			ctx = context.WithValue(ctx, requestIDKey{}, requestID)

			resp, err := next(ctx, req)
			if err != nil {
				return nil, err
			}

			resp.Header().Set(HeaderRequestID, requestID)
			return resp, nil
		}
	}
}

func RequestID(ctx context.Context) string {
	v, _ := ctx.Value(requestIDKey{}).(string)
	return v
}
