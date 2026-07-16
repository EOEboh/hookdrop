// Package cmd defines the hookdrop CLI commands.
package cmd

import (
	"errors"
	"fmt"

	"github.com/EOEboh/hookdrop/cli/internal/api"
	"github.com/EOEboh/hookdrop/cli/internal/config"
	"github.com/spf13/cobra"
)

var rootCmd = &cobra.Command{
	Use:   "hookdrop",
	Short: "Stream and forward your hookdrop webhooks from the terminal",
	Long: `hookdrop streams webhooks captured at hookdrop.app into your terminal and
forwards each one to a local server the hosted backend can't reach directly —
a local webhook forwarder for developing against real webhook traffic.`,
	Example: `  hookdrop login                     # authenticate (opens your browser)
  hookdrop listen my-slug -f 3000    # stream + forward to localhost:3000
  hookdrop endpoints                 # list your endpoints`,
	SilenceUsage:  true,
	SilenceErrors: false,
}

var versionString = "dev"

var versionCmd = &cobra.Command{
	Use:   "version",
	Short: "Print the hookdrop CLI version",
	Args:  cobra.NoArgs,
	Run: func(cmd *cobra.Command, args []string) {
		fmt.Println("hookdrop " + versionString)
	},
}

func SetVersion(version, commit, date string) {
	versionString = fmt.Sprintf("%s (commit %s, built %s)", version, commit, date)
	rootCmd.Version = versionString
}

func init() {
	rootCmd.AddCommand(versionCmd)
}

func Execute() error {
	return rootCmd.Execute()
}

// loadClient builds an authenticated API client, or fails with the standard
// not-logged-in message.
func loadClient() (*api.Client, *config.Config, error) {
	cfg, err := config.Load()
	if err != nil {
		return nil, nil, err
	}
	if cfg.Token == "" {
		return nil, nil, errors.New("you're not logged in. Run 'hookdrop login' first")
	}
	return api.New(cfg.APIURL, cfg.Token), cfg, nil
}

// friendlyAuthErr rewrites sentinel API errors into actionable messages.
func friendlyAuthErr(err error) error {
	switch {
	case errors.Is(err, api.ErrUnauthorized):
		return errors.New("your token is invalid or was revoked. Run 'hookdrop login' again")
	case errors.Is(err, api.ErrPaymentRequired):
		return errors.New("your plan doesn't include this. Run 'hookdrop whoami' to see limits, or upgrade at " + config.DefaultFrontendURL + "/settings/billing")
	default:
		return err
	}
}
