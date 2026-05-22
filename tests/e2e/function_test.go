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

// createProjectForTest creates a project and returns its ID.
func createProjectForTest(t *testing.T, tenantID uuid.UUID) string {
	t.Helper()
	client := newProjectClient()
	resp, err := client.CreateProject(context.Background(), authedRequest(t, tenantID, &projectv1.CreateProjectRequest{
		DisplayName:     "Test Project " + uuid.NewString()[:4],
		PostgresEnabled: true,
	}))
	if err != nil {
		t.Fatalf("CreateProject: %v", err)
	}
	return resp.Msg.Project.Id
}

func TestFunction_Create(t *testing.T) {
	truncateAll(t)

	tenantID := uuid.New()
	createTestTenant(t, tenantID, "alice@example.com")
	projectID := createProjectForTest(t, tenantID)

	client := newFunctionClient()
	ctx := context.Background()

	resp, err := client.CreateFunction(ctx, authedRequest(t, tenantID, &functionv1.CreateFunctionRequest{
		ProjectId:   projectID,
		Name:        "my-func",
		DisplayName: "My Function",
		Kind:        "light-deployment",
		Mode:        "sync",
		Source:      &functionv1.CreateFunctionRequest_InlineSource{InlineSource: &functionv1.InlineSource{Code: "print('hello')"}},
		Runtime:     &functionv1.CreateFunctionRequest_PresetRuntime{PresetRuntime: &functionv1.PresetRuntime{Preset: "python-3.11"}},
	}))
	if err != nil {
		t.Fatalf("CreateFunction: %v", err)
	}

	fn := resp.Msg.Function
	if fn == nil {
		t.Fatal("expected function in response")
	}
	if fn.Name != "my-func" {
		t.Fatalf("got name %q, want %q", fn.Name, "my-func")
	}
	if fn.Status != "pending" {
		t.Fatalf("got status %q, want %q", fn.Status, "pending")
	}
}

func TestFunction_Get(t *testing.T) {
	truncateAll(t)

	tenantID := uuid.New()
	createTestTenant(t, tenantID, "alice@example.com")
	projectID := createProjectForTest(t, tenantID)

	client := newFunctionClient()
	ctx := context.Background()

	createResp, err := client.CreateFunction(ctx, authedRequest(t, tenantID, &functionv1.CreateFunctionRequest{
		ProjectId:   projectID,
		Name:        "get-func",
		DisplayName: "Get Func",
		Kind:        "heavy-deployment",
		Mode:        "async",
		Source:      &functionv1.CreateFunctionRequest_InlineSource{InlineSource: &functionv1.InlineSource{Code: "def handler(): pass"}},
		Runtime:     &functionv1.CreateFunctionRequest_PresetRuntime{PresetRuntime: &functionv1.PresetRuntime{Preset: "python-3.11"}},
	}))
	if err != nil {
		t.Fatalf("CreateFunction: %v", err)
	}
	fnID := createResp.Msg.Function.Id

	getResp, err := client.GetFunction(ctx, authedRequest(t, tenantID, &functionv1.GetFunctionRequest{
		ProjectId:  projectID,
		FunctionId: fnID,
	}))
	if err != nil {
		t.Fatalf("GetFunction: %v", err)
	}
	if getResp.Msg.Function.DisplayName != "Get Func" {
		t.Fatalf("got display_name %q, want %q", getResp.Msg.Function.DisplayName, "Get Func")
	}
}

func TestFunction_List(t *testing.T) {
	truncateAll(t)

	tenantID := uuid.New()
	createTestTenant(t, tenantID, "alice@example.com")
	projectID := createProjectForTest(t, tenantID)

	client := newFunctionClient()
	ctx := context.Background()

	for i := 0; i < 3; i++ {
		_, err := client.CreateFunction(ctx, authedRequest(t, tenantID, &functionv1.CreateFunctionRequest{
			ProjectId:   projectID,
			Name:        "func-" + uuid.NewString()[:4],
			DisplayName: "Function",
			Kind:        "light-deployment",
			Mode:        "sync",
			Source:      &functionv1.CreateFunctionRequest_InlineSource{InlineSource: &functionv1.InlineSource{Code: "code"}},
			Runtime:     &functionv1.CreateFunctionRequest_PresetRuntime{PresetRuntime: &functionv1.PresetRuntime{Preset: "node-20"}},
		}))
		if err != nil {
			t.Fatalf("CreateFunction[%d]: %v", i, err)
		}
	}

	listResp, err := client.ListFunctions(ctx, authedRequest(t, tenantID, &functionv1.ListFunctionsRequest{
		ProjectId: projectID,
	}))
	if err != nil {
		t.Fatalf("ListFunctions: %v", err)
	}
	if len(listResp.Msg.Functions) != 3 {
		t.Fatalf("got %d functions, want 3", len(listResp.Msg.Functions))
	}
}

