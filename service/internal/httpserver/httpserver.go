package httpserver

import (
	"database/sql"

	clientapiconnect "github.com/jamesread/StencilBox/gen/StencilBox/clientapi/v1/clientapi_pbconnect"
	"github.com/jamesread/StencilBox/internal/clientapi"
	"github.com/jamesread/StencilBox/internal/config"
	"github.com/jamesread/StencilBox/internal/database"
	"github.com/jamesread/StencilBox/internal/iam"
	"github.com/jamesread/StencilBox/internal/mcp"
	"github.com/jamesread/StencilBox/internal/reqctx"
	iamsqlite "github.com/jamesread/armature-iam/store/sqlite"
	"github.com/jamesread/golure/pkg/dirs"

	"connectrpc.com/connect"
	connectcors "connectrpc.com/cors"

	"github.com/rs/cors"

	"context"
	"errors"
	"net/http"
	"os"
	"os/signal"
	"path/filepath"
	"strings"
	"syscall"
	"time"

	log "github.com/sirupsen/logrus"
)

func withCors(h http.Handler) http.Handler {
	mw := cors.New(cors.Options{
		AllowedOrigins: []string{"*"},
		AllowedMethods: connectcors.AllowedMethods(),
		AllowedHeaders: connectcors.AllowedHeaders(),
		ExposedHeaders: connectcors.ExposedHeaders(),
	})

	return mw.Handler(h)
}

func getNewApiHandler(authenticator iam.Authenticator, authLayer *iam.AuthLayer, db *sql.DB) (string, http.Handler, *clientapi.ClientApi) {
	apiServer := clientapi.NewServer(authLayer, db)

	path, handler := clientapiconnect.NewStencilBoxApiServiceHandler(apiServer,
		connect.WithInterceptors(apiServer.ScopeInterceptor()))
	// Strip /api first: the auth middleware infers the procedure from the URL path.
	handler = withCors(http.StripPrefix("/api", withHttpRequest(authenticator.WrapHandler(handler))))

	log.Infof("API path: /api/%s", path)

	return path, handler, apiServer
}

func findWebuiDir() string {
	webuidir, err := dirs.GetFirstExistingDirectory("webui", []string{
		"../frontend/dist/",
		"../frontend/",
		"/frontend/",
		"/usr/share/StencilBox/frontend/",
		"/var/www/StencilBox/webui/",
		"/etc/StencilBox/frontend/",
	})

	if err != nil {
		log.Warnf("Did not find the webui directory, you will probably get 404 errors.")
	}

	log.Infof("WebUI path: %s", webuidir)

	return webuidir
}

func getNewWebUIHandler(dir string) http.Handler {
	return http.StripPrefix("/webui/", http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.ServeFile(w, r, resolveWebuiFile(dir, r.URL.Path))
	}))
}

func resolveWebuiFile(dir, requestPath string) string {
	index := filepath.Join(dir, "index.html")
	if full := existingWebuiAsset(dir, requestPath); full != "" {
		return full
	}
	return index
}

func existingWebuiAsset(dir, requestPath string) string {
	rel := strings.TrimPrefix(filepath.Clean("/"+requestPath), "/")
	if rel == "" || rel == "." {
		return ""
	}

	full := filepath.Join(dir, rel)
	if !isExistingFileInside(dir, full) {
		return ""
	}
	return full
}

func isExistingFileInside(dir, full string) bool {
	if !isPathInsideDir(dir, full) {
		return false
	}
	info, err := os.Stat(full)
	return err == nil && !info.IsDir()
}

func isPathInsideDir(dir, full string) bool {
	rel, err := filepath.Rel(dir, full)
	if err != nil {
		return false
	}
	return rel != ".." && !strings.HasPrefix(rel, ".."+string(filepath.Separator))
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
			}).Fatalf("Could not create output directory: %v", err)
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

// getHttpServerAddress returns the bind address.
// Priority: $PORT, then $STENCILBOX_ADDRESS, then 0.0.0.0:8080.
// PORT may be a bare port ("8080") or a full address (":8080" / "0.0.0.0:8080").
func getHttpServerAddress() string {
	if port := strings.TrimSpace(os.Getenv("PORT")); port != "" {
		address := normalizeListenAddr(port)
		log.WithFields(log.Fields{"PORT": port, "address": address}).Info("Starting HTTP server")
		return address
	}

	if address := strings.TrimSpace(os.Getenv("STENCILBOX_ADDRESS")); address != "" {
		log.WithFields(log.Fields{"STENCILBOX_ADDRESS": address}).Info("Starting HTTP server")
		return address
	}

	address := "0.0.0.0:8080"
	log.WithFields(log.Fields{"address": address}).Info("Starting HTTP server")
	return address
}

