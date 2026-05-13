package handler

import (
	"fmt"
	"log"
	"net/http"
	"strings"
	"time"

	"github.com/EOEboh/hookdrop/internal/sse"
	"github.com/EOEboh/hookdrop/internal/store"
)

type SSEHandler struct {
	Broadcaster *sse.Broadcaster
	Store       *store.Store
}

func (h *SSEHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {

	sessionID := strings.TrimPrefix(r.URL.Path, "/events/")
	sessionID = strings.Trim(sessionID, "/")

	if sessionID == "" || !h.Store.IdentifierExists(sessionID) {
		http.Error(w, "session not found", http.StatusNotFound)
		return
	}

	// 2. SSE requires these exact headers
	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")
	w.Header().Set("X-Accel-Buffering", "no") // tells nginx: don't buffer this

	// 3. Flush support — needed to push chunks immediately
	flusher, ok := w.(http.Flusher)
	if !ok {
		http.Error(w, "streaming not supported", http.StatusInternalServerError)
		return
	}

	// 4. Register this browser tab as a client
	client := &sse.Client{
		SessionID: sessionID,
		Send:      make(chan []byte, 32), // buffered — absorbs short bursts
	}
	h.Broadcaster.Register(client)
	defer h.Broadcaster.Deregister(client)

	// 5. Send an initial "connected" ping so the browser knows it's live
	fmt.Fprintf(w, "event: connected\ndata: {\"session_id\":\"%s\"}\n\n", sessionID)
	flusher.Flush()

	// 6. Keep-alive ticker — browsers disconnect if nothing arrives for ~30s
	ticker := time.NewTicker(20 * time.Second)
	defer ticker.Stop()

	// 7. Main loop — wait for events or disconnection
	for {
		select {
		case payload := <-client.Send:
			// Write SSE event format: event name + data + blank line
			fmt.Fprintf(w, "event: request\ndata: %s\n\n", payload)
			flusher.Flush()

		case <-ticker.C:
			// SSE comment — keeps the connection alive, browsers ignore it
			fmt.Fprintf(w, ": keepalive\n\n")
			flusher.Flush()

		case <-r.Context().Done():
			// Browser tab closed or navigated away
			log.Printf("SSE client disconnected from session %s", sessionID)
			return
		}
	}
}
