package handler_test

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"connectrpc.com/connect"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"

	"github.com/kou-etal/etalbaas/pkg/requestctx"
	functionv1 "github.com/kou-etal/etalbaas/proto/gen/go/etalbaas/function/v1"
	"github.com/kou-etal/etalbaas/proto/gen/go/etalbaas/function/v1/functionv1connect"
	"github.com/kou-etal/etalbaas/services/function/internal/handler"
	svcpkg "github.com/kou-etal/etalbaas/services/function/internal/service"
	"github.com/kou-etal/etalbaas/services/function/internal/store"
)

var testUserID = uuid.MustParse("550e8400-e29b-41d4-a716-446655440000")

type mockQuerier struct {
	createFunctionFn              func(ctx context.Context, arg store.CreateFunctionParams) (store.Function, error)
	getFunctionByIDAndProjectIDFn func(ctx context.Context, arg store.GetFunctionByIDAndProjectIDParams) (store.Function, error)
	getProjectByIDAndTenantIDFn   func(ctx context.Context, arg store.GetProjectByIDAndTenantIDParams) (store.Project, error)
	listFunctionsByProjectIDFn    func(ctx context.Context, arg store.ListFunctionsByProjectIDParams) ([]store.Function, error)
	updateFunctionFn              func(ctx context.Context, arg store.UpdateFunctionParams) (store.Function, error)
	deleteFunctionFn              func(ctx context.Context, arg store.DeleteFunctionParams) (store.Function, error)
	listInvocationsByFunctionIDFn func(ctx context.Context, arg store.ListInvocationsByFunctionIDParams) ([]store.Invocation, error)
}

func (m *mockQuerier) CreateFunction(ctx context.Context, arg store.CreateFunctionParams) (store.Function, error) {
	if m.createFunctionFn != nil {
		return m.createFunctionFn(ctx, arg)
	}
	return store.Function{}, pgx.ErrNoRows
}

func (m *mockQuerier) GetFunctionByIDAndProjectID(ctx context.Context, arg store.GetFunctionByIDAndProjectIDParams) (store.Function, error) {
	if m.getFunctionByIDAndProjectIDFn != nil {
		return m.getFunctionByIDAndProjectIDFn(ctx, arg)
	}
	return store.Function{}, pgx.ErrNoRows
}

func (m *mockQuerier) GetProjectByIDAndTenantID(ctx context.Context, arg store.GetProjectByIDAndTenantIDParams) (store.Project, error) {
	if m.getProjectByIDAndTenantIDFn != nil {
		return m.getProjectByIDAndTenantIDFn(ctx, arg)
	}
	return store.Project{}, pgx.ErrNoRows
}

func (m *mockQuerier) ListFunctionsByProjectID(ctx context.Context, arg store.ListFunctionsByProjectIDParams) ([]store.Function, error) {
	if m.listFunctionsByProjectIDFn != nil {
		return m.listFunctionsByProjectIDFn(ctx, arg)
	}
	return nil, nil
}

func (m *mockQuerier) UpdateFunction(ctx context.Context, arg store.UpdateFunctionParams) (store.Function, error) {
	if m.updateFunctionFn != nil {
		return m.updateFunctionFn(ctx, arg)
	}
	return store.Function{}, pgx.ErrNoRows
}

func (m *mockQuerier) DeleteFunction(ctx context.Context, arg store.DeleteFunctionParams) (store.Function, error) {
	if m.deleteFunctionFn != nil {
		return m.deleteFunctionFn(ctx, arg)
	}
	return store.Function{}, pgx.ErrNoRows
}

