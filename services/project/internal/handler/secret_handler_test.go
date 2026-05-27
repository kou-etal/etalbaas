package handler_test

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"connectrpc.com/connect"
	"github.com/google/uuid"

	"github.com/kou-etal/etalbaas/pkg/k8s"
	"github.com/kou-etal/etalbaas/pkg/requestctx"
	secretv1 "github.com/kou-etal/etalbaas/proto/gen/go/etalbaas/secret/v1"
	"github.com/kou-etal/etalbaas/proto/gen/go/etalbaas/secret/v1/secretv1connect"
	"github.com/kou-etal/etalbaas/services/project/internal/handler"
	"github.com/kou-etal/etalbaas/services/project/internal/service"
	"github.com/kou-etal/etalbaas/services/project/internal/store"
)

var testSecretID = uuid.MustParse("770e8400-e29b-41d4-a716-446655440000")

type mockSecretManager struct {
	createSecretFn func(ctx context.Context, namespace, name, key, value string) error
	updateSecretFn func(ctx context.Context, namespace, name, key, value string) error
	deleteSecretFn func(ctx context.Context, namespace, name string) error
}

func (m *mockSecretManager) CreateSecret(ctx context.Context, namespace, name, key, value string) error {
	if m.createSecretFn != nil {
		return m.createSecretFn(ctx, namespace, name, key, value)
	}
	return nil
}

func (m *mockSecretManager) UpdateSecret(ctx context.Context, namespace, name, key, value string) error {
	if m.updateSecretFn != nil {
		return m.updateSecretFn(ctx, namespace, name, key, value)
	}
	return nil
}

func (m *mockSecretManager) DeleteSecret(ctx context.Context, namespace, name string) error {
	if m.deleteSecretFn != nil {
		return m.deleteSecretFn(ctx, namespace, name)
	}
	return nil
}

var _ k8s.SecretManager = (*mockSecretManager)(nil)

func newTestSecretMetadata() store.SecretsMetadatum {
	return store.SecretsMetadatum{
		ID:          testSecretID,
		ProjectID:   "abc12345",
		Name:        "OPENAI_API_KEY",
		Description: "OpenAI API key",
		CreatedAt:   time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC),
		UpdatedAt:   time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC),
	}
}

func setupSecretTestServer(t *testing.T, q store.Querier, secMgr k8s.SecretManager) (secretv1connect.SecretServiceClient, func()) {
	t.Helper()
	svc := service.NewSecretService(q, secMgr)
	h := handler.NewSecretHandler(svc)
	path, hnd := secretv1connect.NewSecretServiceHandler(h,
		connect.WithInterceptors(injectUserID()),
	)
	mux := http.NewServeMux()
	mux.Handle(path, hnd)
	srv := httptest.NewServer(mux)
	client := secretv1connect.NewSecretServiceClient(http.DefaultClient, srv.URL)
	return client, srv.Close
}

func TestCreateSecret_Success(t *testing.T) {
	meta := newTestSecretMetadata()
	proj := newTestProject()
	secMgr := &mockSecretManager{
		createSecretFn: func(_ context.Context, namespace, name, key, value string) error {
			if namespace != "project-abc12345" {
				t.Fatalf("unexpected namespace: %q", namespace)
			}
			if name != "openai-api-key" {
				t.Fatalf("unexpected k8s name: %q", name)
			}
			if key != "OPENAI_API_KEY" {
				t.Fatalf("unexpected key: %q", key)
			}
			return nil
		},
	}
	q := &mockQuerier{
		getProjectByIDAndTenantIDFn: func(_ context.Context, _ store.GetProjectByIDAndTenantIDParams) (store.Project, error) {
			return proj, nil
		},
		createSecretMetadataFn: func(_ context.Context, arg store.CreateSecretMetadataParams) (store.SecretsMetadatum, error) {
			if arg.ProjectID != "abc12345" {
				t.Fatalf("unexpected project ID: %q", arg.ProjectID)
			}
			if arg.Name != "OPENAI_API_KEY" {
				t.Fatalf("unexpected name: %q", arg.Name)
			}
			return meta, nil
		},
	}
	client, cleanup := setupSecretTestServer(t, q, secMgr)
	defer cleanup()

	req := connect.NewRequest(&secretv1.CreateSecretRequest{
		ProjectId:   "abc12345",
		Name:        "OPENAI_API_KEY",
		Value:       "sk-test-123",
		Description: "OpenAI API key",
	})
	req.Header().Set(requestctx.HeaderUserID, testUserID.String())

	resp, err := client.CreateSecret(context.Background(), req)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if resp.Msg.Secret.Name != "OPENAI_API_KEY" {
		t.Fatalf("got name %q, want %q", resp.Msg.Secret.Name, "OPENAI_API_KEY")
	}
}

