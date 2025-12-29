package clientapi

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"sync"
	"time"

	"gopkg.in/yaml.v3"

	"connectrpc.com/connect"
	pb "github.com/jamesread/StencilBox/gen/StencilBox/clientapi/v1"
	client "github.com/jamesread/StencilBox/gen/StencilBox/clientapi/v1/clientapi_pbconnect"
	"github.com/jamesread/StencilBox/internal/buildconfigs"
	"github.com/jamesread/StencilBox/internal/buildinfo"
	"github.com/jamesread/StencilBox/internal/generator"
	"github.com/jamesread/StencilBox/internal/watcher"
	"github.com/jamesread/golure/pkg/dirs"
	auth "github.com/jamesread/httpauthshim"
	log "github.com/sirupsen/logrus"
)

type ClientApi struct {
	buildConfigs  map[string]*buildconfigs.BuildConfig
	BaseOutputDir string
	templates     map[string]*Template
	dataWatcher   *watcher.DataFileWatcher
	watcherCtx    context.Context
	watcherCancel context.CancelFunc
	buildHistory  map[string][]*pb.BuildHistoryEntry // config name -> history entries
	historyMu     sync.RWMutex                       // protects buildHistory
	authCtx       *auth.AuthShimContext              // authentication context (may be nil)

	client.StencilBoxApiServiceClient
}

type Template struct {
	Name             string
	Source           string
	Status           string
	DocumentationURL string
	Description      string
}

func NewServer(authCtx *auth.AuthShimContext) *ClientApi {
	api := &ClientApi{}
	api.authCtx = authCtx
	api.buildConfigs = buildconfigs.ReadConfigFiles()
	api.BaseOutputDir, _ = filepath.Abs(findOutputDir())
	api.buildHistory = make(map[string][]*pb.BuildHistoryEntry)

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

	return api
}

func findOutputDir() string {
	outputdir, err := dirs.GetFirstExistingDirectory("output", []string{
		"/var/www/StencilBox/",
		"../sb-output/",
	})

	if err != nil {
		log.Warnf("Did not find the output directory, using default ./sb-output")
		return "./sb-output"
	}

	return outputdir
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

	return connect.NewResponse(response), nil
}

type TemplateMetadata struct {
	DocumentationUrl string `yaml:"documentation_url"`
	Description      string `yaml:"description"`
}

func readTemplates() map[string]*Template {
	templates := make(map[string]*Template)

	res, err := filepath.Glob(filepath.Join(generator.FindTemplateDir(), "**", "index.html"))

	log.Infof("Reading templates from: %s", filepath.Join("templates", "**", "index.html"))

	if err != nil {
		log.Errorf("Error reading templates: %v", err)
		return templates
	}

	for _, fp := range res {
		templateName := filepath.Base(filepath.Dir(fp))

		metadataFile := filepath.Join(filepath.Dir(fp), "metadata.yaml")
		metadata, err := os.ReadFile(metadataFile)

		if err != nil {
			log.Errorf("Error reading metadata: %v", err)
			continue
		}

		metadataMap := TemplateMetadata{}
		err = yaml.Unmarshal(metadata, &metadataMap)

		if err != nil {
			log.Errorf("Error unmarshalling metadata: %v", err)
			continue
		}

		templates[templateName] = &Template{
			Name:             templateName,
			Source:           "built-in",
			Status:           "OK",
			DocumentationURL: metadataMap.DocumentationUrl,
			Description:      metadataMap.Description,
		}
	}

	return templates
}

func (c *ClientApi) StartBuild(ctx context.Context, req *connect.Request[pb.BuildRequest], srv *connect.ServerStream[pb.BuildUpdateResponse]) error {
	response := &pb.BuildUpdateResponse{
		ConfigName: req.Msg.ConfigName,
		Status:     "Build starting",
	}

	buildConfig, found := c.buildConfigs[req.Msg.ConfigName]

	updateChan := make(chan string)

	if !found {
		log.Errorf("Build config %s not found", req.Msg.ConfigName)
		response.IsError = true
		response.Status = "Build config not found"
		srv.Send(response)
		return connect.NewError(connect.CodeNotFound, fmt.Errorf("build config %s not found", req.Msg.ConfigName))
	}

	buildStatus := &generator.BuildStatus{}
	go generator.Generate(c.BaseOutputDir, buildConfig, buildStatus, updateChan)

	for update := range updateChan {
		log.Infof("Build update: %s", update)

		response.Status = update
		srv.Send(response)
	}

	log.Infof("Build completed for config %s", req.Msg.ConfigName)
	response.IsError = buildStatus.IsError
	response.BuildUrlBase = buildStatus.BuildUrlBase
	response.OutputSizeHumanReadable = buildStatus.OutputSizeHumanReadable
	response.BaseOutputDir = c.BaseOutputDir
	response.InContainer = inContainer()
	response.RelativePath = buildConfig.OutputDir
	response.Found = found
	response.IsComplete = true

	srv.Send(response)

	// Record build history
	c.recordBuildHistory(req.Msg.ConfigName, buildStatus, buildConfig.OutputDir, false)

	return nil
}

