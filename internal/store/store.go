package store

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"time"

	"github.com/EOEboh/hookdrop/internal/models"
	"github.com/google/uuid"
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

// addColumnIfNotExists safely runs ALTER TABLE ADD COLUMN —
func (s *Store) addColumnIfNotExists(table, column, definition string) error {
	// Query the table's column info
	rows, err := s.db.Query(fmt.Sprintf("PRAGMA table_info(%s)", table))
	if err != nil {
		return err
	}
	defer rows.Close()

	for rows.Next() {
		var cid int
		var name, colType string
		var notNull, pk int
		var defaultVal sql.NullString

		if err := rows.Scan(&cid, &name, &colType, &notNull, &defaultVal, &pk); err != nil {
			return err
		}
		if name == column {
			return nil // Column already exists, no need to add
		}
	}

	// Column doesn't exist: add it
	_, err = s.db.Exec(
		fmt.Sprintf("ALTER TABLE %s ADD COLUMN %s %s", table, column, definition),
	)
	return err
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
        headers     TEXT NOT NULL,
        body        BLOB,
        body_size   INTEGER NOT NULL,
        remote_ip   TEXT NOT NULL,
        received_at DATETIME NOT NULL,
        FOREIGN KEY (session_id) REFERENCES sessions(id) ON DELETE CASCADE
    );

    CREATE INDEX IF NOT EXISTS idx_requests_session
        ON requests(session_id, received_at DESC);

    CREATE TABLE IF NOT EXISTS users (
        id         TEXT PRIMARY KEY,
        email      TEXT UNIQUE NOT NULL,
        created_at DATETIME NOT NULL
    );

    CREATE TABLE IF NOT EXISTS magic_links (
        id         TEXT PRIMARY KEY,
        user_id    TEXT NOT NULL,
        token      TEXT UNIQUE NOT NULL,
        expires_at DATETIME NOT NULL,
        used       INTEGER NOT NULL DEFAULT 0,
        FOREIGN KEY (user_id) REFERENCES users(id) ON DELETE CASCADE
    );

    CREATE INDEX IF NOT EXISTS idx_magic_links_token
        ON magic_links(token);

    CREATE TABLE IF NOT EXISTS endpoints (
        id          TEXT PRIMARY KEY,
        user_id     TEXT NOT NULL,
        slug        TEXT UNIQUE NOT NULL,
        name        TEXT NOT NULL,
        description TEXT,
        created_at  DATETIME NOT NULL,
        FOREIGN KEY (user_id) REFERENCES users(id) ON DELETE CASCADE
    );

    CREATE INDEX IF NOT EXISTS idx_endpoints_user
        ON endpoints(user_id);

    CREATE INDEX IF NOT EXISTS idx_endpoints_slug
        ON endpoints(slug);
    `

	if _, err := s.db.Exec(schema); err != nil {
		return err
	}

	// Safe column additions — idempotent, safe to run on every startup
	migrations := []struct {
		table      string
		column     string
		definition string
	}{
		{"sessions", "user_id", "TEXT REFERENCES users(id)"},
		{"requests", "endpoint_id", "TEXT REFERENCES endpoints(id)"},
	}

	for _, m := range migrations {
		if err := s.addColumnIfNotExists(m.table, m.column, m.definition); err != nil {
			return fmt.Errorf("add column %s.%s: %w", m.table, m.column, err)
		}
	}

	return nil
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

func (s *Store) GetOrCreateUser(email string) (*models.User, error) {
	// Try fetching existing user first
	user := &models.User{}
	err := s.db.QueryRow(
		`SELECT id, email, created_at FROM users WHERE email = ?`, email,
	).Scan(&user.ID, &user.Email, &user.CreatedAt)

	if err == nil {
		return user, nil // already exists
	}
	if err != sql.ErrNoRows {
		return nil, err
	}

	// Create new user
	user = &models.User{
		ID:        uuid.NewString(),
		Email:     email,
		CreatedAt: time.Now().UTC(),
	}
	_, err = s.db.Exec(
		`INSERT INTO users (id, email, created_at) VALUES (?, ?, ?)`,
		user.ID, user.Email, user.CreatedAt,
	)
	return user, err
}

func (s *Store) CreateMagicLink(userID string) (*models.MagicLink, error) {
	// Invalidate any existing unused links for this user
	_, err := s.db.Exec(
		`UPDATE magic_links SET used = 1 WHERE user_id = ? AND used = 0`, userID,
	)
	if err != nil {
		return nil, err
	}

	link := &models.MagicLink{
		ID:        uuid.NewString(),
		UserID:    userID,
		Token:     uuid.NewString() + uuid.NewString(), // long token — harder to brute force
		ExpiresAt: time.Now().UTC().Add(15 * time.Minute),
		Used:      false,
	}
	_, err = s.db.Exec(
		`INSERT INTO magic_links (id, user_id, token, expires_at, used)
         VALUES (?, ?, ?, ?, 0)`,
		link.ID, link.UserID, link.Token, link.ExpiresAt,
	)
	return link, err
}

func (s *Store) ConsumeMagicLink(token string) (*models.User, error) {
	// Fetch the link and join with the user in one query
	var link models.MagicLink
	var user models.User

	err := s.db.QueryRow(`
        SELECT ml.id, ml.user_id, ml.expires_at, ml.used,
               u.id, u.email, u.created_at
        FROM magic_links ml
        JOIN users u ON u.id = ml.user_id
        WHERE ml.token = ?`, token,
	).Scan(
		&link.ID, &link.UserID, &link.ExpiresAt, &link.Used,
		&user.ID, &user.Email, &user.CreatedAt,
	)
	if err == sql.ErrNoRows {
		return nil, nil // token not found
	}
	if err != nil {
		return nil, err
	}
	if link.Used {
		return nil, fmt.Errorf("token already used")
	}
	if time.Now().After(link.ExpiresAt) {
		return nil, fmt.Errorf("token expired")
	}

	// Mark as consumed — one-time use
	_, err = s.db.Exec(
		`UPDATE magic_links SET used = 1 WHERE id = ?`, link.ID,
	)
	if err != nil {
		return nil, err
	}
	return &user, nil
}

func (s *Store) GetUserByID(id string) (*models.User, error) {
	user := &models.User{}
	err := s.db.QueryRow(
		`SELECT id, email, created_at FROM users WHERE id = ?`, id,
	).Scan(&user.ID, &user.Email, &user.CreatedAt)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	return user, err
}

func (s *Store) CreateEndpoint(ep *models.Endpoint) error {
	_, err := s.db.Exec(
		`INSERT INTO endpoints (id, user_id, slug, name, description, created_at)
         VALUES (?, ?, ?, ?, ?, ?)`,
		ep.ID, ep.UserID, ep.Slug, ep.Name, ep.Description, ep.CreatedAt,
	)
	return err
}

func (s *Store) GetEndpointBySlug(slug string) (*models.Endpoint, error) {
	ep := &models.Endpoint{}
	err := s.db.QueryRow(
		`SELECT id, user_id, slug, name, description, created_at
         FROM endpoints WHERE slug = ?`, slug,
	).Scan(&ep.ID, &ep.UserID, &ep.Slug, &ep.Name, &ep.Description, &ep.CreatedAt)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	return ep, err
}

func (s *Store) GetEndpointsByUser(userID string) ([]*models.Endpoint, error) {
	rows, err := s.db.Query(
		`SELECT id, user_id, slug, name, description, created_at
         FROM endpoints WHERE user_id = ?
         ORDER BY created_at DESC`, userID,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var results []*models.Endpoint
	for rows.Next() {
		ep := &models.Endpoint{}
		err := rows.Scan(
			&ep.ID, &ep.UserID, &ep.Slug,
			&ep.Name, &ep.Description, &ep.CreatedAt,
		)
		if err != nil {
			return nil, err
		}
		results = append(results, ep)
	}
	return results, rows.Err()
}

func (s *Store) DeleteEndpoint(id, userID string) error {
	// userID check ensures users can only delete their own endpoints
	_, err := s.db.Exec(
		`DELETE FROM endpoints WHERE id = ? AND user_id = ?`, id, userID,
	)
	return err
}

func (s *Store) SlugExists(slug string) (bool, error) {
	var count int
	err := s.db.QueryRow(
		`SELECT COUNT(*) FROM endpoints WHERE slug = ?`, slug,
	).Scan(&count)
	return count > 0, err
}

// EndpointIDExists checks if an ID belongs to a named endpoint
func (s *Store) EndpointIDExists(id string) bool {
	var count int
	err := s.db.QueryRow(
		`SELECT COUNT(*) FROM endpoints WHERE id = ?`, id,
	).Scan(&count)
	return err == nil && count > 0
}

// IdentifierExists checks both sessions and named endpoints
func (s *Store) IdentifierExists(id string) bool {
	return s.SessionExists(id) || s.EndpointIDExists(id)
}
