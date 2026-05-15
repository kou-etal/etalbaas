package apperror

import (
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
	return &AppError{Code: code, Message: msg, Err: err}
}

func ToConnectError(err error) *connect.Error {
	if err == nil {
		return nil
	}
	var appErr *AppError
	if !errors.As(err, &appErr) {
		return connect.NewError(connect.CodeInternal, errors.New("internal error"))
	}
	return connect.NewError(toConnectCode(appErr.Code), errors.New(appErr.Message))
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
	default:
		return connect.CodeInternal
	}
}
