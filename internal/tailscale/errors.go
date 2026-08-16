package tailscale

import (
	"context"
	"errors"
	"strings"

	"github.com/headscaleclient/headscaleclient/internal/domain"
	"tailscale.com/client/local"
)

func classifyError(err error) error {
	if err == nil {
		return nil
	}
	if errors.Is(err, context.Canceled) {
		return domain.WrapError(domain.ErrorCancelled, "The operation was cancelled.", err)
	}
	if errors.Is(err, context.DeadlineExceeded) {
		return domain.WrapError(domain.ErrorTimeout, "The local Tailscale service did not respond in time.", err).
			WithRetryable(true)
	}

	var accessDenied *local.AccessDeniedError
	if errors.As(err, &accessDenied) {
		return domain.WrapError(domain.ErrorDaemonUnauthorized, "HeadscaleClient cannot access the local Tailscale service.", err).
			WithRetryable(true)
	}

	var precondition *local.PreconditionsFailedError
	if errors.As(err, &precondition) {
		return domain.WrapError(domain.ErrorPreconditionFailed, "The local Tailscale service rejected the requested state change.", err)
	}

	message := strings.ToLower(err.Error())
	if strings.Contains(message, "failed to connect to local tailscale daemon") ||
		strings.Contains(message, "the system cannot find the file specified") ||
		strings.Contains(message, "connection refused") ||
		strings.Contains(message, "no such file or directory") {
		return domain.WrapError(domain.ErrorDaemonUnavailable, "The local Tailscale service is not running or is not installed.", err).
			WithRetryable(true)
	}

	return domain.WrapError(domain.ErrorDaemonIncompatible, "The local Tailscale service returned an unsupported response.", err).
		WithRetryable(true)
}
