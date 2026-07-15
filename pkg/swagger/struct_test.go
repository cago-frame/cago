package swagger

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestParseDirHonorsBuildTags(t *testing.T) {
	dir := t.TempDir()
	require.NoError(t, os.WriteFile(filepath.Join(dir, "go.mod"), []byte("module example.com/buildtags\n\ngo 1.26.0\n"), 0o600))
	require.NoError(t, os.WriteFile(filepath.Join(dir, "included.go"), []byte("package fixture\n\ntype Included struct{}\n"), 0o600))
	require.NoError(t, os.WriteFile(filepath.Join(dir, "excluded.go"), []byte("//go:build ignore\n\npackage fixture\n\ntype Excluded struct{}\n"), 0o600))

	p := &parseStruct{Swagger: NewSwagger(dir)}
	assert.EqualError(t, p.parseDir(dir, "Excluded"), "从"+dir+"中找不到类型: Excluded")
}
