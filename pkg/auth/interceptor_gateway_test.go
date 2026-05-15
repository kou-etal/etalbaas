package auth_test

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"connectrpc.com/connect"
	"google.golang.org/protobuf/types/known/wrapperspb"

	"github.com/kou-etal/etalbaas/pkg/auth"
	"github.com/kou-etal/etalbaas/pkg/requestctx"
)

type capturedContext struct {
	userID    string
	projectID string
}

func newEchoHandler(interceptors ...connect.Interceptor) (http.Handler, *capturedContext) {
	cap := &capturedContext{}
	mux := http.NewServeMux()
	mux.Handle("/test.v1.TestService/Echo", connect.NewUnaryHandler(
		"/test.v1.TestService/Echo",
		func(ctx context.Context, req *connect.Request[wrapperspb.StringValue]) (*connect.Response[wrapperspb.StringValue], error) {
			cap.userID, _ = requestctx.UserID(ctx)
			cap.projectID, _ = requestctx.ProjectID(ctx)
			return connect.NewResponse(wrapperspb.String(req.Msg.GetValue())), nil
		},
		connect.WithInterceptors(interceptors...),
	))
	return mux, cap
}

func newEchoClient(serverURL string) *connect.Client[wrapperspb.StringValue, wrapperspb.StringValue] {
	return connect.NewClient[wrapperspb.StringValue, wrapperspb.StringValue](
		http.DefaultClient,
		serverURL+"/test.v1.TestService/Echo",
	)
}

func TestGatewayInterceptor_ValidToken(t *testing.T) {
	tokenStr := createTestToken(t, "user-456", time.Now().Add(time.Hour))

	interceptor := auth.NewGatewayInterceptor(testSigningKey)
	handler, cap := newEchoHandler(interceptor)

	server := httptest.NewServer(handler)
	defer server.Close()

	client := newEchoClient(server.URL)
	req := connect.NewRequest(wrapperspb.String("hello"))
	req.Header().Set("Authorization", "Bearer "+tokenStr)

	_, err := client.CallUnary(context.Background(), req)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if cap.userID != "user-456" {
		t.Fatalf("got user_id %q, want %q", cap.userID, "user-456")
	}
}

func TestGatewayInterceptor_MissingAuth(t *testing.T) {
	interceptor := auth.NewGatewayInterceptor(testSigningKey)
	handler, _ := newEchoHandler(interceptor)

	server := httptest.NewServer(handler)
	defer server.Close()

	client := newEchoClient(server.URL)
	req := connect.NewRequest(wrapperspb.String("hello"))

	_, err := client.CallUnary(context.Background(), req)
	if err == nil {
		t.Fatal("expected error for missing auth")
	}
	connectErr := new(connect.Error)
	if ok := connect.CodeOf(err); ok != connect.CodeUnauthenticated {
		t.Fatalf("got code %v, want Unauthenticated", ok)
	}
	_ = connectErr
}

func TestGatewayInterceptor_InvalidToken(t *testing.T) {
	interceptor := auth.NewGatewayInterceptor(testSigningKey)
	handler, _ := newEchoHandler(interceptor)

	server := httptest.NewServer(handler)
	defer server.Close()

	client := newEchoClient(server.URL)
	req := connect.NewRequest(wrapperspb.String("hello"))
	req.Header().Set("Authorization", "Bearer invalid-token")

	_, err := client.CallUnary(context.Background(), req)
	if err == nil {
		t.Fatal("expected error for invalid token")
	}
	if code := connect.CodeOf(err); code != connect.CodeUnauthenticated {
		t.Fatalf("got code %v, want Unauthenticated", code)
	}
}
