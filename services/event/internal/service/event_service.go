package service

import (
	"context"
	"errors"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"

	"github.com/kou-etal/etalbaas/pkg/apperror"
	"github.com/kou-etal/etalbaas/services/event/internal/store"
)

// --- JSONB Go types (read path only) ---

type DatabaseChangeData struct {
	Table string `json:"table"`
	Event string `json:"event"`
}

type ObjectStorageData struct {
	Bucket    string `json:"bucket"`
	ObjectKey string `json:"object_key"`
	Event     string `json:"event"`
}

// --- Service ---

type EventService struct {
	q store.Querier
}

func NewEventService(q store.Querier) *EventService {
	return &EventService{q: q}
}

// --- ListEventHistory ---

type ListEventHistoryParams struct {
	TenantID   uuid.UUID
	ProjectID  string
	FunctionID *uuid.UUID

	StatusFilter *string
	Since        *time.Time
	Until        *time.Time

	Limit           int32
	CursorCreatedAt *time.Time
	CursorID        *uuid.UUID
}

func (s *EventService) ListEventHistory(ctx context.Context, p ListEventHistoryParams) ([]store.EventHistory, error) {
	if err := s.verifyProjectOwnership(ctx, p.ProjectID, p.TenantID); err != nil {
		return nil, err
	}

	if p.Limit <= 0 {
		p.Limit = 21
	}
	if p.Limit > 101 {
		p.Limit = 101
	}

	var since, until pgtype.Timestamptz
	if p.Since != nil {
		since = pgtype.Timestamptz{Time: *p.Since, Valid: true}
	}
	if p.Until != nil {
		until = pgtype.Timestamptz{Time: *p.Until, Valid: true}
	}

	var cursorTS pgtype.Timestamptz
	if p.CursorCreatedAt != nil {
		cursorTS = pgtype.Timestamptz{Time: *p.CursorCreatedAt, Valid: true}
	}
	var cursorUUID pgtype.UUID
	if p.CursorID != nil {
		cursorUUID = pgtype.UUID{Bytes: *p.CursorID, Valid: true}
	}

	var (
		rows []store.EventHistory
		err  error
	)
	if p.FunctionID != nil {
		rows, err = s.q.ListEventHistoryByFunctionID(ctx, store.ListEventHistoryByFunctionIDParams{
			FunctionID:      *p.FunctionID,
			ProjectID:       p.ProjectID,
			StatusFilter:    p.StatusFilter,
			Since:           since,
			Until:           until,
			CursorCreatedAt: cursorTS,
			CursorID:        cursorUUID,
			PageSize:        p.Limit,
		})
	} else {
		rows, err = s.q.ListEventHistoryByProjectID(ctx, store.ListEventHistoryByProjectIDParams{
			ProjectID:       p.ProjectID,
			StatusFilter:    p.StatusFilter,
			Since:           since,
			Until:           until,
			CursorCreatedAt: cursorTS,
			CursorID:        cursorUUID,
			PageSize:        p.Limit,
		})
	}
	if err != nil {
		return nil, wrapDBError(err, "list event history")
	}
	return rows, nil
}

// --- Helpers ---

func (s *EventService) verifyProjectOwnership(ctx context.Context, projectID string, tenantID uuid.UUID) error {
	if _, err := s.q.GetProjectByIDAndTenantID(ctx, store.GetProjectByIDAndTenantIDParams{
		ID:       projectID,
		TenantID: tenantID,
	}); errors.Is(err, pgx.ErrNoRows) {
		return apperror.New(apperror.CodeNotFound, "project not found")
	} else if err != nil {
		return wrapDBError(err, "verify project ownership")
	}
	return nil
}

func wrapDBError(err error, msg string) *apperror.AppError {
	if errors.Is(err, context.Canceled) || errors.Is(err, context.DeadlineExceeded) {
		return apperror.Wrap(apperror.CodeCanceled, msg, err)
	}
	return apperror.Wrap(apperror.CodeInternal, msg, err)
}
