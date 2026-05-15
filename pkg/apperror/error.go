package apperror

import (
	"context"
	"errors"
	"fmt"

	"connectrpc.com/connect"
)

type Code int

const (
	CodeUnknown Code = iota
	CodeNotFound
	CodeAlreadyExists
	CodeInvalidArgument
	CodePermissionDenied
	CodeUnauthenticated
	CodeFailedPrecondition
	CodeResourceExhausted
	CodeInternal
	CodeCanceled
	CodeDeadlineExceeded
)

type AppError struct {
	Code    Code
	Message string
	Err     error
}

func (e *AppError) Error() string {
	if e.Err != nil {
		return fmt.Sprintf("%s: %v", e.Message, e.Err)
	}
	return e.Message
}

func (e *AppError) Unwrap() error {
	return e.Err
}

func New(code Code, msg string) *AppError {
	return &AppError{Code: code, Message: msg}
}

func Wrap(code Code, msg string, err error) *AppError {
	if err == nil {
		return nil
	}
	return &AppError{Code: code, Message: msg, Err: err}
}

func ToConnectError(err error) *connect.Error {
	if err == nil {
		return nil
	}
	// Map context errors before falling through to generic internal.
	if errors.Is(err, context.Canceled) {
		return connect.NewError(connect.CodeCanceled, errors.New("request canceled"))
	}
	if errors.Is(err, context.DeadlineExceeded) {
		return connect.NewError(connect.CodeDeadlineExceeded, errors.New("deadline exceeded"))
	}
	var appErr *AppError
	if !errors.As(err, &appErr) {
		return connect.NewError(connect.CodeInternal, errors.New("internal error"))
	}
	code := toConnectCode(appErr.Code)
	// Mask internal error messages to avoid leaking implementation details.
	if code == connect.CodeInternal {
		return connect.NewError(code, errors.New("internal error"))
	}
	return connect.NewError(code, errors.New(appErr.Message))
}

func CodeOf(err error) Code {
	var appErr *AppError
	if errors.As(err, &appErr) {
		return appErr.Code
	}
	return CodeUnknown
}

func toConnectCode(code Code) connect.Code {
	switch code {
	case CodeUnknown:
		return connect.CodeUnknown
	case CodeNotFound:
		return connect.CodeNotFound
	case CodeAlreadyExists:
		return connect.CodeAlreadyExists
	case CodeInvalidArgument:
		return connect.CodeInvalidArgument
	case CodePermissionDenied:
		return connect.CodePermissionDenied
	case CodeUnauthenticated:
		return connect.CodeUnauthenticated
	case CodeFailedPrecondition:
		return connect.CodeFailedPrecondition
	case CodeResourceExhausted:
		return connect.CodeResourceExhausted
	case CodeInternal:
		return connect.CodeInternal
	case CodeCanceled:
		return connect.CodeCanceled
	case CodeDeadlineExceeded:
		return connect.CodeDeadlineExceeded
	default:
		return connect.CodeInternal
	}
}
