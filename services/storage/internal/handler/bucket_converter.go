package handler

import (
	"encoding/base64"
	"fmt"
	"strconv"

	"github.com/kou-etal/etalbaas/pkg/apperror"
	commonv1 "github.com/kou-etal/etalbaas/proto/gen/go/etalbaas/common/v1"
	storagev1 "github.com/kou-etal/etalbaas/proto/gen/go/etalbaas/storage/v1"
	"github.com/kou-etal/etalbaas/services/storage/internal/tenantstore"
	"google.golang.org/protobuf/types/known/timestamppb"
)

func BucketToProto(b tenantstore.StorageBucket) *storagev1.Bucket {
	pb := &storagev1.Bucket{
		Id:               b.ID,
		ProjectId:        b.ProjectID,
		Name:             b.Name,
		AccessLevel:      b.AccessLevel,
		AllowedMimeTypes: b.AllowedMimeTypes,
		CreatedAt:        timestamppb.New(b.CreatedAt),
		UpdatedAt:        timestamppb.New(b.UpdatedAt),
	}
	if b.FileSizeLimit != nil {
		pb.FileSizeLimit = *b.FileSizeLimit
	}
	return pb
}

func BucketsToProto(buckets []tenantstore.StorageBucket) []*storagev1.Bucket {
	result := make([]*storagev1.Bucket, len(buckets))
	for i, b := range buckets {
		result[i] = BucketToProto(b)
	}
	return result
}

const defaultPageSize = 20

func OffsetPaginationFromProto(p *commonv1.PaginationRequest) (int32, int32, error) {
	pageSize := int32(defaultPageSize)
	if p != nil && p.PageSize != 0 {
		if p.PageSize < 0 {
			return 0, 0, apperror.New(apperror.CodeInvalidArgument, "page_size must be positive")
		}
		pageSize = p.PageSize
	}
	if pageSize > 100 {
		pageSize = 100
	}

	var offset int32
	if p != nil && p.PageToken != "" {
		o, err := decodeOffsetToken(p.PageToken)
		if err != nil {
			return 0, 0, apperror.New(apperror.CodeInvalidArgument, "invalid page_token")
		}
		offset = o
	}
	return pageSize, offset, nil
}

func OffsetPaginationToProto(offset, pageSize, returned int32) *commonv1.PaginationResponse {
	if int32(returned) < pageSize {
		return nil
	}
	nextOffset := offset + int32(returned)
	return &commonv1.PaginationResponse{
		NextPageToken: encodeOffsetToken(nextOffset),
	}
}

func encodeOffsetToken(offset int32) string {
	return base64.RawURLEncoding.EncodeToString([]byte(strconv.Itoa(int(offset))))
}

func decodeOffsetToken(s string) (int32, error) {
	if len(s) > 64 {
		return 0, fmt.Errorf("token too long")
	}
	b, err := base64.RawURLEncoding.DecodeString(s)
	if err != nil {
		return 0, fmt.Errorf("decode base64: %w", err)
	}
	n, err := strconv.Atoi(string(b))
	if err != nil || n < 0 {
		return 0, fmt.Errorf("invalid offset")
	}
	return int32(n), nil
}
