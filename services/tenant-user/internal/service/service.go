package service

import (
	"context"
	"errors"
	"log/slog"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"

	"github.com/kou-etal/etalbaas/pkg/apperror"
	"github.com/kou-etal/etalbaas/services/tenant-user/internal/store"
)

const (
	maxDisplayNameLen = 100
	maxAvatarURLLen   = 2048
)

type Service struct {
	q store.Querier
}

func New(q store.Querier) *Service {
	return &Service{q: q}
}

func (s *Service) GetMe(ctx context.Context, userID uuid.UUID) (store.Tenant, error) {
	row, err := s.q.GetTenantByID(ctx, userID)
	if errors.Is(err, pgx.ErrNoRows) {
		return store.Tenant{}, apperror.New(apperror.CodeNotFound, "tenant not found")
	}
	if err != nil {
		return store.Tenant{}, wrapDBError(err, "get tenant")
	}
	return row, nil
}

func (s *Service) UpdateProfile(ctx context.Context, userID uuid.UUID, displayName, avatarURL *string) (store.Tenant, error) {
	if displayName == nil && avatarURL == nil {
		return store.Tenant{}, apperror.New(apperror.CodeInvalidArgument, "no fields to update")
	}
	if displayName != nil {
		if len(*displayName) == 0 {
			return store.Tenant{}, apperror.New(apperror.CodeInvalidArgument, "display_name must not be empty")
		}
		if len(*displayName) > maxDisplayNameLen {
			return store.Tenant{}, apperror.New(apperror.CodeInvalidArgument, "display_name exceeds maximum length")
		}
	}
	if avatarURL != nil && len(*avatarURL) > maxAvatarURLLen {
		return store.Tenant{}, apperror.New(apperror.CodeInvalidArgument, "avatar_url exceeds maximum length")
	}
	row, err := s.q.UpdateTenantProfile(ctx, store.UpdateTenantProfileParams{
		ID:          userID,
		DisplayName: displayName,
		AvatarUrl:   avatarURL,
	})
	if errors.Is(err, pgx.ErrNoRows) {
		return store.Tenant{}, apperror.New(apperror.CodeNotFound, "tenant not found")
	}
	if err != nil {
		return store.Tenant{}, wrapDBError(err, "update profile")
	}
	return row, nil
}

func (s *Service) ListTenants(ctx context.Context, limit int32, cursorCreatedAt *time.Time, cursorID *uuid.UUID) ([]store.Tenant, error) {
	if limit <= 0 {
		limit = 21
	}
	if limit > 101 {
		limit = 101
	}
	var cursorTS pgtype.Timestamptz
	if cursorCreatedAt != nil {
		cursorTS = pgtype.Timestamptz{Time: *cursorCreatedAt, Valid: true}
	}
	var cursorUUID pgtype.UUID
	if cursorID != nil {
		cursorUUID = pgtype.UUID{Bytes: *cursorID, Valid: true}
	}
	rows, err := s.q.ListTenants(ctx, store.ListTenantsParams{
		CursorCreatedAt: cursorTS,
		CursorID:        cursorUUID,
		PageSize:        limit,
	})
	if err != nil {
		return nil, wrapDBError(err, "list tenants")
	}
	return rows, nil
}

// wrapDBError wraps a database error, preserving context cancellation semantics.
func wrapDBError(err error, msg string) *apperror.AppError {
	slog.Error("database error", "msg", msg, "error", err)
	if errors.Is(err, context.Canceled) || errors.Is(err, context.DeadlineExceeded) {
		return apperror.Wrap(apperror.CodeCanceled, msg, err)
	}
	return apperror.Wrap(apperror.CodeInternal, msg, err)
}
