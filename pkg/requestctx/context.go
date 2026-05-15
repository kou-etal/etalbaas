package requestctx

import (
	"context"
	"errors"
)

const (
	HeaderUserID    = "X-User-ID"
	HeaderProjectID = "X-Project-ID"
)

type userIDKey struct{}
type projectIDKey struct{}

var (
	ErrUserIDNotFound    = errors.New("user_id not found in context")
	ErrProjectIDNotFound = errors.New("project_id not found in context")
)

func WithUserID(ctx context.Context, id string) context.Context {
	return context.WithValue(ctx, userIDKey{}, id)
}

func UserID(ctx context.Context) (string, error) {
	v, ok := ctx.Value(userIDKey{}).(string)
	if !ok || v == "" {
		return "", ErrUserIDNotFound
	}
	return v, nil
}

func WithProjectID(ctx context.Context, id string) context.Context {
	return context.WithValue(ctx, projectIDKey{}, id)
}

func ProjectID(ctx context.Context) (string, error) {
	v, ok := ctx.Value(projectIDKey{}).(string)
	if !ok || v == "" {
		return "", ErrProjectIDNotFound
	}
	return v, nil
}
