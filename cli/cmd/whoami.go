package cmd

import (
	"fmt"

	"github.com/spf13/cobra"
)

var whoamiCmd = &cobra.Command{
	Use:   "whoami",
	Short: "Show the logged-in account, plan, and limits",
	Args:  cobra.NoArgs,
	RunE: func(cmd *cobra.Command, args []string) error {
		client, _, err := loadClient()
		if err != nil {
			return err
		}
		me, err := client.Me(cmd.Context())
		if err != nil {
			return friendlyAuthErr(err)
		}

		fmt.Println(me.User.Email)
		fmt.Printf("Plan: %s\n", me.Plan)
		fmt.Printf("Limits: %s named endpoints · %d requests/month · %d days history\n",
			formatLimit(me.Limits.MaxNamedEndpoints),
			me.Limits.MaxRequestsPerMonth,
			me.Limits.HistoryDays,
		)
		return nil
	},
}

func formatLimit(n int) string {
	if n < 0 {
		return "unlimited"
	}
	return fmt.Sprintf("%d", n)
}

func init() {
	rootCmd.AddCommand(whoamiCmd)
}
