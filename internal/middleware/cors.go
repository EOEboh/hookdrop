package middleware

import (
	"log"
	"net/http"
	"strings"
)

func CORS(next http.Handler, allowedOrigins string) http.Handler {
	if allowedOrigins == "" {
		log.Fatal("CORS: ALLOWED_ORIGIN is not set — refusing to start with open CORS")
	}

	// Build a lookup map — supports comma-separated list of origins
	// e.g. "https://hookdrop.app,https://www.hookdrop.app"
	allowed := make(map[string]bool)
	for _, origin := range strings.Split(allowedOrigins, ",") {
		cleaned := strings.TrimSpace(strings.TrimRight(strings.TrimSpace(origin), "/"))
		if cleaned != "" {
			allowed[cleaned] = true
			log.Printf("CORS: allowing origin %q", cleaned)
		}
	}

	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		origin := r.Header.Get("Origin")

		if origin != "" && allowed[origin] {
			w.Header().Set("Access-Control-Allow-Origin", origin)
			w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
			w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization")
			w.Header().Set("Access-Control-Allow-Credentials", "true")
			w.Header().Set("Vary", "Origin")
		} else if origin != "" {

			log.Printf("CORS: rejected origin %q (allowed: %v)", origin, allowedOrigins)
		}

		// Always handle preflight — even for rejected origins, return 204

		if r.Method == http.MethodOptions {
			w.WriteHeader(http.StatusNoContent)
			return
		}

		next.ServeHTTP(w, r)
	})
}
