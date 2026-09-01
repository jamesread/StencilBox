package generator

import "testing"

func TestWebuiBuildConfigURL(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name       string
		base       string
		configName string
		want       string
	}{
		{
			name:       "root-relative when base is empty",
			base:       "",
			configName: "homepage",
			want:       "/webui/build-config/homepage",
		},
		{
			name:       "trims trailing slash on base",
			base:       "https://stencilbox.example.com/",
			configName: "homepage",
			want:       "https://stencilbox.example.com/webui/build-config/homepage",
		},
		{
			name:       "absolute base without trailing slash",
			base:       "https://stencilbox.example.com",
			configName: "homepage",
			want:       "https://stencilbox.example.com/webui/build-config/homepage",
		},
		{
			name:       "escapes spaces in config name",
			base:       "",
			configName: "My links homepage",
			want:       "/webui/build-config/My%20links%20homepage",
		},
		{
			name:       "supports reverse-proxy path prefix",
			base:       "https://example.com/stencilbox",
			configName: "homepage",
			want:       "https://example.com/stencilbox/webui/build-config/homepage",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			got := webuiBuildConfigURL(tt.base, tt.configName)
			if got != tt.want {
				t.Fatalf("webuiBuildConfigURL(%q, %q) = %q, want %q", tt.base, tt.configName, got, tt.want)
			}
		})
	}
}

func TestGetWebuiUrlBase(t *testing.T) {
	t.Setenv("STENCILBOX_WEBUI_URL_BASE", "https://stencilbox.example.com/")
	got := getWebuiUrlBase()
	want := "https://stencilbox.example.com"
	if got != want {
		t.Fatalf("getWebuiUrlBase() = %q, want %q", got, want)
	}
}

func TestGetWebuiUrlBaseUnset(t *testing.T) {
	t.Setenv("STENCILBOX_WEBUI_URL_BASE", "")
	got := getWebuiUrlBase()
	if got != "" {
		t.Fatalf("getWebuiUrlBase() = %q, want empty", got)
	}
}
