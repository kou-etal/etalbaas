package requestctx

import (
	"context"
	"errors"

	"github.com/google/uuid"
)

const (
	HeaderUserID    = "X-User-ID"
	HeaderProjectID = "X-Project-ID"
	HeaderRole      = "X-API-Key-Role"
)

type userIDKey struct{}
type projectIDKey struct{}
type roleKey struct{}

var (
	ErrUserIDNotFound    = errors.New("user_id not found in context")
	ErrProjectIDNotFound = errors.New("project_id not found in context")
	ErrRoleNotFound      = errors.New("role not found in context")
)

func WithUserID(ctx context.Context, id uuid.UUID) context.Context {
	return context.WithValue(ctx, userIDKey{}, id)
}

func UserID(ctx context.Context) (uuid.UUID, error) {
	v, ok := ctx.Value(userIDKey{}).(uuid.UUID)
	if !ok || v == uuid.Nil {
		return uuid.Nil, ErrUserIDNotFound
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

func WithRole(ctx context.Context, role string) context.Context {
	return context.WithValue(ctx, roleKey{}, role)
}

func Role(ctx context.Context) (string, error) {
	v, ok := ctx.Value(roleKey{}).(string)
	if !ok || v == "" {
		return "", ErrRoleNotFound
	}
	return v, nil
}
