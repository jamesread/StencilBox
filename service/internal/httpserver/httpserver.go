package httpserver

import (
	"github.com/jamesread/golure/pkg/dirs"
	"github.com/jamesread/StencilBox/internal/clientapi"
	"github.com/jamesread/StencilBox/internal/config"
	clientapiconnect "github.com/jamesread/StencilBox/gen/StencilBox/clientapi/v1/clientapi_pbconnect"
	auth "github.com/jamesread/httpauthshim"

	connectcors "connectrpc.com/cors"

	"github.com/rs/cors"

	"context"
	"path/filepath"
	"net/http"
	"os"

	log "github.com/sirupsen/logrus"
)

func withCors(h http.Handler) http.Handler {
	mw := cors.New(cors.Options{
		AllowedOrigins:   []string{"*"},
		AllowedMethods:   connectcors.AllowedMethods(),
		AllowedHeaders:   connectcors.AllowedHeaders(),
		ExposedHeaders:   connectcors.ExposedHeaders(),
	})

	return mw.Handler(h)
}

func getNewApiHandler(authCtx *auth.AuthShimContext) (string, http.Handler, *clientapi.ClientApi) {
	apiServer := clientapi.NewServer(authCtx)

	path, handler := clientapiconnect.NewStencilBoxApiServiceHandler(apiServer)
	handler = withCors(http.StripPrefix("/api", handler))

	log.Infof("API path: /api/%s", path)

	return path, handler, apiServer
}

func findWebuiDir() string {
	webuidir, err := dirs.GetFirstExistingDirectory("webui", []string{
		"../frontend/dist/",
		"../frontend/",
		"/frontend/",
		"/usr/share/SpaghettiCannon/frontend/",
		"/var/www/SpaghettiCannon/",
		"/etc/SpaghettiCannon/frontend/",
	})

	if err != nil {
		log.Warnf("Did not find the webui directory, you will probably get 404 errors.")
	}

	log.Infof("WebUI path: %s", webuidir)

	return webuidir
}

func getNewWebUIHandler(dir string) http.Handler {
	return http.StripPrefix("/webui/", http.FileServer(http.Dir(dir)))
}

func getOutputHandler(dir string) http.Handler {
	index, _ := filepath.Abs(filepath.Join(dir, "index.html"))

	if _, err := os.Stat(index); os.IsNotExist(err) {
		log.WithFields(log.Fields{
			"index": index,
		}).Infof("Creating index.html file")

		err := os.MkdirAll(dir, 0755)

		if err != nil {
			log.WithFields(log.Fields{
				"index": index,
			}).Fatalf("Could not create output directory")
		}

		err = os.WriteFile(index, []byte("<html><body><h1>StencilBox Default Index file</h1><p>This page will be replaced when something is built.</p><a href = 'webui'>webui</a></body></html>"), 0644)

		if err != nil {
			log.WithFields(log.Fields{
				"index": index,
			}).Fatalf("Could not create index.html file")
		}
	}

	return http.FileServer(http.Dir(dir))
}

func getHttpServerAddress() string {
	address := os.Getenv("STENCILBOX_ADDRESS")

	if address == "" {
		address = "0.0.0.0:8080"
	}

	log.WithFields(log.Fields{
		"STENCILBOX_ADDRESS env var": address,
	}).Info("Starting HTTP server")

	return address
}

// Context key for storing http.Request
type contextKey string

const httpRequestKey contextKey = "httpRequest"

// withAuth wraps an http.Handler with authentication middleware
func withAuth(authCtx *auth.AuthShimContext, h http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		user := authCtx.AuthFromHttpReq(r)

		if user == nil || user.IsGuest() {
			// Not authenticated, return 401
			w.Header().Set("WWW-Authenticate", `Basic realm="StencilBox"`)
			http.Error(w, "Unauthorized", http.StatusUnauthorized)
			return
		}

		// Store http.Request in context for use in API handlers
		ctx := r.Context()
		ctx = context.WithValue(ctx, httpRequestKey, r)
		r = r.WithContext(ctx)

		// User is authenticated, proceed with request
		h.ServeHTTP(w, r)
	})
}

// setupAuth initializes and returns the authentication context
func setupAuth() (*auth.AuthShimContext, error) {
	// Load config from config.yaml file
	appConfig := config.LoadConfig()

	// Use auth config from config.yaml if available
	if appConfig.Auth == nil {
		log.Debug("No auth configuration found in config.yaml")
		return nil, nil
	}

	authCfg := appConfig.Auth
	log.Debug("Auth config loaded from config.yaml")

	// Check if auth is enabled via LocalUsers
	if !authCfg.LocalUsers.Enabled {
		log.Info("Authentication is disabled")
		return nil, nil
	}

	// Validate that we have users configured
	if len(authCfg.LocalUsers.Users) == 0 {
		log.Warn("Authentication is enabled but no users are configured. Authentication disabled.")
		return nil, nil
	}

	authCtx, err := auth.NewAuthShimContext(authCfg)
	if err != nil {
		return nil, err
	}

	log.Info("Authentication enabled")
	return authCtx, nil
}

func Start() {
	// Setup authentication first
	authCtx, err := setupAuth()
	if err != nil {
		log.Fatalf("Failed to setup authentication: %v", err)
	}

	// Ensure auth context is cleaned up on shutdown
	if authCtx != nil {
		defer func() {
			if err := authCtx.Shutdown(); err != nil {
				log.Errorf("Error shutting down auth context: %v", err)
			}
		}()
	}

	apiPath, apiHandler, apiServer := getNewApiHandler(authCtx)

	mux := http.NewServeMux()

	// Apply authentication to API and webui routes if auth is enabled
	if authCtx != nil {
		mux.Handle("/api"+apiPath, withAuth(authCtx, apiHandler))
		mux.Handle("/webui/", withAuth(authCtx, getNewWebUIHandler(findWebuiDir())))
	} else {
		mux.Handle("/api"+apiPath, apiHandler)
		mux.Handle("/webui/", getNewWebUIHandler(findWebuiDir()))
	}

	// Output directory (built sites) is always public - no auth required
	mux.Handle("/", getOutputHandler(apiServer.BaseOutputDir))

	srv := &http.Server{
		Addr:    getHttpServerAddress(),
		Handler: mux,
	}

	log.Fatal(srv.ListenAndServe())
}