func (m *mockQuerier) ListInvocationsByFunctionID(ctx context.Context, arg store.ListInvocationsByFunctionIDParams) ([]store.Invocation, error) {
	if m.listInvocationsByFunctionIDFn != nil {
		return m.listInvocationsByFunctionIDFn(ctx, arg)
	}
	return nil, nil
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

func setupTestServer(t *testing.T, q store.Querier) (functionv1connect.FunctionServiceClient, func()) {
	t.Helper()
	svc := svcpkg.NewFunctionService(q)
	h := handler.NewFunctionHandler(svc)
	path, hnd := functionv1connect.NewFunctionServiceHandler(h,
		connect.WithInterceptors(injectUserID()),
	)
	mux := http.NewServeMux()
	mux.Handle(path, hnd)
	srv := httptest.NewServer(mux)
	client := functionv1connect.NewFunctionServiceClient(http.DefaultClient, srv.URL)
	return client, srv.Close
}

func mustJSON(v any) []byte {
	b, err := json.Marshal(v)
	if err != nil {
		panic(err)
	}
	return b
}

func newTestFunction() store.Function {
	preset := "python3.11"
	return store.Function{
		ID:                  uuid.MustParse("aaaaaaaa-aaaa-aaaa-aaaa-aaaaaaaaaaaa"),
		ProjectID:           "proj-123",
		Name:                "my-func",
		DisplayName:         "My Function",
		Kind:                "light-deployment",
		Mode:                "sync",
		SourceType:          "inline",
		SourceConfig:        mustJSON(svcpkg.InlineSourceConfig{Code: "print('hello')", Filename: "main.py"}),
		RuntimePreset:       &preset,
		RuntimeRequirements: []string{},
		TimeoutSec:          30,
		Triggers:            []byte("[]"),
		EnvVars:             []byte("[]"),
		Status:              "pending",
		CreatedAt:           time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC),
		UpdatedAt:           time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC),
	}
}

func projectOwnershipOK() func(context.Context, store.GetProjectByIDAndTenantIDParams) (store.Project, error) {
	return func(_ context.Context, _ store.GetProjectByIDAndTenantIDParams) (store.Project, error) {
		return store.Project{}, nil
	}
}

// ============================================================
// CreateFunction tests
// ============================================================

func TestCreateFunction_Success_InlinePreset(t *testing.T) {
	fn := newTestFunction()
	q := &mockQuerier{
		getProjectByIDAndTenantIDFn: projectOwnershipOK(),
		createFunctionFn: func(_ context.Context, arg store.CreateFunctionParams) (store.Function, error) {
			fn.ID = arg.ID
			return fn, nil
		},
	}
	client, cleanup := setupTestServer(t, q)
	defer cleanup()

	req := connect.NewRequest(&functionv1.CreateFunctionRequest{
		ProjectId:   "proj-123",
		Name:        "my-func",
		DisplayName: "My Function",
		Kind:        "light-deployment",
		Mode:        "sync",
		Source:      &functionv1.CreateFunctionRequest_InlineSource{InlineSource: &functionv1.InlineSource{Code: "print('hello')", Filename: "main.py"}},
		Runtime:     &functionv1.CreateFunctionRequest_PresetRuntime{PresetRuntime: &functionv1.PresetRuntime{Preset: "python3.11"}},
	})
	req.Header().Set(requestctx.HeaderUserID, testUserID.String())

	resp, err := client.CreateFunction(context.Background(), req)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if resp.Msg.Function.Name != "my-func" {
		t.Fatalf("got name %q, want %q", resp.Msg.Function.Name, "my-func")
	}
	if resp.Msg.Function.Status != "pending" {
		t.Fatalf("got status %q, want 'pending'", resp.Msg.Function.Status)
	}
}

func TestCreateFunction_Success_GitCustom(t *testing.T) {
	fn := newTestFunction()
	dockerfile := "FROM python:3.11"
	fn.SourceType = "git"
	fn.SourceConfig = mustJSON(svcpkg.GitSourceConfig{RepoURL: "https://github.com/test/repo", Branch: "main", Subpath: "/"})
	fn.RuntimePreset = nil
	fn.RuntimeDockerfile = &dockerfile
	fn.Kind = "heavy-deployment"

	q := &mockQuerier{
		getProjectByIDAndTenantIDFn: projectOwnershipOK(),
		createFunctionFn: func(_ context.Context, _ store.CreateFunctionParams) (store.Function, error) {
			return fn, nil
		},
	}
	client, cleanup := setupTestServer(t, q)
	defer cleanup()

	req := connect.NewRequest(&functionv1.CreateFunctionRequest{
		ProjectId:   "proj-123",
		Name:        "my-func",
		DisplayName: "My Function",
		Kind:        "heavy-deployment",
		Mode:        "sync",
		Source:      &functionv1.CreateFunctionRequest_GitSource{GitSource: &functionv1.GitSource{RepoUrl: "https://github.com/test/repo", Branch: "main", Subpath: "/"}},
		Runtime:     &functionv1.CreateFunctionRequest_CustomRuntime{CustomRuntime: &functionv1.CustomRuntime{Dockerfile: dockerfile}},
	})
	req.Header().Set(requestctx.HeaderUserID, testUserID.String())

	resp, err := client.CreateFunction(context.Background(), req)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if resp.Msg.Function.GetGitSource() == nil {
		t.Fatal("expected git source")
	}
	if resp.Msg.Function.GetCustomRuntime() == nil {
		t.Fatal("expected custom runtime")
	}
}

