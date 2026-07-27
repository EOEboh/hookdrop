package store

import (
	"errors"
	"path/filepath"
	"testing"
	"time"

	"github.com/EOEboh/hookdrop/internal/models"
)

func newTestStore(t *testing.T) *Store {
	t.Helper()
	s, err := New(filepath.Join(t.TempDir(), "test.db"))
	if err != nil {
		t.Fatalf("new store: %v", err)
	}
	return s
}

func TestAPITokenLifecycle(t *testing.T) {
	s := newTestStore(t)

	user, err := s.GetOrCreateUser("cli@example.com")
	if err != nil {
		t.Fatalf("create user: %v", err)
	}

	tok := &models.APIToken{
		ID:          "tok-1",
		UserID:      user.ID,
		Name:        "CLI on test",
		TokenHash:   "hash-1",
		TokenPrefix: "hkdp_abc1234",
		CreatedAt:   time.Now().UTC(),
	}
	if err := s.CreateAPIToken(tok); err != nil {
		t.Fatalf("create token: %v", err)
	}

	got, err := s.GetAPITokenByHash("hash-1")
	if err != nil || got == nil {
		t.Fatalf("lookup active token: got %v, err %v", got, err)
	}
	if got.UserID != user.ID {
		t.Fatalf("token user = %q, want %q", got.UserID, user.ID)
	}

	if got, _ := s.GetAPITokenByHash("wrong-hash"); got != nil {
		t.Fatal("unknown hash should return nil")
	}

	// Expired tokens are filtered in SQL
	past := time.Now().UTC().Add(-time.Hour)
	expired := &models.APIToken{
		ID: "tok-2", UserID: user.ID, Name: "expired",
		TokenHash: "hash-2", TokenPrefix: "hkdp_def5678",
		CreatedAt: time.Now().UTC(), ExpiresAt: &past,
	}
	if err := s.CreateAPIToken(expired); err != nil {
		t.Fatalf("create expired token: %v", err)
	}
	if got, _ := s.GetAPITokenByHash("hash-2"); got != nil {
		t.Fatal("expired token should not resolve")
	}

	// Revocation
	if err := s.RevokeAPIToken("tok-1", user.ID); err != nil {
		t.Fatalf("revoke: %v", err)
	}
	if got, _ := s.GetAPITokenByHash("hash-1"); got != nil {
		t.Fatal("revoked token should not resolve")
	}

	tokens, err := s.ListAPITokens(user.ID)
	if err != nil {
		t.Fatalf("list: %v", err)
	}
	if len(tokens) != 2 {
		t.Fatalf("list returned %d tokens, want 2", len(tokens))
	}

	// Revoke-all covers remaining active tokens
	tok3 := &models.APIToken{
		ID: "tok-3", UserID: user.ID, Name: "third",
		TokenHash: "hash-3", TokenPrefix: "hkdp_ghi9012",
		CreatedAt: time.Now().UTC(),
	}
	if err := s.CreateAPIToken(tok3); err != nil {
		t.Fatalf("create third token: %v", err)
	}
	if err := s.RevokeAllAPITokens(user.ID); err != nil {
		t.Fatalf("revoke all: %v", err)
	}
	if got, _ := s.GetAPITokenByHash("hash-3"); got != nil {
		t.Fatal("token should be revoked after revoke-all")
	}
}

func TestResolveIdentifierForUser(t *testing.T) {
	s := newTestStore(t)

	owner, err := s.GetOrCreateUser("owner@example.com")
	if err != nil {
		t.Fatalf("create owner: %v", err)
	}
	other, err := s.GetOrCreateUser("other@example.com")
	if err != nil {
		t.Fatalf("create other: %v", err)
	}

	ep := &models.Endpoint{
		ID:        "ep-uuid-1",
		UserID:    owner.ID,
		Slug:      "my-slug",
		Name:      "My endpoint",
		CreatedAt: time.Now().UTC(),
	}
	if err := s.CreateEndpoint(ep); err != nil {
		t.Fatalf("create endpoint: %v", err)
	}

	ownedSession := &models.Session{
		ID:        "sess1234",
		UserID:    owner.ID,
		CreatedAt: time.Now().UTC(),
		ExpiresAt: time.Now().UTC().Add(time.Hour),
	}
	if err := s.CreateSession(ownedSession); err != nil {
		t.Fatalf("create owned session: %v", err)
	}

	legacySession := &models.Session{
		ID:        "sess5678",
		CreatedAt: time.Now().UTC(),
		ExpiresAt: time.Now().UTC().Add(time.Hour),
	}
	if err := s.CreateSession(legacySession); err != nil {
		t.Fatalf("create legacy session: %v", err)
	}

	expiredSession := &models.Session{
		ID:        "sess9999",
		UserID:    owner.ID,
		CreatedAt: time.Now().UTC().Add(-25 * time.Hour),
		ExpiresAt: time.Now().UTC().Add(-time.Hour),
	}
	if err := s.CreateSession(expiredSession); err != nil {
		t.Fatalf("create expired session: %v", err)
	}

	tests := []struct {
		name       string
		identifier string
		userID     string
		wantID     string
		wantOK     bool
	}{
		{"slug resolves to endpoint ID for owner", "my-slug", owner.ID, ep.ID, true},
		{"slug denied for other user", "my-slug", other.ID, "", false},
		{"endpoint UUID allowed for owner", ep.ID, owner.ID, ep.ID, true},
		{"endpoint UUID denied for other user", ep.ID, other.ID, "", false},
		{"owned session allowed for owner", "sess1234", owner.ID, "sess1234", true},
		{"owned session denied for other user", "sess1234", other.ID, "", false},
		{"legacy NULL-user session open to anyone", "sess5678", other.ID, "sess5678", true},
		{"expired session denied even for owner", "sess9999", owner.ID, "", false},
		{"unknown identifier denied", "nope", owner.ID, "", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gotID, gotOK := s.ResolveIdentifierForUser(tt.identifier, tt.userID)
			if gotOK != tt.wantOK {
				t.Fatalf("ok = %v, want %v", gotOK, tt.wantOK)
			}
			if tt.wantOK && gotID != tt.wantID {
				t.Fatalf("canonical ID = %q, want %q", gotID, tt.wantID)
			}
		})
	}
}

