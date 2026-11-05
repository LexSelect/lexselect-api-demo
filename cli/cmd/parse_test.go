package cmd

import (
	"encoding/json"
	"strings"
	"testing"

	"github.com/spf13/cobra"
)

// decode parses a JSON fixture the way api.Client.Request does, so tests see
// the same map[string]interface{} / float64 shapes the commands do.
func decode(t *testing.T, raw string) map[string]interface{} {
	t.Helper()
	var out map[string]interface{}
	if err := json.Unmarshal([]byte(raw), &out); err != nil {
		t.Fatalf("decode fixture: %v", err)
	}
	return out
}

// parseFixture is a two-page 2026-10-20 /parse response requested with
// tokens=true: page 0 has a line with one struck and one underlined word (its
// TOKEN children also carry the per-word flags), page 1 a struck word nested
// deeper in the tree.
const parseFixture = `{
  "document_id": "doc_1",
  "status": "completed",
  "document_type": "contract",
  "total_pages": 2,
  "page_count": 2,
  "language": {"code": "en", "confidence": 0.981},
  "direction": "ltr",
  "transcript": {"line_numbered": false},
  "pages": [
    {"page_index": 0, "tree": {"class": "root", "content": [
      {"class": "line", "value": {"text": "this clause is void"},
       "meta": {"styling": {
         "struck_tokens": [{"index": 3, "text": "void"}],
         "underlined_tokens": [{"index": 1, "text": "clause"}]}},
       "content": [
         {"class": "token", "value": {"text": "clause"}, "meta": {"styling": {"underline": true}}, "content": []},
         {"class": "token", "value": {"text": "void"}, "meta": {"styling": {"strikethrough": true}}, "content": []}
       ]}
    ]}},
    {"page_index": 1, "tree": {"class": "root", "content": [
      {"class": "section", "content": [
        {"class": "line", "value": {"text": "draft only"},
         "meta": {"styling": {"struck_tokens": [{"index": 0, "text": "draft"}]}},
         "content": []}
      ]}
    ]}},
    {"page_index": 2, "tree": null}
  ]
}`

func TestCollectLineMarksReadsRollupsOnce(t *testing.T) {
	marks := collectLineMarks(decode(t, parseFixture))

	wantStruck := []lineMark{{Page: 0, Index: 3, Text: "void"}, {Page: 1, Index: 0, Text: "draft"}}
	if len(marks.Struck) != len(wantStruck) {
		t.Fatalf("struck = %+v, want %+v", marks.Struck, wantStruck)
	}
	for i, m := range wantStruck {
		if marks.Struck[i] != m {
			t.Errorf("struck[%d] = %+v, want %+v", i, marks.Struck[i], m)
		}
	}
	if len(marks.Underlined) != 1 || marks.Underlined[0] != (lineMark{Page: 0, Index: 1, Text: "clause"}) {
		t.Errorf("underlined = %+v, want one mark for %q on page 0", marks.Underlined, "clause")
	}
}

func TestCollectLineMarksOnOlderVersionIsEmpty(t *testing.T) {
	marks := collectLineMarks(decode(t, `{"pages": [{"page_index": 0, "tree": {"class": "root", "content": [
	  {"class": "line", "value": {"text": "no marks"}, "content": []}]}}]}`))
	if len(marks.Struck) != 0 || len(marks.Underlined) != 0 {
		t.Errorf("marks = %+v, want none", marks)
	}
}

func TestParseSummary(t *testing.T) {
	got := map[string]string{}
	for _, p := range parseSummary(decode(t, parseFixture)) {
		got[p[0]] = p[1]
	}
	want := map[string]string{
		"Document":         "doc_1",
		"Total pages":      "2",
		"Language":         "en (0.98, ltr)",
		"Line-numbered":    "false",
		"Struck words":     `2: "void" (p1), "draft" (p2)`,
		"Underlined words": `1: "clause" (p1)`,
	}
	for k, v := range want {
		if got[k] != v {
			t.Errorf("%s = %q, want %q", k, got[k], v)
		}
	}
}

func TestFormatLanguage(t *testing.T) {
	cases := []struct {
		name      string
		language  interface{}
		direction interface{}
		want      string
	}{
		{"null language", nil, "ltr", "unknown"},
		{"code only", map[string]interface{}{"code": "he"}, nil, "he"},
		{"full", map[string]interface{}{"code": "he", "confidence": 0.5}, "rtl", "he (0.50, rtl)"},
	}
	for _, c := range cases {
		if got := formatLanguage(c.language, c.direction); got != c.want {
			t.Errorf("%s: got %q, want %q", c.name, got, c.want)
		}
	}
}

func TestFormatMarksCapsTheList(t *testing.T) {
	if got := formatMarks(nil); got != "0" {
		t.Errorf("empty = %q, want %q", got, "0")
	}
	marks := make([]lineMark, maxListedMarks+3)
	for i := range marks {
		marks[i] = lineMark{Text: "w"}
	}
	got := formatMarks(marks)
	if !strings.HasPrefix(got, "13: ") || !strings.HasSuffix(got, ", … 3 more") {
		t.Errorf("capped = %q", got)
	}
	if n := strings.Count(got, `"w"`); n != maxListedMarks {
		t.Errorf("listed %d words, want %d", n, maxListedMarks)
	}
}

func TestParseQuery(t *testing.T) {
	cmd := &cobra.Command{}
	cmd.Flags().String("include", "", "")
	cmd.Flags().String("pages", "", "")
	cmd.Flags().String("exclude-node-types", "", "")
	cmd.Flags().Bool("strip-geometry", false, "")
	cmd.Flags().Bool("tokens", false, "")
	if err := cmd.Flags().Parse([]string{"--include", "tree", "--tokens", "--pages", "1-2"}); err != nil {
		t.Fatal(err)
	}
	if got, want := parseQuery(cmd).Encode(), "include=tree&pages=1-2&tokens=true"; got != want {
		t.Errorf("query = %q, want %q", got, want)
	}
}
