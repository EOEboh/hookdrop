package cmd

import (
	"fmt"
	"os"
	"strings"
	"text/tabwriter"

	"github.com/spf13/cobra"
)

var endpointsCmd = &cobra.Command{
	Use:   "endpoints",
	Short: "List your named endpoints (use a slug with 'hookdrop listen --endpoint')",
	Args:  cobra.NoArgs,
	RunE: func(cmd *cobra.Command, args []string) error {
		client, cfg, err := loadClient()
		if err != nil {
			return err
		}
		endpoints, err := client.Endpoints(cmd.Context())
		if err != nil {
			return friendlyAuthErr(err)
		}

		if len(endpoints) == 0 {
			fmt.Printf("No named endpoints yet. Create one at %s\n", strings.TrimRight(cfg.FrontendURL, "/"))
			return nil
		}

		w := tabwriter.NewWriter(os.Stdout, 0, 4, 2, ' ', 0)
		fmt.Fprintln(w, "SLUG\tNAME\tINBOX URL")
		for _, ep := range endpoints {
			fmt.Fprintf(w, "%s\t%s\t%s/i/%s\n", ep.Slug, ep.Name, strings.TrimRight(cfg.APIURL, "/"), ep.Slug)
		}
		return w.Flush()
	},
}

func init() {
	rootCmd.AddCommand(endpointsCmd)
}
