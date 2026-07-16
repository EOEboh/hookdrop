// Package api is the HTTP client for the hookdrop backend.
package api

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"
)

// Sentinel errors commands translate into friendly messages.
var (
	ErrUnauthorized    = errors.New("unauthorized")
	ErrNotFound        = errors.New("not found")
	ErrPaymentRequired = errors.New("plan upgrade required")
)

type Client struct {
	BaseURL string
	Token   string
	HTTP    *http.Client
}

func New(baseURL, token string) *Client {
	return &Client{
		BaseURL: strings.TrimRight(baseURL, "/"),
		Token:   token,
		HTTP:    &http.Client{Timeout: 15 * time.Second},
	}
}

func (c *Client) get(ctx context.Context, path string, out interface{}) error {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, c.BaseURL+path, nil)
	if err != nil {
		return err
	}
	req.Header.Set("Authorization", "Bearer "+c.Token)

	resp, err := c.HTTP.Do(req)
	if err != nil {
		return fmt.Errorf("request %s: %w", path, err)
	}
	defer resp.Body.Close()

	if err := CheckStatus(resp); err != nil {
		return err
	}
	return json.NewDecoder(resp.Body).Decode(out)
}

// CheckStatus maps error statuses to sentinel errors. Shared with the SSE
// client so 401/404/402 behave identically everywhere.
func CheckStatus(resp *http.Response) error {
	switch {
	case resp.StatusCode < 400:
		return nil
	case resp.StatusCode == http.StatusUnauthorized:
		return ErrUnauthorized
	case resp.StatusCode == http.StatusNotFound:
		return ErrNotFound
	case resp.StatusCode == http.StatusPaymentRequired:
		return ErrPaymentRequired
	default:
		body, _ := io.ReadAll(io.LimitReader(resp.Body, 4096))
		return fmt.Errorf("server returned %d: %s", resp.StatusCode, strings.TrimSpace(string(body)))
	}
}

// Me validates the token and returns the account's identity, plan, and limits.
func (c *Client) Me(ctx context.Context) (*Me, error) {
	var me Me
	if err := c.get(ctx, "/me", &me); err != nil {
		return nil, err
	}
	return &me, nil
}

// Endpoints lists the user's named endpoints.
func (c *Client) Endpoints(ctx context.Context) ([]Endpoint, error) {
	var eps []Endpoint
	if err := c.get(ctx, "/endpoints", &eps); err != nil {
		return nil, err
	}
	return eps, nil
}
