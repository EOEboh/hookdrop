package main

import (
	"bufio"
	"log"
	"net/http"
	"os"
	"strings"

	"github.com/EOEboh/hookdrop/internal/email"
	"github.com/EOEboh/hookdrop/internal/handler"
	"github.com/EOEboh/hookdrop/internal/middleware"
	"github.com/EOEboh/hookdrop/internal/replay"
	"github.com/EOEboh/hookdrop/internal/session"
	"github.com/EOEboh/hookdrop/internal/sse"
	"github.com/EOEboh/hookdrop/internal/store"
)

func loadEnvFile(path string) {
	f, err := os.Open(path)
	if err != nil {
		return // file doesn't exist: fine in production
	}
	defer f.Close()

	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		parts := strings.SplitN(line, "=", 2)
		if len(parts) != 2 {
			continue
		}
		key := strings.TrimSpace(parts[0])
		val := strings.TrimSpace(parts[1])
		// Only set if not already set: real env vars take priority
		if os.Getenv(key) == "" {
			os.Setenv(key, val)
		}
	}
}

func main() {
	loadEnvFile(".env.local")

	// Config from environment
	dbPath := getEnv("DB_PATH", "hookdrop.db")
	allowedOrigin := getEnv("ALLOWED_ORIGIN", "http://localhost:5173")
	port := getEnv("PORT", "8080")
	jwtSecret := getEnv("JWT_SECRET", "change-me-in-production")
	resendKey := getEnv("RESEND_API_KEY", "")
	fromAddr := getEnv("EMAIL_FROM", "hookdrop <noreply@hookdrop.app>")
	apiURL := getEnv("API_URL", "http://localhost:8080")
	frontendURL := getEnv("FRONTEND_URL", "http://localhost:5173")

	st, err := store.New(dbPath)
	if err != nil {
		log.Fatalf("failed to init store: %v", err)
	}

	broadcaster := sse.NewBroadcaster()
	replayEngine := replay.NewEngine()
	emailer := email.NewSender(resendKey, fromAddr)

	mgr := session.NewManager(st)
	mgr.StartCleanup()

	authHandler := &handler.AuthHandler{
		Store:       st,
		Emailer:     emailer,
		JWTSecret:   jwtSecret,
		APIURL:      apiURL,
		FrontendURL: frontendURL,
	}

	// Auth middleware: wraps protected routes
	requireAuth := middleware.Auth(jwtSecret)

	mux := http.NewServeMux()

	// Public: no auth needed
	mux.HandleFunc("/auth/request", authHandler.RequestLink)
	mux.HandleFunc("/auth/verify", authHandler.VerifyLink)

	// Protected: require valid JWT
	mux.Handle("/sessions", requireAuth(&handler.SessionHandler{Manager: mgr}))
	mux.Handle("/requests/", requireAuth(&handler.RequestsHandler{Store: st}))
	mux.Handle("/replay", requireAuth(&handler.ReplayHandler{Store: st, Engine: replayEngine}))
	mux.Handle("/events/", requireAuth(&handler.SSEHandler{Broadcaster: broadcaster, Store: st}))
	mux.Handle("/endpoints", requireAuth(&handler.EndpointsHandler{Store: st}))
	mux.Handle("/endpoints/", requireAuth(&handler.EndpointsHandler{Store: st}))

	// Inbox is intentionally public: webhook senders don't authenticate
	mux.Handle("/i/", &handler.InboxHandler{Store: st, Broadcast: broadcaster.Broadcast})

	log.Printf("hookdrop listening on :%s", port)
	if err := http.ListenAndServe(":"+port, middleware.CORS(mux, allowedOrigin)); err != nil {
		log.Fatal(err)
	}
}

func getEnv(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}