func (c *ClientApi) GetBuildConfigs(ctx context.Context, req *connect.Request[pb.GetBuildConfigsRequest]) (*connect.Response[pb.GetBuildConfigsResponse], error) {
	response := &pb.GetBuildConfigsResponse{}
	response.CanGitPull = buildconfigs.CanGitPull()

	for name, bc := range c.buildConfigs {
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
			BuildConfigs: c.getBuildConfigsForTemplate(template.Name),
		})
	}

	return connect.NewResponse(response), nil
}

func (c *ClientApi) getBuildConfigsForTemplate(templateName string) []string {
	buildConfigs := []string{}

	for _, bc := range c.buildConfigs {
		if bc.Template == templateName {
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
	dir, _ := buildconfigs.GetConfigDir()

	response := &pb.GetStatusResponse{
		InContainer:     inContainer(),
		OutputPath:      c.BaseOutputDir,
		TemplatesPath:   filepath.Join(generator.FindTemplateDir(), "templates"),
		BuildConfigsDir: dir,
	}

	return connect.NewResponse(response), nil
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
			BuildConfigs:     c.getBuildConfigsForTemplate(template.Name),
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
		return nil, connect.NewError(connect.CodeNotFound, fmt.Errorf("template not found"))
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

	log.Infof("Starting automatic rebuild for config: %s", configName)

	updateChan := make(chan string)
	buildStatus := &generator.BuildStatus{}

	// Run the build in a goroutine
	go func() {
		generator.Generate(c.BaseOutputDir, buildConfig, buildStatus, updateChan)

		// Log all updates
		for update := range updateChan {
			log.Infof("[Auto-rebuild %s] %s", configName, update)
		}

		if buildStatus.IsError {
			log.Errorf("Auto-rebuild failed for %s: %s", configName, buildStatus.Message)
		} else {
			log.Infof("Auto-rebuild completed successfully for %s", configName)
		}

		// Record build history
		c.recordBuildHistory(configName, buildStatus, buildConfig.OutputDir, true)
	}()
}

// recordBuildHistory records a build completion in the history
func (c *ClientApi) recordBuildHistory(configName string, buildStatus *generator.BuildStatus, relativePath string, isAutoRebuild bool) {
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

// GetCurrentUser returns the currently authenticated user
func (c *ClientApi) GetCurrentUser(ctx context.Context, req *connect.Request[pb.GetCurrentUserRequest]) (*connect.Response[pb.GetCurrentUserResponse], error) {
	response := &pb.GetCurrentUserResponse{
		IsAuthenticated: false,
		Username:        "",
	}

	// If auth is not enabled, return guest user
	if c.authCtx == nil {
		return connect.NewResponse(response), nil
	}

	// Extract http.Request from context (stored by withAuth middleware)
	httpReq, ok := ctx.Value("httpRequest").(*http.Request)
	if !ok || httpReq == nil {
		// If we can't get the request, return guest user
		return connect.NewResponse(response), nil
	}

	// Extract user from request
	user := c.authCtx.AuthFromHttpReq(httpReq)

	if user != nil && !user.IsGuest() {
		response.IsAuthenticated = true
		response.Username = user.Username
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

func (c *ClientApi) GetDataFile(ctx context.Context, req *connect.Request[pb.GetDataFileRequest]) (*connect.Response[pb.GetDataFileResponse], error) {
	response := &pb.GetDataFileResponse{}

	// Find the build config
	buildConfig, found := c.buildConfigs[req.Msg.BuildConfigName]
	if !found {
		return nil, connect.NewError(connect.CodeNotFound, fmt.Errorf("build config %s not found", req.Msg.BuildConfigName))
	}

	// Find the data file path
	datafilePath, found := buildConfig.Datafiles[req.Msg.DatafileName]
	if !found {
		return nil, connect.NewError(connect.CodeNotFound, fmt.Errorf("data file %s not found in build config %s", req.Msg.DatafileName, req.Msg.BuildConfigName))
	}

	// Resolve absolute path
	configDir := filepath.Dir(buildConfig.Path)
	absPath := filepath.Join(configDir, datafilePath)

	// Read the file content
	content, err := os.ReadFile(absPath)
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, fmt.Errorf("failed to read data file: %w", err))
	}

	response.Content = string(content)
	response.Path = absPath
	response.BuildConfigName = req.Msg.BuildConfigName
	response.DatafileName = req.Msg.DatafileName

	return connect.NewResponse(response), nil
}

// Shutdown cleans up resources
func (c *ClientApi) Shutdown() {
	if c.watcherCancel != nil {
		log.Info("Stopping data file watcher...")
		c.watcherCancel()
	}

	if c.dataWatcher != nil {
		c.dataWatcher.Stop()
	}
}
