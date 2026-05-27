package middleware_test

import (
	"bytes"
	"context"
	"log/slog"
	"net/http/httptest"
	"strings"
	"testing"

	"connectrpc.com/connect"
	"google.golang.org/protobuf/types/known/wrapperspb"

	"github.com/kou-etal/etalbaas/pkg/middleware"
)

func TestLoggingInterceptor_Success(t *testing.T) {
	var buf bytes.Buffer
	logger := slog.New(slog.NewTextHandler(&buf, nil))

	interceptor := middleware.NewLoggingInterceptor(logger)

	handler := newEchoHandler(
		func(ctx context.Context, req *connect.Request[wrapperspb.StringValue]) (*connect.Response[wrapperspb.StringValue], error) {
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

	output := buf.String()
	if !strings.Contains(output, "rpc completed") {
		t.Fatalf("expected 'rpc completed' in log, got: %s", output)
	}
}

func TestLoggingInterceptor_Error(t *testing.T) {
	var buf bytes.Buffer
	logger := slog.New(slog.NewTextHandler(&buf, nil))

	interceptor := middleware.NewLoggingInterceptor(logger)

	handler := newEchoHandler(
		func(ctx context.Context, req *connect.Request[wrapperspb.StringValue]) (*connect.Response[wrapperspb.StringValue], error) {
			return nil, connect.NewError(connect.CodeNotFound, nil)
		},
		interceptor,
	)

	server := httptest.NewServer(handler)
	defer server.Close()

	client := newEchoClient(server.URL)
	_, err := client.CallUnary(context.Background(), connect.NewRequest(wrapperspb.String("test")))
	if err == nil {
		t.Fatal("expected error")
	}

	output := buf.String()
	if !strings.Contains(output, "rpc failed") {
		t.Fatalf("expected 'rpc failed' in log, got: %s", output)
	}
}
