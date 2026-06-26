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
