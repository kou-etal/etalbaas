package service

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"

	"github.com/kou-etal/etalbaas/pkg/apperror"
	"github.com/kou-etal/etalbaas/services/storage/internal/metastore"
	"github.com/kou-etal/etalbaas/services/storage/internal/pool"
	"github.com/kou-etal/etalbaas/services/storage/internal/tenantstore"
)

type BucketService struct {
	meta    metastore.Querier
	poolMgr pool.Provider
}

func NewBucketService(meta metastore.Querier, poolMgr pool.Provider) *BucketService {
	return &BucketService{meta: meta, poolMgr: poolMgr}
}

type CreateBucketParams struct {
	TenantID         uuid.UUID
	ProjectID        string
	Name             string
	AccessLevel      string
	FileSizeLimit    *int64
	AllowedMimeTypes []string
}

func (s *BucketService) CreateBucket(ctx context.Context, p CreateBucketParams) (tenantstore.StorageBucket, error) {
	if err := validateBucketName(p.Name); err != nil {
		return tenantstore.StorageBucket{}, apperror.New(apperror.CodeInvalidArgument, err.Error())
	}
	if err := validateAccessLevel(p.AccessLevel); err != nil {
		return tenantstore.StorageBucket{}, apperror.New(apperror.CodeInvalidArgument, err.Error())
	}
	if p.FileSizeLimit != nil {
		if err := validateFileSizeLimit(*p.FileSizeLimit); err != nil {
			return tenantstore.StorageBucket{}, apperror.New(apperror.CodeInvalidArgument, err.Error())
		}
	}
	if err := validateMIMETypes(p.AllowedMimeTypes); err != nil {
		return tenantstore.StorageBucket{}, apperror.New(apperror.CodeInvalidArgument, err.Error())
	}

	if err := s.verifyProjectOwnership(ctx, p.ProjectID, p.TenantID); err != nil {
		return tenantstore.StorageBucket{}, err
	}

	claims := serviceRoleClaims(p.TenantID)
	var result tenantstore.StorageBucket
	err := s.poolMgr.WithRLS(ctx, p.ProjectID, claims, func(tx pgx.Tx) error {
		q := tenantstore.New(tx)
		id := fmt.Sprintf("%s-%s", p.ProjectID, p.Name)
		bucket, err := q.CreateBucket(ctx, tenantstore.CreateBucketParams{
			ID:               id,
			Name:             p.Name,
			ProjectID:        p.ProjectID,
			AccessLevel:      p.AccessLevel,
			FileSizeLimit:    p.FileSizeLimit,
			AllowedMimeTypes: p.AllowedMimeTypes,
		})
		if err != nil {
			if isDuplicateKey(err) {
				return apperror.New(apperror.CodeAlreadyExists, fmt.Sprintf("bucket %q already exists", p.Name))
			}
			return wrapDBError(err, "create bucket")
		}
		result = bucket
		return nil
	})
	if err != nil {
		return tenantstore.StorageBucket{}, err
	}
	return result, nil
}

type ListBucketsParams struct {
	TenantID  uuid.UUID
	ProjectID string
	Limit     int32
	Offset    int32
}

func (s *BucketService) ListBuckets(ctx context.Context, p ListBucketsParams) ([]tenantstore.StorageBucket, error) {
	if err := s.verifyProjectOwnership(ctx, p.ProjectID, p.TenantID); err != nil {
		return nil, err
	}

	if p.Limit <= 0 {
		p.Limit = 20
	}
	if p.Limit > 100 {
		p.Limit = 100
	}

	claims := serviceRoleClaims(p.TenantID)
	var result []tenantstore.StorageBucket
	err := s.poolMgr.WithRLS(ctx, p.ProjectID, claims, func(tx pgx.Tx) error {
		q := tenantstore.New(tx)
		buckets, err := q.ListBuckets(ctx, tenantstore.ListBucketsParams{
			ProjectID: p.ProjectID,
			Limit:     p.Limit,
			Offset:    p.Offset,
		})
		if err != nil {
			return wrapDBError(err, "list buckets")
		}
		result = buckets
		return nil
	})
	if err != nil {
		return nil, err
	}
	return result, nil
}

type UpdateBucketParams struct {
	TenantID         uuid.UUID
	ProjectID        string
	BucketID         string
	AccessLevel      *string
	FileSizeLimit    *int64
	AllowedMimeTypes []string
}

