package config

import (
	"net"
	"net/url"
	"path"
	"strconv"
	"strings"

	"github.com/headscaleclient/headscaleclient/internal/domain"
)

const maxControlURLLength = 2048

// NormalizeControlURL validates a control-plane base URL. Plain HTTP is only
// accepted for loopback development endpoints.
func NormalizeControlURL(raw string) (string, error) {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return "", invalidURL("Control server URL is required.")
	}
	if len(raw) > maxControlURLLength {
		return "", invalidURL("Control server URL is too long.")
	}

	u, err := url.Parse(raw)
	if err != nil || u.Opaque != "" {
		return "", invalidURL("Control server URL is invalid.")
	}
	if u.User != nil {
		return "", invalidURL("Control server URL cannot contain credentials.")
	}
	if strings.Contains(raw, "#") {
		return "", invalidURL("Control server URL cannot contain a fragment.")
	}
	if strings.Contains(raw, "?") {
		return "", invalidURL("Control server URL cannot contain a query string.")
	}

	scheme := strings.ToLower(u.Scheme)
	if scheme != "https" && scheme != "http" {
		return "", invalidURL("Control server URL must use HTTPS.")
	}
	hostname := strings.ToLower(u.Hostname())
	if hostname == "" {
		return "", invalidURL("Control server URL must include a host.")
	}
	if scheme != "https" && !isLoopbackHost(hostname) {
		return "", invalidURL("Non-loopback control servers must use HTTPS.")
	}

	port := u.Port()
	if port != "" {
		portNumber, err := strconv.Atoi(port)
		if err != nil || portNumber < 1 || portNumber > 65535 {
			return "", invalidURL("Control server URL contains an invalid port.")
		}
		port = strconv.Itoa(portNumber)
		if (scheme == "https" && portNumber == 443) || (scheme == "http" && portNumber == 80) {
			port = ""
		}
	}

	if ip := net.ParseIP(hostname); ip != nil {
		hostname = ip.String()
	}
	host := hostname
	if port != "" {
		host = net.JoinHostPort(hostname, port)
	} else if strings.Contains(hostname, ":") {
		host = "[" + hostname + "]"
	}

	cleanPath := u.Path
	if cleanPath == "" || cleanPath == "/" {
		cleanPath = ""
	} else {
		cleanPath = path.Clean("/" + strings.TrimPrefix(cleanPath, "/"))
	}

	normalized := (&url.URL{Scheme: scheme, Host: host, Path: cleanPath}).String()
	return normalized, nil
}

func isLoopbackHost(hostname string) bool {
	trimmed := strings.TrimSuffix(hostname, ".")
	if trimmed == "localhost" || strings.HasSuffix(trimmed, ".localhost") {
		return true
	}
	ip := net.ParseIP(trimmed)
	return ip != nil && ip.IsLoopback()
}

func invalidURL(message string) *domain.AppError {
	return domain.NewError(domain.ErrorInvalidArgument, message).WithDetail("baseURL")
}
