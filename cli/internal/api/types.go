package api

import "time"

// Wire types mirroring the backend's JSON responses. Source of truth:
// internal/models/request.go in the backend module — keep field names and
// JSON tags in sync (internal/ packages can't be imported across modules).

// CapturedRequest is one webhook as delivered over SSE and /requests.
// Body arrives base64-encoded on the wire; encoding/json decodes it into
// raw bytes automatically.
type CapturedRequest struct {
	ID         string            `json:"id"`
	SessionID  string            `json:"session_id"`
	Method     string            `json:"method"`
	Headers    map[string]string `json:"headers"`
	Body       []byte            `json:"body"`
	BodySize   int               `json:"body_size"`
	RemoteIP   string            `json:"remote_ip"`
	ReceivedAt time.Time         `json:"received_at"`
	Verified   string            `json:"verified"` // "verified", "failed", "unverified"
	Provider   string            `json:"provider"`
}

type Endpoint struct {
	ID          string    `json:"id"`
	UserID      string    `json:"user_id"`
	Slug        string    `json:"slug"`
	Name        string    `json:"name"`
	Description string    `json:"description,omitempty"`
	CreatedAt   time.Time `json:"created_at"`
}

type User struct {
	ID        string    `json:"id"`
	Email     string    `json:"email"`
	CreatedAt time.Time `json:"created_at"`
}

// Limits uses the backend's default (un-tagged) Go field serialization.
type Limits struct {
	MaxNamedEndpoints   int  `json:"MaxNamedEndpoints"` // -1 = unlimited
	MaxRequestsPerMonth int  `json:"MaxRequestsPerMonth"`
	HistoryDays         int  `json:"HistoryDays"`
	MaxSecrets          int  `json:"MaxSecrets"`
	HasFiltering        bool `json:"HasFiltering"`
}

// Me is the GET /me response.
type Me struct {
	User       User   `json:"user"`
	Plan       string `json:"plan"`
	AuthMethod string `json:"auth_method"`
	Limits     Limits `json:"limits"`
}
