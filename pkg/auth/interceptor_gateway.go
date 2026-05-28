package auth

import (
	"context"
	"crypto/rsa"
	"errors"
	"strings"

	"connectrpc.com/connect"
	"github.com/google/uuid"

	"github.com/kou-etal/etalbaas/pkg/requestctx"
)

// NewGatewayInterceptor creates a connect-rpc interceptor that verifies RS256 JWTs.
// Only RS256 is accepted to prevent Key Confusion Attack (CVE-2016-5431).
func NewGatewayInterceptor(pubKey *rsa.PublicKey) connect.UnaryInterceptorFunc {
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

			claims, err := VerifyToken(tokenStr, pubKey)
			if err != nil {
				return nil, connect.NewError(connect.CodeUnauthenticated, err)
			}

			userID, err := uuid.Parse(claims.UserID)
			if err != nil {
				return nil, connect.NewError(connect.CodeUnauthenticated, errors.New("invalid user id format in token"))
			}

			ctx = requestctx.WithUserID(ctx, userID)

			resp, err := next(ctx, req)
			if err != nil {
				return nil, err
			}

			resp.Header().Set(requestctx.HeaderUserID, userID.String())

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
