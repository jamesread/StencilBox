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
	c.rearmWatcher()
}

// rearmWatcher points the data file watcher at the current build configs.
func (c *ClientApi) rearmWatcher() {
	if c.dataWatcher == nil {
		return
	}
	if err := c.dataWatcher.UpdateBuildConfigs(c.buildConfigs); err != nil {
		log.WithError(err).Warn("Re-arming the data file watcher")
	}
}
func (c *ClientApi) CreateBuildConfig(ctx context.Context, req *connect.Request[pb.CreateBuildConfigRequest]) (*connect.Response[pb.CreateBuildConfigResponse], error) {
	template := req.Msg.GetTemplate()
	if err := validateTemplateName(template); err != nil {
		return nil, err
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

func validateTemplateName(template string) error {
	if _, ok := readTemplates()[template]; !ok || template == "" {
		return connect.NewError(connect.CodeInvalidArgument, fmt.Errorf("template %q not found", template))
	}
	return nil
}
func (c *ClientApi) CreateDataFile(ctx context.Context, req *connect.Request[pb.CreateDataFileRequest]) (*connect.Response[pb.CreateDataFileResponse], error) {
	name := req.Msg.GetDatafileName()
	if err := buildconfigs.ValidateDatafileName(name); err != nil {
		return nil, writeErr("create data file", err)
	}
	content, err := parseDataFileContent(req.Msg.GetContent())
	if err != nil {
		return nil, err
	}

	c.dataFilesMu.Lock()
	defer c.dataFilesMu.Unlock()

	bc, rel, abs, err := c.planDataFile(req.Msg, name)
	if err != nil {
		return nil, err
	}
	if err = writeAndRegisterDataFile(bc, name, rel, abs, content); err != nil {
		return nil, err
	}
	log.WithFields(log.Fields{"buildConfig": bc.Name, "datafile": name, "path": abs, "user": actorName(ctx)}).Info("Data file added")

	c.reloadBuildConfigs()
	return c.createDataFileResponse(abs)
}

// planDataFile checks the build config can take the data file, and resolves where it goes.
func (c *ClientApi) planDataFile(msg *pb.CreateDataFileRequest, name string) (*buildconfigs.BuildConfig, string, string, error) {
	bc, found := c.buildConfigs[msg.GetBuildConfigName()]
	if !found {
		return nil, "", "", connect.NewError(connect.CodeNotFound, fmt.Errorf("build config %s not found", msg.GetBuildConfigName()))
	}
	if _, taken := bc.Datafiles[name]; taken {
		return nil, "", "", connect.NewError(connect.CodeAlreadyExists, fmt.Errorf("build config %s already has a data file called %s", bc.Name, name))
	}
	rel := msg.GetPath()
	if rel == "" {
		rel = "data/" + buildconfigs.Slug(bc.Name) + "-" + buildconfigs.Slug(name) + ".yaml"
	}
	rel, abs, err := buildconfigs.ResolveDatafilePath(filepath.Dir(bc.Path), rel)
	if err != nil {
		return nil, "", "", writeErr("create data file", err)
	}
	return bc, rel, abs, nil
}

// writeAndRegisterDataFile writes the data file and names it in the build config, removing a
// file it created if the build config could not be updated.
func writeAndRegisterDataFile(bc *buildconfigs.BuildConfig, name, rel, abs string, content []byte) error {
	created, err := writeNewDataFile(abs, rel, content)
	if err != nil {
		return err
	}
	if err = buildconfigs.AddDatafile(bc.Path, name, rel); err != nil {
		if created {
			_ = os.Remove(abs)
		}
		return writeErr("add data file to build config", err)
	}
	return nil
}

func (c *ClientApi) createDataFileResponse(abs string) (*connect.Response[pb.CreateDataFileResponse], error) {
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

// parseDataFileContent checks data file content is small enough and valid YAML.
func parseDataFileContent(text string) ([]byte, error) {
	content := []byte(text)
	if len(content) > maxDataFileSize {
		return nil, connect.NewError(connect.CodeInvalidArgument, fmt.Errorf("data file is larger than %d bytes", maxDataFileSize))
	}
	var parsed any
	if err := yaml.Unmarshal(content, &parsed); err != nil {
		return nil, connect.NewError(connect.CodeInvalidArgument, fmt.Errorf("content is not valid YAML: %w", err))
	}
	return content, nil
}

// writeNewDataFile writes content to a new data file, or, when content is empty, checks an
// existing file can be reused. It reports whether it created the file.
func writeNewDataFile(abs, rel string, content []byte) (bool, error) {
	_, statErr := os.Stat(abs)
	reuse, err := checkDataFileTarget(statErr == nil, len(content) > 0, rel)
	if err != nil || reuse {
		return false, err
	}
	if err = os.MkdirAll(filepath.Dir(abs), 0o750); err != nil {
		return false, writeErr("create data directory", err)
	}
	if err = buildconfigs.WriteFileAtomic(abs, content); err != nil {
		return false, writeErr("write data file", err)
	}
	return true, nil
}

// checkDataFileTarget decides whether an existing file is reused, refused, or (when it does not
// exist) may be created.
func checkDataFileTarget(exists, hasContent bool, rel string) (bool, error) {
	switch {
	case exists && hasContent:
		return false, connect.NewError(connect.CodeAlreadyExists,
			fmt.Errorf("%s already exists; leave the content empty to use it, or choose another path", rel))
	case exists:
		return true, nil
	case !hasContent:
		return false, connect.NewError(connect.CodeInvalidArgument,
			fmt.Errorf("%s does not exist yet, so give it some content", rel))
	}
	return false, nil
}

func actorName(ctx context.Context) string {
	if au := authUser(ctx); au != nil {
		return au.User.Username
	}
	return "unknown"
}
