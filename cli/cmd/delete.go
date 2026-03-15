package cmd

import (
	"fmt"

	"github.com/spf13/cobra"

	"github.com/LexSelect/lexselect-api-demo/cli/api"
	"github.com/LexSelect/lexselect-api-demo/cli/output"
)

func init() {
	rootCmd.AddCommand(&cobra.Command{
		Use:   "delete <document_id>",
		Short: "Delete a document",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			client, err := api.New(cfg)
			if err != nil {
				return err
			}

			resp, err := client.Request(cmd.Context(), "DELETE", "/documents/"+args[0], nil)
			if err != nil {
				return err
			}

			if cfg.JSONOutput {
				output.JSON(resp)
				return nil
			}

			fmt.Printf("Deleted: %v\n", resp["id"])
			return nil
		},
	})
}
