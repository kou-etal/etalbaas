//go:build e2e

package e2e_test

import (
	"context"
	"testing"

	"connectrpc.com/connect"
	"github.com/google/uuid"

	functionv1 "github.com/kou-etal/etalbaas/proto/gen/go/etalbaas/function/v1"
	projectv1 "github.com/kou-etal/etalbaas/proto/gen/go/etalbaas/project/v1"
)

func TestIsolation_ProjectNotVisible(t *testing.T) {
	truncateAll(t)

	tenantA := uuid.New()
	tenantB := uuid.New()
	createTestTenant(t, tenantA, "alice@example.com")
	createTestTenant(t, tenantB, "bob@example.com")

	client := newProjectClient()
	ctx := context.Background()

	// Tenant A creates a project.
	resp, err := client.CreateProject(ctx, authedRequest(t, tenantA, &projectv1.CreateProjectRequest{
		DisplayName:     "Alice's Project",
		PostgresEnabled: true,
	}))
	if err != nil {
		t.Fatalf("CreateProject: %v", err)
	}
	projectID := resp.Msg.Project.Id

	// Tenant B cannot access it.
	_, err = client.GetProject(ctx, authedRequest(t, tenantB, &projectv1.GetProjectRequest{
		ProjectId: projectID,
	}))
	if err == nil {
		t.Fatal("expected error when Tenant B accesses Tenant A's project")
	}
	if code := connect.CodeOf(err); code != connect.CodeNotFound {
		t.Fatalf("got code %v, want NotFound", code)
	}
}

func TestIsolation_ProjectListFiltered(t *testing.T) {
	truncateAll(t)

	tenantA := uuid.New()
	tenantB := uuid.New()
	createTestTenant(t, tenantA, "alice@example.com")
	createTestTenant(t, tenantB, "bob@example.com")

	client := newProjectClient()
	ctx := context.Background()

	// Tenant A creates 2 projects.
	for i := 0; i < 2; i++ {
		_, err := client.CreateProject(ctx, authedRequest(t, tenantA, &projectv1.CreateProjectRequest{
			DisplayName:     "Alice " + uuid.NewString()[:4],
			PostgresEnabled: true,
		}))
		if err != nil {
			t.Fatalf("CreateProject[A]: %v", err)
		}
	}

	// Tenant B creates 1 project.
	_, err := client.CreateProject(ctx, authedRequest(t, tenantB, &projectv1.CreateProjectRequest{
		DisplayName:     "Bob's Project",
		PostgresEnabled: true,
	}))
	if err != nil {
		t.Fatalf("CreateProject[B]: %v", err)
	}

	// Tenant B sees only their own project.
	listResp, err := client.ListProjects(ctx, authedRequest(t, tenantB, &projectv1.ListProjectsRequest{}))
	if err != nil {
		t.Fatalf("ListProjects: %v", err)
	}
	if len(listResp.Msg.Projects) != 1 {
		t.Fatalf("Tenant B sees %d projects, want 1", len(listResp.Msg.Projects))
	}

	// Tenant A sees only their own 2 projects.
	listRespA, err := client.ListProjects(ctx, authedRequest(t, tenantA, &projectv1.ListProjectsRequest{}))
	if err != nil {
		t.Fatalf("ListProjects[A]: %v", err)
	}
	if len(listRespA.Msg.Projects) != 2 {
		t.Fatalf("Tenant A sees %d projects, want 2", len(listRespA.Msg.Projects))
	}
}

func TestIsolation_FunctionNotVisible(t *testing.T) {
	truncateAll(t)

	tenantA := uuid.New()
	tenantB := uuid.New()
	createTestTenant(t, tenantA, "alice@example.com")
	createTestTenant(t, tenantB, "bob@example.com")

	ctx := context.Background()
	projectID := createProjectForTest(t, tenantA)

	fnClient := newFunctionClient()
	fnResp, err := fnClient.CreateFunction(ctx, authedRequest(t, tenantA, &functionv1.CreateFunctionRequest{
		ProjectId:   projectID,
		Name:        "secret-func",
		DisplayName: "Secret",
		Kind:        "light-deployment",
		Mode:        "sync",
		Source:      &functionv1.CreateFunctionRequest_InlineSource{InlineSource: &functionv1.InlineSource{Code: "code"}},
		Runtime:     &functionv1.CreateFunctionRequest_PresetRuntime{PresetRuntime: &functionv1.PresetRuntime{Preset: "python-3.11"}},
	}))
	if err != nil {
		t.Fatalf("CreateFunction: %v", err)
	}
	fnID := fnResp.Msg.Function.Id

	// Tenant B cannot access it.
	_, err = fnClient.GetFunction(ctx, authedRequest(t, tenantB, &functionv1.GetFunctionRequest{
		FunctionId: fnID,
	}))
	if err == nil {
		t.Fatal("expected error when Tenant B accesses Tenant A's function")
	}
	if code := connect.CodeOf(err); code != connect.CodeNotFound {
		t.Fatalf("got code %v, want NotFound", code)
	}
}

func TestIsolation_DeleteCrossTenant(t *testing.T) {
	truncateAll(t)

	tenantA := uuid.New()
	tenantB := uuid.New()
	createTestTenant(t, tenantA, "alice@example.com")
	createTestTenant(t, tenantB, "bob@example.com")

	client := newProjectClient()
	ctx := context.Background()

	resp, err := client.CreateProject(ctx, authedRequest(t, tenantA, &projectv1.CreateProjectRequest{
		DisplayName:     "Protected",
		PostgresEnabled: true,
	}))
	if err != nil {
		t.Fatalf("CreateProject: %v", err)
	}
	projectID := resp.Msg.Project.Id

	// Tenant B cannot delete Tenant A's project.
	_, err = client.DeleteProject(ctx, authedRequest(t, tenantB, &projectv1.DeleteProjectRequest{
		ProjectId: projectID,
	}))
	if err == nil {
		t.Fatal("expected error when Tenant B deletes Tenant A's project")
	}
	if code := connect.CodeOf(err); code != connect.CodeNotFound {
		t.Fatalf("got code %v, want NotFound", code)
	}
}
