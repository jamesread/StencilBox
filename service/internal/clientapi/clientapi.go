package clientapi

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"gopkg.in/yaml.v3"

	"connectrpc.com/connect"
	pb "github.com/jamesread/StencilBox/gen/StencilBox/clientapi/v1"
	client "github.com/jamesread/StencilBox/gen/StencilBox/clientapi/v1/clientapi_pbconnect"
	"github.com/jamesread/StencilBox/internal/buildconfigs"
	"github.com/jamesread/StencilBox/internal/buildinfo"
	"github.com/jamesread/StencilBox/internal/generator"
	"github.com/jamesread/StencilBox/internal/iam"
	"github.com/jamesread/StencilBox/internal/watcher"
	"github.com/jamesread/armature-iam/rbac"
	"github.com/jamesread/golure/pkg/dirs"
	log "github.com/sirupsen/logrus"
)

type ClientApi struct {
	buildConfigs  map[string]*buildconfigs.BuildConfig
	templates     map[string]*Template
	dataWatcher   *watcher.DataFileWatcher
	watcherCancel context.CancelFunc
	buildHistory  map[string][]*pb.BuildHistoryEntry // config name -> history entries
	auth          *iam.AuthLayer                     // IAM store and auth middleware (nil in some tests)
	iamDB         *sql.DB                            // database behind auth.Store
	buildCancel   context.CancelFunc                 // aborts automatic builds when shutdown times out
	watcherCtx    context.Context
	buildCtx      context.Context // parent context for automatic builds

	client.StencilBoxApiServiceClient

	BaseOutputDir string
	historyPath   string // where buildHistory is persisted

	builds sync.WaitGroup // builds in progress, waited for on shutdown

	historyMu   sync.RWMutex // protects buildHistory
	dataFilesMu sync.Mutex   // serialises UpdateDataFile, so expected_sha256 checks are reliable
	shutdownMu  sync.Mutex   // protects shuttingDown and builds.Add

	defaultAdminPassword atomic.Bool // the admin account still has the default password
	shuttingDown         bool
}

type Template struct {
	Name             string
	Source           string
	Status           string
	DocumentationURL string
	Description      string
}

func NewServer(authLayer *iam.AuthLayer, db *sql.DB) *ClientApi {
	api := &ClientApi{}
	api.auth = authLayer
	api.iamDB = db
	api.refreshDefaultAdminPassword(context.Background())
	api.buildConfigs = buildconfigs.ReadConfigFiles()
	api.BaseOutputDir, _ = filepath.Abs(findOutputDir())
	api.historyPath = buildHistoryPath(api.BaseOutputDir)
	api.buildHistory = loadBuildHistory(api.historyPath)
	api.buildCtx, api.buildCancel = context.WithCancel(context.Background())

	// Initialize the data file watcher
	api.watcherCtx, api.watcherCancel = context.WithCancel(context.Background())

	dataWatcher, err := watcher.NewDataFileWatcher(api.buildConfigs, func(configName string) {
		api.autoRebuild(configName)
	})

	if err != nil {
		log.Errorf("Failed to create data file watcher: %v", err)
	} else {
		api.dataWatcher = dataWatcher
		err = api.dataWatcher.Start(api.watcherCtx)
		if err != nil {
			log.Errorf("Failed to start data file watcher: %v", err)
		} else {
			log.Info("Data file watcher started successfully")
		}
	}

	api.queueStartupBuilds()

	return api
}

func (c *ClientApi) queueStartupBuilds() {
	for name, cfg := range c.buildConfigs {
		if !cfg.BuildOnStartup() {
			continue
		}

		if cfg.ErrorMessage != "" {
			log.Warnf("Skipping startup build for config %s: %s", name, cfg.ErrorMessage)
			continue
		}

		log.Infof("Queueing startup build for config: %s", name)
		c.autoRebuild(name)
	}
}

func findOutputDir() string {
	if dir := strings.TrimSpace(os.Getenv("STENCILBOX_OUTPUT_DIR")); dir != "" {
		return dir
	}

	outputdir, err := dirs.GetFirstExistingDirectory("output", []string{
		"/config/output/",
		"/var/www/StencilBox/",
		"../sb-output/",
	})

	if err != nil {
		def := defaultOutputDir()
		log.Warnf("Did not find the output directory, using default %s", def)
		return def
	}

	return outputdir
}

