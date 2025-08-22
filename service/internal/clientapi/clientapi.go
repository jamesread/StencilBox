package clientapi

import (
	"context"
	"fmt"
	"os"
	"path/filepath"

	"gopkg.in/yaml.v3"

	"connectrpc.com/connect"
	pb "github.com/jamesread/StencilBox/gen/StencilBox/clientapi/v1"
	client "github.com/jamesread/StencilBox/gen/StencilBox/clientapi/v1/clientapi_pbconnect"
	"github.com/jamesread/StencilBox/internal/buildconfigs"
	"github.com/jamesread/StencilBox/internal/buildinfo"
	"github.com/jamesread/StencilBox/internal/generator"
	"github.com/jamesread/golure/pkg/dirs"
	log "github.com/sirupsen/logrus"
)

type ClientApi struct {
	buildConfigs  map[string]*buildconfigs.BuildConfig
	BaseOutputDir string
	templates     map[string]*Template

	client.StencilBoxApiServiceClient
}

type Template struct {
	Name             string
	Source           string
	Status           string
	DocumentationURL string
	Description      string
}

func NewServer() *ClientApi {
	api := &ClientApi{}
	api.buildConfigs = buildconfigs.ReadConfigFiles()
	api.BaseOutputDir, _ = filepath.Abs(findOutputDir())

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

func (c *ClientApi) StartBuild(ctx context.Context, req *connect.Request[pb.BuildRequest], srv *connect.ServerStream[pb.BuildUpdateResponse]) (error) {
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
			Name:         buildConfig.Name,
			Template:     buildConfig.Template,
			Filename:     buildConfig.Filename,
			Path:         filepath.Dir(buildConfig.Path),
			DatafilesPath:filepath.Dir(buildConfig.Path),
			InContainer:  inContainer(),
			ErrorMessage: buildConfig.ErrorMessage,
			OutputDir:    buildConfig.OutputDir,
			Datafiles:    buildConfig.Datafiles,
			Repos:        c.getReposForBuildConfig(buildConfig.Name),
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
