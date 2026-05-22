package store

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"strings"
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

// addColumnIfNotExists safely runs ALTER TABLE ADD COLUMN
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

	CREATE TABLE IF NOT EXISTS webhook_secrets (
    id          TEXT PRIMARY KEY,
    endpoint_id TEXT NOT NULL,
    provider    TEXT NOT NULL,
    secret      TEXT NOT NULL,
    created_at  DATETIME NOT NULL,
    UNIQUE(endpoint_id, provider),
    FOREIGN KEY (endpoint_id) REFERENCES endpoints(id) ON DELETE CASCADE
	);

	CREATE INDEX IF NOT EXISTS idx_webhook_secrets_endpoint
    ON webhook_secrets(endpoint_id);

	CREATE TABLE IF NOT EXISTS subscriptions (
    id                  TEXT PRIMARY KEY,
    user_id             TEXT UNIQUE NOT NULL,
    plan                TEXT NOT NULL DEFAULT 'free',
    provider            TEXT,            -- 'stripe' or 'paystack'
    provider_customer_id TEXT,           -- Stripe customer ID or Paystack customer code
    provider_sub_id     TEXT,            -- Stripe subscription ID or Paystack subscription code
    status              TEXT NOT NULL DEFAULT 'active',  -- active, past_due, canceled, trialing
    current_period_end  DATETIME,
    trial_end           DATETIME,
    cancel_at_period_end INTEGER DEFAULT 0,
    currency            TEXT DEFAULT 'usd',
    interval            TEXT DEFAULT 'month',  -- month or year
    created_at          DATETIME NOT NULL,
    updated_at          DATETIME NOT NULL,
    FOREIGN KEY (user_id) REFERENCES users(id) ON DELETE CASCADE
);

	CREATE TABLE IF NOT EXISTS billing_events (
    id           TEXT PRIMARY KEY,
    user_id      TEXT,
    provider     TEXT NOT NULL,
    event_type   TEXT NOT NULL,
    payload      TEXT NOT NULL,
    processed    INTEGER DEFAULT 0,
    created_at   DATETIME NOT NULL
);

	CREATE INDEX IF NOT EXISTS idx_subscriptions_user
    ON subscriptions(user_id);

	CREATE INDEX IF NOT EXISTS idx_subscriptions_provider_sub
    ON subscriptions(provider_sub_id);
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
		{"requests", "verified", "TEXT DEFAULT 'unverified'"},
		{"requests", "provider", "TEXT DEFAULT ''"},
		{"users", "country", "TEXT"},
		{"users", "billing_email", "TEXT"},
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
			(id, session_id, method, headers, body, body_size, remote_ip, received_at, verified, provider)
		 VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		req.ID, req.SessionID, req.Method,
		string(headersJSON), req.Body, req.BodySize,
		req.RemoteIP, req.ReceivedAt,
		req.Verified, req.Provider,
	)
	return err
}

