package handler

import (
	"encoding/json"
	"net/http"
	"strings"
	"time"

	"github.com/EOEboh/hookdrop/internal/endpoint"
	"github.com/EOEboh/hookdrop/internal/middleware"
	"github.com/EOEboh/hookdrop/internal/models"
	"github.com/EOEboh/hookdrop/internal/store"
	"github.com/google/uuid"
)

type EndpointsHandler struct {
	Store *store.Store
}

func (h *EndpointsHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		h.list(w, r)
	case http.MethodPost:
		h.create(w, r)
	case http.MethodDelete:
		h.delete(w, r)
	default:
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
	}
}

func (h *EndpointsHandler) list(w http.ResponseWriter, r *http.Request) {
	user := middleware.GetUser(r)

	endpoints, err := h.Store.GetEndpointsByUser(user.ID)
	if err != nil {
		http.Error(w, "failed to fetch endpoints", http.StatusInternalServerError)
		return
	}
	if endpoints == nil {
		endpoints = []*models.Endpoint{}
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(endpoints)
}

func (h *EndpointsHandler) create(w http.ResponseWriter, r *http.Request) {
	user := middleware.GetUser(r)

	var body struct {
		Slug        string `json:"slug"`
		Name        string `json:"name"`
		Description string `json:"description"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		http.Error(w, "invalid request body", http.StatusBadRequest)
		return
	}

	// Validate slug
	if err := endpoint.ValidateSlug(body.Slug); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	if body.Name == "" {
		http.Error(w, "name is required", http.StatusBadRequest)
		return
	}

	// Check slug availability
	exists, err := h.Store.SlugExists(body.Slug)
	if err != nil {
		http.Error(w, "store error", http.StatusInternalServerError)
		return
	}
	if exists {
		http.Error(w, "slug already taken", http.StatusConflict)
		return
	}

	ep := &models.Endpoint{
		ID:          uuid.NewString(),
		UserID:      user.ID,
		Slug:        body.Slug,
		Name:        body.Name,
		Description: body.Description,
		CreatedAt:   time.Now().UTC(),
	}

	if err := h.Store.CreateEndpoint(ep); err != nil {
		http.Error(w, "failed to create endpoint", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(ep)
}

func (h *EndpointsHandler) delete(w http.ResponseWriter, r *http.Request) {
	user := middleware.GetUser(r)

	// DELETE /endpoints/{id}
	id := strings.TrimPrefix(r.URL.Path, "/endpoints/")
	id = strings.Trim(id, "/")
	if id == "" {
		http.Error(w, "missing endpoint id", http.StatusBadRequest)
		return
	}

	if err := h.Store.DeleteEndpoint(id, user.ID); err != nil {
		http.Error(w, "failed to delete endpoint", http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}
