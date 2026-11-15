package generator

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestFormatDateTime(t *testing.T) {
	when := time.Date(2026, 11, 15, 9, 30, 0, 0, time.UTC)
	assert.Equal(t, "15 Nov 2026, 09:30 UTC", formatDateTime(when))
	assert.Equal(t, "15 Nov 2026, 09:30 UTC", formatDateTime("2026-11-15T09:30:00Z"))
	assert.Equal(t, "15 Nov 2026", formatDateTime("2026-11-15"))
	assert.Equal(t, "Tuesday morning", formatDateTime("Tuesday morning"), "free text is kept")
	assert.Equal(t, "", formatDateTime(nil))
}

func TestSafeURL(t *testing.T) {
	for _, ok := range []string{"https://example.com/a?b=c", "http://x", "mailto:ops@example.com", "/status", "page.html"} {
		assert.Equal(t, ok, safeURL(ok))
	}
	for _, bad := range []any{"javascript:alert(1)", " JavaScript:alert(1)", "data:text/html,x", "", nil} {
		assert.Equal(t, "#", safeURL(bad), bad)
	}
}

func TestListHelpers(t *testing.T) {
	items := []any{
		map[string]any{"id": 1, "cat": "work", "state": "done"},
		map[string]any{"id": 2, "cat": "home"},
		map[string]any{"id": 3, "cat": "work"},
		"not a map",
	}
	assert.Len(t, where(items, "cat", "work"), 2)
	assert.Len(t, whereNot(items, "state", "done"), 3, "missing keys are not equal")
	assert.Len(t, where(items, "id", 2), 1, "numbers compare as text")
	assert.Len(t, whereIn(items, "cat", list("work", "home"), true), 3)
	assert.Len(t, whereIn(items, "cat", list("work", "home"), false), 1, "the non-map item")
	assert.Equal(t, []any{"1", "2", "3", ""}, pluck(items, "id"))
	assert.Empty(t, where(nil, "x", "y"))

	m, err := dict("a", 1, "b", "two")
	assert.NoError(t, err)
	assert.Equal(t, map[string]any{"a": 1, "b": "two"}, m)
	_, err = dict("odd")
	assert.Error(t, err)
}

func TestDueDates(t *testing.T) {
	old := now
	now = func() time.Time { return time.Date(2026, 11, 16, 23, 30, 0, 0, time.UTC) }
	t.Cleanup(func() { now = old })

	assert.Equal(t, "Overdue by 2 days", dueLabel("2026-11-14"))
	assert.Equal(t, "Overdue by 1 day", dueLabel(time.Date(2026, 11, 15, 9, 0, 0, 0, time.UTC)))
	assert.Equal(t, "Due today", dueLabel("2026-11-16T08:00:00Z"), "earlier the same day is still today")
	assert.Equal(t, "Due tomorrow", dueLabel("2026-11-17"))
	assert.Equal(t, "Due in 6 days", dueLabel("2026-11-22"))
	assert.Equal(t, "Due 30 Nov", dueLabel("2026-11-30"))
	assert.Equal(t, "Due next week", dueLabel("next week"), "free text is kept")
	assert.Equal(t, "", dueLabel(nil))

	assert.True(t, isOverdue("2026-11-15"))
	assert.False(t, isOverdue("2026-11-16"))
	assert.Len(t, overdue([]any{
		map[string]any{"due": "2026-11-01"},
		map[string]any{"due": "2026-12-01"},
		map[string]any{},
	}), 1)
}