func TestCreateSecret_InvalidName(t *testing.T) {
	q := &mockQuerier{}
	secMgr := &mockSecretManager{}
	client, cleanup := setupSecretTestServer(t, q, secMgr)
	defer cleanup()

	req := connect.NewRequest(&secretv1.CreateSecretRequest{
		ProjectId: "abc12345",
		Name:      "invalid-name",
		Value:     "some-value",
	})
	req.Header().Set(requestctx.HeaderUserID, testUserID.String())

	_, err := client.CreateSecret(context.Background(), req)
	if err == nil {
		t.Fatal("expected error for invalid name")
	}
	if code := connect.CodeOf(err); code != connect.CodeInvalidArgument {
		t.Fatalf("got code %v, want InvalidArgument", code)
	}
}

func TestCreateSecret_EmptyValue(t *testing.T) {
	q := &mockQuerier{}
	secMgr := &mockSecretManager{}
	client, cleanup := setupSecretTestServer(t, q, secMgr)
	defer cleanup()

	req := connect.NewRequest(&secretv1.CreateSecretRequest{
		ProjectId: "abc12345",
		Name:      "VALID_NAME",
		Value:     "",
	})
	req.Header().Set(requestctx.HeaderUserID, testUserID.String())

	_, err := client.CreateSecret(context.Background(), req)
	if err == nil {
		t.Fatal("expected error for empty value")
	}
	if code := connect.CodeOf(err); code != connect.CodeInvalidArgument {
		t.Fatalf("got code %v, want InvalidArgument", code)
	}
}

func TestUpdateSecretValue_Success(t *testing.T) {
	meta := newTestSecretMetadata()
	proj := newTestProject()
	updated := meta
	updated.UpdatedAt = time.Now()
	secMgr := &mockSecretManager{
		updateSecretFn: func(_ context.Context, namespace, name, key, value string) error {
			if namespace != "project-abc12345" {
				t.Fatalf("unexpected namespace: %q", namespace)
			}
			return nil
		},
	}
	q := &mockQuerier{
		getProjectByIDAndTenantIDFn: func(_ context.Context, _ store.GetProjectByIDAndTenantIDParams) (store.Project, error) {
			return proj, nil
		},
		getSecretMetadataByIDFn: func(_ context.Context, arg store.GetSecretMetadataByIDParams) (store.SecretsMetadatum, error) {
			return meta, nil
		},
		updateSecretMetadataUpdatedAtFn: func(_ context.Context, _ store.UpdateSecretMetadataUpdatedAtParams) (store.SecretsMetadatum, error) {
			return updated, nil
		},
	}
	client, cleanup := setupSecretTestServer(t, q, secMgr)
	defer cleanup()

	req := connect.NewRequest(&secretv1.UpdateSecretValueRequest{
		ProjectId: "abc12345",
		SecretId:  testSecretID.String(),
		Value:     "sk-new-value",
	})
	req.Header().Set(requestctx.HeaderUserID, testUserID.String())

	resp, err := client.UpdateSecretValue(context.Background(), req)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if resp.Msg.Secret.Id != testSecretID.String() {
		t.Fatalf("got id %q, want %q", resp.Msg.Secret.Id, testSecretID.String())
	}
}

func TestUpdateSecretValue_NotFound(t *testing.T) {
	proj := newTestProject()
	q := &mockQuerier{
		getProjectByIDAndTenantIDFn: func(_ context.Context, _ store.GetProjectByIDAndTenantIDParams) (store.Project, error) {
			return proj, nil
		},
	}
	secMgr := &mockSecretManager{}
	client, cleanup := setupSecretTestServer(t, q, secMgr)
	defer cleanup()

	req := connect.NewRequest(&secretv1.UpdateSecretValueRequest{
		ProjectId: "abc12345",
		SecretId:  testSecretID.String(),
		Value:     "new-value",
	})
	req.Header().Set(requestctx.HeaderUserID, testUserID.String())

	_, err := client.UpdateSecretValue(context.Background(), req)
	if err == nil {
		t.Fatal("expected error for not found")
	}
	if code := connect.CodeOf(err); code != connect.CodeNotFound {
		t.Fatalf("got code %v, want NotFound", code)
	}
}

