package handler

import (
	"encoding/base64"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"

	"github.com/kou-etal/etalbaas/pkg/apperror"
	commonv1 "github.com/kou-etal/etalbaas/proto/gen/go/etalbaas/common/v1"
	tenantv1 "github.com/kou-etal/etalbaas/proto/gen/go/etalbaas/tenant/v1"
	"github.com/kou-etal/etalbaas/services/tenant-user/internal/store"
	"google.golang.org/protobuf/types/known/timestamppb"
)

const maxTokenLen = 256

func TenantToProto(row store.Tenant) *tenantv1.Tenant {
	var avatarURL string
	if row.AvatarUrl != nil {
		avatarURL = *row.AvatarUrl
	}
	return &tenantv1.Tenant{
		Id:          row.ID.String(),
		Email:       row.Email,
		DisplayName: row.DisplayName,
		AvatarUrl:   avatarURL,
		Plan:        row.Plan,
		Status:      row.Status,
		CreatedAt:   timestamppb.New(row.CreatedAt),
		UpdatedAt:   timestamppb.New(row.UpdatedAt),
	}
}

func TenantsToProto(rows []store.Tenant, hasMore bool) ([]*tenantv1.Tenant, string) {
	tenants := make([]*tenantv1.Tenant, len(rows))
	for i, r := range rows {
		tenants[i] = TenantToProto(r)
	}
	var nextToken string
	if hasMore && len(rows) > 0 {
		last := rows[len(rows)-1]
		nextToken = encodeCursor(last.CreatedAt, last.ID)
	}
	return tenants, nextToken
}

func PaginationFromProto(p *commonv1.PaginationRequest) (int32, *time.Time, *uuid.UUID, error) {
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
		cursorTime, cursorID, err := decodeCursor(p.PageToken)
		if err != nil {
			return 0, nil, nil, apperror.New(apperror.CodeInvalidArgument, "invalid page_token")
		}
		return pageSize, &cursorTime, &cursorID, nil
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

// EncodeCursorForTest is exported for testing only.
func EncodeCursorForTest(t time.Time, id uuid.UUID) string {
	return encodeCursor(t, id)
}

// encodeCursor encodes a compound cursor (created_at, id) as a URL-safe base64 string.
func encodeCursor(t time.Time, id uuid.UUID) string {
	raw := t.Format(time.RFC3339Nano) + "|" + id.String()
	return base64.RawURLEncoding.EncodeToString([]byte(raw))
}

// decodeCursor decodes a compound cursor token into (created_at, id).
func decodeCursor(s string) (time.Time, uuid.UUID, error) {
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
