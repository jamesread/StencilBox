package generator

import (
	"fmt"
	"net/url"
	"strings"
	"time"
)

// dateTimeLayout is how formatDateTime shows a date and time, eg "15 Nov 2026, 09:30 UTC".
const dateTimeLayout = "2 Jan 2006, 15:04 MST"

// formatDateTime formats a date from a data file for display. YAML turns unquoted timestamps
// into time.Time and keeps quoted ones as strings, so both are accepted. Values that are not a
// recognisable date are returned as they are.
func formatDateTime(v any) string {
	switch t := v.(type) {
	case nil:
		return ""
	case time.Time:
		return formatTime(t)
	case string:
		if parsed, ok := parseDate(t); ok {
			return formatTime(parsed)
		}
		return t
	default:
		return fmt.Sprint(v)
	}
}

func formatTime(t time.Time) string {
	if t.Hour() == 0 && t.Minute() == 0 && t.Second() == 0 {
		return t.Format("2 Jan 2006")
	}
	return t.Format(dateTimeLayout)
}

// safeURL returns a link target that cannot run script: http, https and mailto URLs, and
// relative links, are kept; anything else (eg javascript:) becomes "#". The result still needs
// HTML escaping, eg {{ .url | safeURL | html }}.
func safeURL(v any) string {
	s := strings.TrimSpace(fmt.Sprint(v))
	if s == "" || v == nil {
		return "#"
	}
	u, err := url.Parse(s)
	if err != nil {
		return "#"
	}
	switch strings.ToLower(u.Scheme) {
	case "", "http", "https", "mailto":
		return s
	}
	return "#"
}

// now is the build time used for due dates; a variable so tests can fix it.
var now = time.Now

// list returns its arguments as a list, eg {{ range list "urgent" "high" }}.
func list(values ...any) []any {
	return values
}

// dict builds a map from key/value pairs, to pass several values to a sub-template, eg
// {{ template "card" (dict "item" . "sources" $sources) }}.
func dict(pairs ...any) (map[string]any, error) {
	if len(pairs)%2 != 0 {
		return nil, fmt.Errorf("dict needs key/value pairs, got %d arguments", len(pairs))
	}
	out := make(map[string]any, len(pairs)/2)
	for i := 0; i < len(pairs); i += 2 {
		out[fmt.Sprint(pairs[i])] = pairs[i+1]
	}
	return out, nil
}

// field reads key from a map in a data file list ("" when missing), compared as text so that
// YAML's numbers, booleans and strings all match what a template writes.
func field(item any, key string) string {
	m, ok := item.(map[string]any)
	if !ok || m[key] == nil {
		return ""
	}
	return fmt.Sprint(m[key])
}

func asList(v any) []any {
	l, _ := v.([]any)
	return l
}

// where keeps the items whose key equals value, eg {{ where .items "category" "work" }}.
func where(items any, key string, value any) []any {
	return filter(items, func(item any) bool { return field(item, key) == fmt.Sprint(value) })
}

// whereNot keeps the items whose key does not equal value.
func whereNot(items any, key string, value any) []any {
	return filter(items, func(item any) bool { return field(item, key) != fmt.Sprint(value) })
}

// whereIn keeps the items whose key is one of values; whereIn false keeps the rest, eg items
// whose category is not a known category.
func whereIn(items any, key string, values any, in bool) []any {
	set := map[string]bool{}
	for _, v := range asList(values) {
		set[fmt.Sprint(v)] = true
	}
	return filter(items, func(item any) bool { return set[field(item, key)] == in })
}

// pluck lists one key of every item, eg {{ pluck .categories "id" }}.
func pluck(items any, key string) []any {
	out := []any{}
	for _, item := range asList(items) {
		out = append(out, field(item, key))
	}
	return out
}

func filter(items any, keep func(any) bool) []any {
	out := []any{}
	for _, item := range asList(items) {
		if keep(item) {
			out = append(out, item)
		}
	}
	return out
}

// parseDate reads a date from a data file (time.Time, or a string in a common layout).
func parseDate(v any) (time.Time, bool) {
	switch t := v.(type) {
	case time.Time:
		return t, true
	case string:
		for _, layout := range []string{time.RFC3339, "2006-01-02 15:04:05", "2006-01-02 15:04", "2006-01-02"} {
			if parsed, err := time.Parse(layout, strings.TrimSpace(t)); err == nil {
				return parsed, true
			}
		}
	}
	return time.Time{}, false
}

// daysUntil counts calendar days from the build day to the due day (negative when overdue).
func daysUntil(due time.Time) int {
	n := now()
	today := time.Date(n.Year(), n.Month(), n.Day(), 0, 0, 0, 0, time.UTC)
	day := time.Date(due.Year(), due.Month(), due.Day(), 0, 0, 0, 0, time.UTC)
	return int(day.Sub(today).Hours() / 24)
}

// overdue keeps the items whose due date (key "due") is before the build day.
func overdue(items any) []any {
	return filter(items, func(item any) bool {
		m, _ := item.(map[string]any)
		due, ok := parseDate(m["due"])
		return ok && daysUntil(due) < 0
	})
}

// dueLabel describes a due date relative to the build, eg "Overdue by 2 days", "Due today",
// "Due in 3 days" or "Due 2 Dec". It returns "" for no date.
func dueLabel(v any) string {
	due, ok := parseDate(v)
	if !ok {
		if v == nil {
			return ""
		}
		return "Due " + fmt.Sprint(v)
	}
	switch d := daysUntil(due); {
	case d < -1:
		return fmt.Sprintf("Overdue by %d days", -d)
	case d == -1:
		return "Overdue by 1 day"
	case d == 0:
		return "Due today"
	case d == 1:
		return "Due tomorrow"
	case d <= 6:
		return fmt.Sprintf("Due in %d days", d)
	default:
		return "Due " + due.Format("2 Jan")
	}
}

// isOverdue reports whether a due date is before the build day.
func isOverdue(v any) bool {
	due, ok := parseDate(v)
	return ok && daysUntil(due) < 0
}
