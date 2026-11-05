package cmd

import (
	"fmt"
	"net/url"
	"strings"

	"github.com/spf13/cobra"

	"github.com/LexSelect/lexselect-api-demo/cli/api"
	"github.com/LexSelect/lexselect-api-demo/cli/output"
)

// maxListedMarks caps how many struck-through or underlined words the
// human-readable summary lists; --json carries every one.
const maxListedMarks = 10

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
	parseCmd.Flags().Bool("tokens", false, "Add a token node per word, with its coordinates and styling, to the tree")
	rootCmd.AddCommand(parseCmd)
}

// parseQuery builds the /parse query string from the command's flags.
func parseQuery(cmd *cobra.Command) url.Values {
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
	if tokens, _ := cmd.Flags().GetBool("tokens"); tokens {
		q.Set("tokens", "true")
	}
	return q
}

func runParse(cmd *cobra.Command, args []string) error {
	client, err := api.New(cfg)
	if err != nil {
		return err
	}

	path := "/documents/" + args[0] + "/parse"
	if q := parseQuery(cmd); len(q) > 0 {
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

	output.KV(parseSummary(resp))
	fmt.Println("\nUse --json for the full structure (tree, kvps, text, tables, blocks).")
	return nil
}

// parseSummary turns a /parse response into the key-value lines the
// human-readable output prints: the document's identity and page counts, its
// language, whether it is a line-numbered transcript, and its struck-through
// and underlined words.
func parseSummary(resp map[string]interface{}) [][2]string {
	pairs := [][2]string{
		{"Document", fmt.Sprintf("%v", resp["document_id"])},
		{"Status", fmt.Sprintf("%v", resp["status"])},
		{"Doc type", fmt.Sprintf("%v", resp["document_type"])},
		{"Total pages", fmt.Sprintf("%v", resp["total_pages"])},
		{"Pages returned", fmt.Sprintf("%v", resp["page_count"])},
		{"Language", formatLanguage(resp["language"], resp["direction"])},
	}
	if transcript, ok := resp["transcript"].(map[string]interface{}); ok {
		pairs = append(pairs, [2]string{"Line-numbered", fmt.Sprintf("%v", transcript["line_numbered"])})
	}

	marks := collectLineMarks(resp)
	pairs = append(pairs,
		[2]string{"Struck words", formatMarks(marks.Struck)},
		[2]string{"Underlined words", formatMarks(marks.Underlined)},
	)
	return pairs
}

// formatLanguage renders the document-level `language` ({code, confidence} or
// null) and `direction` as e.g. "en (0.98, ltr)", or "unknown" when the
// response carries no detected language.
func formatLanguage(language, direction interface{}) string {
	lang, ok := language.(map[string]interface{})
	if !ok {
		return "unknown"
	}
	code, _ := lang["code"].(string)
	if code == "" {
		return "unknown"
	}
	details := []string{}
	if conf, ok := lang["confidence"].(float64); ok {
		details = append(details, fmt.Sprintf("%.2f", conf))
	}
	if dir, ok := direction.(string); ok && dir != "" {
		details = append(details, dir)
	}
	if len(details) == 0 {
		return code
	}
	return code + " (" + strings.Join(details, ", ") + ")"
}

// formatMarks renders a list of marked words as a count followed by up to
// maxListedMarks of them, each tagged with its 1-indexed page, e.g.
// `2: "void" (p1), "draft" (p3)`.
func formatMarks(marks []lineMark) string {
	if len(marks) == 0 {
		return "0"
	}
	shown := marks
	if len(shown) > maxListedMarks {
		shown = shown[:maxListedMarks]
	}
	words := make([]string, len(shown))
	for i, m := range shown {
		words[i] = fmt.Sprintf("%q (p%d)", m.Text, m.Page+1)
	}
	out := fmt.Sprintf("%d: %s", len(marks), strings.Join(words, ", "))
	if len(marks) > len(shown) {
		out += fmt.Sprintf(", … %d more", len(marks)-len(shown))
	}
	return out
}
