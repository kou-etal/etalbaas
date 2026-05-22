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
	commonv1 "github.com/kou-etal/etalbaas/proto/gen/go/etalbaas/common/v1"
	projectv1 "github.com/kou-etal/etalbaas/proto/gen/go/etalbaas/project/v1"
	"github.com/kou-etal/etalbaas/proto/gen/go/etalbaas/project/v1/projectv1connect"
	"github.com/kou-etal/etalbaas/services/project/internal/handler"
	"github.com/kou-etal/etalbaas/services/project/internal/service"
	"github.com/kou-etal/etalbaas/services/project/internal/store"
)

var testUserID = uuid.MustParse("550e8400-e29b-41d4-a716-446655440000")

type mockQuerier struct {
	createProjectFn              func(ctx context.Context, arg store.CreateProjectParams) (store.Project, error)
	getProjectByIDAndTenantIDFn  func(ctx context.Context, arg store.GetProjectByIDAndTenantIDParams) (store.Project, error)
	listProjectsByTenantIDFn     func(ctx context.Context, arg store.ListProjectsByTenantIDParams) ([]store.Project, error)
	updateProjectStatusFn        func(ctx context.Context, arg store.UpdateProjectStatusParams) (store.Project, error)
	getProjectStatusFn           func(ctx context.Context, arg store.GetProjectStatusParams) (string, error)
	createApiKeyFn               func(ctx context.Context, arg store.CreateApiKeyParams) (store.ApiKey, error)
	listApiKeysByProjectIDFn     func(ctx context.Context, arg store.ListApiKeysByProjectIDParams) ([]store.ApiKey, error)
	revokeApiKeyFn               func(ctx context.Context, arg store.RevokeApiKeyParams) (store.ApiKey, error)
	createSecretMetadataFn       func(ctx context.Context, arg store.CreateSecretMetadataParams) (store.SecretsMetadatum, error)
	getSecretMetadataByIDFn      func(ctx context.Context, arg store.GetSecretMetadataByIDParams) (store.SecretsMetadatum, error)
	listSecretsByProjectIDFn     func(ctx context.Context, arg store.ListSecretsByProjectIDParams) ([]store.SecretsMetadatum, error)
	updateSecretMetadataUpdatedAtFn func(ctx context.Context, arg store.UpdateSecretMetadataUpdatedAtParams) (store.SecretsMetadatum, error)
	deleteSecretMetadataFn       func(ctx context.Context, arg store.DeleteSecretMetadataParams) (store.SecretsMetadatum, error)
}

func (m *mockQuerier) CreateProject(ctx context.Context, arg store.CreateProjectParams) (store.Project, error) {
	if m.createProjectFn != nil {
		return m.createProjectFn(ctx, arg)
	}
	return store.Project{}, pgx.ErrNoRows
}

func (m *mockQuerier) GetProjectByIDAndTenantID(ctx context.Context, arg store.GetProjectByIDAndTenantIDParams) (store.Project, error) {
	if m.getProjectByIDAndTenantIDFn != nil {
		return m.getProjectByIDAndTenantIDFn(ctx, arg)
	}
	return store.Project{}, pgx.ErrNoRows
}

func (m *mockQuerier) ListProjectsByTenantID(ctx context.Context, arg store.ListProjectsByTenantIDParams) ([]store.Project, error) {
	if m.listProjectsByTenantIDFn != nil {
		return m.listProjectsByTenantIDFn(ctx, arg)
	}
	return nil, nil
}

func (m *mockQuerier) UpdateProjectStatus(ctx context.Context, arg store.UpdateProjectStatusParams) (store.Project, error) {
	if m.updateProjectStatusFn != nil {
		return m.updateProjectStatusFn(ctx, arg)
	}
	return store.Project{}, pgx.ErrNoRows
}

