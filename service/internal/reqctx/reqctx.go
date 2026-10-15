// Package reqctx carries the originating *http.Request through a context, so API
// handlers can see what the auth middleware saw.
package reqctx

import (
	"context"
	"net/http"
)

type contextKey string

const httpRequestKey contextKey = "httpRequest"

// WithHttpRequest returns a copy of ctx that carries r.
func WithHttpRequest(ctx context.Context, r *http.Request) context.Context {
	return context.WithValue(ctx, httpRequestKey, r)
}

// HttpRequest returns the request stored by WithHttpRequest, or nil.
func HttpRequest(ctx context.Context) *http.Request {
	r, _ := ctx.Value(httpRequestKey).(*http.Request)
	return r
}
