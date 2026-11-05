package cmd

import "testing"

func TestFormatPages(t *testing.T) {
	cases := []struct {
		name string
		resp string
		want string
	}{
		{"final total", `{"pages_done": 12, "pages_total": 12, "total_known": true}`, "12/12"},
		{"growing total", `{"pages_done": 7, "pages_total": 12, "total_known": false}`, "7/12+ (total not final)"},
		{"no facts yet", `{"status": "in_progress"}`, "0/0+ (total not final)"},
	}
	for _, c := range cases {
		if got := formatPages(decode(t, c.resp)); got != c.want {
			t.Errorf("%s: got %q, want %q", c.name, got, c.want)
		}
	}
}
