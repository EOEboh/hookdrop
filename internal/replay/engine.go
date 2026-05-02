package replay

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/EOEboh/hookdrop/internal/models"
)

type Engine struct {
	client *http.Client
}

func NewEngine() *Engine {
	return &Engine{
		client: &http.Client{
			Timeout: 30 * time.Second,
			// Don't follow redirects — return them as-is so the
			// developer sees exactly what their server responded
			CheckRedirect: func(req *http.Request, via []*http.Request) error {
				return http.ErrUseLastResponse
			},
		},
	}
}

func (e *Engine) Replay(
	ctx context.Context,
	original *models.CapturedRequest,
	replayReq *models.ReplayRequest,
) (*models.ReplayResponse, error) {

	// 1. Determine body — use override if provided, else original
	bodyBytes := original.Body
	if replayReq.Body != "" {
		bodyBytes = []byte(replayReq.Body)
	}

	// 2. Build the outbound request
	outbound, err := http.NewRequestWithContext(
		ctx,
		original.Method,
		replayReq.TargetURL,
		bytes.NewReader(bodyBytes),
	)
	if err != nil {
		return nil, fmt.Errorf("build request: %w", err)
	}

	// 3. Copy original headers first
	for key, val := range original.Headers {
		// Skip headers that should not be forwarded
		if shouldSkipHeader(key) {
			continue
		}
		outbound.Header.Set(key, val)
	}

	// 4. Apply header overrides from the replay request
	for key, val := range replayReq.Headers {
		outbound.Header.Set(key, val)
	}

	// 5. Mark the request as a replay so your dev server can identify it
	outbound.Header.Set("X-Hookdrop-Replay", "true")
	outbound.Header.Set("X-Hookdrop-Original-Id", original.ID)

	// 6. Fire and measure latency
	start := time.Now()
	resp, err := e.client.Do(outbound)
	latency := time.Since(start).Milliseconds()

	if err != nil {
		return nil, fmt.Errorf("request failed: %w", err)
	}
	defer resp.Body.Close()

	// 7. Read response body — cap at 1MB
	respBody, err := io.ReadAll(io.LimitReader(resp.Body, 1<<20))
	if err != nil {
		return nil, fmt.Errorf("read response: %w", err)
	}

	// 8. Flatten response headers
	respHeaders := make(map[string]string)
	for key, values := range resp.Header {
		respHeaders[http.CanonicalHeaderKey(key)] = strings.Join(values, ", ")
	}

	return &models.ReplayResponse{
		Status:    resp.StatusCode,
		Headers:   respHeaders,
		Body:      string(respBody),
		LatencyMs: latency,
	}, nil
}

// shouldSkipHeader filters headers that break or are meaningless when forwarded
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