func TestCreateFunction_InvalidName(t *testing.T) {
	q := &mockQuerier{
		getProjectByIDAndTenantIDFn: projectOwnershipOK(),
	}
	client, cleanup := setupTestServer(t, q)
	defer cleanup()

	req := connect.NewRequest(&functionv1.CreateFunctionRequest{
		ProjectId:   "proj-123",
		Name:        "ab", // too short
		DisplayName: "Test",
		Kind:        "light-deployment",
		Mode:        "sync",
		Source:      &functionv1.CreateFunctionRequest_InlineSource{InlineSource: &functionv1.InlineSource{Code: "x", Filename: "main.py"}},
		Runtime:     &functionv1.CreateFunctionRequest_PresetRuntime{PresetRuntime: &functionv1.PresetRuntime{Preset: "python3.11"}},
	})
	req.Header().Set(requestctx.HeaderUserID, testUserID.String())

	_, err := client.CreateFunction(context.Background(), req)
	if err == nil {
		t.Fatal("expected error for short name")
	}
	if code := connect.CodeOf(err); code != connect.CodeInvalidArgument {
		t.Fatalf("got code %v, want InvalidArgument", code)
	}
}

func TestCreateFunction_ProjectNotFound(t *testing.T) {
	q := &mockQuerier{} // default returns pgx.ErrNoRows for project lookup
	client, cleanup := setupTestServer(t, q)
	defer cleanup()

	req := connect.NewRequest(&functionv1.CreateFunctionRequest{
		ProjectId:   "nonexist",
		Name:        "my-func",
		DisplayName: "My Function",
		Kind:        "light-deployment",
		Mode:        "sync",
		Source:      &functionv1.CreateFunctionRequest_InlineSource{InlineSource: &functionv1.InlineSource{Code: "x", Filename: "main.py"}},
		Runtime:     &functionv1.CreateFunctionRequest_PresetRuntime{PresetRuntime: &functionv1.PresetRuntime{Preset: "python3.11"}},
	})
	req.Header().Set(requestctx.HeaderUserID, testUserID.String())

	_, err := client.CreateFunction(context.Background(), req)
	if err == nil {
		t.Fatal("expected error for project not found")
	}
	if code := connect.CodeOf(err); code != connect.CodeNotFound {
		t.Fatalf("got code %v, want NotFound", code)
	}
}

// ============================================================
// GetFunction tests
// ============================================================

func TestGetFunction_Success(t *testing.T) {
	fn := newTestFunction()
	q := &mockQuerier{
		getProjectByIDAndTenantIDFn: projectOwnershipOK(),
		getFunctionByIDAndProjectIDFn: func(_ context.Context, arg store.GetFunctionByIDAndProjectIDParams) (store.Function, error) {
			if arg.ID != fn.ID {
				t.Fatalf("unexpected function ID: %v", arg.ID)
			}
			return fn, nil
		},
	}
	client, cleanup := setupTestServer(t, q)
	defer cleanup()

	req := connect.NewRequest(&functionv1.GetFunctionRequest{
		ProjectId:  "proj-123",
		FunctionId: fn.ID.String(),
	})
	req.Header().Set(requestctx.HeaderUserID, testUserID.String())

	resp, err := client.GetFunction(context.Background(), req)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if resp.Msg.Function.Name != "my-func" {
		t.Fatalf("got name %q, want %q", resp.Msg.Function.Name, "my-func")
	}
}