func defaultOutputDir() string {
	if _, err := os.Stat("/config"); err == nil {
		return "/config/output"
	}
	return "./sb-output"
}

func (c *ClientApi) Init(ctx context.Context, req *connect.Request[pb.InitRequest]) (*connect.Response[pb.InitResponse], error) {
	response := &pb.InitResponse{
		Version: buildinfo.Version,
	}

	c.buildConfigs = buildconfigs.ReadConfigFiles()
	c.templates = readTemplates()

	// Update the watcher with the new build configs
	if c.dataWatcher != nil {
		err := c.dataWatcher.UpdateBuildConfigs(c.buildConfigs)
		if err != nil {
			log.Errorf("Failed to update data file watcher: %v", err)
		}
	}

	response.SearchHints = c.buildSearchHints(ctx)

	return connect.NewResponse(response), nil
}

func (c *ClientApi) buildSearchHints(ctx context.Context) *pb.SearchHints {
	hints := &pb.SearchHints{}

	for name := range c.buildConfigs {
		if !buildConfigAllowed(ctx, name) {
			continue
		}
		hints.BuildConfigs = append(hints.BuildConfigs, name)
	}

	sort.Strings(hints.BuildConfigs)

	return hints
}

type TemplateMetadata struct {
	DocumentationUrl string `yaml:"documentation_url"`
	Description      string `yaml:"description"`
}

// readTemplates lists templates from every template directory. Custom directories are
// searched first, so a custom template shadows a built-in template of the same name.
func readTemplates() map[string]*Template {
	templates := make(map[string]*Template)

	for _, dir := range generator.FindTemplateDirs() {
		log.Infof("Reading %s templates from: %s", dir.Source, dir.Path)

		res, err := filepath.Glob(filepath.Join(dir.Path, "*", "index.html"))

		if err != nil {
			log.Errorf("Error reading templates: %v", err)
			continue
		}

		for _, fp := range res {
			templateName := filepath.Base(filepath.Dir(fp))

			if _, exists := templates[templateName]; exists {
				log.Infof("Template %s in %s is shadowed by a custom template", templateName, dir.Path)
				continue
			}

			templates[templateName] = readTemplate(templateName, filepath.Dir(fp), dir.Source)
		}
	}

	return templates
}

// readTemplate builds a Template from its directory. metadata.yaml is optional.
func readTemplate(name, dir, source string) *Template {
	tpl := &Template{
		Name:   name,
		Source: source,
		Status: "OK",
	}

	metadata, err := os.ReadFile(filepath.Join(dir, "metadata.yaml"))

	if os.IsNotExist(err) {
		return tpl
	}

	if err != nil {
		log.Errorf("Error reading metadata for template %s: %v", name, err)
		tpl.Status = "error"
		tpl.Description = "Error reading metadata.yaml: " + err.Error()
		return tpl
	}

	metadataMap := TemplateMetadata{}

	if err := yaml.Unmarshal(metadata, &metadataMap); err != nil {
		log.Errorf("Error unmarshalling metadata for template %s: %v", name, err)
		tpl.Status = "error"
		tpl.Description = "Invalid metadata.yaml: " + err.Error()
		return tpl
	}

	tpl.DocumentationURL = metadataMap.DocumentationUrl
	tpl.Description = metadataMap.Description

	return tpl
}

func (c *ClientApi) StartBuild(ctx context.Context, req *connect.Request[pb.BuildRequest], srv *connect.ServerStream[pb.BuildUpdateResponse]) error {
	response := &pb.BuildUpdateResponse{
		ConfigName: req.Msg.ConfigName,
		Status:     "Build starting",
	}

	buildConfig, found := c.buildConfigs[req.Msg.ConfigName]
	if !found {
		log.Errorf("Build config %s not found", req.Msg.ConfigName)
		return rejectBuild(srv, response, "Build config not found",
			connect.NewError(connect.CodeNotFound, fmt.Errorf("build config %s not found", req.Msg.ConfigName)))
	}

	if !c.beginBuild() {
		return rejectBuild(srv, response, "StencilBox is shutting down",
			connect.NewError(connect.CodeUnavailable, errors.New("server is shutting down")))
	}
	defer c.builds.Done()

	buildStatus := &generator.BuildStatus{}
	buildStarted := time.Now()
	if err := c.streamBuild(ctx, buildConfig, buildStatus, response, srv); err != nil {
		return err
	}

	log.Infof("Build completed for config %s", req.Msg.ConfigName)
	c.completeBuildResponse(response, buildStatus, buildConfig)
	if err := srv.Send(response); err != nil {
		return err
	}

	// Record build history
	c.recordBuildHistory(req.Msg.ConfigName, buildStatus, buildConfig.OutputDir, false, time.Since(buildStarted))

	return nil
}

