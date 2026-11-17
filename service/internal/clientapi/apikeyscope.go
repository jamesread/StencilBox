package clientapi

import (
	"context"
	"database/sql"
	"fmt"
	"strings"

	"connectrpc.com/connect"
	"github.com/jamesread/StencilBox/internal/iam"
	armlayer "github.com/jamesread/armature-iam/layer"
	log "github.com/sirupsen/logrus"
)

// API key scopes are enforced here, after the auth layer has authenticated the key's owner and
// checked the owner holds the procedure's permission. The interceptor then checks the key's
// own scope: its permissions, and the build config each request names.

type apiKeyScopeCtxKey struct{}

// apiKeyScope returns the scope of the API key that authenticated this request, or nil for
// browser sessions and SSO (which are unrestricted beyond their RBAC).
func apiKeyScope(ctx context.Context) *iam.APIKeyScope {
	s, _ := ctx.Value(apiKeyScopeCtxKey{}).(*iam.APIKeyScope)
	return s
}

// buildConfigAllowed reports whether the caller may see or use a build config.
func buildConfigAllowed(ctx context.Context, name string) bool {
	return apiKeyScope(ctx).AllowsBuildConfig(name)
}

// Request messages that name a build config.
type configNamedRequest interface{ GetConfigName() string }
type buildConfigNamedRequest interface{ GetBuildConfigName() string }

func requestedBuildConfig(msg any) (string, bool) {
	switch m := msg.(type) {
	case configNamedRequest:
		return m.GetConfigName(), true
	case buildConfigNamedRequest:
		return m.GetBuildConfigName(), true
	}
	return "", false
}

var allowUnauthenticated = func() map[string]bool {
	out := map[string]bool{}
	for _, p := range iam.AllowUnauthenticated {
		out[p] = true
	}
	return out
}()

func bearerToken(header interface{ Get(string) string }) string {
	token, ok := strings.CutPrefix(header.Get("Authorization"), "Bearer ")
	if !ok {
		return ""
	}
	return strings.TrimSpace(token)
}

// errScope is a PermissionDenied error with a message that tells the key's owner what is missing.
func errScope(format string, args ...any) error {
	return connect.NewError(connect.CodePermissionDenied, fmt.Errorf(format, args...))
}

// resolveAPIKeyScope loads the key's scope into the context, narrows the request's RBAC to it,
// and refuses procedures the key may not call.
func (c *ClientApi) resolveAPIKeyScope(ctx context.Context, procedure string, header interface{ Get(string) string }) (context.Context, error) {
	scope, au, err := c.loadRequestScope(ctx, header)
	if err != nil || scope == nil {
		return ctx, err
	}
	if err = checkProcedureScope(procedure, au, scope); err != nil {
		return ctx, err
	}
	if err = c.narrowForScope(ctx, au, scope); err != nil {
		return ctx, err
	}
	return context.WithValue(ctx, apiKeyScopeCtxKey{}, scope), nil
}

// loadRequestScope returns the scope of the API key on the request, or nil when the request
// was not authenticated by an API key.
func (c *ClientApi) loadRequestScope(ctx context.Context, header interface{ Get(string) string }) (*iam.APIKeyScope, *armlayer.AuthenticatedUser, error) {
	if c.iamDB == nil {
		return nil, nil, nil
	}
	token := bearerToken(header)
	if token == "" {
		return nil, nil, nil
	}
	au := authUser(ctx)
	if au == nil {
		return nil, nil, nil // a guest on an allow-listed procedure; the key was not accepted
	}
	scope, err := iam.LoadAPIKeyScopeByValue(ctx, c.iamDB, token)
	if err != nil {
		return nil, nil, internalErr("load api key scope", err)
	}
	return scope, au, nil
}

