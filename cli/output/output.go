package output

import (
	"encoding/json"
	"fmt"
	"mime"
	"path/filepath"
	"strings"
)

// JSON prints data as indented JSON.
func JSON(data interface{}) {
	out, _ := json.MarshalIndent(data, "", "  ")
	fmt.Println(string(out))
}

// Table prints a formatted table with headers and rows.
func Table(headers []string, rows [][]string) {
	if len(rows) == 0 {
		return
	}

	// Calculate column widths
	widths := make([]int, len(headers))
	for i, h := range headers {
		widths[i] = len(h)
	}
	for _, row := range rows {
		for i, cell := range row {
			if i < len(widths) && len(cell) > widths[i] {
				widths[i] = len(cell)
			}
		}
	}

	// Print header
	for i, h := range headers {
		fmt.Printf("%-*s  ", widths[i], h)
	}
	fmt.Println()
	totalWidth := 0
	for _, w := range widths {
		totalWidth += w + 2
	}
	fmt.Println(strings.Repeat("-", totalWidth))

	// Print rows
	for _, row := range rows {
		for i, cell := range row {
			if i < len(widths) {
				fmt.Printf("%-*s  ", widths[i], cell)
			}
		}
		fmt.Println()
	}
}

// KV prints key-value pairs aligned.
func KV(pairs [][2]string) {
	maxKey := 0
	for _, p := range pairs {
		if len(p[0]) > maxKey {
			maxKey = len(p[0])
		}
	}
	for _, p := range pairs {
		fmt.Printf("%-*s  %s\n", maxKey, p[0]+":", p[1])
	}
}

// Truncate shortens a string with ellipsis if it exceeds max length.
func Truncate(s string, max int) string {
	if len(s) <= max {
		return s
	}
	return s[:max-3] + "..."
}

// ContentType detects MIME type from file extension using the stdlib mime package.
func ContentType(filename string) string {
	ct := mime.TypeByExtension(filepath.Ext(filename))
	if ct == "" {
		return "application/octet-stream"
	}
	return ct
}
