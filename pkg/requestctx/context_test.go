package requestctx_test

import (
	"context"
	"testing"

	"github.com/google/uuid"

	"github.com/kou-etal/etalbaas/pkg/requestctx"
)

func TestSetAndGetUserID(t *testing.T) {
	ctx := context.Background()

	_, err := requestctx.UserID(ctx)
	if err == nil {
		t.Fatal("expected error for empty context")
	}

	id := uuid.MustParse("550e8400-e29b-41d4-a716-446655440000")
	ctx = requestctx.WithUserID(ctx, id)
	got, err := requestctx.UserID(ctx)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got != id {
		t.Fatalf("got %v, want %v", got, id)
	}
}

func TestSetAndGetProjectID(t *testing.T) {
	ctx := context.Background()

	_, err := requestctx.ProjectID(ctx)
	if err == nil {
		t.Fatal("expected error for empty context")
	}

	ctx = requestctx.WithProjectID(ctx, "proj-abc")
	got, err := requestctx.ProjectID(ctx)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got != "proj-abc" {
		t.Fatalf("got %q, want %q", got, "proj-abc")
	}
}

func TestNilUUIDReturnsError(t *testing.T) {
	ctx := requestctx.WithUserID(context.Background(), uuid.Nil)
	_, err := requestctx.UserID(ctx)
	if err == nil {
		t.Fatal("expected error for nil UUID")
	}
}
