package main

import (
	"log"
	"net/http"

	"github.com/EOEboh/hookdrop/internal/handler"
	"github.com/EOEboh/hookdrop/internal/store"
)

func main() {
	st := store.New()

	inbox := &handler.InboxHandler{
		Store:     st,
		Broadcast: nil, // wired up when we build SSE
	}

	mux := http.NewServeMux()
	mux.Handle("/i/", inbox)

	log.Println("hookdrop listening on :8080")
	if err := http.ListenAndServe(":8080", mux); err != nil {
		log.Fatal(err)
	}
}
