package cmd

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"

	"github.com/LexSelect/lexselect-api-demo/cli/config"
)

var cfg *config.Config

var rootCmd = &cobra.Command{
	Use:   "lexselect",
	Short: "LexSelect API CLI",
	Long:  "Command-line tool for the LexSelect External API.",
}

func init() {
	cfg = config.Load()

	rootCmd.PersistentFlags().StringVar(&cfg.APIKey, "api-key", cfg.APIKey, "API key (or LEXSELECT_API_KEY env)")
	rootCmd.PersistentFlags().StringVar(&cfg.APIURL, "api-url", cfg.APIURL, "API base URL (or LEXSELECT_API_URL env)")
	rootCmd.PersistentFlags().BoolVar(&cfg.JSONOutput, "json", false, "Output raw JSON")
}

func Execute() {
	if err := rootCmd.Execute(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
