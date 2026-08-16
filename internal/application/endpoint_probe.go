package application

import (
	"context"
	"crypto/x509"
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/headscaleclient/headscaleclient/internal/domain"
)

const DefaultEndpointProbeTimeout = 5 * time.Second

type EndpointProbe interface {
	Probe(context.Context, string) error
}

type EndpointProbeFunc func(context.Context, string) error

func (f EndpointProbeFunc) Probe(ctx context.Context, baseURL string) error {
	return f(ctx, baseURL)
}

type httpEndpointProbe struct {
	client  *http.Client
	timeout time.Duration
}

func newHTTPEndpointProbe() EndpointProbe {
	return &httpEndpointProbe{
		client:  &http.Client{},
		timeout: DefaultEndpointProbeTimeout,
	}
}

func (p *httpEndpointProbe) Probe(ctx context.Context, baseURL string) error {
	endpoint, err := url.Parse(strings.TrimSpace(baseURL))
	if err != nil || endpoint.Host == "" {
		return domain.NewError(domain.ErrorInvalidArgument, "控制服务器地址无效。")
	}
	healthURL, err := url.JoinPath(endpoint.String(), "health")
	if err != nil {
		return domain.WrapError(domain.ErrorInvalidArgument, "控制服务器地址无效。", err)
	}

	probeCtx, cancel := context.WithTimeout(ctx, p.timeout)
	defer cancel()
	request, err := http.NewRequestWithContext(probeCtx, http.MethodGet, healthURL, nil)
	if err != nil {
		return domain.WrapError(domain.ErrorInvalidArgument, "控制服务器地址无效。", err)
	}
	request.Header.Set("Accept", "application/health+json, application/json, */*")
	request.Header.Set("User-Agent", "HeadscaleClient endpoint probe")

	response, err := p.client.Do(request)
	if err != nil {
		return endpointProbeError(endpoint.Host, probeCtx, err)
	}
	defer response.Body.Close()
	_, _ = io.CopyN(io.Discard, response.Body, 1024)
	if response.StatusCode >= http.StatusInternalServerError {
		return domain.NewError(
			domain.ErrorControlUnavailable,
			fmt.Sprintf("控制服务器 %s 当前不可用（HTTP %d），请检查地址或服务器状态。", endpoint.Host, response.StatusCode),
		).WithRetryable(true)
	}
	return nil
}

func endpointProbeError(host string, probeCtx context.Context, err error) error {
	reason := "无法建立连接"
	if errors.Is(probeCtx.Err(), context.DeadlineExceeded) || isTimeoutError(err) {
		reason = "连接超时"
	} else {
		var dnsError *net.DNSError
		var unknownAuthority x509.UnknownAuthorityError
		var hostnameError x509.HostnameError
		switch {
		case errors.As(err, &dnsError):
			reason = "域名无法解析"
		case errors.As(err, &unknownAuthority), errors.As(err, &hostnameError):
			reason = "TLS 证书验证失败"
		}
	}
	return domain.WrapError(
		domain.ErrorControlUnavailable,
		fmt.Sprintf("无法连接控制服务器 %s：%s。请检查服务器地址和网络。", host, reason),
		err,
	).WithRetryable(true)
}

func isTimeoutError(err error) bool {
	var networkError net.Error
	return errors.As(err, &networkError) && networkError.Timeout()
}
