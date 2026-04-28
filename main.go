package main

import (
	"log"
	"net/http"

	"github.com/EOEboh/hookdrop/internal/handler"
	"github.com/EOEboh/hookdrop/internal/middleware"
	"github.com/EOEboh/hookdrop/internal/session"
	"github.com/EOEboh/hookdrop/internal/sse"
	"github.com/EOEboh/hookdrop/internal/store"
)

func main() {
	st, err := store.New("hookdrop.db")
	if err != nil {
		log.Fatalf("failed to init store: %v", err)
	}

	broadcaster := sse.NewBroadcaster()

	mgr := session.NewManager(st)
	mgr.StartCleanup()

	mux := http.NewServeMux()
	mux.Handle("/sessions", &handler.SessionHandler{Manager: mgr})
	mux.Handle("/i/", &handler.InboxHandler{Store: st, Broadcast: broadcaster.Broadcast})
	mux.Handle("/events/", &handler.SSEHandler{Broadcaster: broadcaster, Store: st})

	// Wrap the whole mux with CORS
	handler := middleware.CORS(mux)

	log.Println("hookdrop listening on :8080")
	if err := http.ListenAndServe(":8080", handler); err != nil {
		log.Fatal(err)
	}
}
