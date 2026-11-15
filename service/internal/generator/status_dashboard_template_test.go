package generator

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gopkg.in/yaml.v3"
)

// fixNow pins the build time used for due dates.
func fixNow(t *testing.T, at time.Time) {
	t.Helper()
	old := now
	now = func() time.Time { return at }
	t.Cleanup(func() { now = old })
}

// renderStatusDashboard renders the built-in status-dashboard template with the real template
// functions, the way a build does.
func renderStatusDashboard(t *testing.T, dashboard any) string {
	t.Helper()
	path := filepath.Join("..", "..", "..", "templates", "status-dashboard", "index.html")
	tmpl, err := getNewTemplater(t.TempDir()).ParseFiles(path)
	require.NoError(t, err)

	var buf bytes.Buffer
	require.NoError(t, tmpl.Execute(&buf, map[string]any{
		"dashboard":          dashboard,
		"hooks":              map[string]string{"head": "", "body": ""},
		"buildDateFormatted": "November 16, 2026 at 8:01 AM UTC",
	}))
	return buf.String()
}

func loadYAML(t *testing.T, path string) any {
	t.Helper()
	raw, err := os.ReadFile(path)
	require.NoError(t, err)
	var v any
	require.NoError(t, yaml.Unmarshal(raw, &v))
	return v
}

// tile returns the number shown on a summary tile.
func tile(t *testing.T, out, label string) string {
	t.Helper()
	i := strings.Index(out, `<span class="subtle">`+label+`</span>`)
	require.GreaterOrEqual(t, i, 0, label)
	before := out[:i]
	start := strings.LastIndex(before, `<span class="stat">`) + len(`<span class="stat">`)
	return before[start:strings.LastIndex(before, "</span>")]
}

func TestStatusDashboardExample(t *testing.T) {
	fixNow(t, time.Date(2026, 11, 16, 8, 0, 0, 0, time.UTC))
	data := loadYAML(t, filepath.Join("..", "..", "..", "var", "config-skel", "buildconfigs", "data", "status-dashboard.yaml"))
	out := renderStatusDashboard(t, data)

	assert.Contains(t, out, "<title>My dashboard</title>")
	assert.Equal(t, "11", tile(t, out, "Need attention"))
	assert.Equal(t, "1", tile(t, out, "Urgent"))
	assert.Equal(t, "1", tile(t, out, "Overdue"), "only the 14 Nov reply is overdue")
	assert.Equal(t, "2", tile(t, out, "Waiting or blocked"))

	assert.Contains(t, out, "Overdue by 2 days")
	assert.Contains(t, out, "Due today")
	assert.Contains(t, out, "Due in 4 days")
	assert.Contains(t, out, "Due 1 Dec")
	assert.NotContains(t, out, "<summary>Done", "the example has no done items")
	assert.Contains(t, out, `<span class="tag bad">Blocked</span>`)
	assert.Contains(t, out, `<span class="annotation warning">`, "CI's sync status")
	assert.Contains(t, out, `<link rel="stylesheet" href="femtocrank.css" />`)

	// Within the Work column, the urgent item comes before the high ones, then normal.
	work := out[strings.Index(out, "Work</h2>"):strings.Index(out, "Side projects</h2>")]
	deploy := strings.Index(work, "Production deploy failed")
	review := strings.Index(work, "Review PR #482")
	meeting := strings.Index(work, "1:1 with Alex")
	assert.True(t, deploy >= 0 && deploy < review && review < meeting, "urgent, then high, then normal")
}

func TestStatusDashboardEmptyAndUncategorised(t *testing.T) {
	out := renderStatusDashboard(t, map[string]any{"items": []any{}})
	assert.Contains(t, out, "Nothing needs your attention right now.")

	out = renderStatusDashboard(t, map[string]any{
		"categories": []any{map[string]any{"id": "work", "name": "Work"}},
		"items": []any{
			map[string]any{"title": "Stray item", "category": "unknown"},
			map[string]any{"title": "No category"},
			map[string]any{"title": "Odd priority", "category": "work", "priority": "whenever"},
		},
	})
	assert.Contains(t, out, "Other</h2>")
	assert.Contains(t, out, "Stray item")
	assert.Contains(t, out, "No category")
	assert.Contains(t, out, "Odd priority", "unknown priorities are shown as normal, not dropped")

	out = renderStatusDashboard(t, map[string]any{
		"categories": []any{map[string]any{"id": "work", "name": "Work"}},
		"items": []any{
			map[string]any{"title": "Still open", "category": "work"},
			map[string]any{"title": "Finished thing", "category": "work", "state": "done"},
		},
	})
	assert.Contains(t, out, "<summary>Done (1)</summary>", "done items are folded away")
	work := out[strings.Index(out, "Work</h2>"):strings.Index(out, "<summary>")]
	assert.NotContains(t, work, "Finished thing", "done items are not in the columns")
}

// Data files are editable through the API and MCP: values must be escaped, links must not run
// script, and source colours cannot inject CSS.
func TestStatusDashboardEscapesData(t *testing.T) {
	out := renderStatusDashboard(t, map[string]any{
		"title":    `<script>alert("title")</script>`,
		"greeting": `<img src=x onerror=alert(1)>`,
		"sources": []any{map[string]any{
			"id": "s", "name": "<b>src</b>", "url": "javascript:alert(2)",
			"color": `x" style="background:url(//evil)`, "status": `ok"><script>alert(3)</script>`,
		}},
		"categories": []any{map[string]any{"id": "c", "name": "<i>cat</i>"}},
		"items": []any{map[string]any{
			"title": "<script>alert(4)</script>", "source": "s", "category": "c",
			"url": " JAVASCRIPT:alert(5)", "note": "<iframe src=//evil>", "tags": []any{"<u>t</u>"},
			"priority": `urgent"><script>alert(6)</script>`,
		}},
	})
	for _, bad := range []string{"<script>alert", "<img src=x", "<iframe", "<b>src", "<i>cat", "<u>t", `style="background`, "javascript:", "JAVASCRIPT:"} {
		assert.NotContains(t, out, bad)
	}
	assert.Contains(t, out, "&lt;script&gt;")
}
