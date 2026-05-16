package service

import (
	"context"
	"crypto/md5"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"

	"github.com/kou-etal/etalbaas/pkg/apperror"
	storageproviders "github.com/kou-etal/etalbaas/storage-providers"

	"github.com/kou-etal/etalbaas/services/storage/internal/pool"
	"github.com/kou-etal/etalbaas/services/storage/internal/tenantstore"
)

const (
	maxProxyUploadSize       = 5 * 1024 * 1024 // 5MB
	defaultPresignExpiry     = 15 * time.Minute
	maxPresignExpiry         = 7 * 24 * time.Hour
	defaultObjectListLimit   = 100
	maxObjectListLimit       = 1000
)

type ObjectService struct {
	poolMgr  *pool.Manager
	provider storageproviders.ObjectStorage
}

func NewObjectService(poolMgr *pool.Manager, provider storageproviders.ObjectStorage) *ObjectService {
	return &ObjectService{poolMgr: poolMgr, provider: provider}
}

type UploadParams struct {
	ProjectID   string
	BucketName  string
	ObjectPath  string
	ContentType string
	Size        int64
	Body        io.Reader
	Claims      json.RawMessage
}

type UploadResult struct {
	Object tenantstore.StorageObject
	Key    string
}

func (s *ObjectService) Upload(ctx context.Context, p UploadParams) (UploadResult, error) {
	if p.Size > maxProxyUploadSize {
		return UploadResult{}, apperror.New(apperror.CodeInvalidArgument, fmt.Sprintf("upload size %d exceeds maximum %d bytes", p.Size, maxProxyUploadSize))
	}

	var bucket tenantstore.StorageBucket
	err := s.poolMgr.WithRLS(ctx, p.ProjectID, p.Claims, func(tx pgx.Tx) error {
		q := tenantstore.New(tx)
		b, err := q.GetBucketByName(ctx, tenantstore.GetBucketByNameParams{
			Name:      p.BucketName,
			ProjectID: p.ProjectID,
		})
		if err != nil {
			return wrapBucketNotFound(err, p.BucketName)
		}
		bucket = b
		return nil
	})
	if err != nil {
		return UploadResult{}, err
	}

	if err := enforceBucketLimits(bucket, p.ContentType, p.Size); err != nil {
		return UploadResult{}, err
	}

	key := objectKey(p.ProjectID, p.BucketName, p.ObjectPath)

	hash := md5.New()
	body := io.TeeReader(p.Body, hash)

	if err := s.provider.Put(ctx, key, body, p.Size, p.ContentType); err != nil {
		return UploadResult{}, apperror.Wrap(apperror.CodeInternal, "upload to storage provider", err)
	}
	etag := hex.EncodeToString(hash.Sum(nil))

	ownerUUID := ownerFromClaims(p.Claims)

	var obj tenantstore.StorageObject
	err = s.poolMgr.WithRLS(ctx, p.ProjectID, p.Claims, func(tx pgx.Tx) error {
		q := tenantstore.New(tx)
		o, err := q.UpsertObject(ctx, tenantstore.UpsertObjectParams{
			BucketID: bucket.ID,
			Name:     p.ObjectPath,
			Owner:    ownerUUID,
			Size:     &p.Size,
			MimeType: &p.ContentType,
			Etag:     &etag,
			Metadata: []byte("{}"),
		})
		if err != nil {
			return wrapDBError(err, "upsert object metadata")
		}
		obj = o
		return nil
	})
	if err != nil {
		return UploadResult{}, err
	}
	return UploadResult{Object: obj, Key: key}, nil
}

type DownloadParams struct {
	ProjectID  string
	BucketName string
	ObjectPath string
	Claims     json.RawMessage
}

type DownloadResult struct {
	Body        io.ReadCloser
	ContentType string
	Size        int64
	ETag        string
}

func (s *ObjectService) Download(ctx context.Context, p DownloadParams) (DownloadResult, error) {
	err := s.poolMgr.WithRLS(ctx, p.ProjectID, p.Claims, func(tx pgx.Tx) error {
		q := tenantstore.New(tx)
		_, err := q.GetBucketByName(ctx, tenantstore.GetBucketByNameParams{
			Name:      p.BucketName,
			ProjectID: p.ProjectID,
		})
		return wrapBucketNotFound(err, p.BucketName)
	})
	if err != nil {
		return DownloadResult{}, err
	}

	key := objectKey(p.ProjectID, p.BucketName, p.ObjectPath)
	body, info, err := s.provider.Get(ctx, key)
	if err != nil {
		return DownloadResult{}, apperror.Wrap(apperror.CodeNotFound, "object not found in storage", err)
	}
	return DownloadResult{
		Body:        body,
		ContentType: info.ContentType,
		Size:        info.Size,
		ETag:        info.ETag,
	}, nil
}

