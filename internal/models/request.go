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