// rejectBuild tells the client why a build did not start, then returns the error for the RPC.
func rejectBuild(srv *connect.ServerStream[pb.BuildUpdateResponse], response *pb.BuildUpdateResponse, status string, rpcErr error) error {
	response.IsError = true
	response.Status = status
	if err := srv.Send(response); err != nil {
		return err
	}
	return rpcErr
}

// streamBuild runs the build and forwards each progress update to the client.
func (c *ClientApi) streamBuild(ctx context.Context, buildConfig *buildconfigs.BuildConfig, buildStatus *generator.BuildStatus, response *pb.BuildUpdateResponse, srv *connect.ServerStream[pb.BuildUpdateResponse]) error {
	updateChan := make(chan string)
	go generator.Generate(ctx, c.BaseOutputDir, buildConfig, buildStatus, updateChan)

	for update := range updateChan {
		log.Infof("Build update: %s", update)

		response.Status = update
		if err := srv.Send(response); err != nil {
			return err
		}
	}
	return nil
}

func (c *ClientApi) completeBuildResponse(response *pb.BuildUpdateResponse, buildStatus *generator.BuildStatus, buildConfig *buildconfigs.BuildConfig) {
	response.IsError = buildStatus.IsError
	if buildStatus.Message != "" {
		response.Status = buildStatus.Message
	}
	response.BuildUrlBase = buildStatus.BuildUrlBase
	response.OutputSizeHumanReadable = buildStatus.OutputSizeHumanReadable
	response.BaseOutputDir = c.BaseOutputDir
	response.InContainer = inContainer()
	response.RelativePath = buildConfig.OutputDir
	response.Found = true
	response.IsComplete = true
}
func (c *ClientApi) ClearBuildCache(ctx context.Context, req *connect.Request[pb.ClearBuildCacheRequest]) (*connect.Response[pb.ClearBuildCacheResponse], error) {
	buildConfig, found := c.buildConfigs[req.Msg.ConfigName]
	if !found {
		return nil, connect.NewError(connect.CodeNotFound, fmt.Errorf("build config %s not found", req.Msg.ConfigName))
	}

	cachePath, err := generator.ClearTemporaryOutputDir(c.BaseOutputDir, buildConfig.OutputDir)
	if err != nil {
		log.Errorf("Failed to clear build cache for %s: %v", req.Msg.ConfigName, err)
		return connect.NewResponse(&pb.ClearBuildCacheResponse{
			Success:   false,
			Message:   err.Error(),
			CachePath: cachePath,
		}), nil
	}

	log.Infof("Cleared build cache for %s at %s", req.Msg.ConfigName, cachePath)
	return connect.NewResponse(&pb.ClearBuildCacheResponse{
		Success:   true,
		Message:   "Build cache cleared",
		CachePath: cachePath,
	}), nil
}

func (c *ClientApi) GetBuildConfigs(ctx context.Context, req *connect.Request[pb.GetBuildConfigsRequest]) (*connect.Response[pb.GetBuildConfigsResponse], error) {
	response := &pb.GetBuildConfigsResponse{}
	response.CanGitPull = buildconfigs.CanGitPull()

	for name, bc := range c.buildConfigs {
		if !buildConfigAllowed(ctx, name) {
			continue
		}
		response.BuildConfigs = append(response.BuildConfigs, &pb.BuildConfig{
			Name:         name,
			Template:     bc.Template,
			OutputDir:    bc.OutputDir,
			Datafiles:    bc.Datafiles,
			Filename:     bc.Filename,
			Path:         bc.Path,
			ErrorMessage: bc.ErrorMessage,
		})
	}

	return connect.NewResponse(response), nil
}