func (m *mockQuerier) GetProjectStatus(ctx context.Context, arg store.GetProjectStatusParams) (string, error) {
	if m.getProjectStatusFn != nil {
		return m.getProjectStatusFn(ctx, arg)
	}
	return "", pgx.ErrNoRows
}

func (m *mockQuerier) CreateApiKey(ctx context.Context, arg store.CreateApiKeyParams) (store.ApiKey, error) {
	if m.createApiKeyFn != nil {
		return m.createApiKeyFn(ctx, arg)
	}
	return store.ApiKey{}, pgx.ErrNoRows
}

func (m *mockQuerier) ListApiKeysByProjectID(ctx context.Context, arg store.ListApiKeysByProjectIDParams) ([]store.ApiKey, error) {
	if m.listApiKeysByProjectIDFn != nil {
		return m.listApiKeysByProjectIDFn(ctx, arg)
	}
	return nil, nil
}

func (m *mockQuerier) RevokeApiKey(ctx context.Context, arg store.RevokeApiKeyParams) (store.ApiKey, error) {
	if m.revokeApiKeyFn != nil {
		return m.revokeApiKeyFn(ctx, arg)
	}
	return store.ApiKey{}, pgx.ErrNoRows
}

func (m *mockQuerier) CreateSecretMetadata(ctx context.Context, arg store.CreateSecretMetadataParams) (store.SecretsMetadatum, error) {
	if m.createSecretMetadataFn != nil {
		return m.createSecretMetadataFn(ctx, arg)
	}
	return store.SecretsMetadatum{}, pgx.ErrNoRows
}

func (m *mockQuerier) GetSecretMetadataByID(ctx context.Context, arg store.GetSecretMetadataByIDParams) (store.SecretsMetadatum, error) {
	if m.getSecretMetadataByIDFn != nil {
		return m.getSecretMetadataByIDFn(ctx, arg)
	}
	return store.SecretsMetadatum{}, pgx.ErrNoRows
}

func (m *mockQuerier) ListSecretsByProjectID(ctx context.Context, arg store.ListSecretsByProjectIDParams) ([]store.SecretsMetadatum, error) {
	if m.listSecretsByProjectIDFn != nil {
		return m.listSecretsByProjectIDFn(ctx, arg)
	}
	return nil, nil
}

func (m *mockQuerier) UpdateSecretMetadataUpdatedAt(ctx context.Context, arg store.UpdateSecretMetadataUpdatedAtParams) (store.SecretsMetadatum, error) {
	if m.updateSecretMetadataUpdatedAtFn != nil {
		return m.updateSecretMetadataUpdatedAtFn(ctx, arg)
	}
	return store.SecretsMetadatum{}, pgx.ErrNoRows
}

func (m *mockQuerier) DeleteSecretMetadata(ctx context.Context, arg store.DeleteSecretMetadataParams) (store.SecretsMetadatum, error) {
	if m.deleteSecretMetadataFn != nil {
		return m.deleteSecretMetadataFn(ctx, arg)
	}
	return store.SecretsMetadatum{}, pgx.ErrNoRows
}

func newTestProject() store.Project {
	desc := "A test project"
	return store.Project{
		ID:                 "abc12345",
		TenantID:           testUserID,
		DisplayName:        "Test Project",
		Description:        &desc,
		Status:             "pending",
		PostgresEnabled:    true,
		PostgresExtensions: []string{"pgvector"},
		RedisEnabled:       false,
		PostgrestEnabled:   false,
		CreatedAt:          time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC),
		UpdatedAt:          time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC),
	}
}

