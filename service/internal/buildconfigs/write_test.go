package buildconfigs

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestValidateOutputDir(t *testing.T) {
	existing := map[string]*BuildConfig{"home": {OutputDir: "homepage"}, "docs": {OutputDir: "sites/docs"}}
	for _, ok := range []string{"status", "sites/blog", "a.b-c_d"} {
		require.NoError(t, ValidateOutputDir(ok, existing), ok)
	}
	for _, bad := range []string{"", "..", "../x", "a/../b", "/abs", "a//b", ".hidden", "webui", "API", "mcp/x",
		"homepage", "homepage/sub", "sites", "sites/docs", "x_tmp", "a/b/c/d/e"} {
		require.ErrorIs(t, ValidateOutputDir(bad, existing), ErrInvalid, bad)
	}
	rootSite := map[string]*BuildConfig{"root": {OutputDir: ""}}
	require.Error(t, ValidateOutputDir("anything", rootSite), "a site built at the output root overlaps everything")
}

func TestCreateAndAddDatafile(t *testing.T) {
	dir := t.TempDir()
	path, err := Create(dir, NewBuildConfig{Name: "My Links", Template: "links-homepage", OutputDir: "links"}, nil)
	require.NoError(t, err)
	assert.Equal(t, filepath.Join(dir, "my-links.yaml"), path)

	_, err = Create(dir, NewBuildConfig{Name: "My Links", Template: "links-homepage", OutputDir: "other"}, nil)
	require.ErrorIs(t, err, ErrExists, "the file name is taken")

	raw, err := os.ReadFile(path)
	require.NoError(t, err)
	assert.Contains(t, string(raw), "\nname: My Links\ntemplate: links-homepage\noutputdir: links\n", "keys in reading order")

	require.NoError(t, AddDatafile(path, "links", "data/my-links-links.yaml"))
	require.ErrorIs(t, AddDatafile(path, "links", "data/x.yaml"), ErrExists)

	bc := readBuildConfig(path)
	require.Empty(t, bc.ErrorMessage)
	assert.Equal(t, "My Links", bc.Name)
	assert.Equal(t, "links", bc.OutputDir)
	assert.False(t, bc.BuildOnStartup())
	assert.Equal(t, map[string]string{"links": "data/my-links-links.yaml"}, bc.Datafiles)
}

func TestAddDatafileKeepsHandWrittenFile(t *testing.T) {
	path := filepath.Join(t.TempDir(), "hand.yaml")
	require.NoError(t, os.WriteFile(path, []byte("# my notes\nname: Hand\n\noutputdir: hand # where it goes\ntemplate: status-page\n"), 0o600))

	require.NoError(t, AddDatafile(path, "status", "data/status.yaml"))

	raw, err := os.ReadFile(path)
	require.NoError(t, err)
	assert.Contains(t, string(raw), "# my notes")
	assert.Contains(t, string(raw), "# where it goes")
	bc := readBuildConfig(path)
	require.Empty(t, bc.ErrorMessage)
	assert.Equal(t, "data/status.yaml", bc.Datafiles["status"])

	info, err := os.Stat(path)
	require.NoError(t, err)
	assert.Equal(t, os.FileMode(0o600), info.Mode().Perm())
}

func TestResolveDatafilePath(t *testing.T) {
	dir := t.TempDir()
	rel, abs, err := ResolveDatafilePath(dir, "./data/links.yaml")
	require.NoError(t, err)
	assert.Equal(t, "data/links.yaml", rel)
	assert.Equal(t, filepath.Join(dir, "data", "links.yaml"), abs)

	for _, bad := range []string{"", "../x.yaml", "/etc/x.yaml", "data/../../x.yaml", "data/x.sh", "data/.x.yaml", "data/a b.yaml"} {
		_, _, badErr := ResolveDatafilePath(dir, bad)
		require.ErrorIs(t, badErr, ErrInvalid, bad)
	}

	// A symlink inside the directory that points out of it is refused.
	outside := t.TempDir()
	require.NoError(t, os.Symlink(outside, filepath.Join(dir, "escape")))
	_, _, err = ResolveDatafilePath(dir, "escape/x.yaml")
	require.ErrorIs(t, err, ErrInvalid)
}
