package replication

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	"github.com/jackc/pglogrepl"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgproto3"

	"github.com/kou-etal/etalbaas/services/cdc-pod/internal/config"
	"github.com/kou-etal/etalbaas/services/cdc-pod/internal/publisher"
)

// Replicator orchestrates PostgreSQL WAL logical replication and event publishing.
type Replicator struct {
	cfg    *config.Config
	hnd    *Handler
	pub    *publisher.NATSPublisher
	logger *slog.Logger
}

// NewReplicator creates a new Replicator.
func NewReplicator(cfg *config.Config, hnd *Handler, pub *publisher.NATSPublisher, logger *slog.Logger) *Replicator {
	return &Replicator{
		cfg:    cfg,
		hnd:    hnd,
		pub:    pub,
		logger: logger,
	}
}

// Run starts the CDC pipeline: publication setup, slot creation, WAL streaming.
// It blocks until the context is cancelled or a fatal error occurs.
func (r *Replicator) Run(ctx context.Context) error {
	// 1. Standard connection for DDL (publication management).
	stdConn, err := pgx.Connect(ctx, r.cfg.StandardConnString())
	if err != nil {
		return fmt.Errorf("standard db connect: %w", err)
	}

	pubName := r.cfg.EffectivePublicationName()
	if err := EnsurePublication(ctx, stdConn, pubName, r.logger); err != nil {
		stdConn.Close(ctx)
		return err
	}
	stdConn.Close(ctx) // No longer needed.

	// 2. Replication connection.
	replConn, err := pgconn.Connect(ctx, r.cfg.ReplicationConnString())
	if err != nil {
		return fmt.Errorf("replication connect: %w", err)
	}
	defer replConn.Close(ctx)

	// 3. Ensure replication slot exists and determine start LSN.
	startLSN, err := r.ensureReplicationSlot(ctx, replConn)
	if err != nil {
		return err
	}

	// 4. Start replication.
	err = pglogrepl.StartReplication(ctx, replConn, r.cfg.SlotName, startLSN,
		pglogrepl.StartReplicationOptions{
			PluginArgs: []string{
				"proto_version '1'",
				fmt.Sprintf("publication_names '%s'", pubName),
			},
		})
	if err != nil {
		return fmt.Errorf("start replication: %w", err)
	}
	r.logger.Info("replication started",
		"slot", r.cfg.SlotName,
		"start_lsn", startLSN,
		"publication", pubName,
	)

	// 5. Stream loop.
	return r.streamLoop(ctx, replConn, startLSN)
}

// ensureReplicationSlot creates the slot if absent and returns the start LSN.
func (r *Replicator) ensureReplicationSlot(ctx context.Context, conn *pgconn.PgConn) (pglogrepl.LSN, error) {
	// Try to read existing slot's confirmed flush LSN via a replication-protocol query.
	result := conn.Exec(ctx,
		fmt.Sprintf("IDENTIFY_SYSTEM"))
	_, err := result.ReadAll()
	if err != nil {
		return 0, fmt.Errorf("identify system: %w", err)
	}

	// Query confirmed_flush_lsn through a simple query over the replication connection.
	rows := conn.Exec(ctx,
		fmt.Sprintf("SELECT confirmed_flush_lsn FROM pg_replication_slots WHERE slot_name = '%s'",
			r.cfg.SlotName))
	resultSet, err := rows.ReadAll()
	if err != nil {
		return 0, fmt.Errorf("query replication slot: %w", err)
	}

	// If the slot exists, parse the flush LSN.
	if len(resultSet) > 0 && len(resultSet[0].Rows) > 0 && len(resultSet[0].Rows[0]) > 0 {
		lsnStr := string(resultSet[0].Rows[0][0])
		if lsnStr != "" {
			lsn, err := pglogrepl.ParseLSN(lsnStr)
			if err != nil {
				return 0, fmt.Errorf("parse flush LSN %q: %w", lsnStr, err)
			}
			r.logger.Info("existing replication slot found", "slot", r.cfg.SlotName, "flush_lsn", lsn)
			return lsn, nil
		}
	}

	// Slot does not exist — create it.
	res, err := pglogrepl.CreateReplicationSlot(ctx, conn, r.cfg.SlotName, "pgoutput",
		pglogrepl.CreateReplicationSlotOptions{Temporary: false})
	if err != nil {
		return 0, fmt.Errorf("create replication slot: %w", err)
	}

	var startLSN pglogrepl.LSN
	if res.ConsistentPoint != "" {
		startLSN, err = pglogrepl.ParseLSN(res.ConsistentPoint)
		if err != nil {
			return 0, fmt.Errorf("parse consistent point: %w", err)
		}
	}
	r.logger.Info("replication slot created", "slot", r.cfg.SlotName, "start_lsn", startLSN)
	return startLSN, nil
}

