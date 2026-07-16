package cmd

import (
	"bufio"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"math/rand"
	"net/url"
	"os"
	"os/signal"
	"strconv"
	"strings"
	"syscall"
	"time"

	"github.com/EOEboh/hookdrop/cli/internal/api"
	"github.com/EOEboh/hookdrop/cli/internal/forward"
	"github.com/EOEboh/hookdrop/cli/internal/output"
	"github.com/EOEboh/hookdrop/cli/internal/sseclient"
	"github.com/spf13/cobra"
	"golang.org/x/term"
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

var (
	listenForwardFlag string
	listenPath        string
	listenEndpointF   string // hidden --endpoint alias for scripting
)

var listenCmd = &cobra.Command{
	Use:   "listen [endpoint] [flags]",
	Short: "Stream webhooks live into your terminal (and forward them locally)",
	Long: `Streams every webhook hitting your hookdrop endpoint into the terminal,
one line per event. With -f/--forward, each webhook is also re-sent to a local
server — the way the web UI's replay works — so your dev server receives traffic
the hosted backend can't deliver directly.

The endpoint is optional: with one named endpoint it's picked automatically,
with several you're prompted to choose. --forward accepts a bare port (3000),
host:port, or a full URL.`,
	Example: `  # Watch webhooks (choose the endpoint interactively)
  hookdrop listen

  # Watch a specific endpoint
  hookdrop listen my-slug

  # Forward each webhook to a local port
  hookdrop listen my-slug -f 3000

  # Forward to a specific path
  hookdrop listen my-slug -f 3000 --path /webhook

  # Forward to a full URL
  hookdrop listen my-slug -f https://localhost:8443/hook`,
	Args: cobra.MaximumNArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		client, cfg, err := loadClient()
		if err != nil {
			return err
		}

		ctx, stop := signal.NotifyContext(cmd.Context(), os.Interrupt, syscall.SIGTERM)
		defer stop()

		explicit := listenEndpointF
		if len(args) == 1 {
			explicit = args[0]
		}
		endpoint, err := resolveEndpoint(ctx, client, explicit, cfg.FrontendURL)
		if err != nil {
			return friendlyAuthErr(err)
		}

		forwardURL, err := normalizeForward(listenForwardFlag, listenPath)
		if err != nil {
			return err
		}

		return runListen(ctx, cfg.APIURL, client.Token, endpoint, forwardURL)
	},
}

func init() {
	listenCmd.Flags().StringVarP(&listenForwardFlag, "forward", "f", "", "forward webhooks to a local target: a port (3000), host:port, or full URL")
	listenCmd.Flags().StringVar(&listenPath, "path", "", "path appended to the forward target (e.g. /webhook)")
	listenCmd.Flags().StringVar(&listenEndpointF, "endpoint", "", "endpoint slug or session id (usually passed as a positional argument)")
	_ = listenCmd.Flags().MarkHidden("endpoint")
	rootCmd.AddCommand(listenCmd)
}

// normalizeForward turns a user-friendly forward target into a full http(s)
// URL. Accepts "" (watch-only), a bare port, host:port, host:port/path, or a
// complete URL. --path, if set, is appended to the resulting path.
func normalizeForward(raw, path string) (string, error) {
	raw = strings.TrimSpace(raw)
	if raw == "" && path == "" {
		return "", nil
	}

	switch {
	case raw == "" && path != "":
		return "", errors.New("--path needs -f/--forward to be set")
	case isAllDigits(raw):
		raw = "http://localhost:" + raw
	case !strings.Contains(raw, "://"):
		raw = "http://" + raw
	}

	u, err := url.Parse(raw)
	if err != nil || (u.Scheme != "http" && u.Scheme != "https") || u.Host == "" {
		return "", fmt.Errorf("invalid --forward target %q (use a port, host:port, or http(s) URL)", raw)
	}
	if path != "" {
		u.Path = "/" + strings.TrimLeft(strings.TrimRight(u.Path, "/")+"/"+strings.TrimLeft(path, "/"), "/")
	}
	return u.String(), nil
}

func isAllDigits(s string) bool {
	if s == "" {
		return false
	}
	for _, r := range s {
		if r < '0' || r > '9' {
			return false
		}
	}
	return true
}

// resolveEndpoint returns the endpoint to listen on. An explicit value passes
// straight through (the server resolves + ownership-checks it). Otherwise it
// auto-selects a lone endpoint, prompts among several, or guides the user to
// create one.
func resolveEndpoint(ctx context.Context, client *api.Client, explicit, frontendURL string) (string, error) {
	if explicit != "" {
		return explicit, nil
	}

	endpoints, err := client.Endpoints(ctx)
	if err != nil {
		return "", err
	}

	switch len(endpoints) {
	case 0:
		return "", fmt.Errorf("you have no named endpoints yet. Create one at %s, then run 'hookdrop listen <slug>'. (For a temporary session, pass its id.)",
			strings.TrimRight(frontendURL, "/"))
	case 1:
		fmt.Fprintf(os.Stderr, "Using endpoint %s\n", endpoints[0].Slug)
		return endpoints[0].Slug, nil
	default:
		if !term.IsTerminal(int(os.Stdin.Fd())) {
			var slugs []string
			for _, e := range endpoints {
				slugs = append(slugs, e.Slug)
			}
			return "", fmt.Errorf("multiple endpoints (%s) — specify one: hookdrop listen <slug>", strings.Join(slugs, ", "))
		}
		return pickEndpoint(endpoints)
	}
}

// pickEndpoint shows a numbered list and reads a choice from stdin. Portable
// across platforms (line-buffered, no raw terminal mode).
func pickEndpoint(endpoints []api.Endpoint) (string, error) {
	fmt.Fprintln(os.Stderr, "Select an endpoint:")
	for i, e := range endpoints {
		name := e.Slug
		if e.Name != "" {
			name = fmt.Sprintf("%s  (%s)", e.Slug, e.Name)
		}
		fmt.Fprintf(os.Stderr, "  %d) %s\n", i+1, name)
	}
	fmt.Fprint(os.Stderr, "> ")

	line, err := bufio.NewReader(os.Stdin).ReadString('\n')
	if err != nil {
		return "", errors.New("no selection made")
	}
	choice, err := strconv.Atoi(strings.TrimSpace(line))
	if err != nil || choice < 1 || choice > len(endpoints) {
		return "", fmt.Errorf("invalid selection %q", strings.TrimSpace(line))
	}
	return endpoints[choice-1].Slug, nil
}

func runListen(ctx context.Context, apiURL, token, endpoint, forwardURL string) error {
	printer := output.NewPrinter()
	defer printer.Close()

	forwarding := forwardURL != ""
	var fwd *forward.Forwarder
	if forwarding {
		fwd = forward.New(forwardURL, func(res forward.Result) {
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
				inbox := strings.TrimRight(apiURL, "/") + "/i/" + endpoint
				printer.Line(printer.Ready(inbox, forwardURL))
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
		err := sse.Stream(ctx, endpoint, events)

		switch {
		case ctx.Err() != nil:
			printer.Line(printer.Status("stopped"))
			return nil
		case errors.Is(err, api.ErrUnauthorized):
			return errors.New("your session is no longer valid — the token may have been revoked. Run 'hookdrop login' again")
		case errors.Is(err, api.ErrNotFound):
			return fmt.Errorf("endpoint %q not found on your account (it may have expired if it was a temporary session). Run 'hookdrop endpoints' to see yours", endpoint)
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
