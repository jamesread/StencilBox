package httpserver

import (
	"testing"
)

func TestNormalizeListenAddr(t *testing.T) {
	cases := []struct {
		in, want string
	}{
		{"8080", "0.0.0.0:8080"},
		{":8080", ":8080"},
		{"127.0.0.1:9000", "127.0.0.1:9000"},
	}

	for _, c := range cases {
		got := normalizeListenAddr(c.in)
		if got != c.want {
			t.Errorf("normalizeListenAddr(%q) = %q, want %q", c.in, got, c.want)
		}
	}
}

func TestGetHttpServerAddressPort(t *testing.T) {
	t.Setenv("PORT", "19090")
	t.Setenv("STENCILBOX_ADDRESS", "")
	got := getHttpServerAddress()
	if got != "0.0.0.0:19090" {
		t.Errorf("got %q, want 0.0.0.0:19090", got)
	}
}

func TestGetHttpServerAddressStencilBoxAddress(t *testing.T) {
	t.Setenv("PORT", "")
	t.Setenv("STENCILBOX_ADDRESS", "127.0.0.1:9999")
	got := getHttpServerAddress()
	if got != "127.0.0.1:9999" {
		t.Errorf("got %q, want 127.0.0.1:9999", got)
	}
}
