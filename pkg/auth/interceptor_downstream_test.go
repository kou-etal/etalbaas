package auth_test

import (
	"context"
	"net/http/httptest"
	"testing"

	"connectrpc.com/connect"
	"github.com/google/uuid"
	"google.golang.org/protobuf/types/known/wrapperspb"

	"github.com/kou-etal/etalbaas/pkg/auth"
	"github.com/kou-etal/etalbaas/pkg/requestctx"
)

func TestDownstreamInterceptor_WithHeaders(t *testing.T) {
	interceptor := auth.NewDownstreamInterceptor()
	handler, cap := newEchoHandler(interceptor)

	server := httptest.NewServer(handler)
	defer server.Close()

	userID := uuid.MustParse("a0eebc99-9c0b-4ef8-bb6d-6bb9bd380a11")
	client := newEchoClient(server.URL)
	req := connect.NewRequest(wrapperspb.String("hello"))
	req.Header().Set(requestctx.HeaderUserID, userID.String())
	req.Header().Set(requestctx.HeaderProjectID, "proj-abc")

	_, err := client.CallUnary(context.Background(), req)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if cap.userID != userID {
		t.Fatalf("got user_id %v, want %v", cap.userID, userID)
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

func TestDownstreamInterceptor_InvalidUserID(t *testing.T) {
	interceptor := auth.NewDownstreamInterceptor()
	handler, _ := newEchoHandler(interceptor)

	server := httptest.NewServer(handler)
	defer server.Close()

	client := newEchoClient(server.URL)
	req := connect.NewRequest(wrapperspb.String("hello"))
	req.Header().Set(requestctx.HeaderUserID, "not-a-uuid")

	_, err := client.CallUnary(context.Background(), req)
	if err == nil {
		t.Fatal("expected error for invalid user_id")
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

	userID := uuid.MustParse("a0eebc99-9c0b-4ef8-bb6d-6bb9bd380a11")
	client := newEchoClient(server.URL)
	req := connect.NewRequest(wrapperspb.String("hello"))
	req.Header().Set(requestctx.HeaderUserID, userID.String())

	_, err := client.CallUnary(context.Background(), req)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if cap.userID != userID {
		t.Fatalf("got user_id %v, want %v", cap.userID, userID)
	}
	if cap.projectID != "" {
		t.Fatalf("got project_id %q, want empty", cap.projectID)
	}
}
