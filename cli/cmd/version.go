package cmd

import (
	"fmt"

	"github.com/spf13/cobra"

	"github.com/LexSelect/lexselect-api-demo/cli/config"
)

func init() {
	rootCmd.AddCommand(&cobra.Command{
		Use:   "version",
		Short: "Show CLI and API version",
		Run: func(cmd *cobra.Command, args []string) {
			fmt.Printf("lexselect CLI %s\n", config.CLIVersion)
			fmt.Printf("API version:  %s\n", config.APIVersion)
			fmt.Printf("API URL:      %s\n", cfg.APIURL)
		},
	})
}
