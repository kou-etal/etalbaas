package handler

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"connectrpc.com/connect"
	"github.com/google/uuid"

	"github.com/kou-etal/etalbaas/pkg/apperror"
	"github.com/kou-etal/etalbaas/pkg/requestctx"
	storagev1 "github.com/kou-etal/etalbaas/proto/gen/go/etalbaas/storage/v1"
	"github.com/kou-etal/etalbaas/proto/gen/go/etalbaas/storage/v1/storagev1connect"
	"github.com/kou-etal/etalbaas/services/storage/internal/service"
	"github.com/kou-etal/etalbaas/services/storage/internal/tenantstore"
)

// stubBucketService wraps a real BucketService but is nil-safe for testing auth.
type testServer struct {
	handler *BucketHandler
	server  *httptest.Server
	client  storagev1connect.StorageServiceClient
}

func newTestServer(svc *service.BucketService) *testServer {
	h := NewBucketHandler(svc)
	path, handler := storagev1connect.NewStorageServiceHandler(h)
	mux := http.NewServeMux()
	mux.Handle(path, handler)
	srv := httptest.NewServer(mux)
	client := storagev1connect.NewStorageServiceClient(srv.Client(), srv.URL)
	return &testServer{handler: h, server: srv, client: client}
}

func (ts *testServer) close() {
	ts.server.Close()
}

func TestBucketHandler_CreateBucket_Unauthenticated(t *testing.T) {
	ts := newTestServer(nil) // svc is nil, but should fail before reaching it
	defer ts.close()

	_, err := ts.client.CreateBucket(context.Background(), connect.NewRequest(&storagev1.CreateBucketRequest{
		ProjectId: "proj1",
		Name:      "test-bucket",
	}))
	if err == nil {
		t.Fatal("expected unauthenticated error, got nil")
	}
	if connect.CodeOf(err) != connect.CodeUnauthenticated {
		t.Errorf("expected Unauthenticated code, got %v", connect.CodeOf(err))
	}
}

func TestBucketHandler_ListBuckets_Unauthenticated(t *testing.T) {
	ts := newTestServer(nil)
	defer ts.close()

	_, err := ts.client.ListBuckets(context.Background(), connect.NewRequest(&storagev1.ListBucketsRequest{
		ProjectId: "proj1",
	}))
	if err == nil {
		t.Fatal("expected unauthenticated error, got nil")
	}
	if connect.CodeOf(err) != connect.CodeUnauthenticated {
		t.Errorf("expected Unauthenticated code, got %v", connect.CodeOf(err))
	}
}

func TestBucketConverter_ToProto(t *testing.T) {
	limit := int64(10485760)
	bucket := tenantstore.StorageBucket{
		ID:               "proj1-images",
		ProjectID:        "proj1",
		Name:             "images",
		AccessLevel:      "public",
		FileSizeLimit:    &limit,
		AllowedMimeTypes: []string{"image/png", "image/jpeg"},
		CreatedAt:        time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC),
		UpdatedAt:        time.Date(2025, 1, 2, 0, 0, 0, 0, time.UTC),
	}

	pb := BucketToProto(bucket)

	if pb.Id != "proj1-images" {
		t.Errorf("Id = %q, want %q", pb.Id, "proj1-images")
	}
	if pb.Name != "images" {
		t.Errorf("Name = %q, want %q", pb.Name, "images")
	}
	if pb.AccessLevel != "public" {
		t.Errorf("AccessLevel = %q, want %q", pb.AccessLevel, "public")
	}
	if pb.FileSizeLimit != 10485760 {
		t.Errorf("FileSizeLimit = %d, want %d", pb.FileSizeLimit, 10485760)
	}
	if len(pb.AllowedMimeTypes) != 2 {
		t.Errorf("AllowedMimeTypes len = %d, want 2", len(pb.AllowedMimeTypes))
	}
}

func TestBucketConverter_OffsetPagination(t *testing.T) {
	// Encode then decode
	token := encodeOffsetToken(40)
	offset, err := decodeOffsetToken(token)
	if err != nil {
		t.Fatalf("decodeOffsetToken error: %v", err)
	}
	if offset != 40 {
		t.Errorf("offset = %d, want 40", offset)
	}

	// OffsetPaginationToProto returns nil when we returned fewer than page_size
	resp := OffsetPaginationToProto(0, 20, 15)
	if resp != nil {
		t.Error("expected nil pagination when returned < page_size")
	}

	// Returns next token when returned == page_size
	resp = OffsetPaginationToProto(0, 20, 20)
	if resp == nil {
		t.Fatal("expected pagination response")
	}
	if resp.NextPageToken == "" {
		t.Error("expected non-empty NextPageToken")
	}
}

func TestExtractTenantID(t *testing.T) {
	// No user ID in context
	_, err := extractTenantID(context.Background())
	if err == nil {
		t.Fatal("expected error for missing user ID")
	}

	// With user ID
	uid := uuid.New()
	ctx := requestctx.WithUserID(context.Background(), uid)
	got, err := extractTenantID(ctx)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got != uid {
		t.Errorf("extractTenantID = %v, want %v", got, uid)
	}
}

// Verify the handler compiles with the correct interface.
var _ storagev1connect.StorageServiceHandler = (*BucketHandler)(nil)

// Verify apperror import works for connect error conversion.
var _ = apperror.ToConnectError
