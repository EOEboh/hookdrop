package cmd

import (
	"context"
	"fmt"
	"net"
	"net/http"
	"testing"
	"time"
)

// listen binds a fresh loopback listener the way browserLogin does (port 0).
func listen(t *testing.T) net.Listener {
	t.Helper()
	l, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("listen: %v", err)
	}
	return l
}

func callbackURL(l net.Listener, query string) string {
	return fmt.Sprintf("http://%s/callback%s", l.Addr().String(), query)
}

func TestWaitForCallbackDeliversToken(t *testing.T) {
	l := listen(t)
	result := make(chan string, 1)
	go func() {
		tok, err := waitForCallback(context.Background(), l, "s3cret", 5*time.Second)
		if err != nil {
			t.Errorf("waitForCallback: %v", err)
		}
		result <- tok
	}()

	// give the server a moment to start
	var resp *http.Response
	var err error
	for i := 0; i < 50; i++ {
		resp, err = http.Get(callbackURL(l, "?state=s3cret&token=hkdp_abc"))
		if err == nil {
			break
		}
		time.Sleep(10 * time.Millisecond)
	}
	if err != nil {
		t.Fatalf("callback GET: %v", err)
	}
	resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("callback status = %d, want 200", resp.StatusCode)
	}

	select {
	case tok := <-result:
		if tok != "hkdp_abc" {
			t.Fatalf("token = %q, want hkdp_abc", tok)
		}
	case <-time.After(2 * time.Second):
		t.Fatal("waitForCallback did not return the token")
	}
}

func TestWaitForCallbackRejectsWrongState(t *testing.T) {
	l := listen(t)
	done := make(chan struct{})
	go func() {
		// short timeout: the wrong-state hit must NOT resolve, so this ends
		// via timeout, proving the mismatched callback was ignored.
		_, err := waitForCallback(context.Background(), l, "correct", 400*time.Millisecond)
		if err == nil {
			t.Error("expected timeout error, got nil (wrong state resolved the wait)")
		}
		close(done)
	}()

	var resp *http.Response
	var err error
	for i := 0; i < 50; i++ {
		resp, err = http.Get(callbackURL(l, "?state=wrong&token=hkdp_x"))
		if err == nil {
			break
		}
		time.Sleep(10 * time.Millisecond)
	}
	if err != nil {
		t.Fatalf("callback GET: %v", err)
	}
	if resp.StatusCode != http.StatusNotFound {
		t.Fatalf("wrong-state status = %d, want 404", resp.StatusCode)
	}
	resp.Body.Close()

	select {
	case <-done:
	case <-time.After(2 * time.Second):
		t.Fatal("waitForCallback should have timed out")
	}
}

func TestWaitForCallbackMissingToken(t *testing.T) {
	l := listen(t)
	go waitForCallback(context.Background(), l, "s", 400*time.Millisecond)

	var resp *http.Response
	var err error
	for i := 0; i < 50; i++ {
		resp, err = http.Get(callbackURL(l, "?state=s"))
		if err == nil {
			break
		}
		time.Sleep(10 * time.Millisecond)
	}
	if err != nil {
		t.Fatalf("callback GET: %v", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusBadRequest {
		t.Fatalf("missing-token status = %d, want 400", resp.StatusCode)
	}
}

func TestWaitForCallbackTimesOut(t *testing.T) {
	l := listen(t)
	start := time.Now()
	_, err := waitForCallback(context.Background(), l, "s", 200*time.Millisecond)
	if err == nil {
		t.Fatal("expected timeout error")
	}
	if elapsed := time.Since(start); elapsed < 150*time.Millisecond {
		t.Fatalf("returned too early (%v) — timeout not honored", elapsed)
	}
}

// Two listeners bound back-to-back get distinct OS-assigned ports and each
// delivers independently — demonstrating the port-0 design has no
// fixed-port collision even with concurrent logins.
func TestConcurrentCallbacksDistinctPorts(t *testing.T) {
	l1, l2 := listen(t), listen(t)
	if l1.Addr().String() == l2.Addr().String() {
		t.Fatal("expected distinct ports")
	}

	res := make(chan string, 2)
	for _, pair := range []struct {
		l     net.Listener
		state string
		tok   string
	}{{l1, "s1", "hkdp_one"}, {l2, "s2", "hkdp_two"}} {
		p := pair
		go func() {
			tok, err := waitForCallback(context.Background(), p.l, p.state, 5*time.Second)
			if err != nil {
				t.Errorf("waitForCallback: %v", err)
			}
			res <- tok
		}()
	}

	deliver := func(l net.Listener, state, tok string) {
		for i := 0; i < 50; i++ {
			resp, err := http.Get(callbackURL(l, "?state="+state+"&token="+tok))
			if err == nil {
				resp.Body.Close()
				return
			}
			time.Sleep(10 * time.Millisecond)
		}
		t.Errorf("could not reach callback on %s", l.Addr())
	}
	deliver(l1, "s1", "hkdp_one")
	deliver(l2, "s2", "hkdp_two")

	got := map[string]bool{}
	for i := 0; i < 2; i++ {
		select {
		case tok := <-res:
			got[tok] = true
		case <-time.After(2 * time.Second):
			t.Fatal("a callback did not return")
		}
	}
	if !got["hkdp_one"] || !got["hkdp_two"] {
		t.Fatalf("expected both tokens, got %v", got)
	}
}