func newTestApiKey() store.ApiKey {
	return store.ApiKey{
		ID:        uuid.MustParse("660e8400-e29b-41d4-a716-446655440000"),
		ProjectID: "abc12345",
		Name:      "my-key",
		KeyHash:   "somehash",
		KeyPrefix: "abcd1234",
		Role:      "anon",
		ExpiresAt: pgtype.Timestamptz{Time: time.Date(2025, 4, 1, 0, 0, 0, 0, time.UTC), Valid: true},
		RevokedAt: pgtype.Timestamptz{},
		CreatedAt: time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC),
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

func setupProjectTestServer(t *testing.T, q store.Querier) (projectv1connect.ProjectServiceClient, func()) {
	t.Helper()
	svc := service.NewProjectService(q, nil)
	h := handler.NewProjectHandler(svc)
	path, hnd := projectv1connect.NewProjectServiceHandler(h,
		connect.WithInterceptors(injectUserID()),
	)
	mux := http.NewServeMux()
	mux.Handle(path, hnd)
	srv := httptest.NewServer(mux)
	client := projectv1connect.NewProjectServiceClient(http.DefaultClient, srv.URL)
	return client, srv.Close
}

func TestCreateProject_Success(t *testing.T) {
	proj := newTestProject()
	q := &mockQuerier{
		createProjectFn: func(_ context.Context, arg store.CreateProjectParams) (store.Project, error) {
			if arg.TenantID != testUserID {
				t.Fatalf("unexpected tenant ID: %v", arg.TenantID)
			}
			if arg.DisplayName != "Test Project" {
				t.Fatalf("unexpected display_name: %q", arg.DisplayName)
			}
			return proj, nil
		},
	}
	client, cleanup := setupProjectTestServer(t, q)
	defer cleanup()

	req := connect.NewRequest(&projectv1.CreateProjectRequest{
		DisplayName:     "Test Project",
		PostgresEnabled: true,
	})
	req.Header().Set(requestctx.HeaderUserID, testUserID.String())

	resp, err := client.CreateProject(context.Background(), req)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if resp.Msg.Project.DisplayName != "Test Project" {
		t.Fatalf("got display_name %q, want %q", resp.Msg.Project.DisplayName, "Test Project")
	}
}

func TestCreateProject_InvalidArgument(t *testing.T) {
	q := &mockQuerier{}
	client, cleanup := setupProjectTestServer(t, q)
	defer cleanup()

	req := connect.NewRequest(&projectv1.CreateProjectRequest{
		DisplayName: "",
	})
	req.Header().Set(requestctx.HeaderUserID, testUserID.String())

	_, err := client.CreateProject(context.Background(), req)
	if err == nil {
		t.Fatal("expected error for empty display_name")
	}
	if code := connect.CodeOf(err); code != connect.CodeInvalidArgument {
		t.Fatalf("got code %v, want InvalidArgument", code)
	}
}

func TestGetProject_Success(t *testing.T) {
	proj := newTestProject()
	q := &mockQuerier{
		getProjectByIDAndTenantIDFn: func(_ context.Context, arg store.GetProjectByIDAndTenantIDParams) (store.Project, error) {
			if arg.ID != "abc12345" {
				t.Fatalf("unexpected project ID: %q", arg.ID)
			}
			if arg.TenantID != testUserID {
				t.Fatalf("unexpected tenant ID: %v", arg.TenantID)
			}
			return proj, nil
		},
	}
	client, cleanup := setupProjectTestServer(t, q)
	defer cleanup()

	req := connect.NewRequest(&projectv1.GetProjectRequest{
		ProjectId: "abc12345",
	})
	req.Header().Set(requestctx.HeaderUserID, testUserID.String())

	resp, err := client.GetProject(context.Background(), req)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if resp.Msg.Project.Id != "abc12345" {
		t.Fatalf("got ID %q, want %q", resp.Msg.Project.Id, "abc12345")
	}
}

func TestGetProject_NotFound(t *testing.T) {
	q := &mockQuerier{}
	client, cleanup := setupProjectTestServer(t, q)
	defer cleanup()

	req := connect.NewRequest(&projectv1.GetProjectRequest{
		ProjectId: "nonexist",
	})
	req.Header().Set(requestctx.HeaderUserID, testUserID.String())

	_, err := client.GetProject(context.Background(), req)
	if err == nil {
		t.Fatal("expected error for not found")
	}
	if code := connect.CodeOf(err); code != connect.CodeNotFound {
		t.Fatalf("got code %v, want NotFound", code)
	}
}

func TestDeleteProject_Success(t *testing.T) {
	proj := newTestProject()
	proj.Status = "deleted"
	q := &mockQuerier{
		updateProjectStatusFn: func(_ context.Context, arg store.UpdateProjectStatusParams) (store.Project, error) {
			if arg.Status != "deleted" {
				t.Fatalf("expected status 'deleted', got %q", arg.Status)
			}
			return proj, nil
		},
	}
	client, cleanup := setupProjectTestServer(t, q)
	defer cleanup()

	req := connect.NewRequest(&projectv1.DeleteProjectRequest{
		ProjectId: "abc12345",
	})
	req.Header().Set(requestctx.HeaderUserID, testUserID.String())

	resp, err := client.DeleteProject(context.Background(), req)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if resp.Msg.Project.Status != "deleted" {
		t.Fatalf("got status %q, want %q", resp.Msg.Project.Status, "deleted")
	}
}

func TestPauseProject_Success(t *testing.T) {
	proj := newTestProject()
	proj.Status = "paused"
	q := &mockQuerier{
		getProjectStatusFn: func(_ context.Context, _ store.GetProjectStatusParams) (string, error) {
			return "ready", nil
		},
		updateProjectStatusFn: func(_ context.Context, arg store.UpdateProjectStatusParams) (store.Project, error) {
			if arg.Status != "paused" {
				t.Fatalf("expected status 'paused', got %q", arg.Status)
			}
			return proj, nil
		},
	}
	client, cleanup := setupProjectTestServer(t, q)
	defer cleanup()

	req := connect.NewRequest(&projectv1.PauseProjectRequest{
		ProjectId: "abc12345",
	})
	req.Header().Set(requestctx.HeaderUserID, testUserID.String())

	resp, err := client.PauseProject(context.Background(), req)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if resp.Msg.Project.Status != "paused" {
		t.Fatalf("got status %q, want %q", resp.Msg.Project.Status, "paused")
	}
}

func TestPauseProject_AlreadyPaused(t *testing.T) {
	q := &mockQuerier{
		getProjectStatusFn: func(_ context.Context, _ store.GetProjectStatusParams) (string, error) {
			return "paused", nil
		},
	}
	client, cleanup := setupProjectTestServer(t, q)
	defer cleanup()

	req := connect.NewRequest(&projectv1.PauseProjectRequest{
		ProjectId: "abc12345",
	})
	req.Header().Set(requestctx.HeaderUserID, testUserID.String())

	_, err := client.PauseProject(context.Background(), req)
	if err == nil {
		t.Fatal("expected error for already paused")
	}
	if code := connect.CodeOf(err); code != connect.CodeFailedPrecondition {
		t.Fatalf("got code %v, want FailedPrecondition", code)
	}
}

func TestResumeProject_Success(t *testing.T) {
	proj := newTestProject()
	proj.Status = "ready"
	q := &mockQuerier{
		getProjectStatusFn: func(_ context.Context, _ store.GetProjectStatusParams) (string, error) {
			return "paused", nil
		},
		updateProjectStatusFn: func(_ context.Context, arg store.UpdateProjectStatusParams) (store.Project, error) {
			if arg.Status != "ready" {
				t.Fatalf("expected status 'ready', got %q", arg.Status)
			}
			return proj, nil
		},
	}
	client, cleanup := setupProjectTestServer(t, q)
	defer cleanup()

	req := connect.NewRequest(&projectv1.ResumeProjectRequest{
		ProjectId: "abc12345",
	})
	req.Header().Set(requestctx.HeaderUserID, testUserID.String())

	resp, err := client.ResumeProject(context.Background(), req)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if resp.Msg.Project.Status != "ready" {
		t.Fatalf("got status %q, want %q", resp.Msg.Project.Status, "ready")
	}
}

func TestResumeProject_NotPaused(t *testing.T) {
	q := &mockQuerier{
		getProjectStatusFn: func(_ context.Context, _ store.GetProjectStatusParams) (string, error) {
			return "ready", nil
		},
	}
	client, cleanup := setupProjectTestServer(t, q)
	defer cleanup()

	req := connect.NewRequest(&projectv1.ResumeProjectRequest{
		ProjectId: "abc12345",
	})
	req.Header().Set(requestctx.HeaderUserID, testUserID.String())

	_, err := client.ResumeProject(context.Background(), req)
	if err == nil {
		t.Fatal("expected error for not paused")
	}
	if code := connect.CodeOf(err); code != connect.CodeFailedPrecondition {
		t.Fatalf("got code %v, want FailedPrecondition", code)
	}
}

func TestListProjects_Success(t *testing.T) {
	projects := []store.Project{newTestProject()}
	q := &mockQuerier{
		listProjectsByTenantIDFn: func(_ context.Context, arg store.ListProjectsByTenantIDParams) ([]store.Project, error) {
			if arg.TenantID != testUserID {
				t.Fatalf("unexpected tenant ID: %v", arg.TenantID)
			}
			if arg.PageSize != 21 {
				t.Fatalf("expected limit 21, got %d", arg.PageSize)
			}
			return projects, nil
		},
	}
	client, cleanup := setupProjectTestServer(t, q)
	defer cleanup()

	req := connect.NewRequest(&projectv1.ListProjectsRequest{})
	req.Header().Set(requestctx.HeaderUserID, testUserID.String())

	resp, err := client.ListProjects(context.Background(), req)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(resp.Msg.Projects) != 1 {
		t.Fatalf("got %d projects, want 1", len(resp.Msg.Projects))
	}
	if resp.Msg.Pagination != nil {
		t.Fatal("expected nil pagination")
	}
}

func TestListProjects_HasMore(t *testing.T) {
	projects := make([]store.Project, 3)
	for i := range projects {
		p := newTestProject()
		p.ID = "proj" + string(rune('a'+i))
		p.CreatedAt = time.Date(2025, 1, 3-i, 0, 0, 0, 0, time.UTC)
		projects[i] = p
	}
	q := &mockQuerier{
		listProjectsByTenantIDFn: func(_ context.Context, _ store.ListProjectsByTenantIDParams) ([]store.Project, error) {
			return projects, nil
		},
	}
	client, cleanup := setupProjectTestServer(t, q)
	defer cleanup()

	req := connect.NewRequest(&projectv1.ListProjectsRequest{
		Pagination: &commonv1.PaginationRequest{PageSize: 2},
	})
	req.Header().Set(requestctx.HeaderUserID, testUserID.String())

	resp, err := client.ListProjects(context.Background(), req)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(resp.Msg.Projects) != 2 {
		t.Fatalf("got %d projects, want 2", len(resp.Msg.Projects))
	}
	if resp.Msg.Pagination == nil || resp.Msg.Pagination.NextPageToken == "" {
		t.Fatal("expected pagination with next_page_token")
	}
}

func TestCreateApiKey_Success(t *testing.T) {
	proj := newTestProject()
	ak := newTestApiKey()
	q := &mockQuerier{
		getProjectByIDAndTenantIDFn: func(_ context.Context, _ store.GetProjectByIDAndTenantIDParams) (store.Project, error) {
			return proj, nil
		},
		createApiKeyFn: func(_ context.Context, arg store.CreateApiKeyParams) (store.ApiKey, error) {
			if arg.ProjectID != "abc12345" {
				t.Fatalf("unexpected project ID: %q", arg.ProjectID)
			}
			return ak, nil
		},
	}
	client, cleanup := setupProjectTestServer(t, q)
	defer cleanup()

	req := connect.NewRequest(&projectv1.CreateApiKeyRequest{
		ProjectId: "abc12345",
		Name:      "my-key",
		Role:      "anon",
	})
	req.Header().Set(requestctx.HeaderUserID, testUserID.String())

	resp, err := client.CreateApiKey(context.Background(), req)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if resp.Msg.ApiKey.Name != "my-key" {
		t.Fatalf("got name %q, want %q", resp.Msg.ApiKey.Name, "my-key")
	}
	if resp.Msg.RawKey == "" {
		t.Fatal("expected raw_key to be non-empty")
	}
}

func TestCreateApiKey_ProjectNotFound(t *testing.T) {
	q := &mockQuerier{}
	client, cleanup := setupProjectTestServer(t, q)
	defer cleanup()

	req := connect.NewRequest(&projectv1.CreateApiKeyRequest{
		ProjectId: "nonexist",
		Name:      "my-key",
		Role:      "anon",
	})
	req.Header().Set(requestctx.HeaderUserID, testUserID.String())

	_, err := client.CreateApiKey(context.Background(), req)
	if err == nil {
		t.Fatal("expected error for project not found")
	}
	if code := connect.CodeOf(err); code != connect.CodeNotFound {
		t.Fatalf("got code %v, want NotFound", code)
	}
}

func TestRevokeApiKey_Success(t *testing.T) {
	proj := newTestProject()
	ak := newTestApiKey()
	ak.RevokedAt = pgtype.Timestamptz{Time: time.Now(), Valid: true}
	q := &mockQuerier{
		getProjectByIDAndTenantIDFn: func(_ context.Context, _ store.GetProjectByIDAndTenantIDParams) (store.Project, error) {
			return proj, nil
		},
		revokeApiKeyFn: func(_ context.Context, arg store.RevokeApiKeyParams) (store.ApiKey, error) {
			return ak, nil
		},
	}
	client, cleanup := setupProjectTestServer(t, q)
	defer cleanup()

	req := connect.NewRequest(&projectv1.RevokeApiKeyRequest{
		ProjectId: "abc12345",
		ApiKeyId:  ak.ID.String(),
	})
	req.Header().Set(requestctx.HeaderUserID, testUserID.String())

	resp, err := client.RevokeApiKey(context.Background(), req)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if resp.Msg.ApiKey.RevokedAt == nil {
		t.Fatal("expected revoked_at to be set")
	}
}

func TestRevokeApiKey_NotFound(t *testing.T) {
	proj := newTestProject()
	q := &mockQuerier{
		getProjectByIDAndTenantIDFn: func(_ context.Context, _ store.GetProjectByIDAndTenantIDParams) (store.Project, error) {
			return proj, nil
		},
	}
	client, cleanup := setupProjectTestServer(t, q)
	defer cleanup()

	req := connect.NewRequest(&projectv1.RevokeApiKeyRequest{
		ProjectId: "abc12345",
		ApiKeyId:  uuid.New().String(),
	})
	req.Header().Set(requestctx.HeaderUserID, testUserID.String())

	_, err := client.RevokeApiKey(context.Background(), req)
	if err == nil {
		t.Fatal("expected error for api key not found")
	}
	if code := connect.CodeOf(err); code != connect.CodeNotFound {
		t.Fatalf("got code %v, want NotFound", code)
	}
}

func TestRevokeApiKey_InvalidID(t *testing.T) {
	q := &mockQuerier{}
	client, cleanup := setupProjectTestServer(t, q)
	defer cleanup()

	req := connect.NewRequest(&projectv1.RevokeApiKeyRequest{
		ProjectId: "abc12345",
		ApiKeyId:  "not-a-uuid",
	})
	req.Header().Set(requestctx.HeaderUserID, testUserID.String())

	_, err := client.RevokeApiKey(context.Background(), req)
	if err == nil {
		t.Fatal("expected error for invalid api_key_id")
	}
	if code := connect.CodeOf(err); code != connect.CodeInvalidArgument {
		t.Fatalf("got code %v, want InvalidArgument", code)
	}
}
