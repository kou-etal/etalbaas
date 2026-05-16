package replication

import (
	"context"
	"fmt"
	"log/slog"

	"github.com/jackc/pgx/v5"
)

// EnsurePublication creates the publication if it does not already exist.
// The publication is created FOR ALL TABLES so that any new table created by
// the tenant is automatically included in CDC.
func EnsurePublication(ctx context.Context, conn *pgx.Conn, pubName string, logger *slog.Logger) error {
	var exists bool
	err := conn.QueryRow(ctx,
		"SELECT EXISTS(SELECT 1 FROM pg_publication WHERE pubname = $1)",
		pubName,
	).Scan(&exists)
	if err != nil {
		return fmt.Errorf("check publication: %w", err)
	}

	if exists {
		logger.Info("publication already exists", "name", pubName)
		return nil
	}

	// Publication names are internal identifiers (cdc_{project_id}) and not user-supplied,
	// so string interpolation is safe here. pgx does not support parameterized DDL names.
	_, err = conn.Exec(ctx, fmt.Sprintf("CREATE PUBLICATION %s FOR ALL TABLES", pgx.Identifier{pubName}.Sanitize()))
	if err != nil {
		return fmt.Errorf("create publication %s: %w", pubName, err)
	}
	logger.Info("publication created", "name", pubName)
	return nil
}
