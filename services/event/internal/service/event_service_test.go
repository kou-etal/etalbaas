package service

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"

	"github.com/kou-etal/etalbaas/pkg/apperror"
	"github.com/kou-etal/etalbaas/services/event/internal/store"
)

// --- Mock Querier ---

type mockQuerier struct {
	getProjectByIDAndTenantIDFn    func(ctx context.Context, arg store.GetProjectByIDAndTenantIDParams) (store.Project, error)
	listEventHistoryByFunctionIDFn func(ctx context.Context, arg store.ListEventHistoryByFunctionIDParams) ([]store.EventHistory, error)
	getEventByIDFn                 func(ctx context.Context, arg store.GetEventByIDParams) (store.EventHistory, error)
}

func (m *mockQuerier) GetProjectByIDAndTenantID(ctx context.Context, arg store.GetProjectByIDAndTenantIDParams) (store.Project, error) {
	if m.getProjectByIDAndTenantIDFn != nil {
		return m.getProjectByIDAndTenantIDFn(ctx, arg)
	}
	return store.Project{}, pgx.ErrNoRows
}

func (m *mockQuerier) ListEventHistoryByFunctionID(ctx context.Context, arg store.ListEventHistoryByFunctionIDParams) ([]store.EventHistory, error) {
	if m.listEventHistoryByFunctionIDFn != nil {
		return m.listEventHistoryByFunctionIDFn(ctx, arg)
	}
	return nil, nil
}

func (m *mockQuerier) ListEventHistoryByProjectID(ctx context.Context, arg store.ListEventHistoryByProjectIDParams) ([]store.EventHistory, error) {
	return nil, nil
}

func (m *mockQuerier) GetEventByID(ctx context.Context, arg store.GetEventByIDParams) (store.EventHistory, error) {
	if m.getEventByIDFn != nil {
		return m.getEventByIDFn(ctx, arg)
	}
	return store.EventHistory{}, pgx.ErrNoRows
}

// --- Helpers ---

func projectOwnershipOK() func(context.Context, store.GetProjectByIDAndTenantIDParams) (store.Project, error) {
	return func(_ context.Context, _ store.GetProjectByIDAndTenantIDParams) (store.Project, error) {
		return store.Project{}, nil
	}
}

func newTestEventHistory() store.EventHistory {
	return store.EventHistory{
		ID:           uuid.MustParse("aaaaaaaa-aaaa-aaaa-aaaa-aaaaaaaaaaaa"),
		ProjectID:    "proj-123",
		FunctionID:   uuid.MustParse("bbbbbbbb-bbbb-bbbb-bbbb-bbbbbbbbbbbb"),
		InvocationID: pgtype.UUID{Bytes: uuid.MustParse("cccccccc-cccc-cccc-cccc-cccccccccccc"), Valid: true},
		TriggerType:  "database_change",
		TriggerData:  []byte(`{"table":"orders","event":"INSERT"}`),
		Status:       "delivered",
		AttemptCount: 1,
		CreatedAt:    time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC),
	}
}

// --- Tests ---

func TestListEventHistory_ProjectNotFound(t *testing.T) {
	svc := NewEventService(&mockQuerier{})

	fnID := uuid.New()
	_, err := svc.ListEventHistory(context.Background(), ListEventHistoryParams{
		TenantID:   uuid.New(),
		ProjectID:  "proj-123",
		FunctionID: &fnID,
		Limit:      21,
	})
	if err == nil {
		t.Fatal("expected error")
	}
	var appErr *apperror.AppError
	if !errors.As(err, &appErr) || appErr.Code != apperror.CodeNotFound {
		t.Fatalf("expected CodeNotFound, got %v", err)
	}
}

func TestListEventHistory_Success(t *testing.T) {
	ev := newTestEventHistory()
	svc := NewEventService(&mockQuerier{
		getProjectByIDAndTenantIDFn: projectOwnershipOK(),
		listEventHistoryByFunctionIDFn: func(_ context.Context, _ store.ListEventHistoryByFunctionIDParams) ([]store.EventHistory, error) {
			return []store.EventHistory{ev}, nil
		},
	})

	fnID := uuid.MustParse("bbbbbbbb-bbbb-bbbb-bbbb-bbbbbbbbbbbb")
	rows, err := svc.ListEventHistory(context.Background(), ListEventHistoryParams{
		TenantID:   uuid.New(),
		ProjectID:  "proj-123",
		FunctionID: &fnID,
		Limit:      21,
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(rows) != 1 {
		t.Fatalf("expected 1 row, got %d", len(rows))
	}
	if rows[0].ID != ev.ID {
		t.Fatalf("expected ID %s, got %s", ev.ID, rows[0].ID)
	}
}

func TestListEventHistory_LimitClamping(t *testing.T) {
	tests := []struct {
		name     string
		input    int32
		expected int32
	}{
		{"zero default", 0, 21},
		{"negative default", -1, 21},
		{"over max", 200, 101},
		{"valid", 50, 50},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var captured int32
			svc := NewEventService(&mockQuerier{
				getProjectByIDAndTenantIDFn: projectOwnershipOK(),
				listEventHistoryByFunctionIDFn: func(_ context.Context, arg store.ListEventHistoryByFunctionIDParams) ([]store.EventHistory, error) {
					captured = arg.PageSize
					return nil, nil
				},
			})

			fnID := uuid.New()
			_, err := svc.ListEventHistory(context.Background(), ListEventHistoryParams{
				TenantID:   uuid.New(),
				ProjectID:  "proj-123",
				FunctionID: &fnID,
				Limit:      tt.input,
			})
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if captured != tt.expected {
				t.Fatalf("expected limit %d, got %d", tt.expected, captured)
			}
		})
	}
}

func TestListEventHistory_EmptyResult(t *testing.T) {
	svc := NewEventService(&mockQuerier{
		getProjectByIDAndTenantIDFn: projectOwnershipOK(),
		listEventHistoryByFunctionIDFn: func(_ context.Context, _ store.ListEventHistoryByFunctionIDParams) ([]store.EventHistory, error) {
			return nil, nil
		},
	})

	fnID := uuid.New()
	rows, err := svc.ListEventHistory(context.Background(), ListEventHistoryParams{
		TenantID:   uuid.New(),
		ProjectID:  "proj-123",
		FunctionID: &fnID,
		Limit:      21,
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if rows != nil {
		t.Fatalf("expected nil rows, got %d", len(rows))
	}
}
