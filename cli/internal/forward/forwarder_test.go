package forward

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/EOEboh/hookdrop/cli/internal/api"
)

// TestSkipListParity pins the exact header skip list ported from the
// backend's internal/replay/engine.go. If this test needs changing, change
// the server list too.
func TestSkipListParity(t *testing.T) {
	skipped := []string{"Host", "Content-Length", "Transfer-Encoding", "Connection", "Te", "Trailers", "Upgrade"}
	for _, h := range skipped {
		if !shouldSkipHeader(h) {
			t.Errorf("%s should be skipped", h)
		}
	}
	// case-insensitive via canonicalization
	if !shouldSkipHeader("content-length") || !shouldSkipHeader("HOST") {
		t.Error("skip list must be case-insensitive")
	}
	for _, h := range []string{"Content-Type", "X-Signature", "Stripe-Signature", "User-Agent", "Authorization"} {
		if shouldSkipHeader(h) {
			t.Errorf("%s should be forwarded", h)
		}
	}
}

func TestDeliverForwardsHeadersAndBody(t *testing.T) {
	var got *http.Request
	var gotBody []byte
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		got = r.Clone(context.Background())
		buf := make([]byte, r.ContentLength)
		r.Body.Read(buf)
		gotBody = buf
		w.WriteHeader(http.StatusAccepted)
	}))
	defer srv.Close()

	f := New(srv.URL, func(Result) {})
	res := f.deliver(context.Background(), &api.CapturedRequest{
		ID:     "req-1",
		Method: "POST",
		Headers: map[string]string{
			"Content-Type":     "application/json",
			"Stripe-Signature": "sig123",
			"Host":             "should-not-forward.example",
			"Content-Length":   "999",
		},
		Body: []byte(`{"a":1}`),
	})

	if res.Err != nil {
		t.Fatalf("deliver: %v", res.Err)
	}
	if res.Status != http.StatusAccepted {
		t.Fatalf("status = %d, want 202", res.Status)
	}
	if string(gotBody) != `{"a":1}` {
		t.Fatalf("body = %q", gotBody)
	}
	if got.Header.Get("Stripe-Signature") != "sig123" || got.Header.Get("Content-Type") != "application/json" {
		t.Fatal("expected original headers forwarded")
	}
	if got.Host == "should-not-forward.example" {
		t.Fatal("Host header must be rewritten, not forwarded")
	}
	if got.Header.Get("X-Hookdrop-Forwarded") != "true" || got.Header.Get("X-Hookdrop-Original-Id") != "req-1" {
		t.Fatal("expected forward marker headers")
	}
	if got.ContentLength != 7 {
		t.Fatalf("Content-Length = %d, want recalculated 7", got.ContentLength)
	}
}

func TestDeliverUnreachableTarget(t *testing.T) {
	f := New("http://127.0.0.1:1", func(Result) {}) // port 1: nothing listens
	res := f.deliver(context.Background(), &api.CapturedRequest{ID: "x", Method: "POST"})
	if res.Err == nil {
		t.Fatal("expected connection error")
	}
}

func TestEnqueueOverflowDropsInsteadOfBlocking(t *testing.T) {
	f := New("http://127.0.0.1:1", func(Result) {})
	// worker not started — queue just fills
	for i := 0; i < queueSize; i++ {
		if !f.Enqueue(&api.CapturedRequest{}) {
			t.Fatalf("enqueue %d should succeed", i)
		}
	}
	done := make(chan bool, 1)
	go func() { done <- f.Enqueue(&api.CapturedRequest{}) }()
	select {
	case ok := <-done:
		if ok {
			t.Fatal("expected overflow enqueue to report a drop")
		}
	case <-time.After(time.Second):
		t.Fatal("Enqueue blocked on a full queue")
	}
}