func (c *ClientApi) GetTemplates(ctx context.Context, req *connect.Request[pb.GetTemplatesRequest]) (*connect.Response[pb.GetTemplatesResponse], error) {
	response := &pb.GetTemplatesResponse{}

	for _, template := range c.templates {
		response.Templates = append(response.Templates, &pb.Template{
			Name:         template.Name,
			Source:       template.Source,
			Status:       template.Status,
			BuildConfigs: c.getBuildConfigsForTemplate(ctx, template.Name),
		})
	}

	return connect.NewResponse(response), nil
}

func (c *ClientApi) getBuildConfigsForTemplate(ctx context.Context, templateName string) []string {
	buildConfigs := []string{}

	for _, bc := range c.buildConfigs {
		if bc.Template == templateName && buildConfigAllowed(ctx, bc.Name) {
			buildConfigs = append(buildConfigs, bc.Name)
		}
	}

	return buildConfigs
}

func inContainer() bool {
	containerFileExists := false

	if _, err := os.Stat("/.dockerenv"); err == nil {
		containerFileExists = true
	}

	return containerFileExists
}

func (c *ClientApi) GetStatus(ctx context.Context, req *connect.Request[pb.GetStatusRequest]) (*connect.Response[pb.GetStatusResponse], error) {
	response := &pb.GetStatusResponse{
		InContainer:     inContainer(),
		RbacPermissions: []string{},
		DatabaseEnabled: c.iamDB != nil,
	}

	// GetStatus is callable by guests: identity and server paths only for logged-in users.
	au := authUser(ctx)
	if au == nil {
		return connect.NewResponse(response), nil
	}

	response.IsLoggedIn = true
	response.Username = au.User.Username
	response.RbacPermissions = c.rbacPermissionNames(ctx, au)
	response.RbacIsSuperuser = au.RBAC != nil && au.RBAC.IsSuperuser
	response.AdminUsesDefaultPassword = response.RbacIsSuperuser && c.defaultAdminPassword.Load()

	if au.HasPermission(rbac.PermissionAppAccess) {
		dir, _ := buildconfigs.GetConfigDir()
		response.OutputPath = c.BaseOutputDir
		response.TemplatesPath = templateDirsString()
		response.BuildConfigsDir = dir
		response.CustomTemplatesPath = generator.CustomTemplatesDir()
	}

	return connect.NewResponse(response), nil
}

func templateDirsString() string {
	paths := []string{}

	for _, dir := range generator.FindTemplateDirs() {
		paths = append(paths, dir.Path)
	}

	return strings.Join(paths, ", ")
}

func (c *ClientApi) GetTemplate(ctx context.Context, req *connect.Request[pb.GetTemplateRequest]) (*connect.Response[pb.GetTemplateResponse], error) {
	response := &pb.GetTemplateResponse{}

	template, found := c.templates[req.Msg.TemplateName]

	log.Infof("Getting template: %+v %+v", c.templates, req.Msg.TemplateName)

	if found {
		response.Template = &pb.Template{
			Name:             template.Name,
			Source:           template.Source,
			Status:           template.Status,
			DocumentationUrl: template.DocumentationURL,
			BuildConfigs:     c.getBuildConfigsForTemplate(ctx, template.Name),
			Description:      template.Description,
		}

		return connect.NewResponse(response), nil
	} else {
		return nil, connect.NewError(connect.CodeNotFound, fmt.Errorf("template not found"))
	}
}

func (c *ClientApi) GetBuildConfig(ctx context.Context, req *connect.Request[pb.GetBuildConfigRequest]) (*connect.Response[pb.GetBuildConfigResponse], error) {
	response := &pb.GetBuildConfigResponse{}

	buildConfig, found := c.buildConfigs[req.Msg.ConfigName]

	if found {
		response.BuildConfig = &pb.BuildConfig{
			Name:          buildConfig.Name,
			Template:      buildConfig.Template,
			Filename:      buildConfig.Filename,
			Path:          filepath.Dir(buildConfig.Path),
			DatafilesPath: filepath.Dir(buildConfig.Path),
			InContainer:   inContainer(),
			ErrorMessage:  buildConfig.ErrorMessage,
			OutputDir:     buildConfig.OutputDir,
			Datafiles:     buildConfig.Datafiles,
			Repos:         c.getReposForBuildConfig(buildConfig.Name),
		}

		return connect.NewResponse(response), nil
	} else {
		return nil, connect.NewError(connect.CodeNotFound, fmt.Errorf("build config %s not found", req.Msg.ConfigName))
	}
}

