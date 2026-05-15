package middleware_test

import (
	"context"
	"net/http/httptest"
	"testing"

	"connectrpc.com/connect"
	"google.golang.org/protobuf/types/known/wrapperspb"

	"github.com/kou-etal/etalbaas/pkg/middleware"
)

func TestRequestIDInterceptor_GeneratesID(t *testing.T) {
	var capturedID string
	interceptor := middleware.NewRequestIDInterceptor()

	handler := newEchoHandler(
		func(ctx context.Context, req *connect.Request[wrapperspb.StringValue]) (*connect.Response[wrapperspb.StringValue], error) {
			capturedID = middleware.RequestID(ctx)
			return connect.NewResponse(wrapperspb.String("ok")), nil
		},
		interceptor,
	)

	server := httptest.NewServer(handler)
	defer server.Close()

	client := newEchoClient(server.URL)
	_, err := client.CallUnary(context.Background(), connect.NewRequest(wrapperspb.String("test")))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if capturedID == "" {
		t.Fatal("expected generated request ID")
	}
}

func TestRequestIDInterceptor_PreservesExisting(t *testing.T) {
	var capturedID string
	interceptor := middleware.NewRequestIDInterceptor()

	handler := newEchoHandler(
		func(ctx context.Context, req *connect.Request[wrapperspb.StringValue]) (*connect.Response[wrapperspb.StringValue], error) {
			capturedID = middleware.RequestID(ctx)
			return connect.NewResponse(wrapperspb.String("ok")), nil
		},
		interceptor,
	)

	server := httptest.NewServer(handler)
	defer server.Close()

	client := newEchoClient(server.URL)
	req := connect.NewRequest(wrapperspb.String("test"))
	req.Header().Set(middleware.HeaderRequestID, "existing-id-123")

	_, err := client.CallUnary(context.Background(), req)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if capturedID != "existing-id-123" {
		t.Fatalf("got %q, want %q", capturedID, "existing-id-123")
	}
}
