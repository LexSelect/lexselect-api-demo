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

			pairs := [][2]string{
				{"Document", fmt.Sprintf("%v", resp["document_name"])},
				{"Status", fmt.Sprintf("%v", resp["status"])},
				{"Progress", fmt.Sprintf("%v", resp["processing_progress"])},
				{"Pages", fmt.Sprintf("%v", resp["page_count"])},
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
