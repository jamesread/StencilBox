package httpserver

import (
	"github.com/jamesread/StencilBox/internal/config"
	"github.com/jamesread/StencilBox/internal/clientapi"
	"github.com/jamesread/StencilBox/gen/StencilBox/clientapi/v1/clientapiconnect"

	connectcors "connectrpc.com/cors"

	"github.com/rs/cors"

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

func getNewApiHandler(cfg *config.Config) (string, http.Handler) {
	apiServer := clientapi.NewServer(cfg)

	path, handler := clientapiconnect.NewStencilBoxApiServiceHandler(apiServer)

	return path, withCors(handler)
}

func findWebuiDir() string {
	directoriesToSearch := []string{
		"../frontend/dist/",
		"../frontend/",
		"/frontend/",
		"/usr/share/SpaghettiCannon/frontend/",
		"/var/www/SpaghettiCannon/",
		"/etc/SpaghettiCannon/frontend/",
	}

	for _, dir := range directoriesToSearch {
		if _, err := os.Stat(dir); !os.IsNotExist(err) {
			absdir, _ := filepath.Abs(dir)

			log.WithFields(log.Fields{
				"dir": dir,
				"absdir": absdir,
			}).Infof("Found the webui directory")

			return absdir
		}
	}

	log.Warnf("Did not find the webui directory, you will probably get 404 errors.")

	return "./webui" // Should not exist
}

func getNewWebUIHandler(dir string) http.Handler {
	return http.StripPrefix("/webui/", http.FileServer(http.Dir(dir)))
}

func getOutputHandler(dir string) http.Handler {
	if dir == "" {
		dir = "./sb-output"
	}

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

func Start(cfg *config.Config) {
	log.WithFields(log.Fields{
	}).Info("Starting HTTP server")

	apipath, apihandler := getNewApiHandler(cfg)

	log.Infof("API path: %s", apipath)

	mux := http.NewServeMux()

	mux.HandleFunc("/api"+apipath, func(w http.ResponseWriter, r *http.Request) {
		log.Infof("API request: %s", r.URL.Path)

		http.StripPrefix("/api", apihandler).ServeHTTP(w, r)
	})

	webuiPath := findWebuiDir()

	log.Infof("WebUI path: %s", webuiPath)

	mux.Handle("/webui/", getNewWebUIHandler(webuiPath))

	mux.Handle("/", getOutputHandler(cfg.OutputDir))

	srv := &http.Server{
		Addr:    "0.0.0.0:8080",
		Handler: mux,
	}

	log.Fatal(srv.ListenAndServe())
}
