package sse

import (
	"encoding/json"
	"log"
	"sync"

	"github.com/EOEboh/hookdrop/internal/models"
)

// Client represents one connected browser tab
type Client struct {
	SessionID string
	Send      chan []byte
}

type Broadcaster struct {
	mu      sync.RWMutex
	clients map[string]map[*Client]struct{} // sessionID → set of clients
}

func NewBroadcaster() *Broadcaster {
	return &Broadcaster{
		clients: make(map[string]map[*Client]struct{}),
	}
}

// Register adds a new SSE client for a given session
func (b *Broadcaster) Register(client *Client) {
	b.mu.Lock()
	defer b.mu.Unlock()

	if b.clients[client.SessionID] == nil {
		b.clients[client.SessionID] = make(map[*Client]struct{})
	}
	b.clients[client.SessionID][client] = struct{}{}
	log.Printf("SSE client registered for session %s (total: %d)", client.SessionID, len(b.clients[client.SessionID]))
}

// Deregister removes a client when the browser disconnects
func (b *Broadcaster) Deregister(client *Client) {
	b.mu.Lock()
	defer b.mu.Unlock()

	delete(b.clients[client.SessionID], client)
	if len(b.clients[client.SessionID]) == 0 {
		delete(b.clients, client.SessionID)
	}
	log.Printf("SSE client deregistered for session %s", client.SessionID)
}

// Broadcast sends a captured request to all clients watching that session
func (b *Broadcaster) Broadcast(sessionID string, req *models.CapturedRequest) {
	b.mu.RLock()
	defer b.mu.RUnlock()

	clients, ok := b.clients[sessionID]
	if !ok || len(clients) == 0 {
		return // no one is watching — that's fine
	}

	payload, err := json.Marshal(req)
	if err != nil {
		log.Printf("SSE marshal error: %v", err)
		return
	}

	for client := range clients {
		select {
		case client.Send <- payload:
			// delivered
		default:
			// client's channel is full — it's too slow, skip it
			// it will catch up via the REST history endpoint
			log.Printf("SSE client too slow, dropping event for session %s", sessionID)
		}
	}
}
