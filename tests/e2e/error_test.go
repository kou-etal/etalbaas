//go:build e2e

package e2e_test

import (
	"context"
	"testing"

	"connectrpc.com/connect"

	projectv1 "github.com/kou-etal/etalbaas/proto/gen/go/etalbaas/project/v1"
)

func TestError_Unauthenticated(t *testing.T) {
	client := newProjectClient()
	ctx := context.Background()

	// Request without X-User-ID header.
	req := connect.NewRequest(&projectv1.ListProjectsRequest{})
	_, err := client.ListProjects(ctx, req)
	if err == nil {
		t.Fatal("expected error for unauthenticated request")
	}
	if code := connect.CodeOf(err); code != connect.CodeUnauthenticated {
		t.Fatalf("got code %v, want Unauthenticated", code)
	}
}

func TestError_InvalidUserID(t *testing.T) {
	client := newProjectClient()
	ctx := context.Background()

	// Request with invalid (non-UUID) X-User-ID header.
	req := connect.NewRequest(&projectv1.ListProjectsRequest{})
	req.Header().Set("X-User-ID", "not-a-uuid")
	_, err := client.ListProjects(ctx, req)
	if err == nil {
		t.Fatal("expected error for invalid user ID")
	}
	if code := connect.CodeOf(err); code != connect.CodeUnauthenticated {
		t.Fatalf("got code %v, want Unauthenticated", code)
	}
}

func TestError_ProjectQuota(t *testing.T) {
	t.Skip("project quota enforcement is not implemented in Phase 1")
}

