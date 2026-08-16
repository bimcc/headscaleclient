package tailscale

import (
	"context"
	"errors"
	"io"
	"net/http"
	"strings"
	"testing"

	"github.com/headscaleclient/headscaleclient/internal/domain"
	"tailscale.com/client/local"
)

type roundTripFunc func(*http.Request) (*http.Response, error)

func (f roundTripFunc) RoundTrip(request *http.Request) (*http.Response, error) {
	return f(request)
}

func TestClassifyLocalAPIResponses(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name      string
		status    int
		body      string
		wantCode  domain.ErrorCode
		retryable bool
	}{
		{name: "access denied", status: http.StatusForbidden, body: `{"Error":"forbidden"}`, wantCode: domain.ErrorDaemonUnauthorized, retryable: true},
		{name: "precondition failed", status: http.StatusPreconditionFailed, body: `{"Error":"precondition"}`, wantCode: domain.ErrorPreconditionFailed},
		{name: "malformed JSON", status: http.StatusOK, body: `{`, wantCode: domain.ErrorDaemonIncompatible, retryable: true},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()

			client := &local.Client{
				OmitAuth: true,
				Transport: roundTripFunc(func(request *http.Request) (*http.Response, error) {
					return &http.Response{
						StatusCode: test.status,
						Status:     http.StatusText(test.status),
						Header:     make(http.Header),
						Body:       io.NopCloser(strings.NewReader(test.body)),
						Request:    request,
					}, nil
				}),
			}

			_, err := client.Status(context.Background())
			assertClassifiedError(t, classifyError(err), test.wantCode, test.retryable)
		})
	}
}

func TestClassifyContextErrors(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name      string
		err       error
		wantCode  domain.ErrorCode
		retryable bool
	}{
		{name: "cancelled", err: context.Canceled, wantCode: domain.ErrorCancelled},
		{name: "deadline", err: context.DeadlineExceeded, wantCode: domain.ErrorTimeout, retryable: true},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()
			assertClassifiedError(t, classifyError(test.err), test.wantCode, test.retryable)
		})
	}
}

func assertClassifiedError(t *testing.T, err error, wantCode domain.ErrorCode, retryable bool) {
	t.Helper()

	var appErr *domain.AppError
	if !errors.As(err, &appErr) {
		t.Fatalf("error %T is not *domain.AppError: %v", err, err)
	}
	if appErr.Problem.Code != wantCode {
		t.Fatalf("error code = %q, want %q", appErr.Problem.Code, wantCode)
	}
	if appErr.Problem.Retryable != retryable {
		t.Fatalf("retryable = %t, want %t", appErr.Problem.Retryable, retryable)
	}
}
