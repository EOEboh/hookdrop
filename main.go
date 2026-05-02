package main

import (
	"log"
	"net/http"
	"os"

	"github.com/EOEboh/hookdrop/internal/handler"
	"github.com/EOEboh/hookdrop/internal/middleware"
	"github.com/EOEboh/hookdrop/internal/replay"
	"github.com/EOEboh/hookdrop/internal/session"
	"github.com/EOEboh/hookdrop/internal/sse"
	"github.com/EOEboh/hookdrop/internal/store"
)

func main() {
	dbPath := os.Getenv("DB_PATH")
	if dbPath == "" {
		dbPath = "hookdrop.db"
	}

	allowedOrigin := os.Getenv("ALLOWED_ORIGIN")
	if allowedOrigin == "" {
		allowedOrigin = "http://localhost:5173"
	}

	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}

	st, err := store.New(dbPath)
	if err != nil {
		log.Fatalf("failed to init store: %v", err)
	}

	broadcaster := sse.NewBroadcaster()
	replayEngine := replay.NewEngine()

	mgr := session.NewManager(st)
	mgr.StartCleanup()

	mux := http.NewServeMux()
	mux.Handle("/sessions", &handler.SessionHandler{Manager: mgr})
	mux.Handle("/requests/", &handler.RequestsHandler{Store: st})
	mux.Handle("/replay", &handler.ReplayHandler{Store: st, Engine: replayEngine})
	mux.Handle("/i/", &handler.InboxHandler{Store: st, Broadcast: broadcaster.Broadcast})
	mux.Handle("/events/", &handler.SSEHandler{Broadcaster: broadcaster, Store: st})

	log.Printf("hookdrop listening on :%s", port)
	if err := http.ListenAndServe(":"+port, middleware.CORS(mux, allowedOrigin)); err != nil {
		log.Fatal(err)
	}
}
