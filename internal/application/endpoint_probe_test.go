package application

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/headscaleclient/headscaleclient/internal/domain"
)

func TestHTTPEndpointProbe(t *testing.T) {
	t.Parallel()

	t.Run("healthy", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
			if request.URL.Path != "/health" {
				t.Fatalf("probe path = %q", request.URL.Path)
			}
			writer.WriteHeader(http.StatusOK)
		}))
		defer server.Close()

		probe := &httpEndpointProbe{client: server.Client(), timeout: time.Second}
		if err := probe.Probe(context.Background(), server.URL); err != nil {
			t.Fatalf("Probe() error: %v", err)
		}
	})

	t.Run("unsupported health endpoint still proves reachability", func(t *testing.T) {
		server := httptest.NewServer(http.NotFoundHandler())
		defer server.Close()

		probe := &httpEndpointProbe{client: server.Client(), timeout: time.Second}
		if err := probe.Probe(context.Background(), server.URL); err != nil {
			t.Fatalf("Probe() error: %v", err)
		}
	})

	t.Run("server unavailable", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, _ *http.Request) {
			writer.WriteHeader(http.StatusServiceUnavailable)
		}))
		defer server.Close()

		probe := &httpEndpointProbe{client: server.Client(), timeout: time.Second}
		err := probe.Probe(context.Background(), server.URL)
		assertProbeErrorCode(t, err, domain.ErrorControlUnavailable)
	})

	t.Run("connection failure", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(http.ResponseWriter, *http.Request) {}))
		serverURL := server.URL
		server.Close()

		probe := &httpEndpointProbe{client: server.Client(), timeout: time.Second}
		err := probe.Probe(context.Background(), serverURL)
		assertProbeErrorCode(t, err, domain.ErrorControlUnavailable)
	})
}

func assertProbeErrorCode(t *testing.T, err error, code domain.ErrorCode) {
	t.Helper()
	var appError *domain.AppError
	if !errors.As(err, &appError) || appError.Problem.Code != code {
		t.Fatalf("error = %#v, want code %q", err, code)
	}
}
