package clientapi

import (
	"connectrpc.com/connect"
	"context"
	"github.com/jamesread/golure/pkg/dirs"
	"github.com/jamesread/StencilBox/internal/buildconfigs"
	"github.com/jamesread/StencilBox/internal/generator"
	"github.com/jamesread/StencilBox/internal/buildinfo"
	pb "github.com/jamesread/StencilBox/gen/StencilBox/clientapi/v1"
	client "github.com/jamesread/StencilBox/gen/StencilBox/clientapi/v1/clientapi_pbconnect"
	log "github.com/sirupsen/logrus"
	"path/filepath"
)

type ClientApi struct {
	buildConfigs map[string]*buildconfigs.BuildConfig
	BaseOutputDir string
	templates []string

	client.StencilBoxApiServiceClient
}

func NewServer() *ClientApi {
	api := &ClientApi{}
	api.buildConfigs = buildconfigs.ReadConfigFiles()
	api.BaseOutputDir, _ = filepath.Abs(findOutputDir())

	return api;
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
	c.templates = readTemplates();

	return connect.NewResponse(response), nil
}

func readTemplates() []string {
	templates := make([]string, 0)

	res, err := filepath.Glob(filepath.Join(generator.FindTemplateDir(), "**", "index.html"))

	log.Infof("Reading templates from: %s", filepath.Join("templates", "**", "index.html"))

	if err != nil {
		log.Errorf("Error reading templates: %v", err)
		return templates
	}

	for _, fp := range res {
		templates = append(templates, filepath.Base(filepath.Dir(fp)))
	}

	return templates
}

func (c *ClientApi) StartBuild(ctx context.Context, req *connect.Request[pb.BuildRequest]) (*connect.Response[pb.BuildResponse], error) {
	response := &pb.BuildResponse{
		ConfigName: req.Msg.ConfigName,
		Status: "Build started",
	}

	buildConfig, found := c.buildConfigs[req.Msg.ConfigName]

	if found {
		buildstatus := generator.Generate(c.BaseOutputDir, buildConfig)

		response.Status = buildstatus.Message
		response.IsError = buildstatus.IsError
	}

	response.RelativePath = buildConfig.OutputDir
	response.Found = found

	return connect.NewResponse(response), nil
}

func (c *ClientApi) GetBuildConfigs(ctx context.Context, req *connect.Request[pb.GetBuildConfigsRequest]) (*connect.Response[pb.GetBuildConfigsResponse], error) {
	response := &pb.GetBuildConfigsResponse{}

	for name, bc := range c.buildConfigs {
		response.BuildConfigs = append(response.BuildConfigs, &pb.BuildConfig{
			Name:       name,
			Template:   bc.Template,
		})
	}

	return connect.NewResponse(response), nil
}

func (c *ClientApi) GetTemplates(ctx context.Context, req *connect.Request[pb.GetTemplatesRequest]) (*connect.Response[pb.GetTemplatesResponse], error) {
	response := &pb.GetTemplatesResponse{}

	for _, template := range c.templates {
		response.Templates = append(response.Templates, &pb.Template{
			Name: template,
			Source: "built-in",
			Status: "OK",
		})
	}

	return connect.NewResponse(response), nil
}

func (c *ClientApi) GetStatus(ctx context.Context, req *connect.Request[pb.GetStatusRequest]) (*connect.Response[pb.GetStatusResponse], error) {
	response := &pb.GetStatusResponse{}
	response.OutputPath = c.BaseOutputDir
	response.TemplatesPath = filepath.Join(generator.FindTemplateDir(), "templates")

	return connect.NewResponse(response), nil
}
