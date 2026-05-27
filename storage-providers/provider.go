package storageproviders

import (
	"context"
	"io"
	"time"
)

// ObjectStorage abstracts S3-compatible object storage operations.
// The bucket parameter is part of the Config (single shared S3 bucket).
// The key parameter uses the format: project-{id}/{bucket-name}/{object-path}.
type ObjectStorage interface {
	Put(ctx context.Context, key string, data io.Reader, size int64, contentType string) error
	Get(ctx context.Context, key string) (io.ReadCloser, *ObjectInfo, error)
	Delete(ctx context.Context, key string) error
	Head(ctx context.Context, key string) (*ObjectInfo, error)
	List(ctx context.Context, prefix string, opts ListOptions) ([]ObjectInfo, error)
	PresignedPutURL(ctx context.Context, key string, expiry time.Duration) (string, error)
	PresignedGetURL(ctx context.Context, key string, expiry time.Duration) (string, error)
}

// ObjectInfo holds metadata about a stored object.
type ObjectInfo struct {
	Key          string
	Size         int64
	ContentType  string
	ETag         string
	LastModified time.Time
}

// ListOptions configures object listing behavior.
type ListOptions struct {
	MaxKeys    int
	StartAfter string
}
