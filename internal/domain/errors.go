package domain

import (
	"errors"
	"fmt"
)

type ErrorCode string

const (
	ErrorInvalidArgument          ErrorCode = "invalid-argument"
	ErrorNotFound                 ErrorCode = "not-found"
	ErrorConflict                 ErrorCode = "conflict"
	ErrorPreconditionFailed       ErrorCode = "precondition-failed"
	ErrorPermissionDenied         ErrorCode = "permission-denied"
	ErrorUnsupported              ErrorCode = "unsupported"
	ErrorCancelled                ErrorCode = "cancelled"
	ErrorTimeout                  ErrorCode = "timeout"
	ErrorDaemonMissing            ErrorCode = "daemon-missing"
	ErrorDaemonStopped            ErrorCode = "daemon-stopped"
	ErrorDaemonUnauthorized       ErrorCode = "daemon-unauthorized"
	ErrorDaemonIncompatible       ErrorCode = "daemon-incompatible"
	ErrorDaemonUnavailable        ErrorCode = "daemon-unavailable"
	ErrorControlUnavailable       ErrorCode = "control-unavailable"
	ErrorConfigurationInvalid     ErrorCode = "configuration-invalid"
	ErrorConfigurationReadFailed  ErrorCode = "configuration-read-failed"
	ErrorConfigurationWriteFailed ErrorCode = "configuration-write-failed"
	ErrorConfigurationUnsupported ErrorCode = "configuration-schema-unsupported"
	ErrorInternal                 ErrorCode = "internal"
)

// Problem is safe to serialize across the frontend boundary. Detail must not
// contain stack traces, auth material, or raw secret-bearing payloads.
type Problem struct {
	Code      ErrorCode `json:"code"`
	Message   string    `json:"message"`
	Detail    string    `json:"detail,omitempty"`
	Retryable bool      `json:"retryable"`
}

// AppError retains an internal cause without serializing it to the frontend.
type AppError struct {
	Problem Problem `json:"problem"`
	cause   error
}

func NewError(code ErrorCode, message string) *AppError {
	return &AppError{Problem: Problem{Code: code, Message: message}}
}

func WrapError(code ErrorCode, message string, cause error) *AppError {
	return &AppError{Problem: Problem{Code: code, Message: message}, cause: cause}
}

func (e *AppError) WithDetail(detail string) *AppError {
	e.Problem.Detail = detail
	return e
}

func (e *AppError) WithRetryable(retryable bool) *AppError {
	e.Problem.Retryable = retryable
	return e
}

func (e *AppError) Error() string {
	if e == nil {
		return "<nil>"
	}
	if e.cause != nil {
		return fmt.Sprintf("%s: %v", e.Problem.Message, e.cause)
	}
	return e.Problem.Message
}

func (e *AppError) Unwrap() error {
	if e == nil {
		return nil
	}
	return e.cause
}

func ProblemFromError(err error) *Problem {
	if err == nil {
		return nil
	}
	var appErr *AppError
	if errors.As(err, &appErr) {
		problem := appErr.Problem
		return &problem
	}
	return &Problem{Code: ErrorInternal, Message: "An unexpected error occurred."}
}
