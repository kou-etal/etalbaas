package handler

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"

	"github.com/kou-etal/etalbaas/pkg/apperror"
	commonv1 "github.com/kou-etal/etalbaas/proto/gen/go/etalbaas/common/v1"
	eventv1 "github.com/kou-etal/etalbaas/proto/gen/go/etalbaas/event/v1"
	"github.com/kou-etal/etalbaas/services/event/internal/service"
	"github.com/kou-etal/etalbaas/services/event/internal/store"
	"google.golang.org/protobuf/types/known/timestamppb"
)

const maxTokenLen = 256

// ============================================================
// DB → Proto (read path)
// ============================================================

// EventRecordToProto converts a DB EventHistory row to a proto EventRecord message.
func EventRecordToProto(row store.EventHistory) (*eventv1.EventRecord, error) {
	rec := &eventv1.EventRecord{
		Id:           row.ID.String(),
		ProjectId:    row.ProjectID,
		FunctionId:   row.FunctionID.String(),
		Status:       row.Status,
		AttemptCount: row.AttemptCount,
		CreatedAt:    timestamppb.New(row.CreatedAt),
	}

	// Nullable invocation_id (pgtype.UUID)
	if row.InvocationID.Valid {
		rec.InvocationId = uuid.UUID(row.InvocationID.Bytes).String()
	}

	// Nullable fields
	if row.LastError != nil {
		rec.LastError = *row.LastError
	}
	if row.TraceID != nil {
		rec.TraceId = *row.TraceID
	}

	// TriggerInfo from trigger_type + trigger_data JSONB
	switch row.TriggerType {
	case "database_change":
		var data service.DatabaseChangeData
		if err := json.Unmarshal(row.TriggerData, &data); err != nil {
			return nil, fmt.Errorf("unmarshal database_change trigger_data: %w", err)
		}
		rec.Trigger = &eventv1.TriggerInfo{
			Source: &eventv1.TriggerInfo_DatabaseChange{
				DatabaseChange: &eventv1.DatabaseChangeSource{
					Table: data.Table,
					Event: data.Event,
				},
			},
		}
	case "object_storage":
		var data service.ObjectStorageData
		if err := json.Unmarshal(row.TriggerData, &data); err != nil {
			return nil, fmt.Errorf("unmarshal object_storage trigger_data: %w", err)
		}
		rec.Trigger = &eventv1.TriggerInfo{
			Source: &eventv1.TriggerInfo_ObjectStorage{
				ObjectStorage: &eventv1.ObjectStorageSource{
					Bucket:    data.Bucket,
					ObjectKey: data.ObjectKey,
					Event:     data.Event,
				},
			},
		}
	default:
		return nil, fmt.Errorf("unknown trigger type: %s", row.TriggerType)
	}

	return rec, nil
}

// EventRecordsToProto converts a slice of DB rows to proto messages with pagination cursor.
func EventRecordsToProto(rows []store.EventHistory, hasMore bool) ([]*eventv1.EventRecord, string, error) {
	records := make([]*eventv1.EventRecord, len(rows))
	for i, r := range rows {
		rec, err := EventRecordToProto(r)
		if err != nil {
			return nil, "", err
		}
		records[i] = rec
	}
	var nextToken string
	if hasMore && len(rows) > 0 {
		last := rows[len(rows)-1]
		nextToken = encodeUUIDCursor(last.CreatedAt, last.ID)
	}
	return records, nextToken, nil
}

// ============================================================
// Pagination
// ============================================================

func UUIDPaginationFromProto(p *commonv1.PaginationRequest) (int32, *time.Time, *uuid.UUID, error) {
	pageSize := int32(20)
	if p != nil && p.PageSize != 0 {
		if p.PageSize < 0 {
			return 0, nil, nil, apperror.New(apperror.CodeInvalidArgument, "page_size must be positive")
		}
		pageSize = p.PageSize
	}
	if pageSize > 100 {
		pageSize = 100
	}
	if p != nil && p.PageToken != "" {
		t, id, err := decodeUUIDCursor(p.PageToken)
		if err != nil {
			return 0, nil, nil, apperror.New(apperror.CodeInvalidArgument, "invalid page_token")
		}
		return pageSize, &t, &id, nil
	}
	return pageSize, nil, nil, nil
}

func PaginationToProto(nextToken string) *commonv1.PaginationResponse {
	if nextToken == "" {
		return nil
	}
	return &commonv1.PaginationResponse{
		NextPageToken: nextToken,
	}
}

// --- UUID cursor (time + uuid.UUID) ---

func encodeUUIDCursor(t time.Time, id uuid.UUID) string {
	raw := t.Format(time.RFC3339Nano) + "|" + id.String()
	return base64.RawURLEncoding.EncodeToString([]byte(raw))
}

func decodeUUIDCursor(s string) (time.Time, uuid.UUID, error) {
	if len(s) > maxTokenLen {
		return time.Time{}, uuid.Nil, fmt.Errorf("token too long")
	}
	b, err := base64.RawURLEncoding.DecodeString(s)
	if err != nil {
		return time.Time{}, uuid.Nil, fmt.Errorf("decode base64: %w", err)
	}
	parts := strings.SplitN(string(b), "|", 2)
	if len(parts) != 2 {
		return time.Time{}, uuid.Nil, fmt.Errorf("invalid cursor format")
	}
	t, err := time.Parse(time.RFC3339Nano, parts[0])
	if err != nil {
		return time.Time{}, uuid.Nil, fmt.Errorf("parse time: %w", err)
	}
	id, err := uuid.Parse(parts[1])
	if err != nil {
		return time.Time{}, uuid.Nil, fmt.Errorf("parse uuid: %w", err)
	}
	return t, id, nil
}

// --- Test helpers ---

func EncodeUUIDCursorForTest(t time.Time, id uuid.UUID) string {
	return encodeUUIDCursor(t, id)
}
