//go:build e2e

package e2e_test

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"os/exec"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
)

var (
	tenantUserURL string
	projectURL    string
	functionURL   string
	eventURL      string
	storageURL    string
	dbPool        *pgxpool.Pool
)

func TestMain(m *testing.M) {
	// Start compose unless E2E_SKIP_COMPOSE is set (CI pre-starts it).
	if os.Getenv("E2E_SKIP_COMPOSE") == "" {
		cmd := exec.Command("docker", "compose", "-f", "docker-compose.yml", "up",
			"--build", "--wait", "--wait-timeout", "180")
		cmd.Stdout = os.Stdout
		cmd.Stderr = os.Stderr
		if err := cmd.Run(); err != nil {
			fmt.Fprintf(os.Stderr, "docker compose up failed: %v\n", err)
			os.Exit(1)
		}
		defer func() {
			down := exec.Command("docker", "compose", "-f", "docker-compose.yml", "down", "-v")
			down.Stdout = os.Stdout
			down.Stderr = os.Stderr
			_ = down.Run()
		}()
	}

	// Set service URLs from env or defaults.
	tenantUserURL = envOrDefault("TENANT_USER_URL", "http://localhost:8081")
	projectURL = envOrDefault("PROJECT_URL", "http://localhost:8082")
	functionURL = envOrDefault("FUNCTION_URL", "http://localhost:8083")
	eventURL = envOrDefault("EVENT_URL", "http://localhost:8084")
	storageURL = envOrDefault("STORAGE_URL", "http://localhost:8085")

	// Wait for all services to be healthy.
	ctx, cancel := context.WithTimeout(context.Background(), 90*time.Second)
	defer cancel()
	if err := waitForServices(ctx); err != nil {
		fmt.Fprintf(os.Stderr, "services not ready: %v\n", err)
		os.Exit(1)
	}

	// Connect to DB for test setup/teardown.
	dbURL := envOrDefault("DATABASE_URL",
		"postgres://etalbaas:etalbaas@127.0.0.1:5433/etalbaas_meta?sslmode=disable")
	pool, err := pgxpool.New(ctx, dbURL)
	if err != nil {
		fmt.Fprintf(os.Stderr, "db connect failed: %v\n", err)
		os.Exit(1)
	}
	dbPool = pool

	// Ensure storage schema exists (needed for storage service when K8S_ENABLED=false,
	// as StaticProvider uses meta DB which doesn't have the storage schema by default).
	if err := ensureStorageSchema(ctx, pool); err != nil {
		fmt.Fprintf(os.Stderr, "ensure storage schema failed: %v\n", err)
		os.Exit(1)
	}

	code := m.Run()
	pool.Close()
	os.Exit(code)
}

func ensureStorageSchema(ctx context.Context, pool *pgxpool.Pool) error {
	ddl := `
CREATE SCHEMA IF NOT EXISTS storage;

CREATE TABLE IF NOT EXISTS storage.buckets (
    id TEXT PRIMARY KEY,
    name TEXT NOT NULL,
    project_id TEXT NOT NULL,
    access_level TEXT NOT NULL DEFAULT 'protected',
    file_size_limit BIGINT,
    allowed_mime_types TEXT[],
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    UNIQUE(project_id, name)
);

CREATE TABLE IF NOT EXISTS storage.objects (
    id UUID DEFAULT gen_random_uuid() PRIMARY KEY,
    bucket_id TEXT NOT NULL REFERENCES storage.buckets(id) ON DELETE CASCADE,
    name TEXT NOT NULL,
    owner UUID,
    size BIGINT,
    mime_type TEXT,
    etag TEXT,
    metadata JSONB DEFAULT '{}',
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    UNIQUE(bucket_id, name)
);
`
	_, err := pool.Exec(ctx, ddl)
	return err
}

func waitForServices(ctx context.Context) error {
	endpoints := []string{
		tenantUserURL + "/healthz",
		projectURL + "/healthz",
		functionURL + "/healthz",
		eventURL + "/healthz",
		storageURL + "/healthz",
	}
	for _, ep := range endpoints {
		if err := pollHTTP(ctx, ep); err != nil {
			return fmt.Errorf("%s: %w", ep, err)
		}
	}
	return nil
}

func pollHTTP(ctx context.Context, url string) error {
	client := &http.Client{Timeout: 2 * time.Second}
	for {
		select {
		case <-ctx.Done():
			return fmt.Errorf("timeout waiting for %s", url)
		default:
		}
		resp, err := client.Get(url)
		if err == nil && resp.StatusCode == http.StatusOK {
			resp.Body.Close()
			return nil
		}
		if resp != nil {
			resp.Body.Close()
		}
		time.Sleep(500 * time.Millisecond)
	}
}

func envOrDefault(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}
