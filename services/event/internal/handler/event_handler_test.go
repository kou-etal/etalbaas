package handler_test

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"connectrpc.com/connect"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"

	"github.com/kou-etal/etalbaas/pkg/requestctx"
	eventv1 "github.com/kou-etal/etalbaas/proto/gen/go/etalbaas/event/v1"
	"github.com/kou-etal/etalbaas/proto/gen/go/etalbaas/event/v1/eventv1connect"
	"github.com/kou-etal/etalbaas/services/event/internal/handler"
	svcpkg "github.com/kou-etal/etalbaas/services/event/internal/service"
	"github.com/kou-etal/etalbaas/services/event/internal/store"
)

var testUserID = uuid.MustParse("550e8400-e29b-41d4-a716-446655440000")

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

func (m *mockQuerier) GetEventByID(ctx context.Context, arg store.GetEventByIDParams) (store.EventHistory, error) {
	if m.getEventByIDFn != nil {
		return m.getEventByIDFn(ctx, arg)
	}
	return store.EventHistory{}, pgx.ErrNoRows
}

// --- Helpers ---

func injectUserID() connect.UnaryInterceptorFunc {
	return func(next connect.UnaryFunc) connect.UnaryFunc {
		return func(ctx context.Context, req connect.AnyRequest) (connect.AnyResponse, error) {
			userIDStr := req.Header().Get(requestctx.HeaderUserID)
			if userIDStr != "" {
				uid, err := uuid.Parse(userIDStr)
				if err == nil {
					ctx = requestctx.WithUserID(ctx, uid)
				}
			}
			return next(ctx, req)
		}
	}
}

func setupTestServer(t *testing.T, q store.Querier) (eventv1connect.EventServiceClient, func()) {
	t.Helper()
	svc := svcpkg.NewEventService(q)
	h := handler.NewEventHandler(svc)
	path, hnd := eventv1connect.NewEventServiceHandler(h,
		connect.WithInterceptors(injectUserID()),
	)
	mux := http.NewServeMux()
	mux.Handle(path, hnd)
	srv := httptest.NewServer(mux)
	client := eventv1connect.NewEventServiceClient(http.DefaultClient, srv.URL)
	return client, srv.Close
}

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

func makeRequest(projectID, functionID string) *connect.Request[eventv1.ListEventHistoryRequest] {
	req := connect.NewRequest(&eventv1.ListEventHistoryRequest{
		ProjectId:  projectID,
		FunctionId: functionID,
	})
	req.Header().Set(requestctx.HeaderUserID, testUserID.String())
	return req
}

// --- Tests ---

func TestListEventHistory_Success(t *testing.T) {
	ev := newTestEventHistory()
	client, close := setupTestServer(t, &mockQuerier{
		getProjectByIDAndTenantIDFn: projectOwnershipOK(),
		listEventHistoryByFunctionIDFn: func(_ context.Context, _ store.ListEventHistoryByFunctionIDParams) ([]store.EventHistory, error) {
			return []store.EventHistory{ev}, nil
		},
	})
	defer close()

	resp, err := client.ListEventHistory(context.Background(), makeRequest("proj-123", "bbbbbbbb-bbbb-bbbb-bbbb-bbbbbbbbbbbb"))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(resp.Msg.Events) != 1 {
		t.Fatalf("expected 1 event, got %d", len(resp.Msg.Events))
	}

	rec := resp.Msg.Events[0]
	if rec.Id != "aaaaaaaa-aaaa-aaaa-aaaa-aaaaaaaaaaaa" {
		t.Fatalf("expected event ID aaaaaaaa-..., got %s", rec.Id)
	}
	if rec.Status != "delivered" {
		t.Fatalf("expected status delivered, got %s", rec.Status)
	}
	if rec.InvocationId != "cccccccc-cccc-cccc-cccc-cccccccccccc" {
		t.Fatalf("expected invocation_id cccccccc-..., got %s", rec.InvocationId)
	}
	if rec.Trigger == nil {
		t.Fatal("expected trigger to be set")
	}
	dc := rec.Trigger.GetDatabaseChange()
	if dc == nil {
		t.Fatal("expected database_change trigger")
	}
	if dc.Table != "orders" || dc.Event != "INSERT" {
		t.Fatalf("unexpected trigger data: %+v", dc)
	}
}

