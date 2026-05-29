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

	"github.com/kou-etal/etalbaas/pkg/requestctx"
	commonv1 "github.com/kou-etal/etalbaas/proto/gen/go/etalbaas/common/v1"
	tenantv1 "github.com/kou-etal/etalbaas/proto/gen/go/etalbaas/tenant/v1"
	"github.com/kou-etal/etalbaas/proto/gen/go/etalbaas/tenant/v1/tenantv1connect"
	"github.com/kou-etal/etalbaas/services/tenant-user/internal/handler"
	"github.com/kou-etal/etalbaas/services/tenant-user/internal/service"
	"github.com/kou-etal/etalbaas/services/tenant-user/internal/store"
)

var testUserID = uuid.MustParse("550e8400-e29b-41d4-a716-446655440000")

type mockQuerier struct {
	getTenantByIDFn       func(ctx context.Context, id uuid.UUID) (store.Tenant, error)
	updateTenantProfileFn func(ctx context.Context, arg store.UpdateTenantProfileParams) (store.Tenant, error)
	listTenantsFn         func(ctx context.Context, arg store.ListTenantsParams) ([]store.Tenant, error)
}

func (m *mockQuerier) GetTenantByID(ctx context.Context, id uuid.UUID) (store.Tenant, error) {
	if m.getTenantByIDFn != nil {
		return m.getTenantByIDFn(ctx, id)
	}
	return store.Tenant{}, pgx.ErrNoRows
}

func (m *mockQuerier) UpdateTenantProfile(ctx context.Context, arg store.UpdateTenantProfileParams) (store.Tenant, error) {
	if m.updateTenantProfileFn != nil {
		return m.updateTenantProfileFn(ctx, arg)
	}
	return store.Tenant{}, pgx.ErrNoRows
}

func (m *mockQuerier) ListTenants(ctx context.Context, arg store.ListTenantsParams) ([]store.Tenant, error) {
	if m.listTenantsFn != nil {
		return m.listTenantsFn(ctx, arg)
	}
	return nil, nil
}

func newTestTenant() store.Tenant {
	return store.Tenant{
		ID:          testUserID,
		Email:       "test@example.com",
		DisplayName: "Test User",
		AvatarUrl:   nil,
		Plan:        "free",
		Status:      "active",
		CreatedAt:   time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC),
		UpdatedAt:   time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC),
	}
}

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

func setupTestServer(t *testing.T, q store.Querier) (tenantv1connect.TenantServiceClient, func()) {
	t.Helper()
	svc := service.New(q)
	h := handler.New(svc)
	path, hnd := tenantv1connect.NewTenantServiceHandler(h,
		connect.WithInterceptors(injectUserID()),
	)
	mux := http.NewServeMux()
	mux.Handle(path, hnd)
	srv := httptest.NewServer(mux)
	client := tenantv1connect.NewTenantServiceClient(http.DefaultClient, srv.URL)
	return client, srv.Close
}

func TestGetMe_Success(t *testing.T) {
	tenant := newTestTenant()
	q := &mockQuerier{
		getTenantByIDFn: func(_ context.Context, id uuid.UUID) (store.Tenant, error) {
			if id != testUserID {
				t.Fatalf("unexpected user ID: %v", id)
			}
			return tenant, nil
		},
	}
	client, cleanup := setupTestServer(t, q)
	defer cleanup()

	req := connect.NewRequest(&tenantv1.GetMeRequest{})
	req.Header().Set(requestctx.HeaderUserID, testUserID.String())

	resp, err := client.GetMe(context.Background(), req)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if resp.Msg.Tenant.Id != testUserID.String() {
		t.Fatalf("got ID %q, want %q", resp.Msg.Tenant.Id, testUserID.String())
	}
	if resp.Msg.Tenant.Email != "test@example.com" {
		t.Fatalf("got email %q, want %q", resp.Msg.Tenant.Email, "test@example.com")
	}
}

func TestGetMe_NotFound(t *testing.T) {
	q := &mockQuerier{}
	client, cleanup := setupTestServer(t, q)
	defer cleanup()

	req := connect.NewRequest(&tenantv1.GetMeRequest{})
	req.Header().Set(requestctx.HeaderUserID, testUserID.String())

	_, err := client.GetMe(context.Background(), req)
	if err == nil {
		t.Fatal("expected error for not found tenant")
	}
	if code := connect.CodeOf(err); code != connect.CodeNotFound {
		t.Fatalf("got code %v, want NotFound", code)
	}
}

