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

func (s *StaticProvider) WithRLS(ctx context.Context, _ string, _ json.RawMessage, fn func(pgx.Tx) error) error {
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return fmt.Errorf("begin tx: %w", err)
	}
	defer tx.Rollback(ctx) //nolint:errcheck

	if err := fn(tx); err != nil {
		return err
	}
	return tx.Commit(ctx)
}

func (s *StaticProvider) Close() {
	// Static pool lifecycle is managed externally.
}
