package pool

import (
	"context"
	"encoding/json"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

// Provider is the interface for accessing per-project DB pools.
type Provider interface {
	GetPool(ctx context.Context, projectID string) (*pgxpool.Pool, error)
	WithRLS(ctx context.Context, projectID string, claims json.RawMessage, fn func(pgx.Tx) error) error
	Close()
}
