// Package sseclient is a minimal Server-Sent Events client for the hookdrop
// /events stream. The format is three line types: "event:", "data:", and
// ":" comments (keepalives) — no library needed.
package sseclient

import (
	"bufio"
	"bytes"
	"context"
	"fmt"
	"net"
	"net/http"
	"strings"
	"sync/atomic"
	"time"

	"github.com/EOEboh/hookdrop/cli/internal/api"
)

// watchdogTimeout kills connections that go silent. The server sends a
// keepalive comment every 20s, so 60s = three missed beats. This is what
// detects laptop-sleep and NAT-drop zombies that never error.
const watchdogTimeout = 60 * time.Second

// maxLineSize caps a single SSE line so a pathological payload errors the
// stream (and triggers a reconnect) instead of exhausting memory.
const maxLineSize = 16 << 20 // 16 MB

type Event struct {
	Name string
	Data []byte
}

type Client struct {
	BaseURL string
	Token   string
	http    *http.Client
}

func New(baseURL, token string) *Client {
	return &Client{
		BaseURL: strings.TrimRight(baseURL, "/"),
		Token:   token,
		// No overall client timeout — the stream is long-lived. Individual
		// phases are bounded instead; mid-stream silence is the watchdog's job.
		http: &http.Client{
			Transport: &http.Transport{
				DialContext:           (&net.Dialer{Timeout: 10 * time.Second}).DialContext,
				TLSHandshakeTimeout:   10 * time.Second,
				ResponseHeaderTimeout: 15 * time.Second,
			},
		},
	}
}

// Stream connects to /events/{identifier} and delivers events until the
// connection ends. It always returns a non-nil error describing why the
// stream stopped (including ctx cancellation); auth/ownership problems
// surface as api.ErrUnauthorized / api.ErrNotFound so callers can stop
// retrying.
func (c *Client) Stream(ctx context.Context, identifier string, events chan<- Event) error {
	ctx, cancel := context.WithCancel(ctx)
	defer cancel()

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, c.BaseURL+"/events/"+identifier, nil)
	if err != nil {
		return err
	}
	req.Header.Set("Authorization", "Bearer "+c.Token)
	req.Header.Set("Accept", "text/event-stream")
	req.Header.Set("Cache-Control", "no-cache")

	resp, err := c.http.Do(req)
	if err != nil {
		return fmt.Errorf("connect: %w", err)
	}
	defer resp.Body.Close()

	if err := api.CheckStatus(resp); err != nil {
		return err
	}

	var watchdogFired atomic.Bool
	watchdog := time.AfterFunc(watchdogTimeout, func() {
		watchdogFired.Store(true)
		cancel()
	})
	defer watchdog.Stop()

	scanner := bufio.NewScanner(resp.Body)
	scanner.Buffer(make([]byte, 64<<10), maxLineSize)

	var eventName string
	var data bytes.Buffer

	for scanner.Scan() {
		watchdog.Reset(watchdogTimeout)
		line := scanner.Bytes()

		switch {
		case len(line) == 0:
			// blank line = dispatch
			if eventName != "" || data.Len() > 0 {
				ev := Event{Name: eventName, Data: append([]byte(nil), data.Bytes()...)}
				eventName = ""
				data.Reset()
				select {
				case events <- ev:
				case <-ctx.Done():
					return ctx.Err()
				}
			}
		case line[0] == ':':
			// comment/keepalive — watchdog already reset
		case bytes.HasPrefix(line, []byte("event:")):
			eventName = string(bytes.TrimSpace(line[len("event:"):]))
		case bytes.HasPrefix(line, []byte("data:")):
			if data.Len() > 0 {
				data.WriteByte('\n')
			}
			data.Write(bytes.TrimSpace(line[len("data:"):]))
		}
	}

	if watchdogFired.Load() {
		return fmt.Errorf("no data for %s — connection stale", watchdogTimeout)
	}
	if ctx.Err() != nil {
		return ctx.Err()
	}
	if err := scanner.Err(); err != nil {
		return fmt.Errorf("stream read: %w", err)
	}
	return fmt.Errorf("server closed the stream")
}
