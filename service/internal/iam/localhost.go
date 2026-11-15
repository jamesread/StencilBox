package iam

import (
	"context"
	"encoding/json"
	"net"
	"net/http"

	"connectrpc.com/authn"
	armlayer "github.com/jamesread/armature-iam/layer"
	"github.com/jamesread/armature-iam/rbac"
	"github.com/jamesread/armature-iam/store"
	log "github.com/sirupsen/logrus"
)

// Authenticator wraps the Connect API and the MCP endpoint with authentication. AuthLayer
// (database users) and LocalhostAuth (no database) both implement it.
type Authenticator interface {
	WrapHandler(http.Handler) http.Handler
	WrapMCPHandler(http.Handler) http.Handler
}

// LocalhostUsername is the user that requests from localhost run as without a database.
const LocalhostUsername = "localhost"

// localhostPermissions is everything that works without a database. IAM, user accounts and
// API keys need the database, so they are left out (and hidden in the web UI).
var localhostPermissions = []string{
	rbac.PermissionAppAccess,
	PermissionBuildConfigsView,
	PermissionBuildsRun,
	PermissionBuildConfigsGitPull,
	PermissionDataFilesEdit,
	PermissionBuildConfigsManage,
}

// forwardingHeaders mark a request that came through a proxy. On the same host a proxy makes
// every client look like localhost, so such requests are never trusted as local.
var forwardingHeaders = []string{"X-Forwarded-For", "X-Real-Ip", "Forwarded", "X-Forwarded-Host"}

// IsLocalRequest reports whether a request came straight from a loopback address, not through a
// proxy.
func IsLocalRequest(r *http.Request) bool {
	for _, h := range forwardingHeaders {
		if r.Header.Get(h) != "" {
			return false
		}
	}
	host, _, err := net.SplitHostPort(r.RemoteAddr)
	if err != nil {
		host = r.RemoteAddr
	}
	ip := net.ParseIP(host)
	return ip != nil && ip.IsLoopback()
}

// LocalhostAuth authenticates without a database: requests from localhost run as the
// "localhost" user, everyone else is refused (except the allow-listed status procedures).
type LocalhostAuth struct{}

func localhostUser() *armlayer.AuthenticatedUser {
	perms := map[string]bool{}
	for _, p := range localhostPermissions {
		perms[p] = true
	}
	return &armlayer.AuthenticatedUser{
		User: &store.UserAccountRow{ID: 0, Username: LocalhostUsername, CreatedBy: "local"},
		RBAC: &rbac.EffectiveRBAC{Permissions: perms},
	}
}

var errLocalOnly = "StencilBox is running without a database, so it only accepts requests made directly from localhost"

// Handle is the Connect authn function.
func (LocalhostAuth) Handle(_ context.Context, req *http.Request) (any, error) {
	procedure, _ := authn.InferProcedure(req.URL)
	allowed := false
	for _, p := range AllowUnauthenticated {
		if p == procedure {
			allowed = true
		}
	}

	if !IsLocalRequest(req) {
		if allowed {
			return nil, nil // a guest: GetStatus tells the web UI why it cannot log in
		}
		return nil, authn.Errorf("%s", errLocalOnly)
	}

	au := localhostUser()
	if required := RequiredPermission(procedure); !allowed && required != "" && !au.HasPermission(required) {
		log.WithFields(log.Fields{"procedure": procedure, "perm": required}).Info("Not available without a database")
		return nil, authn.Errorf("%s needs a database (permission %s)", procedure, required)
	}
	return au, nil
}

func (l LocalhostAuth) WrapHandler(in http.Handler) http.Handler {
	return authn.NewMiddleware(l.Handle).Wrap(in)
}

// WrapMCPHandler lets localhost use MCP without an API key (there are none without a database).
func (LocalhostAuth) WrapMCPHandler(in http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if !IsLocalRequest(r) {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusUnauthorized)
			_ = json.NewEncoder(w).Encode(map[string]string{"error": errLocalOnly})
			return
		}
		in.ServeHTTP(w, r.WithContext(authn.SetInfo(r.Context(), localhostUser())))
	})
}
