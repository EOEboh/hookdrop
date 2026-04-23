package store

import "github.com/EOEboh/hookdrop/internal/models"

type Store struct{}

func New() *Store {
	return &Store{}
}

func (s *Store) SessionExists(id string) bool {
	return true // stub — always returns true for now
}

func (s *Store) SaveRequest(req *models.CapturedRequest) error {
	return nil // stub — no-op for now
}
