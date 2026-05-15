package observability

import (
	"fmt"

	"connectrpc.com/connect"
	"connectrpc.com/otelconnect"
)

func ConnectInterceptor() (connect.Interceptor, error) {
	interceptor, err := otelconnect.NewInterceptor()
	if err != nil {
		return nil, fmt.Errorf("create otelconnect interceptor: %w", err)
	}
	return interceptor, nil
}
