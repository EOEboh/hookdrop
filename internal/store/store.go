package store

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"time"

	"github.com/EOEboh/hookdrop/internal/models"
	_ "github.com/mattn/go-sqlite3"
)

type Store struct {
	db *sql.DB
}

func New(dbPath string) (*Store, error) {
	db, err := sql.Open("sqlite3", dbPath+"?_journal_mode=WAL&_busy_timeout=5000")
	if err != nil {
		return nil, fmt.Errorf("open db: %w", err)
	}

	// SQLite works best with a single writer
	db.SetMaxOpenConns(1)

	s := &Store{db: db}
	if err := s.migrate(); err != nil {
		return nil, fmt.Errorf("migrate: %w", err)
	}
	return s, nil
}

func (s *Store) migrate() error {
	schema := `
    CREATE TABLE IF NOT EXISTS sessions (
        id         TEXT PRIMARY KEY,
        created_at DATETIME NOT NULL,
        expires_at DATETIME NOT NULL
    );

    CREATE TABLE IF NOT EXISTS requests (
        id          TEXT PRIMARY KEY,
        session_id  TEXT NOT NULL,
        method      TEXT NOT NULL,
        headers     TEXT NOT NULL,   -- JSON blob
        body        BLOB,
        body_size   INTEGER NOT NULL,
        remote_ip   TEXT NOT NULL,
        received_at DATETIME NOT NULL,
        FOREIGN KEY (session_id) REFERENCES sessions(id) ON DELETE CASCADE
    );

    CREATE INDEX IF NOT EXISTS idx_requests_session
        ON requests(session_id, received_at DESC);
    `
	_, err := s.db.Exec(schema)
	return err
}

func (s *Store) CreateSession(session *models.Session) error {
	_, err := s.db.Exec(
		`INSERT INTO sessions (id, created_at, expires_at) VALUES (?, ?, ?)`,
		session.ID, session.CreatedAt, session.ExpiresAt,
	)
	return err
}

func (s *Store) SessionExists(id string) bool {
	var expiresAt time.Time
	err := s.db.QueryRow(
		`SELECT expires_at FROM sessions WHERE id = ?`, id,
	).Scan(&expiresAt)
	if err != nil {
		return false
	}
	return time.Now().Before(expiresAt) // expired sessions are treated as gone
}

func (s *Store) DeleteExpiredSessions() error {
	_, err := s.db.Exec(
		`DELETE FROM sessions WHERE expires_at < ?`, time.Now(),
	)
	return err
}

func (s *Store) SaveRequest(req *models.CapturedRequest) error {
	headersJSON, err := json.Marshal(req.Headers)
	if err != nil {
		return fmt.Errorf("marshal headers: %w", err)
	}

	_, err = s.db.Exec(
		`INSERT INTO requests
            (id, session_id, method, headers, body, body_size, remote_ip, received_at)
         VALUES (?, ?, ?, ?, ?, ?, ?, ?)`,
		req.ID, req.SessionID, req.Method,
		string(headersJSON), req.Body, req.BodySize,
		req.RemoteIP, req.ReceivedAt,
	)
	return err
}

func (s *Store) GetRequests(sessionID string, limit int) ([]*models.CapturedRequest, error) {
	rows, err := s.db.Query(
		`SELECT id, session_id, method, headers, body, body_size, remote_ip, received_at
         FROM requests
         WHERE session_id = ?
         ORDER BY received_at DESC
         LIMIT ?`,
		sessionID, limit,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var results []*models.CapturedRequest
	for rows.Next() {
		req := &models.CapturedRequest{}
		var headersJSON string

		err := rows.Scan(
			&req.ID, &req.SessionID, &req.Method,
			&headersJSON, &req.Body, &req.BodySize,
			&req.RemoteIP, &req.ReceivedAt,
		)
		if err != nil {
			return nil, err
		}

		if err := json.Unmarshal([]byte(headersJSON), &req.Headers); err != nil {
			return nil, fmt.Errorf("unmarshal headers: %w", err)
		}

		results = append(results, req)
	}
	return results, rows.Err()
}

func (s *Store) GetRequest(id string) (*models.CapturedRequest, error) {
	req := &models.CapturedRequest{}
	var headersJSON string

	err := s.db.QueryRow(
		`SELECT id, session_id, method, headers, body, body_size, remote_ip, received_at
         FROM requests WHERE id = ?`, id,
	).Scan(
		&req.ID, &req.SessionID, &req.Method,
		&headersJSON, &req.Body, &req.BodySize,
		&req.RemoteIP, &req.ReceivedAt,
	)
	if err == sql.ErrNoRows {
		return nil, nil // not found, not an error
	}
	if err != nil {
		return nil, err
	}

	if err := json.Unmarshal([]byte(headersJSON), &req.Headers); err != nil {
		return nil, err
	}
	return req, nil
}
