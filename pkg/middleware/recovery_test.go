package middleware_test

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"connectrpc.com/connect"
	"google.golang.org/protobuf/types/known/wrapperspb"

	"github.com/kou-etal/etalbaas/pkg/middleware"
)

func newEchoHandler(handler func(context.Context, *connect.Request[wrapperspb.StringValue]) (*connect.Response[wrapperspb.StringValue], error), interceptors ...connect.Interceptor) http.Handler {
	mux := http.NewServeMux()
	mux.Handle("/test.v1.TestService/Echo", connect.NewUnaryHandler(
		"/test.v1.TestService/Echo",
		handler,
		connect.WithInterceptors(interceptors...),
	))
	return mux
}

func newEchoClient(serverURL string) *connect.Client[wrapperspb.StringValue, wrapperspb.StringValue] {
	return connect.NewClient[wrapperspb.StringValue, wrapperspb.StringValue](
		http.DefaultClient,
		serverURL+"/test.v1.TestService/Echo",
	)
}

func TestRecoveryInterceptor_NoPanic(t *testing.T) {
	interceptor := middleware.NewRecoveryInterceptor()

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
}

func TestRecoveryInterceptor_WithPanic(t *testing.T) {
	interceptor := middleware.NewRecoveryInterceptor()

	handler := newEchoHandler(
		func(ctx context.Context, req *connect.Request[wrapperspb.StringValue]) (*connect.Response[wrapperspb.StringValue], error) {
			panic("something went wrong")
		},
		interceptor,
	)

	server := httptest.NewServer(handler)
	defer server.Close()

	client := newEchoClient(server.URL)
	_, err := client.CallUnary(context.Background(), connect.NewRequest(wrapperspb.String("test")))
	if err == nil {
		t.Fatal("expected error after panic")
	}
	if code := connect.CodeOf(err); code != connect.CodeInternal {
		t.Fatalf("got code %v, want Internal", code)
	}
}
