package pool

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

// StaticProvider uses a single pre-configured pool for all projects.
// Used when K8S_ENABLED=false (no per-project tenant DB routing).
type StaticProvider struct {
	pool *pgxpool.Pool
}

// NewStaticProvider creates a Provider backed by a single static pool.
func NewStaticProvider(pool *pgxpool.Pool) *StaticProvider {
	return &StaticProvider{pool: pool}
}

func (s *StaticProvider) GetPool(_ context.Context, _ string) (*pgxpool.Pool, error) {
	return s.pool, nil
}

func (s *StaticProvider) WithRLS(ctx context.Context, _ string, claims json.RawMessage, fn func(pgx.Tx) error) error {
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return fmt.Errorf("begin tx: %w", err)
	}
	defer tx.Rollback(ctx) //nolint:errcheck

	// Apply RLS session variables even in static mode to enforce row-level security.
	if _, err := tx.Exec(ctx, "SELECT set_config('request.jwt.claims', $1, true)", string(claims)); err != nil {
		return fmt.Errorf("set jwt claims: %w", err)
	}

	role := "authenticated"
	var parsed struct {
		Role string `json:"role"`
	}
	if json.Unmarshal(claims, &parsed) == nil && parsed.Role != "" {
		role = parsed.Role
	}
	if role != "anon" && role != "authenticated" && role != "service_role" {
		role = "authenticated"
	}
	if _, err := tx.Exec(ctx, "SET LOCAL ROLE "+role); err != nil {
		return fmt.Errorf("set role: %w", err)
	}

	if err := fn(tx); err != nil {
		return err
	}
	return tx.Commit(ctx)
}

func (s *StaticProvider) Close() {
	// Static pool lifecycle is managed externally.
}
