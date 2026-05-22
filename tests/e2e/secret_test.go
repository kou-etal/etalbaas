//go:build e2e

package e2e_test

import (
	"context"
	"testing"

	"connectrpc.com/connect"
	"github.com/google/uuid"

	projectv1 "github.com/kou-etal/etalbaas/proto/gen/go/etalbaas/project/v1"
	secretv1 "github.com/kou-etal/etalbaas/proto/gen/go/etalbaas/secret/v1"
)

func TestSecret_Create(t *testing.T) {
	truncateAll(t)

	tenantID := uuid.New()
	createTestTenant(t, tenantID, "alice@example.com")

	projClient := newProjectClient()
	ctx := context.Background()

	createResp, err := projClient.CreateProject(ctx, authedRequest(t, tenantID, &projectv1.CreateProjectRequest{
		DisplayName:     "Secret Project",
		PostgresEnabled: true,
	}))
	if err != nil {
		t.Fatalf("CreateProject: %v", err)
	}
	projectID := createResp.Msg.Project.Id

	secClient := newSecretClient()

	resp, err := secClient.CreateSecret(ctx, authedRequest(t, tenantID, &secretv1.CreateSecretRequest{
		ProjectId:   projectID,
		Name:        "MY_SECRET",
		Value:       "super-secret-value",
		Description: "Test secret",
	}))
	if err != nil {
		t.Fatalf("CreateSecret: %v", err)
	}
	if resp.Msg.Secret == nil {
		t.Fatal("expected secret in response")
	}
	if resp.Msg.Secret.Name != "MY_SECRET" {
		t.Fatalf("got name %q, want %q", resp.Msg.Secret.Name, "MY_SECRET")
	}
}

func TestSecret_List(t *testing.T) {
	truncateAll(t)

	tenantID := uuid.New()
	createTestTenant(t, tenantID, "alice@example.com")

	projClient := newProjectClient()
	ctx := context.Background()

	createResp, err := projClient.CreateProject(ctx, authedRequest(t, tenantID, &projectv1.CreateProjectRequest{
		DisplayName:     "Secret List",
		PostgresEnabled: true,
	}))
	if err != nil {
		t.Fatalf("CreateProject: %v", err)
	}
	projectID := createResp.Msg.Project.Id

	secClient := newSecretClient()

	names := []string{"API_KEY", "DB_PASSWORD", "JWT_TOKEN"}
	for _, name := range names {
		_, err := secClient.CreateSecret(ctx, authedRequest(t, tenantID, &secretv1.CreateSecretRequest{
			ProjectId: projectID,
			Name:      name,
			Value:     "value-" + name,
		}))
		if err != nil {
			t.Fatalf("CreateSecret(%s): %v", name, err)
		}
	}

	listResp, err := secClient.ListSecrets(ctx, authedRequest(t, tenantID, &secretv1.ListSecretsRequest{
		ProjectId: projectID,
	}))
	if err != nil {
		t.Fatalf("ListSecrets: %v", err)
	}
	if len(listResp.Msg.Secrets) != 3 {
		t.Fatalf("got %d secrets, want 3", len(listResp.Msg.Secrets))
	}
}

func TestSecret_Delete(t *testing.T) {
	truncateAll(t)

	tenantID := uuid.New()
	createTestTenant(t, tenantID, "alice@example.com")

	projClient := newProjectClient()
	ctx := context.Background()

	createResp, err := projClient.CreateProject(ctx, authedRequest(t, tenantID, &projectv1.CreateProjectRequest{
		DisplayName:     "Secret Delete",
		PostgresEnabled: true,
	}))
	if err != nil {
		t.Fatalf("CreateProject: %v", err)
	}
	projectID := createResp.Msg.Project.Id

	secClient := newSecretClient()

	secResp, err := secClient.CreateSecret(ctx, authedRequest(t, tenantID, &secretv1.CreateSecretRequest{
		ProjectId: projectID,
		Name:      "DELETE_ME",
		Value:     "to-be-deleted",
	}))
	if err != nil {
		t.Fatalf("CreateSecret: %v", err)
	}
	secretID := secResp.Msg.Secret.Id

	_, err = secClient.DeleteSecret(ctx, authedRequest(t, tenantID, &secretv1.DeleteSecretRequest{
		ProjectId: projectID,
		SecretId:  secretID,
	}))
	if err != nil {
		t.Fatalf("DeleteSecret: %v", err)
	}

	// List should be empty.
	listResp, err := secClient.ListSecrets(ctx, authedRequest(t, tenantID, &secretv1.ListSecretsRequest{
		ProjectId: projectID,
	}))
	if err != nil {
		t.Fatalf("ListSecrets: %v", err)
	}
	if len(listResp.Msg.Secrets) != 0 {
		t.Fatalf("got %d secrets after delete, want 0", len(listResp.Msg.Secrets))
	}
}

func TestSecret_Create_DuplicateName(t *testing.T) {
	truncateAll(t)

	tenantID := uuid.New()
	createTestTenant(t, tenantID, "alice@example.com")
	projectID := createProjectForTest(t, tenantID)

	secClient := newSecretClient()
	ctx := context.Background()

	_, err := secClient.CreateSecret(ctx, authedRequest(t, tenantID, &secretv1.CreateSecretRequest{
		ProjectId: projectID,
		Name:      "SAME_NAME",
		Value:     "first",
	}))
	if err != nil {
		t.Fatalf("CreateSecret[1]: %v", err)
	}

	_, err = secClient.CreateSecret(ctx, authedRequest(t, tenantID, &secretv1.CreateSecretRequest{
		ProjectId: projectID,
		Name:      "SAME_NAME",
		Value:     "second",
	}))
	if err == nil {
		t.Fatal("expected error for duplicate secret name")
	}
	if code := connect.CodeOf(err); code != connect.CodeAlreadyExists {
		t.Fatalf("got code %v, want AlreadyExists", code)
	}
}