func (c *ClientApi) getReposForBuildConfig(buildConfigName string) []string {
	repos := []string{}

	for _, repo := range c.buildConfigs[buildConfigName].Repos {
		repos = append(repos, repo.URL)
	}

	return repos
}

// autoRebuild is called by the watcher when a data file changes
func (c *ClientApi) autoRebuild(configName string) {
	buildConfig, found := c.buildConfigs[configName]

	if !found {
		log.Errorf("Auto-rebuild failed: build config %s not found", configName)
		return
	}

	if !c.beginBuild() {
		log.Infof("Skipping automatic rebuild for %s: shutting down", configName)
		return
	}

	log.Infof("Starting automatic rebuild for config: %s", configName)

	updateChan := make(chan string)
	buildStatus := &generator.BuildStatus{}

	// Run the build in a goroutine; read updates concurrently so Generate does not block on the channel.
	go func() {
		defer c.builds.Done()

		buildStarted := time.Now()
		go generator.Generate(c.buildCtx, c.BaseOutputDir, buildConfig, buildStatus, updateChan)

		for update := range updateChan {
			log.Infof("[Auto-rebuild %s] %s", configName, update)
		}

		if buildStatus.IsError {
			log.Errorf("Auto-rebuild failed for %s: %s", configName, buildStatus.Message)
		} else {
			log.Infof("Auto-rebuild completed successfully for %s", configName)
		}

		c.recordBuildHistory(configName, buildStatus, buildConfig.OutputDir, true, time.Since(buildStarted))
	}()
}

// recordBuildHistory records a build completion in the history
func (c *ClientApi) recordBuildHistory(configName string, buildStatus *generator.BuildStatus, relativePath string, isAutoRebuild bool, duration time.Duration) {
	c.historyMu.Lock()
	defer c.historyMu.Unlock()

	// Limit history to last 50 entries per config
	const maxHistoryEntries = 50

	entry := &pb.BuildHistoryEntry{
		Timestamp:               time.Now().Unix(),
		Status:                  buildStatus.Message,
		IsError:                 buildStatus.IsError,
		OutputSizeHumanReadable: buildStatus.OutputSizeHumanReadable,
		IsAutoRebuild:           isAutoRebuild,
		DurationMs:              duration.Milliseconds(),
	}

	// Build the build URL
	if buildStatus.BuildUrlBase != "" {
		entry.BuildUrl = buildStatus.BuildUrlBase + "/" + relativePath
	} else {
		// Store relative path only - client will construct full URL
		// Note: BuildHistoryEntry doesn't have relativePath field, so we'll leave it empty
		// The frontend will need to construct it from the config if needed
		entry.BuildUrl = ""
	}

	// Initialize history slice if needed
	if c.buildHistory[configName] == nil {
		c.buildHistory[configName] = make([]*pb.BuildHistoryEntry, 0)
	}

	// Add to history (most recent first)
	c.buildHistory[configName] = append([]*pb.BuildHistoryEntry{entry}, c.buildHistory[configName]...)

	// Limit to maxHistoryEntries
	if len(c.buildHistory[configName]) > maxHistoryEntries {
		c.buildHistory[configName] = c.buildHistory[configName][:maxHistoryEntries]
	}

	if err := saveBuildHistory(c.historyPath, c.buildHistory); err != nil {
		log.Warnf("Failed to save build history to %s: %v", c.historyPath, err)
	}
}

// GetBuildHistory returns the build history for a specific config
func (c *ClientApi) GetBuildHistory(ctx context.Context, req *connect.Request[pb.GetBuildHistoryRequest]) (*connect.Response[pb.GetBuildHistoryResponse], error) {
	c.historyMu.RLock()
	defer c.historyMu.RUnlock()

	history, exists := c.buildHistory[req.Msg.ConfigName]
	if !exists {
		history = []*pb.BuildHistoryEntry{}
	}

	response := &pb.GetBuildHistoryResponse{
		Entries: history,
	}

	return connect.NewResponse(response), nil
}

