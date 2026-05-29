package cmd

import (
	"fmt"

	"github.com/spf13/cobra"

	"github.com/LexSelect/lexselect-api-demo/cli/api"
	"github.com/LexSelect/lexselect-api-demo/cli/output"
)

func init() {
	listCmd := &cobra.Command{
		Use:   "list",
		Short: "List documents",
		RunE:  runList,
	}

	listCmd.Flags().Int("limit", 25, "Maximum items per page (1-100)")
	listCmd.Flags().String("sort", "", "Sort field: name, size, created_at, modified_at")
	listCmd.Flags().String("dir", "asc", "Sort direction: asc or desc")
	listCmd.Flags().String("type", "", "Filter by type: file, folder, project")
	listCmd.Flags().String("parent", "", "Parent folder ID")
	listCmd.Flags().String("cursor", "", "Pagination cursor from previous response")
	listCmd.Flags().Bool("flat", false, "Return a flat listing instead of one level of the tree")

	rootCmd.AddCommand(listCmd)
}

func runList(cmd *cobra.Command, args []string) error {
	client, err := api.New(cfg)
	if err != nil {
		return err
	}

	limit, _ := cmd.Flags().GetInt("limit")
	sortBy, _ := cmd.Flags().GetString("sort")
	sortDir, _ := cmd.Flags().GetString("dir")
	itemType, _ := cmd.Flags().GetString("type")
	parentID, _ := cmd.Flags().GetString("parent")
	cursor, _ := cmd.Flags().GetString("cursor")
	flat, _ := cmd.Flags().GetBool("flat")

	query := fmt.Sprintf("?limit=%d", limit)
	if flat {
		query += "&flat=true"
	}
	if sortBy != "" {
		query += "&sort_by=" + sortBy
	}
	if sortDir != "" {
		query += "&sort_direction=" + sortDir
	}
	if itemType != "" {
		query += "&type=" + itemType
	}
	if parentID != "" {
		query += "&parent_id=" + parentID
	}
	if cursor != "" {
		query += "&cursor=" + cursor
	}

	resp, err := client.Request(cmd.Context(), "GET", "/documents"+query, nil)
	if err != nil {
		return err
	}

	if cfg.JSONOutput {
		output.JSON(resp)
		return nil
	}

	items, _ := resp["items"].([]interface{})
	if len(items) == 0 {
		fmt.Println("No documents found.")
		return nil
	}

	rows := make([][]string, 0, len(items))
	for _, item := range items {
		doc, _ := item.(map[string]interface{})
		size := ""
		if s, ok := doc["size"].(float64); ok {
			size = fmt.Sprintf("%d", int(s))
		}
		created := ""
		if c, ok := doc["created_at"].(string); ok && len(c) > 10 {
			created = c[:10]
		}
		rows = append(rows, []string{
			fmt.Sprintf("%v", doc["id"]),
			output.Truncate(fmt.Sprintf("%v", doc["name"]), 30),
			fmt.Sprintf("%v", doc["type"]),
			size,
			created,
		})
	}

	output.Table([]string{"ID", "NAME", "TYPE", "SIZE", "CREATED"}, rows)

	if nextCursor, ok := resp["next_cursor"].(string); ok && nextCursor != "" {
		fmt.Printf("\nMore results available: --cursor %s\n", nextCursor)
	}
	return nil
}
