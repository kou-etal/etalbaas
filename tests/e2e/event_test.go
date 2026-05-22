//go:build e2e

package e2e_test

import (
	"context"
	"testing"

	"github.com/google/uuid"

	eventv1 "github.com/kou-etal/etalbaas/proto/gen/go/etalbaas/event/v1"
	functionv1 "github.com/kou-etal/etalbaas/proto/gen/go/etalbaas/function/v1"
)

func TestEvent_ListHistory_Empty(t *testing.T) {
	truncateAll(t)

	tenantID := uuid.New()
	createTestTenant(t, tenantID, "alice@example.com")
	projectID := createProjectForTest(t, tenantID)

	// Create a function (required for ListEventHistory which requires a valid function_id).
	fnClient := newFunctionClient()
	ctx := context.Background()

	fnResp, err := fnClient.CreateFunction(ctx, authedRequest(t, tenantID, &functionv1.CreateFunctionRequest{
		ProjectId:   projectID,
		Name:        "event-empty-func",
		DisplayName: "Event Empty Func",
		Kind:        "light-deployment",
		Mode:        "sync",
		Source:      &functionv1.CreateFunctionRequest_InlineSource{InlineSource: &functionv1.InlineSource{Code: "code"}},
		Runtime:     &functionv1.CreateFunctionRequest_PresetRuntime{PresetRuntime: &functionv1.PresetRuntime{Preset: "python-3.11"}},
	}))
	if err != nil {
		t.Fatalf("CreateFunction: %v", err)
	}
	fnID := fnResp.Msg.Function.Id

	client := newEventClient()

	resp, err := client.ListEventHistory(ctx, authedRequest(t, tenantID, &eventv1.ListEventHistoryRequest{
		ProjectId:  projectID,
		FunctionId: fnID,
	}))
	if err != nil {
		t.Fatalf("ListEventHistory: %v", err)
	}
	if len(resp.Msg.Events) != 0 {
		t.Fatalf("got %d events, want 0", len(resp.Msg.Events))
	}
}

func TestEvent_ListHistory_WithData(t *testing.T) {
	truncateAll(t)

	tenantID := uuid.New()
	createTestTenant(t, tenantID, "alice@example.com")
	projectID := createProjectForTest(t, tenantID)

	// Create a function (required FK for event_history).
	fnClient := newFunctionClient()
	ctx := context.Background()

	fnResp, err := fnClient.CreateFunction(ctx, authedRequest(t, tenantID, &functionv1.CreateFunctionRequest{
		ProjectId:   projectID,
		Name:        "event-func",
		DisplayName: "Event Func",
		Kind:        "light-deployment",
		Mode:        "sync",
		Source:      &functionv1.CreateFunctionRequest_InlineSource{InlineSource: &functionv1.InlineSource{Code: "code"}},
		Runtime:     &functionv1.CreateFunctionRequest_PresetRuntime{PresetRuntime: &functionv1.PresetRuntime{Preset: "python-3.11"}},
	}))
	if err != nil {
		t.Fatalf("CreateFunction: %v", err)
	}
	fnID := fnResp.Msg.Function.Id

	// Insert test event data directly into DB.
	eventID := uuid.New()
	_, err = dbPool.Exec(ctx,
		`INSERT INTO event_history (id, project_id, function_id, trigger_type, trigger_data, status)
		 VALUES ($1, $2, $3, 'database_change', '{"table":"users","operation":"INSERT"}', 'delivered')`,
		eventID, projectID, fnID)
	if err != nil {
		t.Fatalf("insert event_history: %v", err)
	}

	evClient := newEventClient()
	resp, err := evClient.ListEventHistory(ctx, authedRequest(t, tenantID, &eventv1.ListEventHistoryRequest{
		ProjectId:  projectID,
		FunctionId: fnID,
	}))
	if err != nil {
		t.Fatalf("ListEventHistory: %v", err)
	}
	if len(resp.Msg.Events) != 1 {
		t.Fatalf("got %d events, want 1", len(resp.Msg.Events))
	}
	if resp.Msg.Events[0].Status != "delivered" {
		t.Fatalf("got status %q, want %q", resp.Msg.Events[0].Status, "delivered")
	}
}
