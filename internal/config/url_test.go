package config

import (
	"errors"
	"testing"

	"github.com/headscaleclient/headscaleclient/internal/domain"
)

func TestNormalizeControlURL(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		input   string
		want    string
		wantErr bool
	}{
		{name: "canonical HTTPS", input: "HTTPS://Example.COM:443/", want: "https://example.com"},
		{name: "canonical padded default port", input: "https://example.com:00443", want: "https://example.com"},
		{name: "preserve custom port and clean path", input: " https://Hs.Example:8443/headscale/ ", want: "https://hs.example:8443/headscale"},
		{name: "loopback IPv4 HTTP", input: "http://127.0.0.1:8080/", want: "http://127.0.0.1:8080"},
		{name: "loopback IPv6 HTTP", input: "http://[0:0:0:0:0:0:0:1]:8080", want: "http://[::1]:8080"},
		{name: "localhost subdomain HTTP", input: "http://dev.localhost:8080", want: "http://dev.localhost:8080"},
		{name: "empty", input: "", wantErr: true},
		{name: "relative", input: "example.com", wantErr: true},
		{name: "non-loopback HTTP", input: "http://headscale.example", wantErr: true},
		{name: "unsupported scheme", input: "ftp://localhost", wantErr: true},
		{name: "credentials", input: "https://user:password@example.com", wantErr: true},
		{name: "fragment", input: "https://example.com/#token", wantErr: true},
		{name: "empty fragment", input: "https://example.com/#", wantErr: true},
		{name: "query", input: "https://example.com/?tenant=a", wantErr: true},
		{name: "invalid port", input: "https://example.com:99999", wantErr: true},
		{name: "missing host", input: "https:///control", wantErr: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			got, err := NormalizeControlURL(tt.input)
			if tt.wantErr {
				if err == nil {
					t.Fatalf("NormalizeControlURL(%q) unexpectedly succeeded with %q", tt.input, got)
				}
				assertErrorCode(t, err, domain.ErrorInvalidArgument)
				return
			}
			if err != nil {
				t.Fatalf("NormalizeControlURL(%q) error: %v", tt.input, err)
			}
			if got != tt.want {
				t.Fatalf("NormalizeControlURL(%q) = %q, want %q", tt.input, got, tt.want)
			}
		})
	}
}

func assertErrorCode(t *testing.T, err error, want domain.ErrorCode) {
	t.Helper()
	var appErr *domain.AppError
	if !errors.As(err, &appErr) {
		t.Fatalf("error %T is not *domain.AppError: %v", err, err)
	}
	if appErr.Problem.Code != want {
		t.Fatalf("error code = %q, want %q (error: %v)", appErr.Problem.Code, want, err)
	}
}
