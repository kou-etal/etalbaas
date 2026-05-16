package handler

import (
	"context"

	"connectrpc.com/connect"
	"github.com/google/uuid"

	"github.com/kou-etal/etalbaas/pkg/apperror"
	"github.com/kou-etal/etalbaas/pkg/requestctx"
	storagev1 "github.com/kou-etal/etalbaas/proto/gen/go/etalbaas/storage/v1"
	"github.com/kou-etal/etalbaas/proto/gen/go/etalbaas/storage/v1/storagev1connect"
	"github.com/kou-etal/etalbaas/services/storage/internal/service"
)

type BucketHandler struct {
	storagev1connect.UnimplementedStorageServiceHandler
	svc *service.BucketService
}

func NewBucketHandler(svc *service.BucketService) *BucketHandler {
	return &BucketHandler{svc: svc}
}

func (h *BucketHandler) CreateBucket(ctx context.Context, req *connect.Request[storagev1.CreateBucketRequest]) (*connect.Response[storagev1.CreateBucketResponse], error) {
	tenantID, err := extractTenantID(ctx)
	if err != nil {
		return nil, err
	}

	msg := req.Msg
	accessLevel := msg.AccessLevel
	if accessLevel == "" {
		accessLevel = "protected"
	}

	var fileSizeLimit *int64
	if msg.FileSizeLimit != 0 {
		fileSizeLimit = &msg.FileSizeLimit
	}

	bucket, err := h.svc.CreateBucket(ctx, service.CreateBucketParams{
		TenantID:         tenantID,
		ProjectID:        msg.ProjectId,
		Name:             msg.Name,
		AccessLevel:      accessLevel,
		FileSizeLimit:    fileSizeLimit,
		AllowedMimeTypes: msg.AllowedMimeTypes,
	})
	if err != nil {
		return nil, apperror.ToConnectError(err)
	}

	return connect.NewResponse(&storagev1.CreateBucketResponse{
		Bucket: BucketToProto(bucket),
	}), nil
}

func (h *BucketHandler) ListBuckets(ctx context.Context, req *connect.Request[storagev1.ListBucketsRequest]) (*connect.Response[storagev1.ListBucketsResponse], error) {
	tenantID, err := extractTenantID(ctx)
	if err != nil {
		return nil, err
	}

	msg := req.Msg
	pageSize, offset, err := OffsetPaginationFromProto(msg.GetPagination())
	if err != nil {
		return nil, apperror.ToConnectError(err)
	}

	buckets, err := h.svc.ListBuckets(ctx, service.ListBucketsParams{
		TenantID:  tenantID,
		ProjectID: msg.ProjectId,
		Limit:     pageSize,
		Offset:    offset,
	})
	if err != nil {
		return nil, apperror.ToConnectError(err)
	}

	return connect.NewResponse(&storagev1.ListBucketsResponse{
		Buckets:    BucketsToProto(buckets),
		Pagination: OffsetPaginationToProto(offset, pageSize, int32(len(buckets))),
	}), nil
}

func (h *BucketHandler) DeleteBucket(ctx context.Context, req *connect.Request[storagev1.DeleteBucketRequest]) (*connect.Response[storagev1.DeleteBucketResponse], error) {
	tenantID, err := extractTenantID(ctx)
	if err != nil {
		return nil, err
	}

	msg := req.Msg
	bucket, err := h.svc.DeleteBucket(ctx, service.DeleteBucketParams{
		TenantID:  tenantID,
		ProjectID: msg.ProjectId,
		BucketID:  msg.BucketId,
	})
	if err != nil {
		return nil, apperror.ToConnectError(err)
	}

	return connect.NewResponse(&storagev1.DeleteBucketResponse{
		Bucket: BucketToProto(bucket),
	}), nil
}

func (h *BucketHandler) UpdateBucket(ctx context.Context, req *connect.Request[storagev1.UpdateBucketRequest]) (*connect.Response[storagev1.UpdateBucketResponse], error) {
	tenantID, err := extractTenantID(ctx)
	if err != nil {
		return nil, err
	}

	msg := req.Msg

	var allowedMimeTypes []string
	if msg.AllowedMimeTypes != nil {
		allowedMimeTypes = msg.AllowedMimeTypes.GetValues()
	}

	bucket, err := h.svc.UpdateBucket(ctx, service.UpdateBucketParams{
		TenantID:         tenantID,
		ProjectID:        msg.ProjectId,
		BucketID:         msg.BucketId,
		AccessLevel:      msg.AccessLevel,
		FileSizeLimit:    msg.FileSizeLimit,
		AllowedMimeTypes: allowedMimeTypes,
	})
	if err != nil {
		return nil, apperror.ToConnectError(err)
	}

	return connect.NewResponse(&storagev1.UpdateBucketResponse{
		Bucket: BucketToProto(bucket),
	}), nil
}

func extractTenantID(ctx context.Context) (uuid.UUID, error) {
	userID, err := requestctx.UserID(ctx)
	if err != nil {
		return uuid.Nil, apperror.ToConnectError(apperror.New(apperror.CodeUnauthenticated, "user id not found"))
	}
	return userID, nil
}