func TestGetFunction_NotFound(t *testing.T) {
	q := &mockQuerier{
		getProjectByIDAndTenantIDFn: projectOwnershipOK(),
	}
	client, cleanup := setupTestServer(t, q)
	defer cleanup()

	req := connect.NewRequest(&functionv1.GetFunctionRequest{
		ProjectId:  "proj-123",
		FunctionId: uuid.New().String(),
	})
	req.Header().Set(requestctx.HeaderUserID, testUserID.String())

	_, err := client.GetFunction(context.Background(), req)
	if err == nil {
		t.Fatal("expected error for not found")
	}
	if code := connect.CodeOf(err); code != connect.CodeNotFound {
		t.Fatalf("got code %v, want NotFound", code)
	}
}

// ============================================================
// ListFunctions tests
// ============================================================

func TestListFunctions_Success(t *testing.T) {
	fn := newTestFunction()
	q := &mockQuerier{
		getProjectByIDAndTenantIDFn: projectOwnershipOK(),
		listFunctionsByProjectIDFn: func(_ context.Context, _ store.ListFunctionsByProjectIDParams) ([]store.Function, error) {
			return []store.Function{fn}, nil
		},
	}
	client, cleanup := setupTestServer(t, q)
	defer cleanup()

	req := connect.NewRequest(&functionv1.ListFunctionsRequest{
		ProjectId: "proj-123",
	})
	req.Header().Set(requestctx.HeaderUserID, testUserID.String())

	resp, err := client.ListFunctions(context.Background(), req)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(resp.Msg.Functions) != 1 {
		t.Fatalf("got %d functions, want 1", len(resp.Msg.Functions))
	}
	if resp.Msg.Pagination != nil {
		t.Fatal("expected nil pagination")
	}
}

// ============================================================
// DeleteFunction tests
// ============================================================

func TestDeleteFunction_Success(t *testing.T) {
	fn := newTestFunction()
	fn.Status = "deleted"
	q := &mockQuerier{
		getProjectByIDAndTenantIDFn: projectOwnershipOK(),
		deleteFunctionFn: func(_ context.Context, _ store.DeleteFunctionParams) (store.Function, error) {
			return fn, nil
		},
	}
	client, cleanup := setupTestServer(t, q)
	defer cleanup()

	req := connect.NewRequest(&functionv1.DeleteFunctionRequest{
		ProjectId:  "proj-123",
		FunctionId: fn.ID.String(),
	})
	req.Header().Set(requestctx.HeaderUserID, testUserID.String())

	resp, err := client.DeleteFunction(context.Background(), req)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if resp.Msg.Function.Status != "deleted" {
		t.Fatalf("got status %q, want 'deleted'", resp.Msg.Function.Status)
	}
}

func TestDeleteFunction_NotFound(t *testing.T) {
	q := &mockQuerier{
		getProjectByIDAndTenantIDFn: projectOwnershipOK(),
	}
	client, cleanup := setupTestServer(t, q)
	defer cleanup()

	req := connect.NewRequest(&functionv1.DeleteFunctionRequest{
		ProjectId:  "proj-123",
		FunctionId: uuid.New().String(),
	})
	req.Header().Set(requestctx.HeaderUserID, testUserID.String())

	_, err := client.DeleteFunction(context.Background(), req)
	if err == nil {
		t.Fatal("expected error for not found")
	}
	if code := connect.CodeOf(err); code != connect.CodeNotFound {
		t.Fatalf("got code %v, want NotFound", code)
	}
}

// ============================================================
// UpdateFunction tests
// ============================================================

func TestUpdateFunction_Success_Partial(t *testing.T) {
	fn := newTestFunction()
	fn.DisplayName = "Updated Name"
	q := &mockQuerier{
		getProjectByIDAndTenantIDFn: projectOwnershipOK(),
		getFunctionByIDAndProjectIDFn: func(_ context.Context, _ store.GetFunctionByIDAndProjectIDParams) (store.Function, error) {
			return newTestFunction(), nil
		},
		updateFunctionFn: func(_ context.Context, arg store.UpdateFunctionParams) (store.Function, error) {
			fn.DisplayName = arg.DisplayName
			return fn, nil
		},
	}
	client, cleanup := setupTestServer(t, q)
	defer cleanup()

	newName := "Updated Name"
	req := connect.NewRequest(&functionv1.UpdateFunctionRequest{
		ProjectId:   "proj-123",
		FunctionId:  fn.ID.String(),
		DisplayName: &newName,
	})
	req.Header().Set(requestctx.HeaderUserID, testUserID.String())

	resp, err := client.UpdateFunction(context.Background(), req)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if resp.Msg.Function.DisplayName != "Updated Name" {
		t.Fatalf("got display_name %q, want %q", resp.Msg.Function.DisplayName, "Updated Name")
	}
}