func (s *ObjectService) PublicDownload(ctx context.Context, projectID, bucketName, objectPath string) (DownloadResult, error) {
	// For public downloads, use anon claims to let RLS check public bucket access.
	claims := anonClaims()
	var bucket tenantstore.StorageBucket
	err := s.poolMgr.WithRLS(ctx, projectID, claims, func(tx pgx.Tx) error {
		q := tenantstore.New(tx)
		b, err := q.GetBucketByName(ctx, tenantstore.GetBucketByNameParams{
			Name:      bucketName,
			ProjectID: projectID,
		})
		if err != nil {
			return wrapBucketNotFound(err, bucketName)
		}
		bucket = b
		return nil
	})
	if err != nil {
		return DownloadResult{}, err
	}

	if bucket.AccessLevel != "public" {
		return DownloadResult{}, apperror.New(apperror.CodePermissionDenied, "bucket is not public")
	}

	key := objectKey(projectID, bucketName, objectPath)
	body, info, err := s.provider.Get(ctx, key)
	if err != nil {
		return DownloadResult{}, apperror.Wrap(apperror.CodeNotFound, "object not found in storage", err)
	}
	return DownloadResult{
		Body:        body,
		ContentType: info.ContentType,
		Size:        info.Size,
		ETag:        info.ETag,
	}, nil
}

type DeleteObjectParams struct {
	ProjectID  string
	BucketName string
	ObjectPath string
	Claims     json.RawMessage
}

func (s *ObjectService) DeleteObject(ctx context.Context, p DeleteObjectParams) error {
	var bucket tenantstore.StorageBucket
	err := s.poolMgr.WithRLS(ctx, p.ProjectID, p.Claims, func(tx pgx.Tx) error {
		q := tenantstore.New(tx)
		b, err := q.GetBucketByName(ctx, tenantstore.GetBucketByNameParams{
			Name:      p.BucketName,
			ProjectID: p.ProjectID,
		})
		if err != nil {
			return wrapBucketNotFound(err, p.BucketName)
		}
		bucket = b
		return nil
	})
	if err != nil {
		return err
	}

	key := objectKey(p.ProjectID, p.BucketName, p.ObjectPath)
	if err := s.provider.Delete(ctx, key); err != nil {
		return apperror.Wrap(apperror.CodeInternal, "delete from storage provider", err)
	}

	return s.poolMgr.WithRLS(ctx, p.ProjectID, p.Claims, func(tx pgx.Tx) error {
		q := tenantstore.New(tx)
		if err := q.DeleteObject(ctx, tenantstore.DeleteObjectParams{
			BucketID: bucket.ID,
			Name:     p.ObjectPath,
		}); err != nil {
			return wrapDBError(err, "delete object metadata")
		}
		return nil
	})
}

type ListObjectsParams struct {
	ProjectID  string
	BucketName string
	Limit      int32
	Offset     int32
	Claims     json.RawMessage
}

func (s *ObjectService) ListObjects(ctx context.Context, p ListObjectsParams) ([]tenantstore.StorageObject, error) {
	if p.Limit <= 0 {
		p.Limit = defaultObjectListLimit
	}
	if p.Limit > maxObjectListLimit {
		p.Limit = maxObjectListLimit
	}

	var bucket tenantstore.StorageBucket
	var objects []tenantstore.StorageObject
	err := s.poolMgr.WithRLS(ctx, p.ProjectID, p.Claims, func(tx pgx.Tx) error {
		q := tenantstore.New(tx)
		b, err := q.GetBucketByName(ctx, tenantstore.GetBucketByNameParams{
			Name:      p.BucketName,
			ProjectID: p.ProjectID,
		})
		if err != nil {
			return wrapBucketNotFound(err, p.BucketName)
		}
		bucket = b
		_ = bucket

		objs, err := q.ListObjectsByBucket(ctx, tenantstore.ListObjectsByBucketParams{
			BucketID: b.ID,
			Limit:    p.Limit,
			Offset:   p.Offset,
		})
		if err != nil {
			return wrapDBError(err, "list objects")
		}
		objects = objs
		return nil
	})
	if err != nil {
		return nil, err
	}
	return objects, nil
}

type PresignUploadParams struct {
	ProjectID  string
	BucketName string
	ObjectPath string
	Expiry     time.Duration
	Claims     json.RawMessage
}

type PresignResult struct {
	URL string
}

func (s *ObjectService) PresignUpload(ctx context.Context, p PresignUploadParams) (PresignResult, error) {
	expiry := normalizeExpiry(p.Expiry)

	err := s.poolMgr.WithRLS(ctx, p.ProjectID, p.Claims, func(tx pgx.Tx) error {
		q := tenantstore.New(tx)
		_, err := q.GetBucketByName(ctx, tenantstore.GetBucketByNameParams{
			Name:      p.BucketName,
			ProjectID: p.ProjectID,
		})
		return wrapBucketNotFound(err, p.BucketName)
	})
	if err != nil {
		return PresignResult{}, err
	}

	key := objectKey(p.ProjectID, p.BucketName, p.ObjectPath)
	url, err := s.provider.PresignedPutURL(ctx, key, expiry)
	if err != nil {
		return PresignResult{}, apperror.Wrap(apperror.CodeInternal, "generate presigned upload URL", err)
	}
	return PresignResult{URL: url}, nil
}

