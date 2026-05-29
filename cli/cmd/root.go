package cmd

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"

	"github.com/LexSelect/lexselect-api-demo/cli/config"
)

var cfg *config.Config

// Flag-backing vars are kept separate from cfg so the loaded API key is never
// used as a flag default — otherwise Cobra prints it verbatim in `--help` and
// on every usage error, leaking the secret. Empty flags fall back to env/.env.
var (
	apiKeyFlag string
	apiURLFlag string
)

var rootCmd = &cobra.Command{
	Use:   "lexselect",
	Short: "LexSelect API CLI",
	Long:  "Command-line tool for the LexSelect External API.",
	PersistentPreRun: func(cmd *cobra.Command, args []string) {
		if apiKeyFlag != "" {
			cfg.APIKey = apiKeyFlag
		}
		if apiURLFlag != "" {
			cfg.APIURL = apiURLFlag
		}
	},
}

func init() {
	cfg = config.Load()

	rootCmd.PersistentFlags().StringVar(&apiKeyFlag, "api-key", "", "API key (overrides LEXSELECT_API_KEY env)")
	rootCmd.PersistentFlags().StringVar(&apiURLFlag, "api-url", "", "API base URL (overrides LEXSELECT_API_URL env)")
	rootCmd.PersistentFlags().BoolVar(&cfg.JSONOutput, "json", false, "Output raw JSON")
}

func Execute() {
	if err := rootCmd.Execute(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