func TestUpdateProfile_Success(t *testing.T) {
	displayName := "New Name"
	tenant := newTestTenant()
	tenant.DisplayName = displayName

	q := &mockQuerier{
		updateTenantProfileFn: func(_ context.Context, arg store.UpdateTenantProfileParams) (store.Tenant, error) {
			if arg.ID != testUserID {
				t.Fatalf("unexpected user ID: %v", arg.ID)
			}
			if arg.DisplayName == nil || *arg.DisplayName != displayName {
				t.Fatalf("unexpected display_name: %v", arg.DisplayName)
			}
			return tenant, nil
		},
	}
	client, cleanup := setupTestServer(t, q)
	defer cleanup()

	req := connect.NewRequest(&tenantv1.UpdateProfileRequest{
		DisplayName: &displayName,
	})
	req.Header().Set(requestctx.HeaderUserID, testUserID.String())

	resp, err := client.UpdateProfile(context.Background(), req)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if resp.Msg.Tenant.DisplayName != displayName {
		t.Fatalf("got display_name %q, want %q", resp.Msg.Tenant.DisplayName, displayName)
	}
}

func TestUpdateProfile_NoFields(t *testing.T) {
	q := &mockQuerier{}
	client, cleanup := setupTestServer(t, q)
	defer cleanup()

	req := connect.NewRequest(&tenantv1.UpdateProfileRequest{})
	req.Header().Set(requestctx.HeaderUserID, testUserID.String())

	_, err := client.UpdateProfile(context.Background(), req)
	if err == nil {
		t.Fatal("expected error for no fields")
	}
	if code := connect.CodeOf(err); code != connect.CodeInvalidArgument {
		t.Fatalf("got code %v, want InvalidArgument", code)
	}
}

func TestListTenants_Success(t *testing.T) {
	tenants := []store.Tenant{newTestTenant()}
	q := &mockQuerier{
		listTenantsFn: func(_ context.Context, arg store.ListTenantsParams) ([]store.Tenant, error) {
			// Handler passes pageSize+1=21 as limit
			if arg.PageSize != 21 {
				t.Fatalf("expected limit 21 (pageSize+1), got %d", arg.PageSize)
			}
			return tenants, nil
		},
	}
	client, cleanup := setupTestServer(t, q)
	defer cleanup()

	req := connect.NewRequest(&tenantv1.ListTenantsRequest{})
	req.Header().Set(requestctx.HeaderUserID, testUserID.String())

	resp, err := client.ListTenants(context.Background(), req)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(resp.Msg.Tenants) != 1 {
		t.Fatalf("got %d tenants, want 1", len(resp.Msg.Tenants))
	}
	if resp.Msg.Pagination != nil {
		t.Fatal("expected nil pagination when fewer results than page size")
	}
}

func TestListTenants_WithPagination(t *testing.T) {
	q := &mockQuerier{
		listTenantsFn: func(_ context.Context, arg store.ListTenantsParams) ([]store.Tenant, error) {
			// Handler passes pageSize+1=11 as limit
			if arg.PageSize != 11 {
				t.Fatalf("expected limit 11 (pageSize+1), got %d", arg.PageSize)
			}
			return nil, nil
		},
	}
	client, cleanup := setupTestServer(t, q)
	defer cleanup()

	req := connect.NewRequest(&tenantv1.ListTenantsRequest{
		Pagination: &commonv1.PaginationRequest{
			PageSize: 10,
		},
	})
	req.Header().Set(requestctx.HeaderUserID, testUserID.String())

	resp, err := client.ListTenants(context.Background(), req)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(resp.Msg.Tenants) != 0 {
		t.Fatalf("got %d tenants, want 0", len(resp.Msg.Tenants))
	}
	if resp.Msg.Pagination != nil {
		t.Fatal("expected nil pagination for empty result")
	}
}

