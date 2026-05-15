package auth_test

import (
	"context"
	"net/http/httptest"
	"testing"

	"connectrpc.com/connect"
	"google.golang.org/protobuf/types/known/wrapperspb"

	"github.com/kou-etal/etalbaas/pkg/auth"
	"github.com/kou-etal/etalbaas/pkg/requestctx"
)

func TestDownstreamInterceptor_WithHeaders(t *testing.T) {
	interceptor := auth.NewDownstreamInterceptor()
	handler, cap := newEchoHandler(interceptor)

	server := httptest.NewServer(handler)
	defer server.Close()

	client := newEchoClient(server.URL)
	req := connect.NewRequest(wrapperspb.String("hello"))
	req.Header().Set(requestctx.HeaderUserID, "user-789")
	req.Header().Set(requestctx.HeaderProjectID, "proj-abc")

	_, err := client.CallUnary(context.Background(), req)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if cap.userID != "user-789" {
		t.Fatalf("got user_id %q, want %q", cap.userID, "user-789")
	}
	if cap.projectID != "proj-abc" {
		t.Fatalf("got project_id %q, want %q", cap.projectID, "proj-abc")
	}
}

func TestDownstreamInterceptor_MissingUserID(t *testing.T) {
	interceptor := auth.NewDownstreamInterceptor()
	handler, _ := newEchoHandler(interceptor)

	server := httptest.NewServer(handler)
	defer server.Close()

	client := newEchoClient(server.URL)
	req := connect.NewRequest(wrapperspb.String("hello"))

	_, err := client.CallUnary(context.Background(), req)
	if err == nil {
		t.Fatal("expected error for missing user_id")
	}
	if code := connect.CodeOf(err); code != connect.CodeUnauthenticated {
		t.Fatalf("got code %v, want Unauthenticated", code)
	}
}

func TestDownstreamInterceptor_OptionalProjectID(t *testing.T) {
	interceptor := auth.NewDownstreamInterceptor()
	handler, cap := newEchoHandler(interceptor)

	server := httptest.NewServer(handler)
	defer server.Close()

	client := newEchoClient(server.URL)
	req := connect.NewRequest(wrapperspb.String("hello"))
	req.Header().Set(requestctx.HeaderUserID, "user-789")

	_, err := client.CallUnary(context.Background(), req)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if cap.userID != "user-789" {
		t.Fatalf("got user_id %q, want %q", cap.userID, "user-789")
	}
	if cap.projectID != "" {
		t.Fatalf("got project_id %q, want empty", cap.projectID)
	}
}