func TestListEventHistory_ProjectNotFound(t *testing.T) {
	client, close := setupTestServer(t, &mockQuerier{})
	defer close()

	_, err := client.ListEventHistory(context.Background(), makeRequest("proj-123", "bbbbbbbb-bbbb-bbbb-bbbb-bbbbbbbbbbbb"))
	if err == nil {
		t.Fatal("expected error")
	}
	if connect.CodeOf(err) != connect.CodeNotFound {
		t.Fatalf("expected CodeNotFound, got %v", connect.CodeOf(err))
	}
}

func TestListEventHistory_InvalidFunctionID(t *testing.T) {
	client, close := setupTestServer(t, &mockQuerier{
		getProjectByIDAndTenantIDFn: projectOwnershipOK(),
	})
	defer close()

	_, err := client.ListEventHistory(context.Background(), makeRequest("proj-123", "not-a-uuid"))
	if err == nil {
		t.Fatal("expected error")
	}
	if connect.CodeOf(err) != connect.CodeInvalidArgument {
		t.Fatalf("expected CodeInvalidArgument, got %v", connect.CodeOf(err))
	}
}

func TestListEventHistory_WithStatusFilter(t *testing.T) {
	var capturedFilter *string
	client, close := setupTestServer(t, &mockQuerier{
		getProjectByIDAndTenantIDFn: projectOwnershipOK(),
		listEventHistoryByFunctionIDFn: func(_ context.Context, arg store.ListEventHistoryByFunctionIDParams) ([]store.EventHistory, error) {
			capturedFilter = arg.StatusFilter
			return nil, nil
		},
	})
	defer close()

	filter := "failed"
	req := connect.NewRequest(&eventv1.ListEventHistoryRequest{
		ProjectId:    "proj-123",
		FunctionId:   "bbbbbbbb-bbbb-bbbb-bbbb-bbbbbbbbbbbb",
		StatusFilter: &filter,
	})
	req.Header().Set(requestctx.HeaderUserID, testUserID.String())

	_, err := client.ListEventHistory(context.Background(), req)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if capturedFilter == nil || *capturedFilter != "failed" {
		t.Fatalf("expected status_filter 'failed', got %v", capturedFilter)
	}
}

func TestListEventHistory_Pagination(t *testing.T) {
	// Return 21 rows to trigger hasMore (default pageSize=20, query with pageSize+1=21)
	rows := make([]store.EventHistory, 21)
	for i := range rows {
		rows[i] = store.EventHistory{
			ID:          uuid.New(),
			ProjectID:   "proj-123",
			FunctionID:  uuid.MustParse("bbbbbbbb-bbbb-bbbb-bbbb-bbbbbbbbbbbb"),
			TriggerType: "database_change",
			TriggerData: []byte(`{"table":"t","event":"INSERT"}`),
			Status:      "delivered",
			CreatedAt:   time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC).Add(-time.Duration(i) * time.Minute),
		}
	}

	client, close := setupTestServer(t, &mockQuerier{
		getProjectByIDAndTenantIDFn: projectOwnershipOK(),
		listEventHistoryByFunctionIDFn: func(_ context.Context, _ store.ListEventHistoryByFunctionIDParams) ([]store.EventHistory, error) {
			return rows, nil
		},
	})
	defer close()

	resp, err := client.ListEventHistory(context.Background(), makeRequest("proj-123", "bbbbbbbb-bbbb-bbbb-bbbb-bbbbbbbbbbbb"))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(resp.Msg.Events) != 20 {
		t.Fatalf("expected 20 events, got %d", len(resp.Msg.Events))
	}
	if resp.Msg.Pagination == nil || resp.Msg.Pagination.NextPageToken == "" {
		t.Fatal("expected pagination token")
	}
}

