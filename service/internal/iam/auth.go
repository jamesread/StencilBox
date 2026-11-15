// Package iam wires armature-iam (database users, group RBAC, httpauthshim) into StencilBox.
package iam

import (
	"net/http"
	"os"
	"strings"

	clientapiconnect "github.com/jamesread/StencilBox/gen/StencilBox/clientapi/v1/clientapi_pbconnect"
	armlayer "github.com/jamesread/armature-iam/layer"
	"github.com/jamesread/armature-iam/store"
	log "github.com/sirupsen/logrus"
)

// CookieName is the browser session cookie; its value is a row in the sessions table.
const CookieName = "stencilbox-sid"

// APIKeyPrefix marks Bearer API keys, so they are never mistaken for session IDs.
const APIKeyPrefix = "sbk_"

// AllowUnauthenticated lists procedures guests may call. Their handlers must cope with no user.
var AllowUnauthenticated = []string{
	clientapiconnect.StencilBoxApiServiceGetStatusProcedure,
	clientapiconnect.StencilBoxApiServiceGetCurrentUserProcedure,
	clientapiconnect.StencilBoxApiServiceLoginWithUsernameAndPasswordProcedure,
	clientapiconnect.StencilBoxApiServiceLogoutProcedure,
}

// AuthLayer is the Connect authentication middleware plus the IAM store.
type AuthLayer struct {
	*armlayer.Layer
	Store store.Store

	// secureCookies is "true", "false" or "" (decide per request from the scheme).
	secureCookies string
}

// NewAuthLayer wires armature-iam Connect auth. authYAML is the config.yaml "auth" block
// (JWT, trusted headers, mTLS); JWT users are auto-created on first login.
func NewAuthLayer(st store.Store, authYAML map[string]any) (*AuthLayer, error) {
	devDisable := os.Getenv("STENCILBOX_DEV_DISABLE_AUTH") == "true"
	if devDisable {
		log.Warn("STENCILBOX_DEV_DISABLE_AUTH=true: every request runs as an anonymous superuser. Never use this in production.")
	}

	inner, err := armlayer.New(st, armlayer.Config{
		CookieName:           CookieName,
		APIKeyPrefix:         APIKeyPrefix,
		AuthYAML:             authYAML,
		Logger:               log.StandardLogger(),
		AllowUnauthenticated: AllowUnauthenticated,
		RequiredPermission:   RequiredPermission,
		DevDisableAuth:       devDisable,
		SecureCookies:        true,
	})
	if err != nil {
		return nil, err
	}

	return &AuthLayer{
		Layer:         inner,
		Store:         st,
		secureCookies: strings.ToLower(strings.TrimSpace(os.Getenv("STENCILBOX_SECURE_COOKIES"))),
	}, nil
}

// SessionCookie returns the Set-Cookie for a new session. The cookie is Secure when the request
// arrived over HTTPS (directly or via a proxy setting X-Forwarded-Proto), unless
// STENCILBOX_SECURE_COOKIES forces it on ("true") or off ("false").
func (a *AuthLayer) SessionCookie(sid string, r *http.Request, header http.Header) http.Cookie {
	c := a.NewSessionCookie(sid)
	c.Secure = a.secure(r, header)
	return c
}

// ClearCookie returns the Set-Cookie that removes the session cookie.
func (a *AuthLayer) ClearCookie(r *http.Request, header http.Header) http.Cookie {
	c := a.ClearSessionCookie()
	c.Secure = a.secure(r, header)
	return c
}

func (a *AuthLayer) secure(r *http.Request, header http.Header) bool {
	switch a.secureCookies {
	case "true":
		return true
	case "false":
		return false
	}
	if r != nil && r.TLS != nil {
		return true
	}
	return strings.EqualFold(strings.TrimSpace(header.Get("X-Forwarded-Proto")), "https")
}
