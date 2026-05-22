//go:build e2e

package e2e_test

import (
	"context"
	"net/http"
	"testing"
	"time"

	"connectrpc.com/connect"
	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"

	"github.com/kou-etal/etalbaas/proto/gen/go/etalbaas/event/v1/eventv1connect"
	"github.com/kou-etal/etalbaas/proto/gen/go/etalbaas/function/v1/functionv1connect"
	"github.com/kou-etal/etalbaas/proto/gen/go/etalbaas/project/v1/projectv1connect"
	"github.com/kou-etal/etalbaas/proto/gen/go/etalbaas/secret/v1/secretv1connect"
	"github.com/kou-etal/etalbaas/proto/gen/go/etalbaas/storage/v1/storagev1connect"
	"github.com/kou-etal/etalbaas/proto/gen/go/etalbaas/tenant/v1/tenantv1connect"
)

const jwtSecret = "e2e-test-secret"

// generateJWT creates a valid JWT for a given user ID.
func generateJWT(userID uuid.UUID) string {
	claims := jwt.MapClaims{
		"sub": userID.String(),
		"iat": time.Now().Unix(),
		"exp": time.Now().Add(1 * time.Hour).Unix(),
	}
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	tokenStr, _ := token.SignedString([]byte(jwtSecret))
	return tokenStr
}

// authedRequest creates a connect.Request with X-User-ID header set.
// Downstream services trust this header (set by gateway in production).
func authedRequest[T any](t *testing.T, userID uuid.UUID, msg *T) *connect.Request[T] {
	t.Helper()
	req := connect.NewRequest(msg)
	req.Header().Set("X-User-ID", userID.String())
	return req
}

// --- Client factories ---

func newTenantClient() tenantv1connect.TenantServiceClient {
	return tenantv1connect.NewTenantServiceClient(http.DefaultClient, tenantUserURL)
}

func newProjectClient() projectv1connect.ProjectServiceClient {
	return projectv1connect.NewProjectServiceClient(http.DefaultClient, projectURL)
}

func newSecretClient() secretv1connect.SecretServiceClient {
	return secretv1connect.NewSecretServiceClient(http.DefaultClient, projectURL)
}

func newFunctionClient() functionv1connect.FunctionServiceClient {
	return functionv1connect.NewFunctionServiceClient(http.DefaultClient, functionURL)
}

func newEventClient() eventv1connect.EventServiceClient {
	return eventv1connect.NewEventServiceClient(http.DefaultClient, eventURL)
}

func newStorageClient() storagev1connect.StorageServiceClient {
	return storagev1connect.NewStorageServiceClient(http.DefaultClient, storageURL)
}

// --- DB helpers ---

// createTestTenant inserts a tenant directly into the DB.
func createTestTenant(t *testing.T, id uuid.UUID, email string) {
	t.Helper()
	_, err := dbPool.Exec(context.Background(),
		`INSERT INTO tenants (id, email, display_name, plan, status)
		 VALUES ($1, $2, $3, 'free', 'active')
		 ON CONFLICT (id) DO NOTHING`,
		id, email, "Test User "+id.String()[:8])
	if err != nil {
		t.Fatalf("create test tenant: %v", err)
	}
}

// truncateAll removes all data from all tables (respecting FK order).
func truncateAll(t *testing.T) {
	t.Helper()
	tables := []string{
		"storage.objects",
		"storage.buckets",
		"event_history",
		"invocations",
		"functions",
		"secrets_metadata",
		"api_keys",
		"usage_daily",
		"platform_events",
		"projects",
		"tenants",
	}
	for _, tbl := range tables {
		_, err := dbPool.Exec(context.Background(), "DELETE FROM "+tbl)
		if err != nil {
			t.Fatalf("delete from %s: %v", tbl, err)
		}
	}
}

// pollUntil polls fn until it returns true or timeout is reached.
func pollUntil(t *testing.T, timeout time.Duration, interval time.Duration, fn func() bool) {
	t.Helper()
	deadline := time.Now().Add(timeout)
	for time.Now().Before(deadline) {
		if fn() {
			return
		}
		time.Sleep(interval)
	}
	t.Fatal("poll timeout exceeded")
}
