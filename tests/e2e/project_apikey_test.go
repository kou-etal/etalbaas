//go:build e2e

package e2e_test

import (
	"context"
	"testing"

	"connectrpc.com/connect"
	"github.com/google/uuid"

	projectv1 "github.com/kou-etal/etalbaas/proto/gen/go/etalbaas/project/v1"
)

func TestAPIKey_CreateAndList(t *testing.T) {
	truncateAll(t)

	tenantID := uuid.New()
	createTestTenant(t, tenantID, "alice@example.com")

	client := newProjectClient()
	ctx := context.Background()

	// Create project.
	createResp, err := client.CreateProject(ctx, authedRequest(t, tenantID, &projectv1.CreateProjectRequest{
		DisplayName:     "Key Project",
		PostgresEnabled: true,
	}))
	if err != nil {
		t.Fatalf("CreateProject: %v", err)
	}
	projectID := createResp.Msg.Project.Id

	// Create API key.
	keyResp, err := client.CreateApiKey(ctx, authedRequest(t, tenantID, &projectv1.CreateApiKeyRequest{
		ProjectId: projectID,
		Name:      "test-key",
		Role:      "anon",
	}))
	if err != nil {
		t.Fatalf("CreateApiKey: %v", err)
	}
	if keyResp.Msg.RawKey == "" {
		t.Fatal("expected non-empty raw_key")
	}
	if keyResp.Msg.ApiKey == nil {
		t.Fatal("expected api_key in response")
	}
	if keyResp.Msg.ApiKey.KeyPrefix == "" {
		t.Fatal("expected non-empty key_prefix")
	}

	// List API keys.
	listResp, err := client.ListApiKeys(ctx, authedRequest(t, tenantID, &projectv1.ListApiKeysRequest{
		ProjectId: projectID,
	}))
	if err != nil {
		t.Fatalf("ListApiKeys: %v", err)
	}
	if len(listResp.Msg.ApiKeys) != 1 {
		t.Fatalf("got %d keys, want 1", len(listResp.Msg.ApiKeys))
	}
	if listResp.Msg.ApiKeys[0].Name != "test-key" {
		t.Fatalf("got name %q, want %q", listResp.Msg.ApiKeys[0].Name, "test-key")
	}
}

func TestAPIKey_Revoke(t *testing.T) {
	truncateAll(t)

	tenantID := uuid.New()
	createTestTenant(t, tenantID, "alice@example.com")

	client := newProjectClient()
	ctx := context.Background()

	createResp, err := client.CreateProject(ctx, authedRequest(t, tenantID, &projectv1.CreateProjectRequest{
		DisplayName:     "Revoke Project",
		PostgresEnabled: true,
	}))
	if err != nil {
		t.Fatalf("CreateProject: %v", err)
	}
	projectID := createResp.Msg.Project.Id

	keyResp, err := client.CreateApiKey(ctx, authedRequest(t, tenantID, &projectv1.CreateApiKeyRequest{
		ProjectId: projectID,
		Name:      "revoke-key",
		Role:      "anon",
	}))
	if err != nil {
		t.Fatalf("CreateApiKey: %v", err)
	}
	keyID := keyResp.Msg.ApiKey.Id

	// Revoke.
	_, err = client.RevokeApiKey(ctx, authedRequest(t, tenantID, &projectv1.RevokeApiKeyRequest{
		ProjectId: projectID,
		ApiKeyId:  keyID,
	}))
	if err != nil {
		t.Fatalf("RevokeApiKey: %v", err)
	}

	// List should still contain the key (revoke is a soft-delete: sets revoked_at).
	listResp, err := client.ListApiKeys(ctx, authedRequest(t, tenantID, &projectv1.ListApiKeysRequest{
		ProjectId: projectID,
	}))
	if err != nil {
		t.Fatalf("ListApiKeys: %v", err)
	}
	if len(listResp.Msg.ApiKeys) != 1 {
		t.Fatalf("got %d keys after revoke, want 1", len(listResp.Msg.ApiKeys))
	}
	if listResp.Msg.ApiKeys[0].RevokedAt == nil {
		t.Fatal("expected revoked_at to be set after revoke")
	}
}

func TestAPIKey_Create_DuplicateName(t *testing.T) {
	truncateAll(t)

	tenantID := uuid.New()
	createTestTenant(t, tenantID, "alice@example.com")

	client := newProjectClient()
	ctx := context.Background()

	createResp, err := client.CreateProject(ctx, authedRequest(t, tenantID, &projectv1.CreateProjectRequest{
		DisplayName:     "Dup Key Project",
		PostgresEnabled: true,
	}))
	if err != nil {
		t.Fatalf("CreateProject: %v", err)
	}
	projectID := createResp.Msg.Project.Id

	// First key — should succeed.
	_, err = client.CreateApiKey(ctx, authedRequest(t, tenantID, &projectv1.CreateApiKeyRequest{
		ProjectId: projectID,
		Name:      "duplicate",
		Role:      "anon",
	}))
	if err != nil {
		t.Fatalf("CreateApiKey[1]: %v", err)
	}

	// Second key with same name — should fail.
	_, err = client.CreateApiKey(ctx, authedRequest(t, tenantID, &projectv1.CreateApiKeyRequest{
		ProjectId: projectID,
		Name:      "duplicate",
		Role:      "anon",
	}))
	if err == nil {
		t.Fatal("expected error for duplicate key name")
	}
	if code := connect.CodeOf(err); code != connect.CodeAlreadyExists {
		t.Fatalf("got code %v, want AlreadyExists", code)
	}
}