type PresignDownloadParams struct {
	ProjectID  string
	BucketName string
	ObjectPath string
	Expiry     time.Duration
	Claims     json.RawMessage
}

func (s *ObjectService) PresignDownload(ctx context.Context, p PresignDownloadParams) (PresignResult, error) {
	expiry := normalizeExpiry(p.Expiry)

	err := s.poolMgr.WithRLS(ctx, p.ProjectID, p.Claims, func(tx pgx.Tx) error {
		q := tenantstore.New(tx)
		_, err := q.GetBucketByName(ctx, tenantstore.GetBucketByNameParams{
			Name:      p.BucketName,
			ProjectID: p.ProjectID,
		})
		return wrapBucketNotFound(err, p.BucketName)
	})
	if err != nil {
		return PresignResult{}, err
	}

	key := objectKey(p.ProjectID, p.BucketName, p.ObjectPath)
	url, err := s.provider.PresignedGetURL(ctx, key, expiry)
	if err != nil {
		return PresignResult{}, apperror.Wrap(apperror.CodeInternal, "generate presigned download URL", err)
	}
	return PresignResult{URL: url}, nil
}

type UploadCompleteParams struct {
	ProjectID  string
	BucketName string
	ObjectPath string
	Claims     json.RawMessage
}

func (s *ObjectService) UploadComplete(ctx context.Context, p UploadCompleteParams) (tenantstore.StorageObject, error) {
	key := objectKey(p.ProjectID, p.BucketName, p.ObjectPath)

	info, err := s.provider.Head(ctx, key)
	if err != nil {
		return tenantstore.StorageObject{}, apperror.Wrap(apperror.CodeNotFound, "object not found in storage", err)
	}

	ownerUUID := ownerFromClaims(p.Claims)

	var obj tenantstore.StorageObject
	err = s.poolMgr.WithRLS(ctx, p.ProjectID, p.Claims, func(tx pgx.Tx) error {
		q := tenantstore.New(tx)
		bucket, err := q.GetBucketByName(ctx, tenantstore.GetBucketByNameParams{
			Name:      p.BucketName,
			ProjectID: p.ProjectID,
		})
		if err != nil {
			return wrapBucketNotFound(err, p.BucketName)
		}

		o, err := q.UpsertObject(ctx, tenantstore.UpsertObjectParams{
			BucketID: bucket.ID,
			Name:     p.ObjectPath,
			Owner:    ownerUUID,
			Size:     &info.Size,
			MimeType: &info.ContentType,
			Etag:     &info.ETag,
			Metadata: []byte("{}"),
		})
		if err != nil {
			return wrapDBError(err, "upsert object metadata")
		}
		obj = o
		return nil
	})
	if err != nil {
		return tenantstore.StorageObject{}, err
	}
	return obj, nil
}

// --- helpers ---

func objectKey(projectID, bucketName, objectPath string) string {
	return fmt.Sprintf("project-%s/%s/%s", projectID, bucketName, objectPath)
}

func enforceBucketLimits(bucket tenantstore.StorageBucket, contentType string, size int64) error {
	if bucket.FileSizeLimit != nil && *bucket.FileSizeLimit > 0 && size > *bucket.FileSizeLimit {
		return apperror.New(apperror.CodeInvalidArgument, fmt.Sprintf("file size %d exceeds bucket limit %d", size, *bucket.FileSizeLimit))
	}
	if len(bucket.AllowedMimeTypes) > 0 {
		allowed := false
		for _, t := range bucket.AllowedMimeTypes {
			if t == contentType {
				allowed = true
				break
			}
		}
		if !allowed {
			return apperror.New(apperror.CodeInvalidArgument, fmt.Sprintf("MIME type %q not allowed for this bucket", contentType))
		}
	}
	return nil
}

func ownerFromClaims(claims json.RawMessage) pgtype.UUID {
	var parsed struct {
		Sub string `json:"sub"`
	}
	if json.Unmarshal(claims, &parsed) == nil && parsed.Sub != "" {
		if uid, err := uuid.Parse(parsed.Sub); err == nil {
			return pgtype.UUID{Bytes: uid, Valid: true}
		}
	}
	return pgtype.UUID{}
}

func anonClaims() json.RawMessage {
	return json.RawMessage(`{"role":"anon"}`)
}

func normalizeExpiry(d time.Duration) time.Duration {
	if d <= 0 {
		return defaultPresignExpiry
	}
	if d > maxPresignExpiry {
		return maxPresignExpiry
	}
	return d
}

func wrapBucketNotFound(err error, name string) error {
	if err == nil {
		return nil
	}
	if isPgNoRows(err) {
		return apperror.New(apperror.CodeNotFound, fmt.Sprintf("bucket %q not found", name))
	}
	return wrapDBError(err, "get bucket")
}

func isPgNoRows(err error) bool {
	return errors.Is(err, pgx.ErrNoRows)
}
