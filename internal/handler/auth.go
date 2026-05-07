package handler

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"strings"

	"github.com/EOEboh/hookdrop/internal/auth"
	"github.com/EOEboh/hookdrop/internal/email"
	"github.com/EOEboh/hookdrop/internal/store"
)

type AuthHandler struct {
	Store     *store.Store
	Emailer   *email.Sender
	JWTSecret string
	AppURL    string // e.g. https://hookdrop.app
}

// POST /auth/request — send a magic link
func (h *AuthHandler) RequestLink(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var body struct {
		Email string `json:"email"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil || body.Email == "" {
		http.Error(w, "valid email required", http.StatusBadRequest)
		return
	}

	body.Email = strings.ToLower(strings.TrimSpace(body.Email))

	// Get or create the user
	user, err := h.Store.GetOrCreateUser(body.Email)
	if err != nil {
		log.Printf("auth: get/create user error: %v", err)
		http.Error(w, "server error", http.StatusInternalServerError)
		return
	}

	// Generate the magic link
	link, err := h.Store.CreateMagicLink(user.ID)
	if err != nil {
		log.Printf("auth: create magic link error: %v", err)
		http.Error(w, "server error", http.StatusInternalServerError)
		return
	}

	magicURL := fmt.Sprintf("%s/auth/verify?token=%s", h.AppURL, link.Token)

	// Send the email
	if err := h.Emailer.SendMagicLink(body.Email, magicURL); err != nil {
		log.Printf("auth: email send error: %v", err)
		http.Error(w, "failed to send email", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{
		"message": "Magic link sent — check your email",
	})
}

// GET /auth/verify?token=xxx — verify token, return JWT
func (h *AuthHandler) VerifyLink(w http.ResponseWriter, r *http.Request) {
	token := r.URL.Query().Get("token")
	if token == "" {
		http.Error(w, "missing token", http.StatusBadRequest)
		return
	}

	user, err := h.Store.ConsumeMagicLink(token)
	if err != nil {
		http.Error(w, err.Error(), http.StatusUnauthorized)
		return
	}
	if user == nil {
		http.Error(w, "invalid token", http.StatusUnauthorized)
		return
	}

	// Issue a JWT
	jwt, err := auth.GenerateToken(user.ID, user.Email, h.JWTSecret)
	if err != nil {
		http.Error(w, "server error", http.StatusInternalServerError)
		return
	}

	// Redirect to the frontend with the token in the URL fragment
	// Fragment (#) is never sent to servers — safer than query params
	redirectURL := fmt.Sprintf("%s/auth/callback#token=%s", h.AppURL, jwt)
	http.Redirect(w, r, redirectURL, http.StatusFound)
}
