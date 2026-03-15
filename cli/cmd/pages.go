package cmd

import (
	"fmt"

	"github.com/spf13/cobra"

	"github.com/LexSelect/lexselect-api-demo/cli/api"
	"github.com/LexSelect/lexselect-api-demo/cli/output"
)

func init() {
	pagesCmd := &cobra.Command{
		Use:   "pages <document_id>",
		Short: "Get page metadata",
		Args:  cobra.ExactArgs(1),
		RunE:  runPages,
	}

	pagesCmd.Flags().Int("limit", 50, "Maximum pages per response (1-100)")
	pagesCmd.Flags().String("cursor", "", "Pagination cursor")

	rootCmd.AddCommand(pagesCmd)
}

func runPages(cmd *cobra.Command, args []string) error {
	client, err := api.New(cfg)
	if err != nil {
		return err
	}

	limit, _ := cmd.Flags().GetInt("limit")
	cursor, _ := cmd.Flags().GetString("cursor")

	query := fmt.Sprintf("?limit=%d", limit)
	if cursor != "" {
		query += "&cursor=" + cursor
	}

	resp, err := client.Request(cmd.Context(), "GET", "/documents/"+args[0]+"/processing/latest/pages"+query, nil)
	if err != nil {
		return err
	}

	if cfg.JSONOutput {
		output.JSON(resp)
		return nil
	}

	total, _ := resp["total"].(float64)
	fmt.Printf("Total pages: %d\n\n", int(total))

	pages, _ := resp["pages"].([]interface{})
	if len(pages) == 0 {
		return nil
	}

	rows := make([][]string, 0, len(pages))
	for _, p := range pages {
		page, _ := p.(map[string]interface{})
		idx, _ := page["page_index"].(float64)
		w, _ := page["width"].(float64)
		h, _ := page["height"].(float64)
		rows = append(rows, []string{
			fmt.Sprintf("%d", int(idx)),
			fmt.Sprintf("%.1f", w),
			fmt.Sprintf("%.1f", h),
			fmt.Sprintf("%v", page["page_type"]),
		})
	}

	output.Table([]string{"INDEX", "WIDTH", "HEIGHT", "TYPE"}, rows)

	if nextCursor, ok := resp["next_cursor"].(string); ok && nextCursor != "" {
		fmt.Printf("\nMore pages available: --cursor %s\n", nextCursor)
	}
	return nil
}