// checkProcedureScope refuses procedures the key may not call.
func checkProcedureScope(procedure string, au *armlayer.AuthenticatedUser, scope *iam.APIKeyScope) error {
	if iam.SessionOnlyProcedures[procedure] {
		return errScope("API keys cannot manage API keys or passwords; log in to the web UI")
	}
	if allowUnauthenticated[procedure] {
		return nil
	}
	required := iam.RequiredPermission(procedure)
	if required != "" && !scope.AllowsPermission(required) {
		log.WithFields(log.Fields{"user": au.User.Username, "procedure": procedure, "perm": required}).Warn("API key scope denied")
		return errScope("this API key does not have the %s permission", required)
	}
	return nil
}

// narrowForScope narrows the request's RBAC to the permissions the key is restricted to.
func (c *ClientApi) narrowForScope(ctx context.Context, au *armlayer.AuthenticatedUser, scope *iam.APIKeyScope) error {
	if !scope.RestrictPermissions {
		return nil
	}
	all, err := c.allPermissionNames(ctx)
	if err != nil {
		return err
	}
	au.RBAC = iam.NarrowRBAC(au.RBAC, all, scope)
	return nil
}
func (c *ClientApi) allPermissionNames(ctx context.Context) ([]string, error) {
	perms, err := c.auth.Store.ListRBACPermissions(ctx)
	if err != nil {
		return nil, internalErr("list permissions", err)
	}
	names := make([]string, 0, len(perms))
	for _, p := range perms {
		names = append(names, p.Name)
	}
	return names, nil
}

func checkRequestBuildConfig(ctx context.Context, msg any) error {
	name, ok := requestedBuildConfig(msg)
	if !ok || buildConfigAllowed(ctx, name) {
		return nil
	}
	return errScope("this API key cannot use build config %q", name)
}

type apiKeyScopeInterceptor struct {
	api *ClientApi
}

// ScopeInterceptor enforces API key scopes; register it on the Connect handler.
func (c *ClientApi) ScopeInterceptor() connect.Interceptor {
	return &apiKeyScopeInterceptor{api: c}
}

// NewScopeInterceptor builds the API key scope interceptor without a full server, for tests of
// handlers that sit behind it.
func NewScopeInterceptor(authLayer *iam.AuthLayer, db *sql.DB) connect.Interceptor {
	return (&ClientApi{auth: authLayer, iamDB: db}).ScopeInterceptor()
}

func (i *apiKeyScopeInterceptor) WrapUnary(next connect.UnaryFunc) connect.UnaryFunc {
	return func(ctx context.Context, req connect.AnyRequest) (connect.AnyResponse, error) {
		ctx, err := i.api.resolveAPIKeyScope(ctx, req.Spec().Procedure, req.Header())
		if err != nil {
			return nil, err
		}
		if err := checkRequestBuildConfig(ctx, req.Any()); err != nil {
			return nil, err
		}
		return next(ctx, req)
	}
}

func (i *apiKeyScopeInterceptor) WrapStreamingClient(next connect.StreamingClientFunc) connect.StreamingClientFunc {
	return next
}

func (i *apiKeyScopeInterceptor) WrapStreamingHandler(next connect.StreamingHandlerFunc) connect.StreamingHandlerFunc {
	return func(ctx context.Context, conn connect.StreamingHandlerConn) error {
		ctx, err := i.api.resolveAPIKeyScope(ctx, conn.Spec().Procedure, conn.RequestHeader())
		if err != nil {
			return err
		}
		return next(ctx, &scopedStreamConn{StreamingHandlerConn: conn, ctx: ctx})
	}
}

// scopedStreamConn checks each received message (eg StartBuild's BuildRequest) against the
// key's build configs.
type scopedStreamConn struct {
	connect.StreamingHandlerConn
	ctx context.Context
}

func (s *scopedStreamConn) Receive(msg any) error {
	if err := s.StreamingHandlerConn.Receive(msg); err != nil {
		return err
	}
	return checkRequestBuildConfig(s.ctx, msg)
}
