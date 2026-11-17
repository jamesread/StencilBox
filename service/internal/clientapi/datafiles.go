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
	content, err := parseDataFileContent(req.Msg.GetContent())
	if err != nil {
		return nil, err
	}
	absPath, err := c.writableDataFilePath(req.Msg.GetBuildConfigName(), req.Msg.GetDatafileName())
	if err != nil {
		return nil, err
	}

	c.dataFilesMu.Lock()
	defer c.dataFilesMu.Unlock()

	if err = checkExpectedSHA(absPath, req.Msg.GetExpectedSha256()); err != nil {
		return nil, err
	}
	if err = buildconfigs.WriteFileAtomic(absPath, content); err != nil {
		log.WithError(err).WithField("path", absPath).Error("Writing data file")
		return nil, connect.NewError(connect.CodeInternal, errors.New("failed to write data file"))
	}
	log.WithFields(log.Fields{"path": absPath, "user": actorName(ctx), "bytes": len(content)}).Info("Data file updated")

	return connect.NewResponse(&pb.UpdateDataFileResponse{
		Sha256:                 sha256Hex(content),
		Path:                   absPath,
		RebuildingBuildConfigs: c.rebuildUsersOf(absPath),
	}), nil
}

// writableDataFilePath resolves a data file, and refuses files StencilBox may not write.
// Hand-written build configs may name files anywhere; StencilBox only writes inside the
// build configs directory.
func (c *ClientApi) writableDataFilePath(buildConfigName, datafileName string) (string, error) {
	absPath, err := c.lookupDataFile(buildConfigName, datafileName)
	if err != nil {
		return "", err
	}
	if !buildconfigs.InsideDir(filepath.Dir(c.buildConfigs[buildConfigName].Path), absPath) {
		return "", connect.NewError(connect.CodePermissionDenied,
			errors.New("this data file is outside the build configs directory, so it can only be changed on disk"))
	}
	return absPath, nil
}

// checkExpectedSHA refuses a write when the file changed since the caller read it.
func checkExpectedSHA(absPath, expected string) error {
	if expected == "" {
		return nil
	}
	current, err := os.ReadFile(absPath)
	if err != nil && !errors.Is(err, os.ErrNotExist) {
		return connect.NewError(connect.CodeInternal, fmt.Errorf("failed to read data file: %w", err))
	}
	if sha256Hex(current) != expected {
		return connect.NewError(connect.CodeFailedPrecondition, errors.New("the data file has changed since it was read; get it again and reapply your change"))
	}
	return nil
}

// rebuildUsersOf re-arms the file watcher (the atomic rename replaced the watched file) and
// rebuilds every build config that uses the file, as an edit on disk would.
func (c *ClientApi) rebuildUsersOf(absPath string) []string {
	c.rearmWatcher()

	names := c.configsUsingFile(absPath)
	for _, name := range names {
		c.autoRebuild(name)
	}
	return names
}

// configsUsingFile returns the sorted names of the build configs that name the file.
func (c *ClientApi) configsUsingFile(absPath string) []string {
	names := []string{}
	for name, bc := range c.buildConfigs {
		if usesDataFile(bc, absPath) {
			names = append(names, name)
		}
	}
	sort.Strings(names)
	return names
}

func usesDataFile(bc *buildconfigs.BuildConfig, absPath string) bool {
	for _, datafile := range bc.Datafiles {
		if dataFilePath(bc, datafile) == absPath {
			return true
		}
	}
	return false
}