// GetCurrentUser returns the logged-in user. Kept for API clients; the web UI uses GetStatus.
func (c *ClientApi) GetCurrentUser(ctx context.Context, req *connect.Request[pb.GetCurrentUserRequest]) (*connect.Response[pb.GetCurrentUserResponse], error) {
	response := &pb.GetCurrentUserResponse{}

	if au := authUser(ctx); au != nil {
		response.IsAuthenticated = true
		response.Username = au.User.Username
	}

	return connect.NewResponse(response), nil
}

func (c *ClientApi) GitPull(ctx context.Context, req *connect.Request[pb.GitPullRequest]) (*connect.Response[pb.GitPullResponse], error) {
	response := &pb.GitPullResponse{
		Success: false,
		Message: "",
	}

	// Check if git pull is possible
	if !buildconfigs.CanGitPull() {
		response.Message = "Build configurations directory is not a git repository"
		return connect.NewResponse(response), nil
	}

	// Perform git pull
	err := buildconfigs.GitPull()
	if err != nil {
		response.Message = fmt.Sprintf("Git pull failed: %v", err)
		log.Errorf("Git pull failed: %v", err)
		return connect.NewResponse(response), nil
	}

	// Reload build configs after successful pull
	c.buildConfigs = buildconfigs.ReadConfigFiles()

	// Update the watcher with the new build configs
	if c.dataWatcher != nil {
		err := c.dataWatcher.UpdateBuildConfigs(c.buildConfigs)
		if err != nil {
			log.Errorf("Failed to update data file watcher after git pull: %v", err)
		}
	}

	response.Success = true
	response.Message = "Git pull completed successfully. Build configurations reloaded."

	return connect.NewResponse(response), nil
}

func (c *ClientApi) ListDataFiles(ctx context.Context, req *connect.Request[pb.ListDataFilesRequest]) (*connect.Response[pb.ListDataFilesResponse], error) {
	response := &pb.ListDataFilesResponse{
		DataFiles:  []*pb.DataFile{},
		CanGitPull: buildconfigs.CanGitPull(),
	}

	// Collect all data files from all build configs
	dataFileMap := make(map[string]*pb.DataFile) // key: buildConfigName:datafileName

	for configName, buildConfig := range c.buildConfigs {
		if !buildConfigAllowed(ctx, configName) {
			continue
		}
		configDir := filepath.Dir(buildConfig.Path)
		for datafileName, datafilePath := range buildConfig.Datafiles {
			// Create unique key to avoid duplicates
			key := configName + ":" + datafileName

			dataFileMap[key] = &pb.DataFile{
				Name:            datafileName,
				Path:            datafilePath,
				BuildConfigName: configName,
				BuildConfigPath: configDir,
			}
		}
	}

	// Convert map to slice
	for _, df := range dataFileMap {
		response.DataFiles = append(response.DataFiles, df)
	}

	return connect.NewResponse(response), nil
}

// beginBuild registers a build so shutdown waits for it. It returns false once shutdown has
// started; callers that get true must call c.builds.Done() when the build finishes.
func (c *ClientApi) beginBuild() bool {
	c.shutdownMu.Lock()
	defer c.shutdownMu.Unlock()

	if c.shuttingDown {
		return false
	}

	c.builds.Add(1)
	return true
}

// Shutdown stops the data file watcher, refuses new builds, and waits for builds in progress
// to finish. If ctx expires first, remaining builds are cancelled.
func (c *ClientApi) Shutdown(ctx context.Context) {
	c.shutdownMu.Lock()
	c.shuttingDown = true
	c.shutdownMu.Unlock()

	c.stopWatcher()

	done := make(chan struct{})
	go func() {
		c.builds.Wait()
		close(done)
	}()

	select {
	case <-done:
		log.Info("All builds finished")
	case <-ctx.Done():
		c.cancelBuilds()
	}
}

func (c *ClientApi) stopWatcher() {
	if c.watcherCancel != nil {
		log.Info("Stopping data file watcher...")
		c.watcherCancel()
	}

	if c.dataWatcher != nil {
		c.dataWatcher.Stop()
	}
}

func (c *ClientApi) cancelBuilds() {
	log.Warn("Timed out waiting for builds to finish, cancelling them")
	if c.buildCancel != nil {
		c.buildCancel()
	}
}