func TestListEventHistory_ObjectStorageTrigger(t *testing.T) {
	ev := store.EventHistory{
		ID:          uuid.MustParse("aaaaaaaa-aaaa-aaaa-aaaa-aaaaaaaaaaaa"),
		ProjectID:   "proj-123",
		FunctionID:  uuid.MustParse("bbbbbbbb-bbbb-bbbb-bbbb-bbbbbbbbbbbb"),
		TriggerType: "object_storage",
		TriggerData: []byte(`{"bucket":"uploads","object_key":"img/photo.png","event":"ObjectCreated"}`),
		Status:      "received",
		CreatedAt:   time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC),
	}

	client, close := setupTestServer(t, &mockQuerier{
		getProjectByIDAndTenantIDFn: projectOwnershipOK(),
		listEventHistoryByFunctionIDFn: func(_ context.Context, _ store.ListEventHistoryByFunctionIDParams) ([]store.EventHistory, error) {
			return []store.EventHistory{ev}, nil
		},
	})
	defer close()

	resp, err := client.ListEventHistory(context.Background(), makeRequest("proj-123", "bbbbbbbb-bbbb-bbbb-bbbb-bbbbbbbbbbbb"))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(resp.Msg.Events) != 1 {
		t.Fatalf("expected 1 event, got %d", len(resp.Msg.Events))
	}

	os := resp.Msg.Events[0].Trigger.GetObjectStorage()
	if os == nil {
		t.Fatal("expected object_storage trigger")
	}
	if os.Bucket != "uploads" || os.ObjectKey != "img/photo.png" || os.Event != "ObjectCreated" {
		t.Fatalf("unexpected trigger data: %+v", os)
	}
}

func TestListEventHistory_NullInvocationID(t *testing.T) {
	ev := store.EventHistory{
		ID:           uuid.MustParse("aaaaaaaa-aaaa-aaaa-aaaa-aaaaaaaaaaaa"),
		ProjectID:    "proj-123",
		FunctionID:   uuid.MustParse("bbbbbbbb-bbbb-bbbb-bbbb-bbbbbbbbbbbb"),
		InvocationID: pgtype.UUID{Valid: false},
		TriggerType:  "database_change",
		TriggerData:  []byte(`{"table":"orders","event":"INSERT"}`),
		Status:       "received",
		CreatedAt:    time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC),
	}

	client, close := setupTestServer(t, &mockQuerier{
		getProjectByIDAndTenantIDFn: projectOwnershipOK(),
		listEventHistoryByFunctionIDFn: func(_ context.Context, _ store.ListEventHistoryByFunctionIDParams) ([]store.EventHistory, error) {
			return []store.EventHistory{ev}, nil
		},
	})
	defer close()

	resp, err := client.ListEventHistory(context.Background(), makeRequest("proj-123", "bbbbbbbb-bbbb-bbbb-bbbb-bbbbbbbbbbbb"))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if resp.Msg.Events[0].InvocationId != "" {
		t.Fatalf("expected empty invocation_id, got %s", resp.Msg.Events[0].InvocationId)
	}
}

func TestRetryEvent_Unimplemented(t *testing.T) {
	client, close := setupTestServer(t, &mockQuerier{})
	defer close()

	req := connect.NewRequest(&eventv1.RetryEventRequest{
		ProjectId: "proj-123",
		EventId:   "aaaaaaaa-aaaa-aaaa-aaaa-aaaaaaaaaaaa",
	})
	req.Header().Set(requestctx.HeaderUserID, testUserID.String())

	_, err := client.RetryEvent(context.Background(), req)
	if err == nil {
		t.Fatal("expected error")
	}
	if connect.CodeOf(err) != connect.CodeUnimplemented {
		t.Fatalf("expected CodeUnimplemented, got %v", connect.CodeOf(err))
	}
}
