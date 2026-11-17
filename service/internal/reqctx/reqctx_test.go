package reqctx

import (
	"context"
	"net/http/httptest"
	"testing"
)

func TestHttpRequestRoundTrip(t *testing.T) {
	r := httptest.NewRequest("GET", "/", nil)

	if got := HttpRequest(WithHttpRequest(context.Background(), r)); got != r {
		t.Fatalf("expected stored request back, got %v", got)
	}

	//nolint:staticcheck // SA1029: intentional untyped string key to prove typed-key isolation
	if got := HttpRequest(context.WithValue(context.Background(), "httpRequest", r)); got != nil {
		t.Fatal("untyped string key must not match")
	}
}
