package store

import (
	"database/sql"
	"time"

	"github.com/EOEboh/hookdrop/internal/models"
)

func (s *Store) CreateAPIToken(t *models.APIToken) error {
	_, err := s.db.Exec(
		`INSERT INTO api_tokens (id, user_id, name, token_hash, token_prefix, created_at, expires_at)
         VALUES (?, ?, ?, ?, ?, ?, ?)`,
		t.ID, t.UserID, t.Name, t.TokenHash, t.TokenPrefix, t.CreatedAt, t.ExpiresAt,
	)
	return err
}

// GetAPITokenByHash returns the token only if it is active — revoked or
// expired tokens are filtered out in SQL so callers can't misuse them.
func (s *Store) GetAPITokenByHash(hash string) (*models.APIToken, error) {
	t := &models.APIToken{}
	err := s.db.QueryRow(
		`SELECT id, user_id, name, token_hash, token_prefix, created_at, last_used_at, expires_at, revoked_at
         FROM api_tokens
         WHERE token_hash = ?
           AND revoked_at IS NULL
           AND (expires_at IS NULL OR expires_at > ?)`,
		hash, time.Now().UTC(),
	).Scan(
		&t.ID, &t.UserID, &t.Name, &t.TokenHash, &t.TokenPrefix,
		&t.CreatedAt, &t.LastUsedAt, &t.ExpiresAt, &t.RevokedAt,
	)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return t, nil
}

func (s *Store) ListAPITokens(userID string) ([]*models.APIToken, error) {
	rows, err := s.db.Query(
		`SELECT id, user_id, name, token_hash, token_prefix, created_at, last_used_at, expires_at, revoked_at
         FROM api_tokens WHERE user_id = ?
         ORDER BY created_at DESC`, userID,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var results []*models.APIToken
	for rows.Next() {
		t := &models.APIToken{}
		err := rows.Scan(
			&t.ID, &t.UserID, &t.Name, &t.TokenHash, &t.TokenPrefix,
			&t.CreatedAt, &t.LastUsedAt, &t.ExpiresAt, &t.RevokedAt,
		)
		if err != nil {
			return nil, err
		}
		results = append(results, t)
	}
	return results, rows.Err()
}

func (s *Store) RevokeAPIToken(id, userID string) error {
	_, err := s.db.Exec(
		`UPDATE api_tokens SET revoked_at = ?
         WHERE id = ? AND user_id = ? AND revoked_at IS NULL`,
		time.Now().UTC(), id, userID,
	)
	return err
}

func (s *Store) RevokeAllAPITokens(userID string) error {
	_, err := s.db.Exec(
		`UPDATE api_tokens SET revoked_at = ?
         WHERE user_id = ? AND revoked_at IS NULL`,
		time.Now().UTC(), userID,
	)
	return err
}

// TouchAPIToken records usage, throttled to one write per 5 minutes per
// token via the WHERE clause — no in-memory state needed.
func (s *Store) TouchAPIToken(id string, now time.Time) error {
	_, err := s.db.Exec(
		`UPDATE api_tokens SET last_used_at = ?
         WHERE id = ? AND (last_used_at IS NULL OR last_used_at < ?)`,
		now, id, now.Add(-5*time.Minute),
	)
	return err
}
