package cmd

import (
	"fmt"
	"net/url"

	"github.com/spf13/cobra"

	"github.com/LexSelect/lexselect-api-demo/cli/api"
)

func init() {
	renderCmd := &cobra.Command{
		Use:   "render <document_id>",
		Short: "Render a processed document to Markdown or HTML (experimental)",
		Long: "Render a processed document to Markdown or HTML.\n\n" +
			"EXPERIMENTAL: output format and rules may change. Markdown output may contain\n" +
			"inline HTML from the source document — sanitize it before rendering to HTML.",
		Args: cobra.ExactArgs(1),
		RunE: runRender,
	}
	renderCmd.Flags().String("format", "markdown", "Output format: markdown or html")
	renderCmd.Flags().String("pages", "", "1-indexed page ranges, e.g. '1-10', '1,3,5' (default: all)")
	renderCmd.Flags().String("exclude-node-types", "", "Node types to drop, e.g. 'header,footer'")
	rootCmd.AddCommand(renderCmd)
}

func runRender(cmd *cobra.Command, args []string) error {
	client, err := api.New(cfg)
	if err != nil {
		return err
	}

	format, _ := cmd.Flags().GetString("format")
	q := url.Values{}
	q.Set("format", format)
	if v, _ := cmd.Flags().GetString("pages"); v != "" {
		q.Set("pages", v)
	}
	if v, _ := cmd.Flags().GetString("exclude-node-types"); v != "" {
		q.Set("exclude_node_types", v)
	}

	path := "/documents/" + args[0] + "/render?" + q.Encode()

	// /render returns a raw Markdown/HTML body, not JSON.
	body, _, err := client.RequestRaw(cmd.Context(), "GET", path)
	if err != nil {
		return err
	}

	fmt.Print(string(body))
	if len(body) > 0 && body[len(body)-1] != '\n' {
		fmt.Println()
	}
	return nil
}
