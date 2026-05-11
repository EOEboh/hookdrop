package models

import "time"

type CapturedRequest struct {
	ID         string            `json:"id"`
	SessionID  string            `json:"session_id"`
	Method     string            `json:"method"`
	Headers    map[string]string `json:"headers"`
	Body       []byte            `json:"body"`
	BodySize   int               `json:"body_size"`
	RemoteIP   string            `json:"remote_ip"`
	ReceivedAt time.Time         `json:"received_at"`
}

type Endpoint struct {
	ID          string    `json:"id"`
	UserID      string    `json:"user_id"`
	Slug        string    `json:"slug"`
	Name        string    `json:"name"`
	Description string    `json:"description,omitempty"`
	CreatedAt   time.Time `json:"created_at"`
}

type Session struct {
	ID        string    `json:"id"`
	CreatedAt time.Time `json:"created_at"`
	ExpiresAt time.Time `json:"expires_at"`
}

type ReplayRequest struct {
	RequestID string            `json:"request_id"`
	TargetURL string            `json:"target_url"`
	Headers   map[string]string `json:"headers,omitempty"` // optional overrides
	Body      string            `json:"body,omitempty"`    // optional override
}

type ReplayResponse struct {
	Status    int               `json:"status"`
	Headers   map[string]string `json:"headers"`
	Body      string            `json:"body"`
	LatencyMs int64             `json:"latency_ms"`
}

type User struct {
	ID        string    `json:"id"`
	Email     string    `json:"email"`
	CreatedAt time.Time `json:"created_at"`
}

type MagicLink struct {
	ID        string    `json:"id"`
	UserID    string    `json:"user_id"`
	Token     string    `json:"token"`
	ExpiresAt time.Time `json:"expires_at"`
	Used      bool      `json:"used"`
}
