package main

import (
	"bufio"
	"log"
	"net/http"
	"os"
	"strconv"
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

	dbPath := getEnv("DB_PATH", "hookdrop.db")
	allowedOrigin := getEnv("ALLOWED_ORIGIN", "http://localhost:5173")
	port := getEnv("PORT", "8080")
	jwtSecret := getEnv("JWT_SECRET", "change-me-in-production")
	resendKey := getEnv("RESEND_API_KEY", "")
	fromAddr := getEnv("EMAIL_FROM", "hookdrop <noreply@hookdrop.app>")
	apiURL := getEnv("API_URL", "http://localhost:8080")
	frontendURL := getEnv("FRONTEND_URL", "http://localhost:5173")

	// Defaults: 30 token capacity, 5 refills/sec per IP
	rlCapacity := getEnvFloat("RATE_LIMIT_CAPACITY", 30)
	rlRefill := getEnvFloat("RATE_LIMIT_REFILL", 5)

	st, err := store.New(dbPath)
	if err != nil {
		log.Fatalf("failed to init store: %v", err)
	}

	broadcaster := sse.NewBroadcaster()
	replayEngine := replay.NewEngine()
	emailer := email.NewSender(resendKey, fromAddr)
	rateLimiter := middleware.NewRateLimiter(rlCapacity, rlRefill)

	mgr := session.NewManager(st)
	mgr.StartCleanup()

	authHandler := &handler.AuthHandler{
		Store:       st,
		Emailer:     emailer,
		JWTSecret:   jwtSecret,
		APIURL:      apiURL,
		FrontendURL: frontendURL,
	}

	requireAuth := middleware.Auth(jwtSecret)
	inboxLimiter := middleware.InboxRateLimit(rateLimiter)

	mux := http.NewServeMux()

	// Public auth routes
	mux.HandleFunc("/auth/request", authHandler.RequestLink)
	mux.HandleFunc("/auth/verify", authHandler.VerifyLink)

	// Protected routes
	mux.Handle("/sessions", requireAuth(&handler.SessionHandler{Manager: mgr}))
	mux.Handle("/requests/", requireAuth(&handler.RequestsHandler{Store: st}))
	mux.Handle("/replay", requireAuth(&handler.ReplayHandler{Store: st, Engine: replayEngine}))
	mux.Handle("/events/", requireAuth(&handler.SSEHandler{Broadcaster: broadcaster, Store: st}))
	mux.Handle("/endpoints", requireAuth(&handler.EndpointsHandler{Store: st}))
	mux.Handle("/endpoints/", requireAuth(&handler.EndpointsHandler{Store: st}))

	// Inbox: public but rate limited
	inboxHandler := &handler.InboxHandler{
		Store:     st,
		Broadcast: broadcaster.Broadcast,
	}
	mux.Handle("/i/", inboxLimiter(inboxHandler))

	log.Printf("hookdrop listening on :%s", port)
	if err := http.ListenAndServe(":"+port, middleware.CORS(mux, allowedOrigin)); err != nil {
		log.Fatal(err)
	}
}

func getEnvFloat(key string, fallback float64) float64 {
	v := os.Getenv(key)
	if v == "" {
		return fallback
	}
	f, err := strconv.ParseFloat(v, 64)
	if err != nil {
		return fallback
	}
	return f
}

func getEnv(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}
