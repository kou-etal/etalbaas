//go:build e2e

package e2e_test

import (
	"context"
	"testing"

	"github.com/google/uuid"

	projectv1 "github.com/kou-etal/etalbaas/proto/gen/go/etalbaas/project/v1"
	storagev1 "github.com/kou-etal/etalbaas/proto/gen/go/etalbaas/storage/v1"
)

func TestBucket_Create(t *testing.T) {
	truncateAll(t)

	tenantID := uuid.New()
	createTestTenant(t, tenantID, "alice@example.com")

	projClient := newProjectClient()
	ctx := context.Background()

	createResp, err := projClient.CreateProject(ctx, authedRequest(t, tenantID, &projectv1.CreateProjectRequest{
		DisplayName:     "Storage Project",
		PostgresEnabled: true,
	}))
	if err != nil {
		t.Fatalf("CreateProject: %v", err)
	}
	projectID := createResp.Msg.Project.Id

	storClient := newStorageClient()
	resp, err := storClient.CreateBucket(ctx, authedRequest(t, tenantID, &storagev1.CreateBucketRequest{
		ProjectId: projectID,
		Name:      "my-bucket",
		AccessLevel: "private",
	}))
	if err != nil {
		t.Fatalf("CreateBucket: %v", err)
	}
	if resp.Msg.Bucket == nil {
		t.Fatal("expected bucket in response")
	}
	if resp.Msg.Bucket.Name != "my-bucket" {
		t.Fatalf("got name %q, want %q", resp.Msg.Bucket.Name, "my-bucket")
	}
}

func TestBucket_List(t *testing.T) {
	truncateAll(t)

	tenantID := uuid.New()
	createTestTenant(t, tenantID, "alice@example.com")

	projClient := newProjectClient()
	ctx := context.Background()

	createResp, err := projClient.CreateProject(ctx, authedRequest(t, tenantID, &projectv1.CreateProjectRequest{
		DisplayName:     "Bucket List",
		PostgresEnabled: true,
	}))
	if err != nil {
		t.Fatalf("CreateProject: %v", err)
	}
	projectID := createResp.Msg.Project.Id

	storClient := newStorageClient()
	for _, name := range []string{"bucket-a", "bucket-b"} {
		_, err := storClient.CreateBucket(ctx, authedRequest(t, tenantID, &storagev1.CreateBucketRequest{
			ProjectId: projectID,
			Name:      name,
			AccessLevel: "private",
		}))
		if err != nil {
			t.Fatalf("CreateBucket(%s): %v", name, err)
		}
	}

	listResp, err := storClient.ListBuckets(ctx, authedRequest(t, tenantID, &storagev1.ListBucketsRequest{
		ProjectId: projectID,
	}))
	if err != nil {
		t.Fatalf("ListBuckets: %v", err)
	}
	if len(listResp.Msg.Buckets) != 2 {
		t.Fatalf("got %d buckets, want 2", len(listResp.Msg.Buckets))
	}
}