// streamLoop is the main WAL message processing loop.
func (r *Replicator) streamLoop(ctx context.Context, conn *pgconn.PgConn, startLSN pglogrepl.LSN) error {
	clientXLogPos := startLSN
	flushedLSN := startLSN
	intervalS := r.cfg.StandbyStatusIntervalS
	if intervalS < 1 {
		intervalS = 10
	}
	standbyInterval := time.Duration(intervalS) * time.Second
	nextStandbyUpdate := time.Now().Add(standbyInterval)

	for {
		if ctx.Err() != nil {
			// Graceful shutdown: send final status before exiting.
			r.sendStandbyStatus(context.Background(), conn, flushedLSN)
			return nil
		}

		// Use a short deadline so we can periodically send standby status.
		deadline := time.Now().Add(standbyInterval)
		recvCtx, cancel := context.WithDeadline(ctx, deadline)
		rawMsg, err := conn.ReceiveMessage(recvCtx)
		cancel()

		if err != nil {
			if pgconn.Timeout(err) {
				// Timeout — send standby status update.
				if time.Now().After(nextStandbyUpdate) {
					r.sendStandbyStatus(ctx, conn, flushedLSN)
					nextStandbyUpdate = time.Now().Add(standbyInterval)
				}
				continue
			}
			if ctx.Err() != nil {
				r.sendStandbyStatus(context.Background(), conn, flushedLSN)
				return nil
			}
			return fmt.Errorf("receive message: %w", err)
		}

		if errMsg, ok := rawMsg.(*pgproto3.ErrorResponse); ok {
			return fmt.Errorf("postgres error: severity=%s code=%s message=%s",
				errMsg.Severity, errMsg.Code, errMsg.Message)
		}

		copyData, ok := rawMsg.(*pgproto3.CopyData)
		if !ok {
			continue
		}

		if len(copyData.Data) == 0 {
			continue
		}

		switch copyData.Data[0] {
		case pglogrepl.XLogDataByteID:
			xld, err := pglogrepl.ParseXLogData(copyData.Data[1:])
			if err != nil {
				return fmt.Errorf("parse xlog data: %w", err)
			}

			newPos := xld.WALStart + pglogrepl.LSN(len(xld.WALData))
			if newPos > clientXLogPos {
				clientXLogPos = newPos
			}

			if err := r.handleWALData(ctx, xld); err != nil {
				return fmt.Errorf("handle wal data: %w", err)
			}

			// Advance flushed LSN after successful processing.
			flushedLSN = clientXLogPos

		case pglogrepl.PrimaryKeepaliveMessageByteID:
			pkm, err := pglogrepl.ParsePrimaryKeepaliveMessage(copyData.Data[1:])
			if err != nil {
				return fmt.Errorf("parse keepalive: %w", err)
			}
			if pkm.ReplyRequested {
				r.sendStandbyStatus(ctx, conn, flushedLSN)
				nextStandbyUpdate = time.Now().Add(standbyInterval)
			}
		}

		// Periodic standby status.
		if time.Now().After(nextStandbyUpdate) {
			r.sendStandbyStatus(ctx, conn, flushedLSN)
			nextStandbyUpdate = time.Now().Add(standbyInterval)
		}
	}
}

// handleWALData parses and processes a single WAL message.
func (r *Replicator) handleWALData(ctx context.Context, xld pglogrepl.XLogData) error {
	logicalMsg, err := pglogrepl.Parse(xld.WALData)
	if err != nil {
		return fmt.Errorf("parse logical message: %w", err)
	}

	switch msg := logicalMsg.(type) {
	case *pglogrepl.RelationMessage:
		r.hnd.HandleRelationMessage(msg)

	case *pglogrepl.BeginMessage:
		r.hnd.HandleBeginMessage(msg)

	case *pglogrepl.InsertMessage:
		event, err := r.hnd.HandleInsertMessage(msg, r.cfg.ProjectID, xld.WALStart)
		if err != nil {
			return err
		}
		if err := r.pub.Publish(ctx, event); err != nil {
			return err
		}

	case *pglogrepl.UpdateMessage:
		event, err := r.hnd.HandleUpdateMessage(msg, r.cfg.ProjectID, xld.WALStart)
		if err != nil {
			return err
		}
		if err := r.pub.Publish(ctx, event); err != nil {
			return err
		}

	case *pglogrepl.DeleteMessage:
		event, err := r.hnd.HandleDeleteMessage(msg, r.cfg.ProjectID, xld.WALStart)
		if err != nil {
			return err
		}
		if err := r.pub.Publish(ctx, event); err != nil {
			return err
		}

	case *pglogrepl.CommitMessage:
		// Transaction boundary — no action needed.

	case *pglogrepl.TypeMessage:
		// Custom type definition — not used in Phase 1.

	case *pglogrepl.OriginMessage:
		// Origin info — not used.

	case *pglogrepl.TruncateMessage:
		r.logger.Warn("TRUNCATE detected, not publishing event",
			"relation_ids", msg.RelationIDs)
	}

	return nil
}

// sendStandbyStatus sends a WAL flush position update to PostgreSQL.
func (r *Replicator) sendStandbyStatus(ctx context.Context, conn *pgconn.PgConn, flushedLSN pglogrepl.LSN) {
	err := pglogrepl.SendStandbyStatusUpdate(ctx, conn, pglogrepl.StandbyStatusUpdate{
		WALWritePosition: flushedLSN,
		WALFlushPosition: flushedLSN,
		WALApplyPosition: flushedLSN,
	})
	if err != nil {
		r.logger.Warn("failed to send standby status", "error", err)
	}
}
