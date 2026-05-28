package cmd

import (
	"fmt"
	"net/url"

	"github.com/spf13/cobra"

	"github.com/LexSelect/lexselect-api-demo/cli/api"
	"github.com/LexSelect/lexselect-api-demo/cli/output"
)

func init() {
	parseCmd := &cobra.Command{
		Use:   "parse <document_id>",
		Short: "Get the parsed structure of a processed document",
		Args:  cobra.ExactArgs(1),
		RunE:  runParse,
	}
	parseCmd.Flags().String("include", "", "Projections to include: tree,kvps,text,tables,blocks (default tree,kvps)")
	parseCmd.Flags().String("pages", "", "1-indexed page ranges, e.g. '1-10', '1,3,5' (default: all)")
	parseCmd.Flags().String("exclude-node-types", "", "Node types to drop, e.g. 'header,footer'")
	parseCmd.Flags().Bool("strip-geometry", false, "Drop per-node coordinates and geometry")
	rootCmd.AddCommand(parseCmd)
}

func runParse(cmd *cobra.Command, args []string) error {
	client, err := api.New(cfg)
	if err != nil {
		return err
	}

	q := url.Values{}
	if v, _ := cmd.Flags().GetString("include"); v != "" {
		q.Set("include", v)
	}
	if v, _ := cmd.Flags().GetString("pages"); v != "" {
		q.Set("pages", v)
	}
	if v, _ := cmd.Flags().GetString("exclude-node-types"); v != "" {
		q.Set("exclude_node_types", v)
	}
	if strip, _ := cmd.Flags().GetBool("strip-geometry"); strip {
		q.Set("strip_geometry", "true")
	}

	path := "/documents/" + args[0] + "/parse"
	if len(q) > 0 {
		path += "?" + q.Encode()
	}

	resp, err := client.Request(cmd.Context(), "GET", path, nil)
	if err != nil {
		return err
	}

	if cfg.JSONOutput {
		output.JSON(resp)
		return nil
	}

	output.KV([][2]string{
		{"Document", fmt.Sprintf("%v", resp["document_id"])},
		{"Status", fmt.Sprintf("%v", resp["status"])},
		{"Doc type", fmt.Sprintf("%v", resp["document_type"])},
		{"Total pages", fmt.Sprintf("%v", resp["total_pages"])},
		{"Pages returned", fmt.Sprintf("%v", resp["page_count"])},
	})
	fmt.Println("\nUse --json for the full structure (tree, kvps, text, tables, blocks).")
	return nil
}