func TestFunction_Delete(t *testing.T) {
	truncateAll(t)

	tenantID := uuid.New()
	createTestTenant(t, tenantID, "alice@example.com")
	projectID := createProjectForTest(t, tenantID)

	client := newFunctionClient()
	ctx := context.Background()

	createResp, err := client.CreateFunction(ctx, authedRequest(t, tenantID, &functionv1.CreateFunctionRequest{
		ProjectId:   projectID,
		Name:        "del-func",
		DisplayName: "Delete Me",
		Kind:        "light-deployment",
		Mode:        "sync",
		Source:      &functionv1.CreateFunctionRequest_InlineSource{InlineSource: &functionv1.InlineSource{Code: "code"}},
		Runtime:     &functionv1.CreateFunctionRequest_PresetRuntime{PresetRuntime: &functionv1.PresetRuntime{Preset: "python-3.11"}},
	}))
	if err != nil {
		t.Fatalf("CreateFunction: %v", err)
	}
	fnID := createResp.Msg.Function.Id

	_, err = client.DeleteFunction(ctx, authedRequest(t, tenantID, &functionv1.DeleteFunctionRequest{
		ProjectId:  projectID,
		FunctionId: fnID,
	}))
	if err != nil {
		t.Fatalf("DeleteFunction: %v", err)
	}

	// Should not appear in list.
	listResp, err := client.ListFunctions(ctx, authedRequest(t, tenantID, &functionv1.ListFunctionsRequest{
		ProjectId: projectID,
	}))
	if err != nil {
		t.Fatalf("ListFunctions: %v", err)
	}
	if len(listResp.Msg.Functions) != 0 {
		t.Fatalf("got %d functions after delete, want 0", len(listResp.Msg.Functions))
	}
}

func TestFunction_Create_ValidationError(t *testing.T) {
	truncateAll(t)

	tenantID := uuid.New()
	createTestTenant(t, tenantID, "alice@example.com")
	projectID := createProjectForTest(t, tenantID)

	client := newFunctionClient()
	ctx := context.Background()

	// Name too short.
	_, err := client.CreateFunction(ctx, authedRequest(t, tenantID, &functionv1.CreateFunctionRequest{
		ProjectId:   projectID,
		Name:        "ab",
		DisplayName: "Too Short",
		Kind:        "light-deployment",
		Mode:        "sync",
		Source:      &functionv1.CreateFunctionRequest_InlineSource{InlineSource: &functionv1.InlineSource{Code: "code"}},
		Runtime:     &functionv1.CreateFunctionRequest_PresetRuntime{PresetRuntime: &functionv1.PresetRuntime{Preset: "python-3.11"}},
	}))
	if err == nil {
		t.Fatal("expected error for short name")
	}
	if code := connect.CodeOf(err); code != connect.CodeInvalidArgument {
		t.Fatalf("got code %v, want InvalidArgument", code)
	}
}

func TestFunction_Create_DuplicateName(t *testing.T) {
	truncateAll(t)

	tenantID := uuid.New()
	createTestTenant(t, tenantID, "alice@example.com")
	projectID := createProjectForTest(t, tenantID)

	client := newFunctionClient()
	ctx := context.Background()

	req := &functionv1.CreateFunctionRequest{
		ProjectId:   projectID,
		Name:        "dup-func",
		DisplayName: "Duplicate",
		Kind:        "light-deployment",
		Mode:        "sync",
		Source:      &functionv1.CreateFunctionRequest_InlineSource{InlineSource: &functionv1.InlineSource{Code: "code"}},
		Runtime:     &functionv1.CreateFunctionRequest_PresetRuntime{PresetRuntime: &functionv1.PresetRuntime{Preset: "python-3.11"}},
	}

	_, err := client.CreateFunction(ctx, authedRequest(t, tenantID, req))
	if err != nil {
		t.Fatalf("CreateFunction[1]: %v", err)
	}

	_, err = client.CreateFunction(ctx, authedRequest(t, tenantID, req))
	if err == nil {
		t.Fatal("expected error for duplicate name")
	}
	if code := connect.CodeOf(err); code != connect.CodeAlreadyExists {
		t.Fatalf("got code %v, want AlreadyExists", code)
	}
}
