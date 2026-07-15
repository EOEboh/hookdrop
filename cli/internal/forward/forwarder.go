// Package forward re-issues captured webhooks against a local target,
// mirroring the server-side replay engine's header semantics.
package forward

import (
	"bytes"
	"context"
	"io"
	"net/http"
	"time"

	"github.com/EOEboh/hookdrop/cli/internal/api"
)

// forwardTimeout is deliberately shorter than the server replay engine's
// 30s: a local dev server that takes longer is effectively down, and a
// short timeout keeps the queue draining during bursts.
const forwardTimeout = 10 * time.Second

// queueSize bounds in-flight webhooks. The worker is sequential, so this is
// also the burst ceiling before we start dropping (visibly) instead of
// hammering the local server or stalling the SSE reader.
const queueSize = 100

type Result struct {
	Request *api.CapturedRequest
	Status  int
	Latency time.Duration
	Err     error
}

type Forwarder struct {
	target   string
	client   *http.Client
	queue    chan *api.CapturedRequest
	onResult func(Result)
}

// New creates a forwarder targeting url. onResult is called from the worker
// goroutine for every attempted delivery.
func New(target string, onResult func(Result)) *Forwarder {
	return &Forwarder{
		target: target,
		client: &http.Client{
			Timeout: forwardTimeout,
			// Match the replay engine: hand redirects back as-is so the
			// developer sees exactly what their server responded.
			CheckRedirect: func(req *http.Request, via []*http.Request) error {
				return http.ErrUseLastResponse
			},
		},
		queue:    make(chan *api.CapturedRequest, queueSize),
		onResult: onResult,
	}
}

// Start runs the single sequential delivery worker until ctx is cancelled.
func (f *Forwarder) Start(ctx context.Context) {
	go func() {
		for {
			select {
			case req := <-f.queue:
				f.onResult(f.deliver(ctx, req))
			case <-ctx.Done():
				return
			}
		}
	}()
}

// Enqueue adds a webhook to the delivery queue. Returns false when the
// queue is full — callers should surface the drop, never block the SSE
// reader on a slow local server.
func (f *Forwarder) Enqueue(req *api.CapturedRequest) bool {
	select {
	case f.queue <- req:
		return true
	default:
		return false
	}
}

func (f *Forwarder) deliver(ctx context.Context, original *api.CapturedRequest) Result {
	outbound, err := http.NewRequestWithContext(ctx, original.Method, f.target, bytes.NewReader(original.Body))
	if err != nil {
		return Result{Request: original, Err: err}
	}

	for key, val := range original.Headers {
		if shouldSkipHeader(key) {
			continue
		}
		outbound.Header.Set(key, val)
	}

	// Distinct from the replay engine's X-Hookdrop-Replay so dev servers can
	// tell live CLI forwards from dashboard replays.
	outbound.Header.Set("X-Hookdrop-Forwarded", "true")
	outbound.Header.Set("X-Hookdrop-Original-Id", original.ID)

	start := time.Now()
	resp, err := f.client.Do(outbound)
	latency := time.Since(start)
	if err != nil {
		return Result{Request: original, Latency: latency, Err: err}
	}
	defer resp.Body.Close()

	// Drain (bounded to the same 1 MB cap the server replay engine uses)
	// so keep-alive connections can be reused
	io.Copy(io.Discard, io.LimitReader(resp.Body, 1<<20))

	return Result{Request: original, Status: resp.StatusCode, Latency: latency}
}

// shouldSkipHeader filters headers that break or are meaningless when
// forwarded. Parity port of shouldSkipHeader in the backend's
// internal/replay/engine.go — keep the two lists identical so behavior
// matches whether a request is replayed from the web UI or forwarded live.
func shouldSkipHeader(key string) bool {
	skip := map[string]bool{
		"Host":              true, // rewritten by Go's HTTP client automatically
		"Content-Length":    true, // recalculated by Go's HTTP client based on body size
		"Transfer-Encoding": true,
		"Connection":        true,
		"Te":                true,
		"Trailers":          true,
		"Upgrade":           true,
	}
	return skip[http.CanonicalHeaderKey(key)]
}