func TestUpdateFunction_NotFound(t *testing.T) {
	q := &mockQuerier{
		getProjectByIDAndTenantIDFn: projectOwnershipOK(),
	}
	client, cleanup := setupTestServer(t, q)
	defer cleanup()

	req := connect.NewRequest(&functionv1.UpdateFunctionRequest{
		ProjectId:  "proj-123",
		FunctionId: uuid.New().String(),
	})
	req.Header().Set(requestctx.HeaderUserID, testUserID.String())

	_, err := client.UpdateFunction(context.Background(), req)
	if err == nil {
		t.Fatal("expected error for not found")
	}
	if code := connect.CodeOf(err); code != connect.CodeNotFound {
		t.Fatalf("got code %v, want NotFound", code)
	}
}

// ============================================================
// ListInvocations tests
// ============================================================

func TestListInvocations_Success(t *testing.T) {
	fn := newTestFunction()
	inv := store.Invocation{
		ID:          uuid.New(),
		FunctionID:  fn.ID,
		ProjectID:   "proj-123",
		TriggerType: "http",
		Mode:        "sync",
		Status:      "succeeded",
		RetryCount:  0,
		CreatedAt:   time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC),
	}
	q := &mockQuerier{
		getProjectByIDAndTenantIDFn: projectOwnershipOK(),
		getFunctionByIDAndProjectIDFn: func(_ context.Context, _ store.GetFunctionByIDAndProjectIDParams) (store.Function, error) {
			return fn, nil
		},
		listInvocationsByFunctionIDFn: func(_ context.Context, _ store.ListInvocationsByFunctionIDParams) ([]store.Invocation, error) {
			return []store.Invocation{inv}, nil
		},
	}
	client, cleanup := setupTestServer(t, q)
	defer cleanup()

	req := connect.NewRequest(&functionv1.ListInvocationsRequest{
		ProjectId:  "proj-123",
		FunctionId: fn.ID.String(),
	})
	req.Header().Set(requestctx.HeaderUserID, testUserID.String())

	resp, err := client.ListInvocations(context.Background(), req)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(resp.Msg.Invocations) != 1 {
		t.Fatalf("got %d invocations, want 1", len(resp.Msg.Invocations))
	}
	if resp.Msg.Invocations[0].Status != "succeeded" {
		t.Fatalf("got status %q, want 'succeeded'", resp.Msg.Invocations[0].Status)
	}
}

func TestListInvocations_WithStatusFilter(t *testing.T) {
	fn := newTestFunction()
	q := &mockQuerier{
		getProjectByIDAndTenantIDFn: projectOwnershipOK(),
		getFunctionByIDAndProjectIDFn: func(_ context.Context, _ store.GetFunctionByIDAndProjectIDParams) (store.Function, error) {
			return fn, nil
		},
		listInvocationsByFunctionIDFn: func(_ context.Context, arg store.ListInvocationsByFunctionIDParams) ([]store.Invocation, error) {
			if arg.StatusFilter == nil || *arg.StatusFilter != "failed" {
				t.Fatal("expected status_filter 'failed'")
			}
			return nil, nil
		},
	}
	client, cleanup := setupTestServer(t, q)
	defer cleanup()

	filter := "failed"
	req := connect.NewRequest(&functionv1.ListInvocationsRequest{
		ProjectId:    "proj-123",
		FunctionId:   fn.ID.String(),
		StatusFilter: &filter,
	})
	req.Header().Set(requestctx.HeaderUserID, testUserID.String())

	_, err := client.ListInvocations(context.Background(), req)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

// Unused variable suppression
var _ pgtype.Timestamptz