func (s *Store) GetRequests(sessionID string, filter models.RequestFilter) ([]*models.CapturedRequest, error) {
	// Build WHERE clauses dynamically based on active filters
	conditions := []string{"session_id = ?"}
	args := []interface{}{sessionID}

	if filter.Method != "" {
		conditions = append(conditions, "method = ?")
		args = append(args, filter.Method)
	}

	if filter.Verified != "" {
		conditions = append(conditions, "COALESCE(verified, 'unverified') = ?")
		args = append(args, filter.Verified)
	}

	if !filter.Since.IsZero() {
		conditions = append(conditions, "received_at >= ?")
		args = append(args, filter.Since)
	}

	if filter.Search != "" {

		// SQLite FTS5 is the upgrade path when this becomes slow
		conditions = append(conditions, "CAST(body AS TEXT) LIKE ?")
		args = append(args, "%"+filter.Search+"%")
	}

	limit := filter.Limit
	if limit <= 0 || limit > 200 {
		limit = 100
	}

	query := fmt.Sprintf(`
		SELECT id, session_id, method, headers, body, body_size,
		       remote_ip, received_at,
		       COALESCE(verified, 'unverified'),
		       COALESCE(provider, '')
		FROM requests
		WHERE %s
		ORDER BY received_at DESC
		LIMIT ?`,
		strings.Join(conditions, " AND "),
	)
	args = append(args, limit)

	rows, err := s.db.Query(query, args...)
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
			&req.Verified, &req.Provider,
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
		`SELECT id, session_id, method, headers, body, body_size, remote_ip, received_at,
		        COALESCE(verified, 'unverified'), COALESCE(provider, '')
		 FROM requests WHERE id = ?`, id,
	).Scan(
		&req.ID, &req.SessionID, &req.Method,
		&headersJSON, &req.Body, &req.BodySize,
		&req.RemoteIP, &req.ReceivedAt,
		&req.Verified, &req.Provider,
	)
	if err == sql.ErrNoRows {
		return nil, nil
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
	// Fetch existing user first
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
		Token:     uuid.NewString() + uuid.NewString(),
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

func (s *Store) SaveWebhookSecret(secret *models.WebhookSecret) error {
	_, err := s.db.Exec(`
        INSERT INTO webhook_secrets (id, endpoint_id, provider, secret, created_at)
        VALUES (?, ?, ?, ?, ?)
        ON CONFLICT(endpoint_id, provider)
        DO UPDATE SET secret = excluded.secret`,
		secret.ID, secret.EndpointID, secret.Provider,
		secret.Secret, secret.CreatedAt,
	)
	return err
}

func (s *Store) GetWebhookSecrets(endpointID string) ([]*models.WebhookSecret, error) {
	rows, err := s.db.Query(`
        SELECT id, endpoint_id, provider, created_at
        FROM webhook_secrets
        WHERE endpoint_id = ?
        ORDER BY created_at DESC`, endpointID,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var results []*models.WebhookSecret
	for rows.Next() {
		ws := &models.WebhookSecret{}
		if err := rows.Scan(&ws.ID, &ws.EndpointID, &ws.Provider, &ws.CreatedAt); err != nil {
			return nil, err
		}
		results = append(results, ws)
	}
	return results, rows.Err()
}

// GetWebhookSecretsWithValues returns secrets including the actual secret value
// Only used internally for verification: never exposed via API
func (s *Store) GetWebhookSecretsWithValues(endpointID string) ([]*models.WebhookSecret, error) {
	rows, err := s.db.Query(`
        SELECT id, endpoint_id, provider, secret, created_at
        FROM webhook_secrets
        WHERE endpoint_id = ?`, endpointID,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var results []*models.WebhookSecret
	for rows.Next() {
		ws := &models.WebhookSecret{}
		if err := rows.Scan(&ws.ID, &ws.EndpointID, &ws.Provider, &ws.Secret, &ws.CreatedAt); err != nil {
			return nil, err
		}
		results = append(results, ws)
	}
	return results, rows.Err()
}

func (s *Store) DeleteWebhookSecret(id, endpointID string) error {
	_, err := s.db.Exec(
		`DELETE FROM webhook_secrets WHERE id = ? AND endpoint_id = ?`,
		id, endpointID,
	)
	return err
}

// UpdateRequestVerification stores the verification result on a captured request
func (s *Store) UpdateRequestVerification(requestID, status, provider string) error {
	_, err := s.db.Exec(
		`UPDATE requests SET verified = ?, provider = ? WHERE id = ?`,
		status, provider, requestID,
	)
	return err
}

func (s *Store) EndpointBelongsToUser(endpointID, userID string) bool {
	var count int
	err := s.db.QueryRow(
		`SELECT COUNT(*) FROM endpoints WHERE id = ? AND user_id = ?`,
		endpointID, userID,
	).Scan(&count)
	return err == nil && count > 0
}

// SUBSCRIPTION/PAYMENT METHODS

func (s *Store) GetSubscription(userID string) (*models.Subscription, error) {
	sub := &models.Subscription{}
	err := s.db.QueryRow(`
        SELECT id, user_id, plan, COALESCE(provider,''), 
               COALESCE(provider_customer_id,''), COALESCE(provider_sub_id,''),
               status, current_period_end, trial_end,
               cancel_at_period_end, COALESCE(currency,'usd'),
               COALESCE(interval,'month'), created_at, updated_at
        FROM subscriptions WHERE user_id = ?`, userID,
	).Scan(
		&sub.ID, &sub.UserID, &sub.Plan, &sub.Provider,
		&sub.ProviderCustomerID, &sub.ProviderSubID,
		&sub.Status, &sub.CurrentPeriodEnd, &sub.TrialEnd,
		&sub.CancelAtPeriodEnd, &sub.Currency, &sub.Interval,
		&sub.CreatedAt, &sub.UpdatedAt,
	)
	if err == sql.ErrNoRows {
		// Return a default free subscription if none exists
		return &models.Subscription{
			UserID:   userID,
			Plan:     "free",
			Status:   "active",
			Currency: "usd",
		}, nil
	}
	return sub, err
}

func (s *Store) UpsertSubscription(sub *models.Subscription) error {
	if sub.ID == "" {
		sub.ID = uuid.NewString()
	}
	sub.UpdatedAt = time.Now().UTC()

	_, err := s.db.Exec(`
        INSERT INTO subscriptions
            (id, user_id, plan, provider, provider_customer_id, provider_sub_id,
             status, current_period_end, trial_end, cancel_at_period_end,
             currency, interval, created_at, updated_at)
        VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
        ON CONFLICT(user_id) DO UPDATE SET
            plan                 = excluded.plan,
            provider             = excluded.provider,
            provider_customer_id = excluded.provider_customer_id,
            provider_sub_id      = excluded.provider_sub_id,
            status               = excluded.status,
            current_period_end   = excluded.current_period_end,
            trial_end            = excluded.trial_end,
            cancel_at_period_end = excluded.cancel_at_period_end,
            currency             = excluded.currency,
            interval             = excluded.interval,
            updated_at           = excluded.updated_at`,
		sub.ID, sub.UserID, sub.Plan, sub.Provider,
		sub.ProviderCustomerID, sub.ProviderSubID,
		sub.Status, sub.CurrentPeriodEnd, sub.TrialEnd,
		sub.CancelAtPeriodEnd, sub.Currency, sub.Interval,
		sub.CreatedAt, sub.UpdatedAt,
	)
	return err
}

func (s *Store) CountUserEndpoints(userID string) (int, error) {
	var count int
	err := s.db.QueryRow(
		`SELECT COUNT(*) FROM endpoints WHERE user_id = ?`, userID,
	).Scan(&count)
	return count, err
}

func (s *Store) CountUserRequestsThisMonth(userID string) (int, error) {
	var count int
	err := s.db.QueryRow(`
        SELECT COUNT(*) FROM requests r
        JOIN endpoints e ON e.id = r.session_id
        WHERE e.user_id = ?
        AND r.received_at >= datetime('now', 'start of month')`, userID,
	).Scan(&count)
	return count, err
}
