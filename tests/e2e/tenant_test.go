//go:build e2e

package e2e_test

import (
	"context"
	"testing"

	"connectrpc.com/connect"
	"github.com/google/uuid"

	tenantv1 "github.com/kou-etal/etalbaas/proto/gen/go/etalbaas/tenant/v1"
)

func TestTenant_GetMe(t *testing.T) {
	truncateAll(t)

	tenantID := uuid.New()
	createTestTenant(t, tenantID, "alice@example.com")

	client := newTenantClient()
	ctx := context.Background()

	resp, err := client.GetMe(ctx, authedRequest(t, tenantID, &tenantv1.GetMeRequest{}))
	if err != nil {
		t.Fatalf("GetMe: %v", err)
	}
	if resp.Msg.Tenant == nil {
		t.Fatal("expected tenant in response")
	}
	if resp.Msg.Tenant.Email != "alice@example.com" {
		t.Fatalf("got email %q, want %q", resp.Msg.Tenant.Email, "alice@example.com")
	}
}

func TestTenant_GetMe_NotFound(t *testing.T) {
	truncateAll(t)

	nonexistentID := uuid.New()
	client := newTenantClient()
	ctx := context.Background()

	_, err := client.GetMe(ctx, authedRequest(t, nonexistentID, &tenantv1.GetMeRequest{}))
	if err == nil {
		t.Fatal("expected error for nonexistent tenant")
	}
	if code := connect.CodeOf(err); code != connect.CodeNotFound {
		t.Fatalf("got code %v, want NotFound", code)
	}
}
