package tailscale

import (
	"net"
	"net/url"
	"strings"
	"unicode"

	"github.com/headscaleclient/headscaleclient/internal/domain"
)

// ValidateLoginURL limits daemon-provided login targets to URLs that are safe
// to hand to the user's browser. HTTP is only accepted for local development.
func ValidateLoginURL(raw string) (string, error) {
	if raw == "" || raw != strings.TrimSpace(raw) || strings.IndexFunc(raw, unicode.IsControl) >= 0 {
		return "", unsafeLoginURLError()
	}

	parsed, err := url.Parse(raw)
	if err != nil || parsed.Opaque != "" || parsed.User != nil || parsed.Hostname() == "" {
		return "", unsafeLoginURLError()
	}

	switch {
	case strings.EqualFold(parsed.Scheme, "https"):
		return raw, nil
	case strings.EqualFold(parsed.Scheme, "http") && isLoopbackHost(parsed.Hostname()):
		return raw, nil
	default:
		return "", unsafeLoginURLError()
	}
}

func isLoopbackHost(host string) bool {
	if strings.EqualFold(host, "localhost") {
		return true
	}
	ip := net.ParseIP(host)
	return ip != nil && ip.IsLoopback()
}

func unsafeLoginURLError() error {
	return domain.NewError(
		domain.ErrorDaemonIncompatible,
		"The local Tailscale service provided an unsafe login URL.",
	)
}