func TestUpdateSecretValue_InvalidID(t *testing.T) {
	q := &mockQuerier{}
	secMgr := &mockSecretManager{}
	client, cleanup := setupSecretTestServer(t, q, secMgr)
	defer cleanup()

	req := connect.NewRequest(&secretv1.UpdateSecretValueRequest{
		ProjectId: "abc12345",
		SecretId:  "not-a-uuid",
		Value:     "new-value",
	})
	req.Header().Set(requestctx.HeaderUserID, testUserID.String())

	_, err := client.UpdateSecretValue(context.Background(), req)
	if err == nil {
		t.Fatal("expected error for invalid secret_id")
	}
	if code := connect.CodeOf(err); code != connect.CodeInvalidArgument {
		t.Fatalf("got code %v, want InvalidArgument", code)
	}
}

func TestDeleteSecret_Success(t *testing.T) {
	meta := newTestSecretMetadata()
	proj := newTestProject()
	secMgr := &mockSecretManager{
		deleteSecretFn: func(_ context.Context, namespace, name string) error {
			if namespace != "project-abc12345" {
				t.Fatalf("unexpected namespace: %q", namespace)
			}
			return nil
		},
	}
	q := &mockQuerier{
		getProjectByIDAndTenantIDFn: func(_ context.Context, _ store.GetProjectByIDAndTenantIDParams) (store.Project, error) {
			return proj, nil
		},
		getSecretMetadataByIDFn: func(_ context.Context, _ store.GetSecretMetadataByIDParams) (store.SecretsMetadatum, error) {
			return meta, nil
		},
		deleteSecretMetadataFn: func(_ context.Context, _ store.DeleteSecretMetadataParams) (store.SecretsMetadatum, error) {
			return meta, nil
		},
	}
	client, cleanup := setupSecretTestServer(t, q, secMgr)
	defer cleanup()

	req := connect.NewRequest(&secretv1.DeleteSecretRequest{
		ProjectId: "abc12345",
		SecretId:  testSecretID.String(),
	})
	req.Header().Set(requestctx.HeaderUserID, testUserID.String())

	resp, err := client.DeleteSecret(context.Background(), req)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if resp.Msg.Secret.Name != "OPENAI_API_KEY" {
		t.Fatalf("got name %q, want %q", resp.Msg.Secret.Name, "OPENAI_API_KEY")
	}
}

func TestDeleteSecret_NotFound(t *testing.T) {
	proj := newTestProject()
	q := &mockQuerier{
		getProjectByIDAndTenantIDFn: func(_ context.Context, _ store.GetProjectByIDAndTenantIDParams) (store.Project, error) {
			return proj, nil
		},
	}
	secMgr := &mockSecretManager{}
	client, cleanup := setupSecretTestServer(t, q, secMgr)
	defer cleanup()

	req := connect.NewRequest(&secretv1.DeleteSecretRequest{
		ProjectId: "abc12345",
		SecretId:  testSecretID.String(),
	})
	req.Header().Set(requestctx.HeaderUserID, testUserID.String())

	_, err := client.DeleteSecret(context.Background(), req)
	if err == nil {
		t.Fatal("expected error for not found")
	}
	if code := connect.CodeOf(err); code != connect.CodeNotFound {
		t.Fatalf("got code %v, want NotFound", code)
	}
}

func TestListSecrets_Success(t *testing.T) {
	secrets := []store.SecretsMetadatum{newTestSecretMetadata()}
	proj := newTestProject()
	q := &mockQuerier{
		getProjectByIDAndTenantIDFn: func(_ context.Context, _ store.GetProjectByIDAndTenantIDParams) (store.Project, error) {
			return proj, nil
		},
		listSecretsByProjectIDFn: func(_ context.Context, arg store.ListSecretsByProjectIDParams) ([]store.SecretsMetadatum, error) {
			if arg.ProjectID != "abc12345" {
				t.Fatalf("unexpected project ID: %q", arg.ProjectID)
			}
			return secrets, nil
		},
	}
	secMgr := &mockSecretManager{}
	client, cleanup := setupSecretTestServer(t, q, secMgr)
	defer cleanup()

	req := connect.NewRequest(&secretv1.ListSecretsRequest{
		ProjectId: "abc12345",
	})
	req.Header().Set(requestctx.HeaderUserID, testUserID.String())

	resp, err := client.ListSecrets(context.Background(), req)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(resp.Msg.Secrets) != 1 {
		t.Fatalf("got %d secrets, want 1", len(resp.Msg.Secrets))
	}
	if resp.Msg.Pagination != nil {
		t.Fatal("expected nil pagination")
	}
}
