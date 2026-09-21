//go:build android

package engine

import (
	"context"
	"errors"
	"io"
	"net"
	"net/http"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/tailscale/tailscale-android/libtailscale"
	"tailscale.com/ipn"
)

type transport struct {
	app   libtailscale.Application
	slots chan struct{}
}

func newTransport(app libtailscale.Application) *transport {
	return &transport{app: app, slots: make(chan struct{}, 8)}
}

func (t *transport) RoundTrip(req *http.Request) (*http.Response, error) {
	if req.URL.Hostname() != "local-tailscaled.sock" || !strings.HasPrefix(req.URL.Path, "/localapi/v0/") {
		return nil, errors.New("embedded transport only accepts LocalAPI requests")
	}
	if req.URL.Path == "/localapi/v0/watch-ipn-bus" {
		return t.watch(req)
	}
	select {
	case t.slots <- struct{}{}:
	case <-req.Context().Done():
		return nil, req.Context().Err()
	}
	type result struct {
		response *http.Response
		err      error
	}
	results := make(chan result, 1)
	go func() {
		defer func() { <-t.slots }()
		timeout := 45 * time.Second
		if deadline, ok := req.Context().Deadline(); ok {
			timeout = time.Until(deadline)
		}
		if timeout <= 0 {
			results <- result{err: context.DeadlineExceeded}
			return
		}
		var input libtailscale.InputStream
		if req.Body != nil {
			input = &inputStream{body: req.Body}
		}
		response, err := t.app.CallLocalAPI(int(timeout.Milliseconds()), req.Method, req.URL.RequestURI(), input)
		if err != nil {
			results <- result{err: err}
			return
		}
		body, ok := response.(interface{ Body() net.Conn })
		if !ok {
			results <- result{err: errors.New("embedded LocalAPI body unavailable")}
			return
		}
		stream := body.Body()
		stop := context.AfterFunc(req.Context(), func() { stream.Close() })
		wrapped := &responseBody{ReadCloser: stream, stop: stop}
		res := result{response: &http.Response{StatusCode: response.StatusCode(), Header: make(http.Header), Body: wrapped, Request: req}}
		results <- res
	}()
	select {
	case res := <-results:
		return res.response, res.err
	case <-req.Context().Done():
		// Close a response that races with cancellation; startup work is bounded
		// by the slot limit even if upstream's readiness wait never returns.
		go func() {
			res := <-results
			if res.response != nil {
				res.response.Body.Close()
			}
		}()
		return nil, req.Context().Err()
	}
}

type inputStream struct{ body io.ReadCloser }

func (s *inputStream) Read() ([]byte, error) {
	b := make([]byte, 32*1024)
	n, err := s.body.Read(b)
	return b[:n], err
}
func (s *inputStream) Close() error { return s.body.Close() }

type responseBody struct {
	io.ReadCloser
	stop func() bool
}

func (b *responseBody) Close() error { b.stop(); return b.ReadCloser.Close() }

type notificationStream struct {
	ctx    context.Context
	writer *io.PipeWriter
}

func (n *notificationStream) OnNotify(data []byte) error {
	select {
	case <-n.ctx.Done():
		return n.ctx.Err()
	default:
	}
	_, err := n.writer.Write(append(data, '\n'))
	return err
}

func (t *transport) watch(req *http.Request) (*http.Response, error) {
	mask, err := strconv.Atoi(req.URL.Query().Get("mask"))
	if err != nil {
		return nil, err
	}
	// Match LocalAPI's synchronous validation so the adapter can fall back
	// before receiving a seemingly successful stream with an error notification.
	if err := ipn.ValidateNotifyWatchOpt(ipn.NotifyWatchOpt(mask)); err != nil {
		return nil, err
	}
	ctx, cancel := context.WithCancel(req.Context())
	reader, writer := io.Pipe()
	var once sync.Once
	closeStream := func() { once.Do(func() { cancel(); reader.Close(); writer.Close() }) }
	// Upstream's explicit watcher survives request completion, unlike
	// CallLocalAPI (which cancels its HTTP context before returning).
	go func() {
		manager := t.app.WatchNotifications(mask, &notificationStream{ctx: ctx, writer: writer})
		<-ctx.Done()
		manager.Stop()
		closeStream()
	}()
	stop := context.AfterFunc(ctx, closeStream)
	return &http.Response{StatusCode: 200, Header: make(http.Header), Body: &watchBody{Reader: reader, close: func() { stop(); closeStream() }}, Request: req}, nil
}

type watchBody struct {
	io.Reader
	close func()
}

func (b *watchBody) Close() error { b.close(); return nil }
