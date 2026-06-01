package main

import (
	"bufio"
	"log"
	"net/http"
	"os"
	"strconv"
	"strings"

	"github.com/EOEboh/hookdrop/internal/billing"
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
		return
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
		if os.Getenv(key) == "" {
			os.Setenv(key, val)
		}
	}
}

func main() {
	loadEnvFile(".env.local")

	// ── Core config
	dbPath := getEnv("DB_PATH", "hookdrop.db")
	allowedOrigin := getEnv("ALLOWED_ORIGIN", "http://localhost:5173")
	port := getEnv("PORT", "8080")
	jwtSecret := getEnv("JWT_SECRET", "change-me-in-production")
	resendKey := getEnv("RESEND_API_KEY", "")
	fromAddr := getEnv("EMAIL_FROM", "hookdrop <noreply@hookdrop.app>")
	apiURL := getEnv("API_URL", "http://localhost:8080")
	frontendURL := getEnv("FRONTEND_URL", "http://localhost:5173")
	rlCapacity := getEnvFloat("RATE_LIMIT_CAPACITY", 30)
	rlRefill := getEnvFloat("RATE_LIMIT_REFILL", 5)

	// ── LemonSqueezy config (international users)
	lsAPIKey := getEnv("LEMONSQUEEZY_API_KEY", "")
	lsWebhookSecret := getEnv("LEMONSQUEEZY_WEBHOOK_SECRET", "")
	lsStoreID := getEnv("LEMONSQUEEZY_STORE_ID", "")
	lsVariantMonthly := getEnv("LEMONSQUEEZY_VARIANT_PRO_MONTHLY", "")
	lsVariantAnnual := getEnv("LEMONSQUEEZY_VARIANT_PRO_ANNUAL", "")

	// ── Paystack config (African users)
	paystackSecretKey := getEnv("PAYSTACK_SECRET_KEY", "")
	paystackWebhookSecret := getEnv("PAYSTACK_WEBHOOK_SECRET", "")
	paystackPlanMonthly := getEnv("PAYSTACK_PLAN_PRO_MONTHLY", "")
	paystackPlanAnnual := getEnv("PAYSTACK_PLAN_PRO_ANNUAL", "")

	// ── Infrastructure
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

	// ── Billing providers
	lemonSqueezyProvider := billing.NewLemonSqueezyProvider(
		lsAPIKey,
		lsWebhookSecret,
		lsStoreID,
		billing.LemonSqueezyVariants{
			ProMonthly: lsVariantMonthly,
			ProAnnual:  lsVariantAnnual,
		},
	)

	paystackProvider := billing.NewPaystackProvider(
		paystackSecretKey,
		paystackWebhookSecret,
		billing.PaystackPlans{
			ProMonthly: paystackPlanMonthly,
			ProAnnual:  paystackPlanAnnual,
		},
	)

	// ── Handlers
	authHandler := &handler.AuthHandler{
		Store:       st,
		Emailer:     emailer,
		JWTSecret:   jwtSecret,
		APIURL:      apiURL,
		FrontendURL: frontendURL,
	}

	billingHandler := &handler.BillingHandler{
		Store:        st,
		LemonSqueezy: lemonSqueezyProvider,
		Paystack:     paystackProvider,
		AppURL:       frontendURL,
	}

	secretsHandler := &handler.SecretsHandler{Store: st}
	requireAuth := middleware.Auth(jwtSecret)
	inboxLimiter := middleware.InboxRateLimit(rateLimiter)

	// ── Routes
	mux := http.NewServeMux()

	// Auth — public
	mux.HandleFunc("/auth/request", authHandler.RequestLink)
	mux.HandleFunc("/auth/verify", authHandler.VerifyLink)

	// Billing webhooks: public but signature-verified internally
	mux.HandleFunc("/billing/webhook/lemonsqueezy", billingHandler.LemonSqueezyWebhook)
	mux.HandleFunc("/billing/webhook/paystack", billingHandler.PaystackWebhook)

	mux.Handle("/billing/verify-paystack",
		requireAuth(http.HandlerFunc(billingHandler.VerifyPaystack)))

	// Billing — authenticated
	mux.Handle("/billing/subscription",
		requireAuth(http.HandlerFunc(billingHandler.GetSubscription)))
	mux.Handle("/billing/checkout",
		requireAuth(http.HandlerFunc(billingHandler.CreateCheckout)))
	mux.Handle("/billing/portal",
		requireAuth(http.HandlerFunc(billingHandler.GetPortal)))
	mux.Handle("/billing/cancel",
		requireAuth(http.HandlerFunc(billingHandler.CancelSubscription)))

	// Core — authenticated
	mux.Handle("/sessions", requireAuth(&handler.SessionHandler{Manager: mgr}))
	mux.Handle("/requests/", requireAuth(&handler.RequestsHandler{Store: st}))
	mux.Handle("/replay", requireAuth(&handler.ReplayHandler{Store: st, Engine: replayEngine}))
	mux.Handle("/events/", requireAuth(&handler.SSEHandler{Broadcaster: broadcaster, Store: st}))
	mux.Handle("/endpoints", requireAuth(&handler.EndpointsHandler{Store: st}))

	// Endpoints + secrets — authenticated, routed by path shape
	mux.Handle("/endpoints/", requireAuth(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if strings.Contains(r.URL.Path, "/secrets") {
			secretsHandler.ServeHTTP(w, r)
		} else {
			(&handler.EndpointsHandler{Store: st}).ServeHTTP(w, r)
		}
	})))

	// Inbox — public but rate limited
	mux.Handle("/i/", inboxLimiter(&handler.InboxHandler{
		Store:     st,
		Broadcast: broadcaster.Broadcast,
	}))

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