func billingEvent(key, objectID string, at time.Time) *models.BillingEvent {
	return &models.BillingEvent{
		Provider:  "lemonsqueezy",
		EventType: "subscription_updated",
		Payload:   `{"meta":{"event_name":"subscription_updated"}}`,
		EventKey:  key,
		ObjectID:  objectID,
		EventAt:   &at,
	}
}

// Retried deliveries are absorbed by the unique index, not by application logic.
func TestRecordBillingEventDeduplicates(t *testing.T) {
	s := newTestStore(t)
	now := time.Now().UTC()

	if err := s.RecordBillingEvent(billingEvent("key-1", "sub-1", now)); err != nil {
		t.Fatalf("first record: %v", err)
	}

	err := s.RecordBillingEvent(billingEvent("key-1", "sub-1", now))
	if !errors.Is(err, ErrDuplicateBillingEvent) {
		t.Fatalf("second record: err = %v, want ErrDuplicateBillingEvent", err)
	}

	// A genuinely different event still records.
	if err := s.RecordBillingEvent(billingEvent("key-2", "sub-1", now)); err != nil {
		t.Fatalf("distinct event: %v", err)
	}
}

func TestLatestProcessedEventAt(t *testing.T) {
	s := newTestStore(t)
	older := time.Now().UTC().Add(-2 * time.Hour).Truncate(time.Second)
	newer := time.Now().UTC().Add(-1 * time.Hour).Truncate(time.Second)

	// Unprocessed events must not count: they were never applied.
	if err := s.RecordBillingEvent(billingEvent("key-unprocessed", "sub-1", newer)); err != nil {
		t.Fatalf("record: %v", err)
	}
	if _, found, err := s.LatestProcessedEventAt("lemonsqueezy", "sub-1"); err != nil || found {
		t.Fatalf("found = %t (err %v), want false — nothing is processed yet", found, err)
	}

	if err := s.RecordBillingEvent(billingEvent("key-old", "sub-1", older)); err != nil {
		t.Fatalf("record: %v", err)
	}
	if err := s.MarkBillingEventProcessed("key-old", "user-1"); err != nil {
		t.Fatalf("mark processed: %v", err)
	}

	got, found, err := s.LatestProcessedEventAt("lemonsqueezy", "sub-1")
	if err != nil || !found {
		t.Fatalf("found = %t (err %v), want true", found, err)
	}
	if !got.Equal(older) {
		t.Errorf("latest = %s, want %s", got, older)
	}

	// A different subscription is scoped separately.
	if _, found, _ := s.LatestProcessedEventAt("lemonsqueezy", "sub-2"); found {
		t.Error("sub-2 picked up sub-1's events")
	}
}

func TestDeleteOldBillingEvents(t *testing.T) {
	s := newTestStore(t)
	now := time.Now().UTC()

	fresh := billingEvent("key-fresh", "sub-1", now)
	fresh.CreatedAt = now
	if err := s.RecordBillingEvent(fresh); err != nil {
		t.Fatalf("record fresh: %v", err)
	}

	stale := billingEvent("key-stale", "sub-1", now)
	stale.CreatedAt = now.Add(-BillingEventRetention - time.Hour)
	if err := s.RecordBillingEvent(stale); err != nil {
		t.Fatalf("record stale: %v", err)
	}

	n, err := s.DeleteOldBillingEvents()
	if err != nil {
		t.Fatalf("delete: %v", err)
	}
	if n != 1 {
		t.Fatalf("deleted %d, want 1 — only the out-of-window row", n)
	}

	// The fresh row survives, so its key still collides.
	if err := s.RecordBillingEvent(billingEvent("key-fresh", "sub-1", now)); !errors.Is(err, ErrDuplicateBillingEvent) {
		t.Errorf("fresh event was deleted: re-record err = %v", err)
	}
}
