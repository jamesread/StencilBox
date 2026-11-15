package generator

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestTemplateLayers(t *testing.T) {
	layers, err := readTemplateLayers(filepath.Join("..", "..", "..", "templates", "status-dashboard"))
	require.NoError(t, err)
	assert.Equal(t, []string{"femtocrank"}, layers)

	none, err := readTemplateLayers(t.TempDir())
	require.NoError(t, err, "metadata.yaml is optional")
	assert.Empty(t, none)

	bad := t.TempDir()
	require.NoError(t, os.WriteFile(filepath.Join(bad, "metadata.yaml"), []byte("layers: [../../etc]\n"), 0o600))
	_, err = readTemplateLayers(bad)
	assert.Error(t, err, "layer names cannot be paths")
}

func TestCopyFemtocrankLayer(t *testing.T) {
	tools := t.TempDir()
	pkg := filepath.Join(tools, "node_modules", "femtocrank")
	require.NoError(t, os.MkdirAll(pkg, 0o750))
	require.NoError(t, os.WriteFile(filepath.Join(pkg, "style.css"), []byte("/* style */"), 0o600))
	require.NoError(t, os.WriteFile(filepath.Join(pkg, "dark.css"), []byte("/* dark */"), 0o600))
	t.Setenv("STENCILBOX_TOOLS_DIR", tools)

	out := t.TempDir()
	require.NoError(t, copyTemplateLayers(filepath.Join("..", "..", "..", "templates", "status-dashboard"), out))
	css, err := os.ReadFile(filepath.Join(out, "femtocrank.css"))
	require.NoError(t, err)
	assert.Equal(t, "/* style */", string(css))
	_, err = os.Stat(filepath.Join(out, "femtocrank-dark.css"))
	assert.NoError(t, err)
}

func TestCopyMissingLayerFails(t *testing.T) {
	tmpl := t.TempDir()
	require.NoError(t, os.WriteFile(filepath.Join(tmpl, "metadata.yaml"), []byte("layers: [no-such-layer]\n"), 0o600))
	assert.Error(t, copyTemplateLayers(tmpl, t.TempDir()))
}
