package s3compat

import (
	"testing"

	storageproviders "github.com/kou-etal/etalbaas/storage-providers"
)

func TestInterfaceCompliance(t *testing.T) {
	// Compile-time check that S3CompatibleStorage implements ObjectStorage.
	var _ storageproviders.ObjectStorage = (*S3CompatibleStorage)(nil)
}

func TestNew(t *testing.T) {
	s, err := New(Config{
		Endpoint:       "localhost:9000",
		AccessKeyID:    "minioadmin",
		SecretAccessKey: "minioadmin",
		Region:         "us-east-1",
		UseSSL:         false,
		BucketName:     "test-bucket",
	})
	if err != nil {
		t.Fatalf("New() error = %v", err)
	}
	if s.bucketName != "test-bucket" {
		t.Errorf("bucketName = %q, want %q", s.bucketName, "test-bucket")
	}
}
