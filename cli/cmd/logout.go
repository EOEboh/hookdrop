package cmd

import (
	"fmt"
	"strings"

	"github.com/EOEboh/hookdrop/cli/internal/config"
	"github.com/spf13/cobra"
)

var logoutCmd = &cobra.Command{
	Use:   "logout",
	Short: "Remove the stored API token from this machine",
	Args:  cobra.NoArgs,
	RunE: func(cmd *cobra.Command, args []string) error {
		cfg, err := config.Load()
		if err != nil {
			return err
		}
		if cfg.Token == "" {
			fmt.Println("You're not logged in.")
			return nil
		}
		cfg.Token = ""
		if err := config.Save(cfg); err != nil {
			return err
		}
		fmt.Printf("Logged out locally. To also revoke the token, visit %s/settings/tokens\n",
			strings.TrimRight(cfg.FrontendURL, "/"))
		return nil
	},
}

func init() {
	rootCmd.AddCommand(logoutCmd)
}
