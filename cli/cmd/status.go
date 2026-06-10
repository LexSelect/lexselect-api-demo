package cmd

import (
	"fmt"

	"github.com/spf13/cobra"

	"github.com/LexSelect/lexselect-api-demo/cli/api"
	"github.com/LexSelect/lexselect-api-demo/cli/output"
)

func init() {
	rootCmd.AddCommand(&cobra.Command{
		Use:   "status <document_id>",
		Short: "Check processing status",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			client, err := api.New(cfg)
			if err != nil {
				return err
			}

			resp, err := client.Request(cmd.Context(), "GET", "/documents/"+args[0]+"/processing/latest", nil)
			if err != nil {
				return err
			}

			if cfg.JSONOutput {
				output.JSON(resp)
				return nil
			}

			pagesDone, _ := resp["pages_done"].(float64)
			pagesTotal, _ := resp["pages_total"].(float64)
			totalKnown, _ := resp["total_known"].(bool)
			// When the total is not final yet, mark it with "+" — the total
			// may still grow, so never present it as a finished fraction.
			pages := fmt.Sprintf("%d/%d", int(pagesDone), int(pagesTotal))
			if !totalKnown {
				pages += "+ (total not final)"
			}

			pairs := [][2]string{
				{"Document", fmt.Sprintf("%v", resp["document_name"])},
				{"Status", fmt.Sprintf("%v", resp["status"])},
				{"Stage", fmt.Sprintf("%v", resp["stage"])},
				{"Pages", pages},
				{"Preview", fmt.Sprintf("%v", resp["preview_available"])},
				{"Engine", fmt.Sprintf("v%v", resp["engine_version"])},
			}
			if errMsg, ok := resp["error_message"].(string); ok && errMsg != "" {
				pairs = append(pairs, [2]string{"Error", errMsg})
			}
			output.KV(pairs)
			return nil
		},
	})
}
