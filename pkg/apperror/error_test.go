package apperror_test

import (
	"context"
	"errors"
	"testing"

	"connectrpc.com/connect"

	"github.com/kou-etal/etalbaas/pkg/apperror"
)

func TestNew(t *testing.T) {
	err := apperror.New(apperror.CodeNotFound, "not found")
	if err.Code != apperror.CodeNotFound {
		t.Fatalf("got code %d, want %d", err.Code, apperror.CodeNotFound)
	}
	if err.Error() != "not found" {
		t.Fatalf("got %q, want %q", err.Error(), "not found")
	}
}

func TestWrap(t *testing.T) {
	cause := errors.New("db error")
	err := apperror.Wrap(apperror.CodeInternal, "query failed", cause)
	if err.Error() != "query failed: db error" {
		t.Fatalf("got %q", err.Error())
	}
	if !errors.Is(err, cause) {
		t.Fatal("Unwrap should return cause")
	}
}

func TestWrapNilReturnsNil(t *testing.T) {
	err := apperror.Wrap(apperror.CodeInternal, "should be nil", nil)
	if err != nil {
		t.Fatalf("Wrap(nil) should return nil, got %v", err)
	}
}

func TestToConnectError(t *testing.T) {
	tests := []struct {
		code     apperror.Code
		wantCode connect.Code
	}{
		{apperror.CodeNotFound, connect.CodeNotFound},
		{apperror.CodeAlreadyExists, connect.CodeAlreadyExists},
		{apperror.CodeInvalidArgument, connect.CodeInvalidArgument},
		{apperror.CodePermissionDenied, connect.CodePermissionDenied},
		{apperror.CodeUnauthenticated, connect.CodeUnauthenticated},
		{apperror.CodeFailedPrecondition, connect.CodeFailedPrecondition},
		{apperror.CodeResourceExhausted, connect.CodeResourceExhausted},
		{apperror.CodeInternal, connect.CodeInternal},
		{apperror.CodeCanceled, connect.CodeCanceled},
		{apperror.CodeDeadlineExceeded, connect.CodeDeadlineExceeded},
	}

	for _, tt := range tests {
		appErr := apperror.New(tt.code, "test")
		connectErr := apperror.ToConnectError(appErr)
		if connectErr.Code() != tt.wantCode {
			t.Errorf("code %d: got connect code %v, want %v", tt.code, connectErr.Code(), tt.wantCode)
		}
	}
}

func TestToConnectErrorNonAppError(t *testing.T) {
	err := errors.New("random error")
	connectErr := apperror.ToConnectError(err)
	if connectErr.Code() != connect.CodeInternal {
		t.Fatalf("non-AppError should map to CodeInternal, got %v", connectErr.Code())
	}
	if connectErr.Message() != "internal error" {
		t.Fatalf("non-AppError message should be generic, got %q", connectErr.Message())
	}
}

func TestToConnectErrorNil(t *testing.T) {
	connectErr := apperror.ToConnectError(nil)
	if connectErr != nil {
		t.Fatal("ToConnectError(nil) should return nil")
	}
}

func TestCodeUnknown(t *testing.T) {
	err := apperror.New(apperror.CodeUnknown, "unknown error")
	connectErr := apperror.ToConnectError(err)
	if connectErr.Code() != connect.CodeUnknown {
		t.Fatalf("CodeUnknown should map to connect.CodeUnknown, got %v", connectErr.Code())
	}
}

func TestToConnectErrorInternalMasksMessage(t *testing.T) {
	cause := errors.New("sensitive db details")
	appErr := apperror.Wrap(apperror.CodeInternal, "operation failed", cause)
	connectErr := apperror.ToConnectError(appErr)
	if connectErr.Message() != "internal error" {
		t.Fatalf("CodeInternal should mask message, got %q", connectErr.Message())
	}
}

func TestToConnectErrorContextCanceled(t *testing.T) {
	connectErr := apperror.ToConnectError(context.Canceled)
	if connectErr.Code() != connect.CodeCanceled {
		t.Fatalf("context.Canceled should map to CodeCanceled, got %v", connectErr.Code())
	}
}

func TestToConnectErrorContextDeadlineExceeded(t *testing.T) {
	connectErr := apperror.ToConnectError(context.DeadlineExceeded)
	if connectErr.Code() != connect.CodeDeadlineExceeded {
		t.Fatalf("context.DeadlineExceeded should map to CodeDeadlineExceeded, got %v", connectErr.Code())
	}
}

func TestCodeOf(t *testing.T) {
	err := apperror.New(apperror.CodeInvalidArgument, "bad input")
	if apperror.CodeOf(err) != apperror.CodeInvalidArgument {
		t.Fatalf("got %d, want CodeInvalidArgument", apperror.CodeOf(err))
	}
}

func TestCodeOfNonAppError(t *testing.T) {
	err := errors.New("random error")
	if apperror.CodeOf(err) != apperror.CodeUnknown {
		t.Fatalf("got %d, want CodeUnknown", apperror.CodeOf(err))
	}
}

func TestCodeOfWrapped(t *testing.T) {
	cause := errors.New("db error")
	appErr := apperror.Wrap(apperror.CodeInternal, "query failed", cause)
	wrapped := errors.Join(errors.New("context"), appErr)
	if apperror.CodeOf(wrapped) != apperror.CodeInternal {
		t.Fatalf("got %d, want CodeInternal", apperror.CodeOf(wrapped))
	}
}
