package session

import (
	"log"
	"time"

	"github.com/EOEboh/hookdrop/internal/models"
	"github.com/EOEboh/hookdrop/internal/store"
	"github.com/google/uuid"
)

const DefaultTTL = 24 * time.Hour

type Manager struct {
	Store *store.Store
}

func NewManager(s *store.Store) *Manager {
	return &Manager{Store: s}
}

func (m *Manager) CreateSession() (*models.Session, error) {
	session := &models.Session{
		ID:        uuid.NewString()[:8], // short ID — easier to read in URLs
		CreatedAt: time.Now().UTC(),
		ExpiresAt: time.Now().UTC().Add(DefaultTTL),
	}
	if err := m.Store.CreateSession(session); err != nil {
		return nil, err
	}
	return session, nil
}

// StartCleanup runs a background goroutine that purges expired sessions
func (m *Manager) StartCleanup() {
	go func() {
		ticker := time.NewTicker(1 * time.Hour)
		defer ticker.Stop()
		for range ticker.C {
			if err := m.Store.DeleteExpiredSessions(); err != nil {
				log.Printf("session cleanup error: %v", err)
			} else {
				log.Println("expired sessions cleaned up")
			}
		}
	}()
}
