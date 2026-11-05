package cmd

// lineMark is one struck-through or underlined word reported by /parse.
type lineMark struct {
	Page  int    // zero-based page_index of the page the word is on
	Index int    // the word's position in its line's word sequence
	Text  string // the word as it appears in the line's value.text
}

// lineMarks holds the struck-through and underlined words of a /parse
// response, in reading order.
type lineMarks struct {
	Struck     []lineMark
	Underlined []lineMark
}

// collectLineMarks walks every page tree of a /parse response and gathers the
// line-level `meta.styling.struck_tokens` / `underlined_tokens` rollups that
// API version 2026-10-20 adds.
//
// It reads the rollups rather than the per-TOKEN `strikethrough`/`underline`
// flags because the server stamps the rollups on every version-2026-10-20
// response, with or without `tokens=true`, so a word is counted exactly once
// whichever way the tree was requested.
func collectLineMarks(resp map[string]interface{}) lineMarks {
	var marks lineMarks
	pages, _ := resp["pages"].([]interface{})
	for _, p := range pages {
		page, ok := p.(map[string]interface{})
		if !ok {
			continue
		}
		pageIndex, _ := page["page_index"].(float64)
		walkLineMarks(page["tree"], int(pageIndex), &marks)
	}
	return marks
}

// walkLineMarks visits a tree node and its `content` children depth-first —
// the order the API gives reading order in — appending each rollup it meets.
func walkLineMarks(n interface{}, page int, marks *lineMarks) {
	node, ok := n.(map[string]interface{})
	if !ok {
		return
	}
	if meta, ok := node["meta"].(map[string]interface{}); ok {
		if styling, ok := meta["styling"].(map[string]interface{}); ok {
			marks.Struck = appendMarks(marks.Struck, styling["struck_tokens"], page)
			marks.Underlined = appendMarks(marks.Underlined, styling["underlined_tokens"], page)
		}
	}
	children, _ := node["content"].([]interface{})
	for _, child := range children {
		walkLineMarks(child, page, marks)
	}
}

// appendMarks decodes a `[{index, text}]` rollup into lineMarks, skipping
// entries that are not objects.
func appendMarks(dst []lineMark, raw interface{}, page int) []lineMark {
	items, _ := raw.([]interface{})
	for _, it := range items {
		entry, ok := it.(map[string]interface{})
		if !ok {
			continue
		}
		index, _ := entry["index"].(float64)
		text, _ := entry["text"].(string)
		dst = append(dst, lineMark{Page: page, Index: int(index), Text: text})
	}
	return dst
}
