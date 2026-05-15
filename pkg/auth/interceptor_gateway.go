package auth

import (
	"context"
	"strings"

	"connectrpc.com/connect"

	"github.com/kou-etal/etalbaas/pkg/requestctx"
)

func NewGatewayInterceptor(signingKey []byte) connect.UnaryInterceptorFunc {
	return func(next connect.UnaryFunc) connect.UnaryFunc {
		return func(ctx context.Context, req connect.AnyRequest) (connect.AnyResponse, error) {
			if isPublicProcedure(req.Spec().Procedure) {
				return next(ctx, req)
			}

			authHeader := req.Header().Get("Authorization")
			if authHeader == "" {
				return nil, connect.NewError(connect.CodeUnauthenticated, nil)
			}

			tokenStr := strings.TrimPrefix(authHeader, "Bearer ")
			if tokenStr == authHeader {
				return nil, connect.NewError(connect.CodeUnauthenticated, nil)
			}

			claims, err := VerifyToken(tokenStr, signingKey)
			if err != nil {
				return nil, connect.NewError(connect.CodeUnauthenticated, err)
			}

			ctx = requestctx.WithUserID(ctx, claims.UserID)

			resp, err := next(ctx, req)
			if err != nil {
				return nil, err
			}

			// 下流 MS 向けにヘッダ付加
			resp.Header().Set(requestctx.HeaderUserID, claims.UserID)

			return resp, nil
		}
	}
}

var publicProcedures = map[string]bool{}

// RegisterPublicProcedure registers a procedure that does not require authentication.
// Must be called during init before the server starts handling requests.
func RegisterPublicProcedure(procedure string) {
	publicProcedures[procedure] = true
}

func isPublicProcedure(procedure string) bool {
	return publicProcedures[procedure]
}