func (s *BucketService) UpdateBucket(ctx context.Context, p UpdateBucketParams) (tenantstore.StorageBucket, error) {
	if p.AccessLevel != nil {
		if err := validateAccessLevel(*p.AccessLevel); err != nil {
			return tenantstore.StorageBucket{}, apperror.New(apperror.CodeInvalidArgument, err.Error())
		}
	}
	if p.FileSizeLimit != nil {
		if err := validateFileSizeLimit(*p.FileSizeLimit); err != nil {
			return tenantstore.StorageBucket{}, apperror.New(apperror.CodeInvalidArgument, err.Error())
		}
	}
	if p.AllowedMimeTypes != nil {
		if err := validateMIMETypes(p.AllowedMimeTypes); err != nil {
			return tenantstore.StorageBucket{}, apperror.New(apperror.CodeInvalidArgument, err.Error())
		}
	}

	if err := s.verifyProjectOwnership(ctx, p.ProjectID, p.TenantID); err != nil {
		return tenantstore.StorageBucket{}, err
	}

	claims := serviceRoleClaims(p.TenantID)
	var result tenantstore.StorageBucket
	err := s.poolMgr.WithRLS(ctx, p.ProjectID, claims, func(tx pgx.Tx) error {
		q := tenantstore.New(tx)
		bucket, err := q.UpdateBucket(ctx, tenantstore.UpdateBucketParams{
			ID:               p.BucketID,
			ProjectID:        p.ProjectID,
			AccessLevel:      p.AccessLevel,
			FileSizeLimit:    p.FileSizeLimit,
			AllowedMimeTypes: p.AllowedMimeTypes,
		})
		if err != nil {
			if errors.Is(err, pgx.ErrNoRows) {
				return apperror.New(apperror.CodeNotFound, "bucket not found")
			}
			return wrapDBError(err, "update bucket")
		}
		result = bucket
		return nil
	})
	if err != nil {
		return tenantstore.StorageBucket{}, err
	}
	return result, nil
}

type DeleteBucketParams struct {
	TenantID  uuid.UUID
	ProjectID string
	BucketID  string
}

func (s *BucketService) DeleteBucket(ctx context.Context, p DeleteBucketParams) (tenantstore.StorageBucket, error) {
	if err := s.verifyProjectOwnership(ctx, p.ProjectID, p.TenantID); err != nil {
		return tenantstore.StorageBucket{}, err
	}

	claims := serviceRoleClaims(p.TenantID)
	var result tenantstore.StorageBucket
	err := s.poolMgr.WithRLS(ctx, p.ProjectID, claims, func(tx pgx.Tx) error {
		q := tenantstore.New(tx)

		count, err := q.CountObjectsByBucketID(ctx, p.BucketID)
		if err != nil {
			return wrapDBError(err, "count objects")
		}
		if count > 0 {
			return apperror.New(apperror.CodeFailedPrecondition, fmt.Sprintf("bucket contains %d objects; delete them first", count))
		}

		bucket, err := q.DeleteBucket(ctx, tenantstore.DeleteBucketParams{
			ID:        p.BucketID,
			ProjectID: p.ProjectID,
		})
		if err != nil {
			if errors.Is(err, pgx.ErrNoRows) {
				return apperror.New(apperror.CodeNotFound, "bucket not found")
			}
			return wrapDBError(err, "delete bucket")
		}
		result = bucket
		return nil
	})
	if err != nil {
		return tenantstore.StorageBucket{}, err
	}
	return result, nil
}

func (s *BucketService) verifyProjectOwnership(ctx context.Context, projectID string, tenantID uuid.UUID) error {
	if _, err := s.meta.GetProjectByIDAndTenantID(ctx, metastore.GetProjectByIDAndTenantIDParams{
		ID:       projectID,
		TenantID: tenantID,
	}); errors.Is(err, pgx.ErrNoRows) {
		return apperror.New(apperror.CodeNotFound, "project not found")
	} else if err != nil {
		return wrapDBError(err, "verify project ownership")
	}
	return nil
}

func serviceRoleClaims(tenantID uuid.UUID) json.RawMessage {
	return json.RawMessage(fmt.Sprintf(`{"role":"service_role","sub":"%s"}`, tenantID.String()))
}

func isDuplicateKey(err error) bool {
	// pgx wraps PostgreSQL errors; SQLSTATE 23505 = unique_violation.
	return err != nil && (errors.Is(err, pgx.ErrNoRows) == false) && containsSQLState(err, "23505")
}

func containsSQLState(err error, code string) bool {
	var pgErr interface{ SQLState() string }
	if errors.As(err, &pgErr) {
		return pgErr.SQLState() == code
	}
	return false
}

func wrapDBError(err error, msg string) *apperror.AppError {
	if errors.Is(err, context.Canceled) || errors.Is(err, context.DeadlineExceeded) {
		return apperror.Wrap(apperror.CodeCanceled, msg, err)
	}
	return apperror.Wrap(apperror.CodeInternal, msg, err)
}
