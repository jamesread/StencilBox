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
)

type ClientApi struct {
	buildConfigs map[string]*buildconfigs.BuildConfig
	BaseOutputDir string

	client.StencilBoxApiServiceClient
}

func NewServer() *ClientApi {
	api := &ClientApi{}
	api.buildConfigs = buildconfigs.ReadConfigFiles()
	api.BaseOutputDir = findOutputDir()

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

	for name, bc := range c.buildConfigs {
		response.BuildConfigs = append(response.BuildConfigs, &pb.BuildConfig{
			Name:       name,
			Template:   bc.Template,
		})
	}

	return connect.NewResponse(response), nil
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
