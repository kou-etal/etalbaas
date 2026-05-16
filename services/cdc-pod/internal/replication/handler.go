package replication

import (
	"encoding/json"
	"fmt"
	"log/slog"
	"strconv"
	"time"

	"github.com/jackc/pglogrepl"

	"github.com/kou-etal/etalbaas/services/cdc-pod/internal/publisher"
)

// PostgreSQL type OIDs for common types.
const (
	oidBool    uint32 = 16
	oidInt2    uint32 = 21
	oidInt4    uint32 = 23
	oidInt8    uint32 = 20
	oidFloat4  uint32 = 700
	oidFloat8  uint32 = 701
	oidNumeric uint32 = 1700
	oidJSON    uint32 = 114
	oidJSONB   uint32 = 3802
)

// Handler parses pglogrepl WAL messages and converts them to CDCEvents.
type Handler struct {
	relations    map[uint32]*pglogrepl.RelationMessage
	currentTxID  uint32
	currentTxTS  time.Time
	logger       *slog.Logger
}

// NewHandler creates a new WAL message handler.
func NewHandler(logger *slog.Logger) *Handler {
	return &Handler{
		relations: make(map[uint32]*pglogrepl.RelationMessage),
		logger:    logger,
	}
}

// HandleRelationMessage stores the relation (table) metadata for later tuple decoding.
func (h *Handler) HandleRelationMessage(msg *pglogrepl.RelationMessage) {
	h.relations[msg.RelationID] = msg
	h.logger.Debug("relation message",
		"relation_id", msg.RelationID,
		"namespace", msg.Namespace,
		"name", msg.RelationName,
	)
}

// HandleBeginMessage records the current transaction ID and timestamp.
func (h *Handler) HandleBeginMessage(msg *pglogrepl.BeginMessage) {
	h.currentTxID = msg.Xid
	h.currentTxTS = msg.CommitTime
}

// HandleInsertMessage converts a WAL INSERT into a CDCEvent.
func (h *Handler) HandleInsertMessage(msg *pglogrepl.InsertMessage, projectID string, lsn pglogrepl.LSN) (*publisher.CDCEvent, error) {
	rel, ok := h.relations[msg.RelationID]
	if !ok {
		return nil, fmt.Errorf("unknown relation ID %d", msg.RelationID)
	}

	newRow, err := h.decodeTuple(msg.Tuple, rel)
	if err != nil {
		return nil, fmt.Errorf("decode insert tuple: %w", err)
	}

	return &publisher.CDCEvent{
		EventID:   h.buildEventID(projectID, lsn),
		ProjectID: projectID,
		Table:     rel.RelationName,
		Operation: "INSERT",
		Timestamp: h.currentTxTS,
		Old:       nil,
		New:       newRow,
	}, nil
}

// HandleUpdateMessage converts a WAL UPDATE into a CDCEvent.
func (h *Handler) HandleUpdateMessage(msg *pglogrepl.UpdateMessage, projectID string, lsn pglogrepl.LSN) (*publisher.CDCEvent, error) {
	rel, ok := h.relations[msg.RelationID]
	if !ok {
		return nil, fmt.Errorf("unknown relation ID %d", msg.RelationID)
	}

	var oldRow map[string]any
	if msg.OldTuple != nil {
		var err error
		oldRow, err = h.decodeTuple(msg.OldTuple, rel)
		if err != nil {
			return nil, fmt.Errorf("decode update old tuple: %w", err)
		}
	}

	newRow, err := h.decodeTuple(msg.NewTuple, rel)
	if err != nil {
		return nil, fmt.Errorf("decode update new tuple: %w", err)
	}

	return &publisher.CDCEvent{
		EventID:   h.buildEventID(projectID, lsn),
		ProjectID: projectID,
		Table:     rel.RelationName,
		Operation: "UPDATE",
		Timestamp: h.currentTxTS,
		Old:       oldRow,
		New:       newRow,
	}, nil
}

// HandleDeleteMessage converts a WAL DELETE into a CDCEvent.
func (h *Handler) HandleDeleteMessage(msg *pglogrepl.DeleteMessage, projectID string, lsn pglogrepl.LSN) (*publisher.CDCEvent, error) {
	rel, ok := h.relations[msg.RelationID]
	if !ok {
		return nil, fmt.Errorf("unknown relation ID %d", msg.RelationID)
	}

	var oldRow map[string]any
	if msg.OldTuple != nil {
		var err error
		oldRow, err = h.decodeTuple(msg.OldTuple, rel)
		if err != nil {
			return nil, fmt.Errorf("decode delete old tuple: %w", err)
		}
	}

	return &publisher.CDCEvent{
		EventID:   h.buildEventID(projectID, lsn),
		ProjectID: projectID,
		Table:     rel.RelationName,
		Operation: "DELETE",
		Timestamp: h.currentTxTS,
		Old:       oldRow,
		New:       nil,
	}, nil
}

// buildEventID creates a unique event identifier: {project_id}-{lsn}-{txid}.
func (h *Handler) buildEventID(projectID string, lsn pglogrepl.LSN) string {
	return fmt.Sprintf("%s-%s-%d", projectID, lsn, h.currentTxID)
}

// decodeTuple converts a pglogrepl TupleData into a map using the relation's column metadata.
func (h *Handler) decodeTuple(tuple *pglogrepl.TupleData, rel *pglogrepl.RelationMessage) (map[string]any, error) {
	if tuple == nil {
		return nil, nil
	}

	row := make(map[string]any, len(tuple.Columns))
	for i, col := range tuple.Columns {
		if i >= len(rel.Columns) {
			break
		}
		colMeta := rel.Columns[i]

		switch col.DataType {
		case 'n': // NULL
			row[colMeta.Name] = nil
		case 'u': // unchanged TOAST value — omit from the map
			continue
		case 't': // text-format value
			row[colMeta.Name] = decodeColumnValue(col.Data, colMeta.DataType)
		default:
			row[colMeta.Name] = string(col.Data)
		}
	}
	return row, nil
}

// decodeColumnValue converts a text-format PostgreSQL value to a typed Go value
// based on the column's type OID.
func decodeColumnValue(data []byte, typeOID uint32) any {
	text := string(data)

	switch typeOID {
	case oidBool:
		return text == "t" || text == "true"

	case oidInt2, oidInt4, oidInt8:
		if v, err := strconv.ParseInt(text, 10, 64); err == nil {
			return v
		}

	case oidFloat4, oidFloat8, oidNumeric:
		if v, err := strconv.ParseFloat(text, 64); err == nil {
			return v
		}

	case oidJSON, oidJSONB:
		var v any
		if err := json.Unmarshal(data, &v); err == nil {
			return v
		}
	}

	// Default: return as string (text, varchar, uuid, timestamp, etc.)
	return text
}
