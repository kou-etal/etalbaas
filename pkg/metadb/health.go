package metadb

import (
	"context"
	"fmt"

	"github.com/jackc/pgx/v5/pgxpool"
)

func HealthCheck(ctx context.Context, pool *pgxpool.Pool) error {
	var result int
	if err := pool.QueryRow(ctx, "SELECT 1").Scan(&result); err != nil {
		return fmt.Errorf("metadb health check: %w", err)
	}
	return nil
}
