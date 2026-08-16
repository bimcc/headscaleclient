package tailscale

import "testing"

func TestValidateLoginURL(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name  string
		value string
		valid bool
	}{
		{name: "https", value: "https://headscale.example/register/device", valid: true},
		{name: "https with port", value: "https://headscale.example:8443/register/device", valid: true},
		{name: "localhost http", value: "http://localhost:8080/register/device", valid: true},
		{name: "ipv4 loopback http", value: "http://127.0.0.1:8080/register/device", valid: true},
		{name: "ipv6 loopback http", value: "http://[::1]:8080/register/device", valid: true},
		{name: "file scheme", value: "file:///tmp/login", valid: false},
		{name: "custom scheme", value: "headscale://login/session", valid: false},
		{name: "remote http", value: "http://headscale.example/register/device", valid: false},
		{name: "credentials", value: "https://user:secret@headscale.example/register/device", valid: false},
		{name: "control character", value: "https://headscale.example/register\n/device", valid: false},
		{name: "missing host", value: "https:///register/device", valid: false},
		{name: "malformed", value: "https://[::1", valid: false},
		{name: "leading whitespace", value: " https://headscale.example/register/device", valid: false},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()
			got, err := ValidateLoginURL(test.value)
			if test.valid {
				if err != nil {
					t.Fatalf("ValidateLoginURL() error: %v", err)
				}
				if got != test.value {
					t.Fatalf("ValidateLoginURL() = %q, want %q", got, test.value)
				}
				return
			}
			if err == nil {
				t.Fatalf("ValidateLoginURL() accepted %q", test.value)
			}
			if got != "" {
				t.Fatalf("ValidateLoginURL() returned unsafe value %q", got)
			}
		})
	}
}
