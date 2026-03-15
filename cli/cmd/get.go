package cmd

import (
	"fmt"

	"github.com/spf13/cobra"

	"github.com/LexSelect/lexselect-api-demo/cli/api"
	"github.com/LexSelect/lexselect-api-demo/cli/output"
)

func init() {
	rootCmd.AddCommand(&cobra.Command{
		Use:   "get <document_id>",
		Short: "Get document details",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			client, err := api.New(cfg)
			if err != nil {
				return err
			}

			resp, err := client.Request(cmd.Context(), "GET", "/documents/"+args[0], nil)
			if err != nil {
				return err
			}

			if cfg.JSONOutput {
				output.JSON(resp)
				return nil
			}

			output.KV([][2]string{
				{"ID", fmt.Sprintf("%v", resp["id"])},
				{"Name", fmt.Sprintf("%v", resp["name"])},
				{"Type", fmt.Sprintf("%v", resp["type"])},
				{"Size", fmt.Sprintf("%v", resp["size"])},
				{"Parent", fmt.Sprintf("%v", resp["parent_id"])},
				{"Created", fmt.Sprintf("%v", resp["created_at"])},
				{"Deleted", fmt.Sprintf("%v", resp["deleted"])},
			})
			return nil
		},
	})
}
