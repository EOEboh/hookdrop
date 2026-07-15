package cmd

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"math/rand"
	"net/url"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"github.com/EOEboh/hookdrop/cli/internal/api"
	"github.com/EOEboh/hookdrop/cli/internal/forward"
	"github.com/EOEboh/hookdrop/cli/internal/output"
	"github.com/EOEboh/hookdrop/cli/internal/sseclient"
	"github.com/spf13/cobra"
)

const (
	reconnectBaseDelay = time.Second
	reconnectMaxDelay  = 30 * time.Second
	// a connection that survives this long resets the backoff
	stableConnection = time.Minute
)

var (
	listenEndpoint string
	listenForward  string
)

var listenCmd = &cobra.Command{
	Use:   "listen --endpoint <slug-or-session-id>",
	Short: "Stream webhooks live into your terminal",
	Long: `Streams every webhook hitting your hookdrop endpoint into the terminal,
one line per event. With --forward, each webhook is also re-sent to a local
URL — the same way the web UI's replay works — so your local dev server
receives traffic the hosted backend can't deliver directly.`,
	Args: cobra.NoArgs,
	RunE: func(cmd *cobra.Command, args []string) error {
		if listenEndpoint == "" {
			return errors.New("--endpoint is required. Run 'hookdrop endpoints' to list yours")
		}
		if listenForward != "" {
			u, err := url.Parse(listenForward)
			if err != nil || (u.Scheme != "http" && u.Scheme != "https") || u.Host == "" {
				return fmt.Errorf("--forward must be an http(s) URL, got %q", listenForward)
			}
		}

		client, cfg, err := loadClient()
		if err != nil {
			return err
		}

		ctx, stop := signal.NotifyContext(cmd.Context(), os.Interrupt, syscall.SIGTERM)
		defer stop()

		return runListen(ctx, cfg.APIURL, client.Token)
	},
}

func init() {
	listenCmd.Flags().StringVar(&listenEndpoint, "endpoint", "", "named endpoint slug or temporary session ID")
	listenCmd.Flags().StringVar(&listenForward, "forward", "", "forward each webhook to this local URL (e.g. http://localhost:3000/webhook)")
	rootCmd.AddCommand(listenCmd)
}

func runListen(ctx context.Context, apiURL, token string) error {
	printer := output.NewPrinter()
	defer printer.Close()

	forwarding := listenForward != ""
	var fwd *forward.Forwarder
	if forwarding {
		fwd = forward.New(listenForward, func(res forward.Result) {
			printer.Line(printer.ForwardResult(res))
		})
		fwd.Start(ctx)
	}

	events := make(chan sseclient.Event, 256)

	// Renderer: the only consumer of the events channel
	go func() {
		for ev := range events {
			switch ev.Name {
			case "connected":
				inbox := strings.TrimRight(apiURL, "/") + "/i/" + listenEndpoint
				msg := "Listening on " + inbox
				if forwarding {
					msg += "  →  forwarding to " + listenForward
				}
				printer.Line(printer.Status(msg))
			case "request":
				var req api.CapturedRequest
				if err := json.Unmarshal(ev.Data, &req); err != nil {
					printer.Line(printer.Status("skipped an event the CLI couldn't parse: " + err.Error()))
					continue
				}
				printer.Line(printer.Event(&req, forwarding))
				if forwarding && !fwd.Enqueue(&req) {
					printer.Line(printer.Status("⚠ forward queue full — dropped " + output.ShortID(req.ID) + " (still visible in the dashboard)"))
				}
			}
		}
	}()
	defer close(events)

	// Connection loop: reconnect forever on transient failures, stop cleanly
	// on auth/ownership errors or ctrl-c.
	sse := sseclient.New(apiURL, token)
	delay := reconnectBaseDelay
	attempt := 0

	for {
		connectedAt := time.Now()
		err := sse.Stream(ctx, listenEndpoint, events)

		switch {
		case ctx.Err() != nil:
			printer.Line(printer.Status("stopped"))
			return nil
		case errors.Is(err, api.ErrUnauthorized):
			return errors.New("your session is no longer valid — the token may have been revoked. Run 'hookdrop login' again")
		case errors.Is(err, api.ErrNotFound):
			return fmt.Errorf("endpoint %q not found on your account (it may have expired if it was a temporary session). Run 'hookdrop endpoints' to see yours", listenEndpoint)
		case errors.Is(err, api.ErrPaymentRequired):
			return friendlyAuthErr(err)
		}

		if time.Since(connectedAt) > stableConnection {
			delay = reconnectBaseDelay
			attempt = 0
		}
		attempt++
		jittered := delay + time.Duration(rand.Int63n(int64(delay/2+1)))
		printer.Line(printer.Status(fmt.Sprintf("⟳ connection lost (%v) — reconnecting in %s (attempt %d)", err, jittered.Round(time.Second), attempt)))

		select {
		case <-time.After(jittered):
		case <-ctx.Done():
			printer.Line(printer.Status("stopped"))
			return nil
		}

		delay *= 2
		if delay > reconnectMaxDelay {
			delay = reconnectMaxDelay
		}
	}
}
