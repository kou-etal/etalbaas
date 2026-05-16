// Package s3compat provides an S3-compatible implementation of ObjectStorage.
// It works with R2, MinIO, and any S3-compatible object storage via minio-go.
package s3compat

import (
	"context"
	"fmt"
	"io"
	"net/url"
	"time"

	"github.com/minio/minio-go/v7"
	"github.com/minio/minio-go/v7/pkg/credentials"

	storageproviders "github.com/kou-etal/etalbaas/storage-providers"
)

// Config holds configuration for S3-compatible storage.
type Config struct {
	Endpoint        string
	AccessKeyID     string
	SecretAccessKey  string
	Region          string
	UseSSL          bool
	BucketName      string // shared S3 bucket (e.g., "etalbaas-storage")
}

// S3CompatibleStorage implements ObjectStorage using minio-go SDK.
type S3CompatibleStorage struct {
	client     *minio.Client
	bucketName string
}

// New creates a new S3CompatibleStorage.
func New(cfg Config) (*S3CompatibleStorage, error) {
	client, err := minio.New(cfg.Endpoint, &minio.Options{
		Creds:  credentials.NewStaticV4(cfg.AccessKeyID, cfg.SecretAccessKey, ""),
		Secure: cfg.UseSSL,
		Region: cfg.Region,
	})
	if err != nil {
		return nil, fmt.Errorf("s3compat: create client: %w", err)
	}
	return &S3CompatibleStorage{
		client:     client,
		bucketName: cfg.BucketName,
	}, nil
}

// EnsureBucket creates the shared S3 bucket if it does not exist.
func (s *S3CompatibleStorage) EnsureBucket(ctx context.Context) error {
	exists, err := s.client.BucketExists(ctx, s.bucketName)
	if err != nil {
		return fmt.Errorf("s3compat: check bucket: %w", err)
	}
	if exists {
		return nil
	}
	if err := s.client.MakeBucket(ctx, s.bucketName, minio.MakeBucketOptions{}); err != nil {
		return fmt.Errorf("s3compat: create bucket: %w", err)
	}
	return nil
}

func (s *S3CompatibleStorage) Put(ctx context.Context, key string, data io.Reader, size int64, contentType string) error {
	opts := minio.PutObjectOptions{
		ContentType: contentType,
	}
	// size -1 means unknown; minio-go handles chunked upload.
	_, err := s.client.PutObject(ctx, s.bucketName, key, data, size, opts)
	if err != nil {
		return fmt.Errorf("s3compat: put %q: %w", key, err)
	}
	return nil
}

func (s *S3CompatibleStorage) Get(ctx context.Context, key string) (io.ReadCloser, *storageproviders.ObjectInfo, error) {
	obj, err := s.client.GetObject(ctx, s.bucketName, key, minio.GetObjectOptions{})
	if err != nil {
		return nil, nil, fmt.Errorf("s3compat: get %q: %w", key, err)
	}
	stat, err := obj.Stat()
	if err != nil {
		obj.Close()
		return nil, nil, fmt.Errorf("s3compat: stat %q: %w", key, err)
	}
	info := &storageproviders.ObjectInfo{
		Key:          stat.Key,
		Size:         stat.Size,
		ContentType:  stat.ContentType,
		ETag:         stat.ETag,
		LastModified: stat.LastModified,
	}
	return obj, info, nil
}

func (s *S3CompatibleStorage) Delete(ctx context.Context, key string) error {
	err := s.client.RemoveObject(ctx, s.bucketName, key, minio.RemoveObjectOptions{})
	if err != nil {
		return fmt.Errorf("s3compat: delete %q: %w", key, err)
	}
	return nil
}

func (s *S3CompatibleStorage) Head(ctx context.Context, key string) (*storageproviders.ObjectInfo, error) {
	stat, err := s.client.StatObject(ctx, s.bucketName, key, minio.StatObjectOptions{})
	if err != nil {
		return nil, fmt.Errorf("s3compat: head %q: %w", key, err)
	}
	return &storageproviders.ObjectInfo{
		Key:          stat.Key,
		Size:         stat.Size,
		ContentType:  stat.ContentType,
		ETag:         stat.ETag,
		LastModified: stat.LastModified,
	}, nil
}

func (s *S3CompatibleStorage) List(ctx context.Context, prefix string, opts storageproviders.ListOptions) ([]storageproviders.ObjectInfo, error) {
	maxKeys := opts.MaxKeys
	if maxKeys <= 0 {
		maxKeys = 1000
	}

	listOpts := minio.ListObjectsOptions{
		Prefix:     prefix,
		MaxKeys:    maxKeys,
		StartAfter: opts.StartAfter,
		Recursive:  true,
	}

	var objects []storageproviders.ObjectInfo
	for obj := range s.client.ListObjects(ctx, s.bucketName, listOpts) {
		if obj.Err != nil {
			return nil, fmt.Errorf("s3compat: list %q: %w", prefix, obj.Err)
		}
		objects = append(objects, storageproviders.ObjectInfo{
			Key:          obj.Key,
			Size:         obj.Size,
			ContentType:  obj.ContentType,
			ETag:         obj.ETag,
			LastModified: obj.LastModified,
		})
		if len(objects) >= maxKeys {
			break
		}
	}
	return objects, nil
}

func (s *S3CompatibleStorage) PresignedPutURL(ctx context.Context, key string, expiry time.Duration) (string, error) {
	u, err := s.client.PresignedPutObject(ctx, s.bucketName, key, expiry)
	if err != nil {
		return "", fmt.Errorf("s3compat: presigned put %q: %w", key, err)
	}
	return u.String(), nil
}

func (s *S3CompatibleStorage) PresignedGetURL(ctx context.Context, key string, expiry time.Duration) (string, error) {
	u, err := s.client.PresignedGetObject(ctx, s.bucketName, key, expiry, url.Values{})
	if err != nil {
		return "", fmt.Errorf("s3compat: presigned get %q: %w", key, err)
	}
	return u.String(), nil
}

// Compile-time interface check.
var _ storageproviders.ObjectStorage = (*S3CompatibleStorage)(nil)
