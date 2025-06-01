package httpserver

import (
	"github.com/jamesread/golure/pkg/dirs"
	"github.com/jamesread/StencilBox/internal/clientapi"
	clientapiconnect "github.com/jamesread/StencilBox/gen/StencilBox/clientapi/v1/clientapi_pbconnect"

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

func getNewApiHandler() (string, http.Handler, *clientapi.ClientApi) {
	apiServer := clientapi.NewServer()

	path, handler := clientapiconnect.NewStencilBoxApiServiceHandler(apiServer)

	return path, withCors(handler), apiServer
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

func Start() {
	log.WithFields(log.Fields{
	}).Info("Starting HTTP server")

	apipath, apihandler, apiServer := getNewApiHandler()

	log.Infof("API path: %s", apipath)

	mux := http.NewServeMux()

	mux.HandleFunc("/api"+apipath, func(w http.ResponseWriter, r *http.Request) {
		http.StripPrefix("/api", apihandler).ServeHTTP(w, r)
	})

	webuiPath := findWebuiDir()

	log.Infof("WebUI path: %s", webuiPath)

	mux.Handle("/webui/", getNewWebUIHandler(webuiPath))

	mux.Handle("/", getOutputHandler(apiServer.BaseOutputDir))

	srv := &http.Server{
		Addr:    "0.0.0.0:8080",
		Handler: mux,
	}

	log.Fatal(srv.ListenAndServe())
}
