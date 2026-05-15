package requestctx_test

import (
	"context"
	"testing"

	"github.com/kou-etal/etalbaas/pkg/requestctx"
)

func TestSetAndGetUserID(t *testing.T) {
	ctx := context.Background()

	_, err := requestctx.UserID(ctx)
	if err == nil {
		t.Fatal("expected error for empty context")
	}

	ctx = requestctx.WithUserID(ctx, "user-123")
	got, err := requestctx.UserID(ctx)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got != "user-123" {
		t.Fatalf("got %q, want %q", got, "user-123")
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

func TestEmptyStringReturnsError(t *testing.T) {
	ctx := requestctx.WithUserID(context.Background(), "")
	_, err := requestctx.UserID(ctx)
	if err == nil {
		t.Fatal("expected error for empty string")
	}
}
