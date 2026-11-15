package clientapi

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"

	"connectrpc.com/connect"
	pb "github.com/jamesread/StencilBox/gen/StencilBox/clientapi/v1"
	"github.com/jamesread/StencilBox/internal/buildconfigs"
	log "github.com/sirupsen/logrus"
	"gopkg.in/yaml.v3"
)

// writeErr maps buildconfigs validation errors to Connect codes, so their messages reach the UI.
func writeErr(op string, err error) error {
	switch {
	case errors.Is(err, buildconfigs.ErrInvalid):
		return connect.NewError(connect.CodeInvalidArgument, err)
	case errors.Is(err, buildconfigs.ErrExists):
		return connect.NewError(connect.CodeAlreadyExists, err)
	}
	log.WithError(err).Errorf("%s failed", op)
	return connect.NewError(connect.CodeInternal, fmt.Errorf("%s failed", op))
}

// reloadBuildConfigs re-reads the build configs directory after a write, and re-arms the data
// file watcher for any new data files.
func (c *ClientApi) reloadBuildConfigs() {
	c.buildConfigs = buildconfigs.ReadConfigFiles()
	if c.dataWatcher != nil {
		if err := c.dataWatcher.UpdateBuildConfigs(c.buildConfigs); err != nil {
			log.WithError(err).Warn("Re-arming the data file watcher")
		}
	}
}

func (c *ClientApi) CreateBuildConfig(ctx context.Context, req *connect.Request[pb.CreateBuildConfigRequest]) (*connect.Response[pb.CreateBuildConfigResponse], error) {
	template := req.Msg.GetTemplate()
	if _, ok := readTemplates()[template]; !ok || template == "" {
		return nil, connect.NewError(connect.CodeInvalidArgument, fmt.Errorf("template %q not found", template))
	}

	dir, err := buildconfigs.GetConfigDir()
	if err != nil {
		return nil, connect.NewError(connect.CodeFailedPrecondition, errors.New("there is no build configs directory to write to"))
	}

	c.dataFilesMu.Lock()
	defer c.dataFilesMu.Unlock()

	c.reloadBuildConfigs() // check names and output dirs against what is on disk now
	path, err := buildconfigs.Create(dir, buildconfigs.NewBuildConfig{
		Name:           req.Msg.GetConfigName(),
		Template:       template,
		OutputDir:      req.Msg.GetOutputDir(),
		BuildOnStartup: req.Msg.GetBuildOnStartup(),
	}, c.buildConfigs)
	if err != nil {
		return nil, writeErr("create build config", err)
	}
	c.reloadBuildConfigs()
	log.WithFields(log.Fields{"path": path, "user": actorName(ctx)}).Info("Build config created")

	created, err := c.GetBuildConfig(ctx, connect.NewRequest(&pb.GetBuildConfigRequest{ConfigName: req.Msg.GetConfigName()}))
	if err != nil {
		return nil, err
	}
	return connect.NewResponse(&pb.CreateBuildConfigResponse{BuildConfig: created.Msg.GetBuildConfig()}), nil
}

func (c *ClientApi) CreateDataFile(ctx context.Context, req *connect.Request[pb.CreateDataFileRequest]) (*connect.Response[pb.CreateDataFileResponse], error) {
	name := req.Msg.GetDatafileName()
	if err := buildconfigs.ValidateDatafileName(name); err != nil {
		return nil, writeErr("create data file", err)
	}
	content := []byte(req.Msg.GetContent())
	if len(content) > maxDataFileSize {
		return nil, connect.NewError(connect.CodeInvalidArgument, fmt.Errorf("data file is larger than %d bytes", maxDataFileSize))
	}
	var parsed any
	if err := yaml.Unmarshal(content, &parsed); err != nil {
		return nil, connect.NewError(connect.CodeInvalidArgument, fmt.Errorf("content is not valid YAML: %w", err))
	}

	c.dataFilesMu.Lock()
	defer c.dataFilesMu.Unlock()

	bc, found := c.buildConfigs[req.Msg.GetBuildConfigName()]
	if !found {
		return nil, connect.NewError(connect.CodeNotFound, fmt.Errorf("build config %s not found", req.Msg.GetBuildConfigName()))
	}
	if _, taken := bc.Datafiles[name]; taken {
		return nil, connect.NewError(connect.CodeAlreadyExists, fmt.Errorf("build config %s already has a data file called %s", bc.Name, name))
	}

	rel := req.Msg.GetPath()
	if rel == "" {
		rel = "data/" + buildconfigs.Slug(bc.Name) + "-" + buildconfigs.Slug(name) + ".yaml"
	}
	dir := filepath.Dir(bc.Path)
	rel, abs, err := buildconfigs.ResolveDatafilePath(dir, rel)
	if err != nil {
		return nil, writeErr("create data file", err)
	}

	created, err := writeNewDataFile(abs, rel, content)
	if err != nil {
		return nil, err
	}
	if err := buildconfigs.AddDatafile(bc.Path, name, rel); err != nil {
		if created {
			_ = os.Remove(abs)
		}
		return nil, writeErr("add data file to build config", err)
	}
	log.WithFields(log.Fields{"buildConfig": bc.Name, "datafile": name, "path": abs, "user": actorName(ctx)}).Info("Data file added")

	c.reloadBuildConfigs()
	onDisk, err := os.ReadFile(abs)
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, errors.New("failed to read the data file back"))
	}
	return connect.NewResponse(&pb.CreateDataFileResponse{
		Path:                   abs,
		Sha256:                 sha256Hex(onDisk),
		RebuildingBuildConfigs: c.rebuildUsersOf(abs),
	}), nil
}

// writeNewDataFile writes content to a new data file, or, when content is empty, checks an
// existing file can be reused. It reports whether it created the file.
func writeNewDataFile(abs, rel string, content []byte) (bool, error) {
	_, statErr := os.Stat(abs)
	exists := statErr == nil
	switch {
	case exists && len(content) > 0:
		return false, connect.NewError(connect.CodeAlreadyExists,
			fmt.Errorf("%s already exists; leave the content empty to use it, or choose another path", rel))
	case exists:
		return false, nil
	case len(content) == 0:
		return false, connect.NewError(connect.CodeInvalidArgument,
			fmt.Errorf("%s does not exist yet, so give it some content", rel))
	}
	if err := os.MkdirAll(filepath.Dir(abs), 0o750); err != nil {
		return false, writeErr("create data directory", err)
	}
	if err := buildconfigs.WriteFileAtomic(abs, content); err != nil {
		return false, writeErr("write data file", err)
	}
	return true, nil
}

func actorName(ctx context.Context) string {
	if au := authUser(ctx); au != nil {
		return au.User.Username
	}
	return "unknown"
}
