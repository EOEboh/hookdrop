package cmd

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"errors"
	"fmt"
	"net"
	"net/http"
	"net/url"
	"os"
	"os/exec"
	"runtime"
	"strings"
	"time"

	"github.com/EOEboh/hookdrop/cli/internal/api"
	"github.com/EOEboh/hookdrop/cli/internal/config"
	"github.com/spf13/cobra"
	"golang.org/x/term"
)

const browserLoginTimeout = 3 * time.Minute

var (
	loginToken     string
	loginNoBrowser bool
)

var loginCmd = &cobra.Command{
	Use:   "login",
	Short: "Authenticate the CLI with your hookdrop account",
	Long: `Opens your browser to authorize the CLI and stores an API token locally.

Use --token to paste a token created at Settings → API tokens instead,
or --no-browser on headless machines.`,
	Args: cobra.NoArgs,
	RunE: func(cmd *cobra.Command, args []string) error {
		cfg, err := config.Load()
		if err != nil {
			return err
		}

		token := strings.TrimSpace(loginToken)
		if token == "" && !loginNoBrowser {
			token, err = browserLogin(cmd.Context(), cfg.FrontendURL)
			if err != nil {
				fmt.Fprintf(os.Stderr, "Browser login didn't complete (%v) — falling back to manual entry.\n", err)
			}
		}
		if token == "" {
			token, err = promptForToken(cfg.FrontendURL)
			if err != nil {
				return err
			}
		}

		if !strings.HasPrefix(token, "hkdp_") {
			return errors.New("that doesn't look like a hookdrop API token (expected it to start with hkdp_)")
		}

		// Verify before saving so a bad paste fails loudly
		me, err := api.New(cfg.APIURL, token).Me(cmd.Context())
		if err != nil {
			if errors.Is(err, api.ErrUnauthorized) {
				return errors.New("that token is invalid, expired, or revoked")
			}
			return err
		}

		cfg.Token = token
		if err := config.Save(cfg); err != nil {
			return err
		}

		fmt.Printf("✓ Logged in as %s (%s plan)\n", me.User.Email, me.Plan)
		return nil
	},
}

func init() {
	loginCmd.Flags().StringVar(&loginToken, "token", "", "API token (skips the browser flow)")
	loginCmd.Flags().BoolVar(&loginNoBrowser, "no-browser", false, "don't open a browser; prompt for a token instead")
	rootCmd.AddCommand(loginCmd)
}

// browserLogin runs a loopback listener, sends the user's browser to the
// web app's /cli-auth page, and waits for it to deliver a freshly minted
// token back to us. The random state ties the callback to this attempt.
func browserLogin(ctx context.Context, frontendURL string) (string, error) {
	stateBytes := make([]byte, 16)
	if _, err := rand.Read(stateBytes); err != nil {
		return "", err
	}
	state := hex.EncodeToString(stateBytes)

	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		return "", fmt.Errorf("start local listener: %w", err)
	}
	defer listener.Close()
	port := listener.Addr().(*net.TCPAddr).Port

	tokenCh := make(chan string, 1)
	server := &http.Server{Handler: http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/callback" || r.URL.Query().Get("state") != state {
			http.NotFound(w, r)
			return
		}
		token := r.URL.Query().Get("token")
		if token == "" {
			http.Error(w, "missing token", http.StatusBadRequest)
			return
		}
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		fmt.Fprint(w, `<!doctype html><html><body style="font-family: system-ui; display: flex; align-items: center; justify-content: center; height: 100vh; margin: 0">
<div style="text-align: center"><h2>✓ You're logged in</h2><p>Return to your terminal — you can close this tab.</p></div>
</body></html>`)
		// Flush before signaling: the main goroutine closes the server as
		// soon as it has the token, which would race the buffered response.
		if f, ok := w.(http.Flusher); ok {
			f.Flush()
		}
		select {
		case tokenCh <- token:
		default:
		}
	})}
	go server.Serve(listener)
	defer server.Close()

	authURL := fmt.Sprintf("%s/cli-auth?port=%d&state=%s",
		strings.TrimRight(frontendURL, "/"), port, url.QueryEscape(state))

	fmt.Printf("Opening your browser to authorize the CLI…\n  %s\n", authURL)
	if err := openBrowser(authURL); err != nil {
		fmt.Fprintln(os.Stderr, "Couldn't open a browser automatically — open the URL above manually.")
	}
	fmt.Println("Waiting for authorization…")

	select {
	case token := <-tokenCh:
		return token, nil
	case <-time.After(browserLoginTimeout):
		return "", errors.New("timed out waiting for the browser")
	case <-ctx.Done():
		return "", ctx.Err()
	}
}

func promptForToken(frontendURL string) (string, error) {
	fmt.Printf("Create a token at %s/settings/tokens\n", strings.TrimRight(frontendURL, "/"))
	fmt.Print("Paste your API token: ")

	if term.IsTerminal(int(os.Stdin.Fd())) {
		raw, err := term.ReadPassword(int(os.Stdin.Fd()))
		fmt.Println()
		if err != nil {
			return "", err
		}
		return strings.TrimSpace(string(raw)), nil
	}

	// Piped stdin (CI, scripts)
	var token string
	if _, err := fmt.Fscanln(os.Stdin, &token); err != nil {
		return "", errors.New("no token provided")
	}
	return strings.TrimSpace(token), nil
}

func openBrowser(u string) error {
	switch runtime.GOOS {
	case "darwin":
		return exec.Command("open", u).Start()
	case "windows":
		return exec.Command("rundll32", "url.dll,FileProtocolHandler", u).Start()
	default:
		return exec.Command("xdg-open", u).Start()
	}
}
