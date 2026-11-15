package clientapi

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sort"

	"connectrpc.com/connect"
	pb "github.com/jamesread/StencilBox/gen/StencilBox/clientapi/v1"
	"github.com/jamesread/StencilBox/internal/buildconfigs"
	log "github.com/sirupsen/logrus"
	"gopkg.in/yaml.v3"
)

// maxDataFileSize bounds UpdateDataFile, so one request cannot fill the disk.
const maxDataFileSize = 5 << 20

func sha256Hex(b []byte) string {
	sum := sha256.Sum256(b)
	return hex.EncodeToString(sum[:])
}

// dataFilePath resolves a data file the way builds and the watcher do: relative to the
// directory of the build config that names it. Only files a build config names are reachable.
func dataFilePath(bc *buildconfigs.BuildConfig, datafilePath string) string {
	return filepath.Join(filepath.Dir(bc.Path), datafilePath)
}

func (c *ClientApi) lookupDataFile(buildConfigName, datafileName string) (string, error) {
	bc, found := c.buildConfigs[buildConfigName]
	if !found {
		return "", connect.NewError(connect.CodeNotFound, fmt.Errorf("build config %s not found", buildConfigName))
	}
	datafile, found := bc.Datafiles[datafileName]
	if !found {
		return "", connect.NewError(connect.CodeNotFound, fmt.Errorf("data file %s not found in build config %s", datafileName, buildConfigName))
	}
	return dataFilePath(bc, datafile), nil
}

func (c *ClientApi) GetDataFile(ctx context.Context, req *connect.Request[pb.GetDataFileRequest]) (*connect.Response[pb.GetDataFileResponse], error) {
	absPath, err := c.lookupDataFile(req.Msg.GetBuildConfigName(), req.Msg.GetDatafileName())
	if err != nil {
		return nil, err
	}

	content, err := os.ReadFile(absPath)
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, fmt.Errorf("failed to read data file: %w", err))
	}

	return connect.NewResponse(&pb.GetDataFileResponse{
		Content:         string(content),
		Path:            absPath,
		BuildConfigName: req.Msg.GetBuildConfigName(),
		DatafileName:    req.Msg.GetDatafileName(),
		Sha256:          sha256Hex(content),
	}), nil
}

func (c *ClientApi) UpdateDataFile(ctx context.Context, req *connect.Request[pb.UpdateDataFileRequest]) (*connect.Response[pb.UpdateDataFileResponse], error) {
	content := []byte(req.Msg.GetContent())
	if len(content) > maxDataFileSize {
		return nil, connect.NewError(connect.CodeInvalidArgument, fmt.Errorf("data file is larger than %d bytes", maxDataFileSize))
	}
	var parsed any
	if err := yaml.Unmarshal(content, &parsed); err != nil {
		return nil, connect.NewError(connect.CodeInvalidArgument, fmt.Errorf("content is not valid YAML: %w", err))
	}

	absPath, err := c.lookupDataFile(req.Msg.GetBuildConfigName(), req.Msg.GetDatafileName())
	if err != nil {
		return nil, err
	}
	// Hand-written build configs may name files anywhere; StencilBox only writes inside the
	// build configs directory.
	configDir := filepath.Dir(c.buildConfigs[req.Msg.GetBuildConfigName()].Path)
	if !buildconfigs.InsideDir(configDir, absPath) {
		return nil, connect.NewError(connect.CodePermissionDenied,
			errors.New("this data file is outside the build configs directory, so it can only be changed on disk"))
	}

	c.dataFilesMu.Lock()
	defer c.dataFilesMu.Unlock()

	if expected := req.Msg.GetExpectedSha256(); expected != "" {
		current, err := os.ReadFile(absPath)
		if err != nil && !errors.Is(err, os.ErrNotExist) {
			return nil, connect.NewError(connect.CodeInternal, fmt.Errorf("failed to read data file: %w", err))
		}
		if sha256Hex(current) != expected {
			return nil, connect.NewError(connect.CodeFailedPrecondition, errors.New("the data file has changed since it was read; get it again and reapply your change"))
		}
	}

	if err := buildconfigs.WriteFileAtomic(absPath, content); err != nil {
		log.WithError(err).WithField("path", absPath).Error("Writing data file")
		return nil, connect.NewError(connect.CodeInternal, errors.New("failed to write data file"))
	}

	actor := "unknown"
	if au := authUser(ctx); au != nil {
		actor = au.User.Username
	}
	log.WithFields(log.Fields{"path": absPath, "user": actor, "bytes": len(content)}).Info("Data file updated")

	rebuilding := c.rebuildUsersOf(absPath)
	return connect.NewResponse(&pb.UpdateDataFileResponse{
		Sha256:                 sha256Hex(content),
		Path:                   absPath,
		RebuildingBuildConfigs: rebuilding,
	}), nil
}

// rebuildUsersOf re-arms the file watcher (the atomic rename replaced the watched file) and
// rebuilds every build config that uses the file, as an edit on disk would.
func (c *ClientApi) rebuildUsersOf(absPath string) []string {
	if c.dataWatcher != nil {
		if err := c.dataWatcher.UpdateBuildConfigs(c.buildConfigs); err != nil {
			log.WithError(err).Warn("Re-arming the data file watcher")
		}
	}

	names := []string{}
	for name, bc := range c.buildConfigs {
		for _, datafile := range bc.Datafiles {
			if dataFilePath(bc, datafile) == absPath {
				names = append(names, name)
				break
			}
		}
	}
	sort.Strings(names)
	for _, name := range names {
		c.autoRebuild(name)
	}
	return names
}
