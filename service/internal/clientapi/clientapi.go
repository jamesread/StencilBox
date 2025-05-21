package clientapi

import (
	"connectrpc.com/connect"
	"context"
	"github.com/jamesread/StencilBox/internal/config"
	pb "github.com/jamesread/StencilBox/gen/StencilBox/clientapi/v1"
)

type ClientApi struct {
	cfg *config.Config
}

func NewServer(cfg *config.Config) *ClientApi {
	return &ClientApi{
		cfg: cfg,
	}
}

func (c *ClientApi) Init(ctx context.Context, req *connect.Request[pb.InitRequest]) (*connect.Response[pb.InitResponse], error) {
	return nil, nil
}