func normalizeListenAddr(port string) string {
	if strings.Contains(port, ":") {
		return port
	}
	return "0.0.0.0:" + port
}

// withHttpRequest stores the *http.Request in the context, so handlers can see TLS state.
func withHttpRequest(h http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		h.ServeHTTP(w, r.WithContext(reqctx.WithHttpRequest(r.Context(), r)))
	})
}

// setupIAM opens the database (applying migrations), bootstraps users and RBAC, and builds the
// Connect authentication layer from the config.yaml auth block.
// openDatabase opens the database (applying migrations) and bootstraps users and RBAC. It
// returns nil when the database is turned off, or cannot be opened: StencilBox then runs with
// localhost-only access rather than not at all.
func openDatabase(ctx context.Context, appConfig *config.Config) *sql.DB {
	if !appConfig.DatabaseEnabled() {
		log.Warn("The database is turned off (database.enabled / STENCILBOX_DB_DISABLED): only requests from localhost are accepted, and user accounts, IAM and API keys are unavailable.")
		return nil
	}

	db, err := database.Open(ctx, database.Path(config.ConfigDir()))
	if err == nil {
		err = iam.Bootstrap(ctx, iamsqlite.New(db))
		if err != nil {
			_ = db.Close()
		}
	}
	if err != nil {
		log.Errorf("Could not open the database, so StencilBox is running WITHOUT one: only requests from localhost are accepted, and user accounts, IAM and API keys are unavailable. Fix the database (or set database.enabled: false to silence this). Error: %v", err)
		return nil
	}
	return db
}

// setupAuth returns the authenticator for the API and MCP, and the IAM layer when there is a
// database. A broken auth block in config.yaml is fatal: that is a mistake to fix, not degrade.
func setupAuth(db *sql.DB, appConfig *config.Config) (iam.Authenticator, *iam.AuthLayer) {
	if db == nil {
		return iam.LocalhostAuth{}, nil
	}
	authLayer, err := iam.NewAuthLayer(iamsqlite.New(db), appConfig.Auth)
	if err != nil {
		log.Fatalf("Failed to set up authentication from config.yaml: %v", err)
	}
	return authLayer, authLayer
}

func Start() {
	appConfig := config.LoadConfig()
	db := openDatabase(context.Background(), appConfig)
	if db != nil {
		defer func() {
			if err := db.Close(); err != nil {
				log.Errorf("Error closing database: %v", err)
			}
		}()
	}
	authenticator, authLayer := setupAuth(db, appConfig)

	apiPath, apiHandler, apiServer := getNewApiHandler(authenticator, authLayer, db)

	mux := http.NewServeMux()

	// API calls are authenticated per procedure by the auth layer. The web UI is static files
	// that show a login form until GetStatus reports a logged-in user.
	mux.Handle("/api"+apiPath, apiHandler)
	mux.Handle("/webui/", getNewWebUIHandler(findWebuiDir()))

	// MCP for AI assistants. With a database: Bearer API keys only (no cookies). Without one:
	// localhost only. Tools call the API above in-process, so they get the same access checks.
	mux.Handle(mcp.Path, authenticator.WrapMCPHandler(mcp.NewHandler(apiHandler)))
	log.Infof("MCP path: %s", mcp.Path)

	// Output directory (built sites) is always public - no auth required
	mux.Handle("/", getOutputHandler(apiServer.BaseOutputDir))

	srv := &http.Server{
		Addr:              getHttpServerAddress(),
		Handler:           mux,
		ReadHeaderTimeout: 30 * time.Second,
	}

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	serveErr := make(chan error, 1)
	go func() {
		serveErr <- srv.ListenAndServe()
	}()

	select {
	case err := <-serveErr:
		log.Fatalf("HTTP server failed: %v", err)
	case <-ctx.Done():
	}

	stop()
	shutdown(srv, apiServer)
}

// shutdownGracePeriod is how long in-progress builds get to finish after SIGTERM. It is below
// the 30s Kubernetes default; Docker only waits 10s unless --stop-timeout is raised.
const shutdownGracePeriod = 25 * time.Second

func shutdown(srv *http.Server, apiServer *clientapi.ClientApi) {
	log.Infof("Shutting down, waiting up to %v for builds in progress", shutdownGracePeriod)

	ctx, cancel := context.WithTimeout(context.Background(), shutdownGracePeriod)
	defer cancel()

	// Stop accepting builds first, so a request arriving during shutdown is refused rather than started.
	apiDone := make(chan struct{})
	go func() {
		apiServer.Shutdown(ctx)
		close(apiDone)
	}()

	if err := srv.Shutdown(ctx); err != nil && !errors.Is(err, http.ErrServerClosed) {
		log.Warnf("HTTP server did not shut down cleanly, closing connections: %v", err)
		_ = srv.Close()
	}

	<-apiDone

	log.Info("StencilBox stopped")
}
