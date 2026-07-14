package generator

import "testing"

func TestLinkFaviconBaseURL(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		linkMap  map[string]any
		expected string
	}{
		{
			name: "uses url when url_internal is absent",
			linkMap: map[string]any{
				"url": "https://example.com",
			},
			expected: "https://example.com",
		},
		{
			name: "uses url_internal when set",
			linkMap: map[string]any{
				"url":          "https://grafana.example.com",
				"url_internal": "http://grafana.internal:3000",
			},
			expected: "http://grafana.internal:3000",
		},
		{
			name: "falls back to url when url_internal is empty",
			linkMap: map[string]any{
				"url":          "https://example.com",
				"url_internal": "",
			},
			expected: "https://example.com",
		},
		{
			name:     "returns empty when no url fields are set",
			linkMap:  map[string]any{},
			expected: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			if got := linkFaviconBaseURL(tt.linkMap); got != tt.expected {
				t.Fatalf("linkFaviconBaseURL() = %q, want %q", got, tt.expected)
			}
		})
	}
}

func TestNormalizeLinksDataURLs(t *testing.T) {
	t.Parallel()

	data := map[string]any{
		"categories": []any{
			map[string]any{
				"title": "tech",
				"links": []any{
					map[string]any{"url": "xkcd.com", "title": "XKCD"},
					map[string]any{"url": "https://github.com", "title": "GitHub"},
				},
			},
		},
		"links": []any{
			map[string]any{"url": "google.com", "title": "Google"},
		},
	}

	normalizeLinksDataURLs(data)

	categories := data["categories"].([]any)
	catLinks := categories[0].(map[string]any)["links"].([]any)
	if got := catLinks[0].(map[string]any)["url"]; got != "https://xkcd.com" {
		t.Fatalf("category link url = %q, want https://xkcd.com", got)
	}
	if got := catLinks[1].(map[string]any)["url"]; got != "https://github.com" {
		t.Fatalf("category link url = %q, want https://github.com", got)
	}

	flatLinks := data["links"].([]any)
	if got := flatLinks[0].(map[string]any)["url"]; got != "https://google.com" {
		t.Fatalf("flat link url = %q, want https://google.com", got)
	}
}