func TestListTenants_HasMore(t *testing.T) {
	// Return 3 rows for pageSize=2 (limit=3) to trigger hasMore
	tenants := make([]store.Tenant, 3)
	for i := range tenants {
		tenants[i] = store.Tenant{
			ID:          uuid.New(),
			Email:       "test@example.com",
			DisplayName: "Test",
			Plan:        "free",
			Status:      "active",
			CreatedAt:   time.Date(2025, 1, 3-i, 0, 0, 0, 0, time.UTC),
			UpdatedAt:   time.Date(2025, 1, 3-i, 0, 0, 0, 0, time.UTC),
		}
	}
	q := &mockQuerier{
		listTenantsFn: func(_ context.Context, arg store.ListTenantsParams) ([]store.Tenant, error) {
			return tenants, nil
		},
	}
	client, cleanup := setupTestServer(t, q)
	defer cleanup()

	req := connect.NewRequest(&tenantv1.ListTenantsRequest{
		Pagination: &commonv1.PaginationRequest{
			PageSize: 2,
		},
	})
	req.Header().Set(requestctx.HeaderUserID, testUserID.String())

	resp, err := client.ListTenants(context.Background(), req)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(resp.Msg.Tenants) != 2 {
		t.Fatalf("got %d tenants, want 2 (trimmed)", len(resp.Msg.Tenants))
	}
	if resp.Msg.Pagination == nil || resp.Msg.Pagination.NextPageToken == "" {
		t.Fatal("expected pagination with next_page_token when hasMore")
	}
}

func TestListTenants_InvalidPageToken(t *testing.T) {
	q := &mockQuerier{}
	client, cleanup := setupTestServer(t, q)
	defer cleanup()

	req := connect.NewRequest(&tenantv1.ListTenantsRequest{
		Pagination: &commonv1.PaginationRequest{
			PageToken: "invalid-token!!!",
		},
	})
	req.Header().Set(requestctx.HeaderUserID, testUserID.String())

	_, err := client.ListTenants(context.Background(), req)
	if err == nil {
		t.Fatal("expected error for invalid page_token")
	}
	if code := connect.CodeOf(err); code != connect.CodeInvalidArgument {
		t.Fatalf("got code %v, want InvalidArgument", code)
	}
}

func TestListTenants_NegativePageSize(t *testing.T) {
	q := &mockQuerier{}
	client, cleanup := setupTestServer(t, q)
	defer cleanup()

	req := connect.NewRequest(&tenantv1.ListTenantsRequest{
		Pagination: &commonv1.PaginationRequest{
			PageSize: -5,
		},
	})
	req.Header().Set(requestctx.HeaderUserID, testUserID.String())

	_, err := client.ListTenants(context.Background(), req)
	if err == nil {
		t.Fatal("expected error for negative page_size")
	}
	if code := connect.CodeOf(err); code != connect.CodeInvalidArgument {
		t.Fatalf("got code %v, want InvalidArgument", code)
	}
}

func TestListTenants_CompoundCursorPassedToStore(t *testing.T) {
	cursorTime := time.Date(2025, 6, 15, 12, 0, 0, 0, time.UTC)
	cursorUUID := uuid.MustParse("aaaaaaaa-bbbb-cccc-dddd-eeeeeeeeeeee")

	q := &mockQuerier{
		listTenantsFn: func(_ context.Context, arg store.ListTenantsParams) ([]store.Tenant, error) {
			if !arg.CursorCreatedAt.Valid {
				t.Fatal("expected valid cursor_created_at")
			}
			if !arg.CursorCreatedAt.Time.Equal(cursorTime) {
				t.Fatalf("cursor time mismatch: got %v, want %v", arg.CursorCreatedAt.Time, cursorTime)
			}
			if !arg.CursorID.Valid {
				t.Fatal("expected valid cursor_id")
			}
			if arg.CursorID.Bytes != cursorUUID {
				t.Fatalf("cursor id mismatch: got %v, want %v", arg.CursorID.Bytes, cursorUUID)
			}
			return nil, nil
		},
	}
	client, cleanup := setupTestServer(t, q)
	defer cleanup()

	// Build a valid cursor token
	token := handler.EncodeCursorForTest(cursorTime, cursorUUID)

	req := connect.NewRequest(&tenantv1.ListTenantsRequest{
		Pagination: &commonv1.PaginationRequest{
			PageSize:  5,
			PageToken: token,
		},
	})
	req.Header().Set(requestctx.HeaderUserID, testUserID.String())

	_, err := client.ListTenants(context.Background(), req)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}
