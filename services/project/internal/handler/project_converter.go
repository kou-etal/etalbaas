package handler

import (
	"encoding/base64"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"

	"github.com/kou-etal/etalbaas/pkg/apperror"
	commonv1 "github.com/kou-etal/etalbaas/proto/gen/go/etalbaas/common/v1"
	projectv1 "github.com/kou-etal/etalbaas/proto/gen/go/etalbaas/project/v1"
	"github.com/kou-etal/etalbaas/services/project/internal/store"
	"google.golang.org/protobuf/types/known/timestamppb"
)

const maxTokenLen = 256

// --- Project converters ---

func ProjectToProto(row store.Project) *projectv1.Project {
	var desc string
	if row.Description != nil {
		desc = *row.Description
	}
	return &projectv1.Project{
		Id:                 row.ID,
		TenantId:           row.TenantID.String(),
		DisplayName:        row.DisplayName,
		Description:        desc,
		Status:             row.Status,
		PostgresEnabled:    row.PostgresEnabled,
		PostgresExtensions: row.PostgresExtensions,
		RedisEnabled:       row.RedisEnabled,
		PostgrestEnabled:   row.PostgrestEnabled,
		CreatedAt:          timestamppb.New(row.CreatedAt),
		UpdatedAt:          timestamppb.New(row.UpdatedAt),
	}
}

func ProjectsToProto(rows []store.Project, hasMore bool) ([]*projectv1.Project, string) {
	projects := make([]*projectv1.Project, len(rows))
	for i, r := range rows {
		projects[i] = ProjectToProto(r)
	}
	var nextToken string
	if hasMore && len(rows) > 0 {
		last := rows[len(rows)-1]
		nextToken = encodeProjectCursor(last.CreatedAt, last.ID)
	}
	return projects, nextToken
}

// --- ApiKey converters ---

func ApiKeyToProto(row store.ApiKey) *projectv1.ApiKey {
	ak := &projectv1.ApiKey{
		Id:        row.ID.String(),
		ProjectId: row.ProjectID,
		Name:      row.Name,
		KeyPrefix: row.KeyPrefix,
		Role:      row.Role,
		CreatedAt: timestamppb.New(row.CreatedAt),
	}
	if row.ExpiresAt.Valid {
		ak.ExpiresAt = timestamppb.New(row.ExpiresAt.Time)
	}
	if row.RevokedAt.Valid {
		ak.RevokedAt = timestamppb.New(row.RevokedAt.Time)
	}
	return ak
}

func ApiKeysToProto(rows []store.ApiKey, hasMore bool) ([]*projectv1.ApiKey, string) {
	keys := make([]*projectv1.ApiKey, len(rows))
	for i, r := range rows {
		keys[i] = ApiKeyToProto(r)
	}
	var nextToken string
	if hasMore && len(rows) > 0 {
		last := rows[len(rows)-1]
		nextToken = encodeUUIDCursor(last.CreatedAt, last.ID)
	}
	return keys, nextToken
}

// --- Pagination ---

// ProjectPaginationFromProto decodes pagination for Project lists (text ID cursor).
func ProjectPaginationFromProto(p *commonv1.PaginationRequest) (int32, *time.Time, *string, error) {
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
		t, id, err := decodeProjectCursor(p.PageToken)
		if err != nil {
			return 0, nil, nil, apperror.New(apperror.CodeInvalidArgument, "invalid page_token")
		}
		return pageSize, &t, &id, nil
	}
	return pageSize, nil, nil, nil
}

// UUIDPaginationFromProto decodes pagination for UUID-keyed lists (ApiKey, Secret).
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

// --- Project cursor (time + string ID) ---

func encodeProjectCursor(t time.Time, id string) string {
	raw := t.Format(time.RFC3339Nano) + "|" + id
	return base64.RawURLEncoding.EncodeToString([]byte(raw))
}

func decodeProjectCursor(s string) (time.Time, string, error) {
	if len(s) > maxTokenLen {
		return time.Time{}, "", fmt.Errorf("token too long")
	}
	b, err := base64.RawURLEncoding.DecodeString(s)
	if err != nil {
		return time.Time{}, "", fmt.Errorf("decode base64: %w", err)
	}
	parts := strings.SplitN(string(b), "|", 2)
	if len(parts) != 2 {
		return time.Time{}, "", fmt.Errorf("invalid cursor format")
	}
	t, err := time.Parse(time.RFC3339Nano, parts[0])
	if err != nil {
		return time.Time{}, "", fmt.Errorf("parse time: %w", err)
	}
	return t, parts[1], nil
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

func EncodeProjectCursorForTest(t time.Time, id string) string {
	return encodeProjectCursor(t, id)
}

func EncodeUUIDCursorForTest(t time.Time, id uuid.UUID) string {
	return encodeUUIDCursor(t, id)
}
