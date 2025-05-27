package clientapi

import (
	"connectrpc.com/connect"
	"context"
	"github.com/jamesread/StencilBox/internal/config"
	"github.com/jamesread/StencilBox/internal/buildinfo"
	pb "github.com/jamesread/StencilBox/gen/StencilBox/clientapi/v1"
	client "github.com/jamesread/StencilBox/gen/StencilBox/clientapi/v1/clientapi_pbconnect"
)

type ClientApi struct {
	cfg *config.Config

	client.StencilBoxApiServiceClient
}

func NewServer(cfg *config.Config) *ClientApi {
	return &ClientApi{
		cfg: cfg,
	}
}

func (c *ClientApi) Init(ctx context.Context, req *connect.Request[pb.InitRequest]) (*connect.Response[pb.InitResponse], error) {
	response := &pb.InitResponse{
		Version: buildinfo.Version,
	}

	return connect.NewResponse(response), nil
}

func (c *ClientApi) StartBuild(ctx context.Context, req *connect.Request[pb.BuildRequest]) (*connect.Response[pb.BuildResponse], error) {
	response := &pb.BuildResponse{
		Status: "Build started",
	}

	return connect.NewResponse(response), nil
}
