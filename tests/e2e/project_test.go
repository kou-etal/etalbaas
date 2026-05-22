//go:build e2e

package e2e_test

import (
	"context"
	"testing"

	"connectrpc.com/connect"
	"github.com/google/uuid"

	projectv1 "github.com/kou-etal/etalbaas/proto/gen/go/etalbaas/project/v1"
)

func TestProject_Create(t *testing.T) {
	truncateAll(t)

	tenantID := uuid.New()
	createTestTenant(t, tenantID, "alice@example.com")

	client := newProjectClient()
	ctx := context.Background()

	resp, err := client.CreateProject(ctx, authedRequest(t, tenantID, &projectv1.CreateProjectRequest{
		DisplayName:     "My Project",
		Description:     "A test project",
		PostgresEnabled: true,
	}))
	if err != nil {
		t.Fatalf("CreateProject: %v", err)
	}

	project := resp.Msg.Project
	if project == nil {
		t.Fatal("expected project in response")
	}
	if len(project.Id) != 8 {
		t.Fatalf("expected 8-char project ID, got %q (len %d)", project.Id, len(project.Id))
	}
	if project.DisplayName != "My Project" {
		t.Fatalf("got display_name %q, want %q", project.DisplayName, "My Project")
	}
	if project.Status != "pending" {
		t.Fatalf("got status %q, want %q", project.Status, "pending")
	}
}

func TestProject_Get(t *testing.T) {
	truncateAll(t)

	tenantID := uuid.New()
	createTestTenant(t, tenantID, "alice@example.com")

	client := newProjectClient()
	ctx := context.Background()

	createResp, err := client.CreateProject(ctx, authedRequest(t, tenantID, &projectv1.CreateProjectRequest{
		DisplayName:     "Get Test",
		PostgresEnabled: true,
	}))
	if err != nil {
		t.Fatalf("CreateProject: %v", err)
	}
	projectID := createResp.Msg.Project.Id

	getResp, err := client.GetProject(ctx, authedRequest(t, tenantID, &projectv1.GetProjectRequest{
		ProjectId: projectID,
	}))
	if err != nil {
		t.Fatalf("GetProject: %v", err)
	}
	if getResp.Msg.Project.DisplayName != "Get Test" {
		t.Fatalf("got display_name %q, want %q", getResp.Msg.Project.DisplayName, "Get Test")
	}
}

func TestProject_List(t *testing.T) {
	truncateAll(t)

	tenantID := uuid.New()
	createTestTenant(t, tenantID, "alice@example.com")

	client := newProjectClient()
	ctx := context.Background()

	for i := 0; i < 3; i++ {
		_, err := client.CreateProject(ctx, authedRequest(t, tenantID, &projectv1.CreateProjectRequest{
			DisplayName:     "Project " + uuid.NewString()[:4],
			PostgresEnabled: true,
		}))
		if err != nil {
			t.Fatalf("CreateProject[%d]: %v", i, err)
		}
	}

	listResp, err := client.ListProjects(ctx, authedRequest(t, tenantID, &projectv1.ListProjectsRequest{}))
	if err != nil {
		t.Fatalf("ListProjects: %v", err)
	}
	if len(listResp.Msg.Projects) != 3 {
		t.Fatalf("got %d projects, want 3", len(listResp.Msg.Projects))
	}
}

func TestProject_Delete(t *testing.T) {
	truncateAll(t)

	tenantID := uuid.New()
	createTestTenant(t, tenantID, "alice@example.com")

	client := newProjectClient()
	ctx := context.Background()

	createResp, err := client.CreateProject(ctx, authedRequest(t, tenantID, &projectv1.CreateProjectRequest{
		DisplayName:     "To Delete",
		PostgresEnabled: true,
	}))
	if err != nil {
		t.Fatalf("CreateProject: %v", err)
	}
	projectID := createResp.Msg.Project.Id

	_, err = client.DeleteProject(ctx, authedRequest(t, tenantID, &projectv1.DeleteProjectRequest{
		ProjectId: projectID,
	}))
	if err != nil {
		t.Fatalf("DeleteProject: %v", err)
	}

	// Should not appear in list after delete.
	listResp, err := client.ListProjects(ctx, authedRequest(t, tenantID, &projectv1.ListProjectsRequest{}))
	if err != nil {
		t.Fatalf("ListProjects: %v", err)
	}
	if len(listResp.Msg.Projects) != 0 {
		t.Fatalf("got %d projects after delete, want 0", len(listResp.Msg.Projects))
	}
}

func TestProject_Create_ValidationError(t *testing.T) {
	truncateAll(t)

	tenantID := uuid.New()
	createTestTenant(t, tenantID, "alice@example.com")

	client := newProjectClient()
	ctx := context.Background()

	// Empty display_name should fail.
	_, err := client.CreateProject(ctx, authedRequest(t, tenantID, &projectv1.CreateProjectRequest{
		DisplayName: "",
	}))
	if err == nil {
		t.Fatal("expected error for empty display_name")
	}
	if code := connect.CodeOf(err); code != connect.CodeInvalidArgument {
		t.Fatalf("got code %v, want InvalidArgument", code)
	}
}

func TestProject_Get_NotFound(t *testing.T) {
	truncateAll(t)

	tenantID := uuid.New()
	createTestTenant(t, tenantID, "alice@example.com")

	client := newProjectClient()
	ctx := context.Background()

	_, err := client.GetProject(ctx, authedRequest(t, tenantID, &projectv1.GetProjectRequest{
		ProjectId: "nonexist",
	}))
	if err == nil {
		t.Fatal("expected error for nonexistent project")
	}
	if code := connect.CodeOf(err); code != connect.CodeNotFound {
		t.Fatalf("got code %v, want NotFound", code)
	}
}
