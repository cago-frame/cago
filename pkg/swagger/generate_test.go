package swagger

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/go-openapi/spec"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"golang.org/x/tools/go/packages"
)

// writeRecursiveFixture 写一个最小 api 目录：TreeNode 自引用（Children []TreeNode），
// TreeNode 与 TreeEntry 互相引用（Entry *TreeEntry / Parent *TreeNode），且同一个类型被多处引用。
func writeRecursiveFixture(t *testing.T) string {
	t.Helper()
	dir := t.TempDir()
	files := map[string]string{
		"go.mod": "module example.com/recursive\n\ngo 1.24\n",
		"router.go": `package api

// Router 路由
// @title    递归
// @version  1.0
func Router() error { return nil }
`,
		"tree.go": `package api

type TreeResponse struct {
	Groups []TreeNode ` + "`json:\"groups\"`" + `
	Root   TreeNode   ` + "`json:\"root\"`" + `
	Entry  TreeEntry  ` + "`json:\"entry\"`" + `
}

type TreeNode struct {
	Label    string     ` + "`json:\"label\"`" + `
	Children []TreeNode ` + "`json:\"children\"`" + `
	Entry    *TreeEntry ` + "`json:\"entry\"`" + `
}

type TreeEntry struct {
	Name   string    ` + "`json:\"name\"`" + `
	Parent *TreeNode ` + "`json:\"parent\"`" + `
}
`,
	}
	for name, content := range files {
		require.NoError(t, os.WriteFile(filepath.Join(dir, name), []byte(content), 0o600))
	}
	return dir
}

// genWithin 在限定时间内跑完 gen；自引用类型曾让 gen 无限递归，测试不能跟着挂死。
func genWithin(t *testing.T, s *Swagger, d time.Duration) {
	t.Helper()
	done := make(chan error, 1)
	go func() { done <- s.gen() }()
	select {
	case err := <-done:
		require.NoError(t, err)
	case <-time.After(d):
		t.Fatalf("gen 在 %s 内没有结束（自引用类型无限展开？）", d)
	}
}

func refOf(s spec.Schema) string { return s.Ref.String() }

func TestGenSelfReferencingStruct(t *testing.T) {
	s := NewSwagger(writeRecursiveFixture(t))
	genWithin(t, s, 60*time.Second)

	defs := s.swagger.Definitions
	require.Contains(t, defs, "api.TreeNode")
	require.Contains(t, defs, "api.TreeEntry")

	children := defs["api.TreeNode"].Properties["children"]
	require.NotNil(t, children.Items)
	require.NotNil(t, children.Items.Schema)
	assert.Equal(t, "#/definitions/api.TreeNode", refOf(*children.Items.Schema), "自引用落成指向自己的 $ref")
	assert.Equal(t, "#/definitions/api.TreeEntry", refOf(defs["api.TreeNode"].Properties["entry"]))
	assert.Equal(t, "#/definitions/api.TreeNode", refOf(defs["api.TreeEntry"].Properties["parent"]), "互相引用同样落成 $ref")
	assert.Equal(t, []string{"object"}, []string(defs["api.TreeNode"].Type), "最终定义是完整的结构，不是占位")
	assert.Contains(t, defs["api.TreeNode"].Properties, "label")
}

func TestGenLoadsEachPackageOnce(t *testing.T) {
	s := NewSwagger(writeRecursiveFixture(t))
	calls := map[string]int{}
	s.loadPackages = func(cfg *packages.Config, patterns ...string) ([]*packages.Package, error) {
		for _, p := range patterns {
			calls[cfg.Dir+"|"+p]++
		}
		return packages.Load(cfg, patterns...)
	}
	genWithin(t, s, 60*time.Second)

	require.NotEmpty(t, calls)
	for key, n := range calls {
		assert.Equal(t, 1, n, "同一个包只该加载一次：%s", key)
	}
}

// 别的包里的类型再引用它同包的类型：要回到那个包的目录里找，而不是项目根目录。
func TestGenResolvesSamePackageTypeInsideImportedPackage(t *testing.T) {
	root := t.TempDir()
	files := map[string]string{
		"go.mod": "module example.com/nested\n\ngo 1.24\n",
		"api/router.go": `package api

// Router 路由
// @title    嵌套
// @version  1.0
func Router() error { return nil }
`,
		"api/user.go": `package api

import "example.com/nested/model"

type UserResponse struct {
	User model.User ` + "`json:\"user\"`" + `
}
`,
		"model/user.go": `package model

type User struct {
	Profile Profile ` + "`json:\"profile\"`" + `
}

type Profile struct {
	Name string ` + "`json:\"name\"`" + `
}
`,
	}
	for name, content := range files {
		require.NoError(t, os.MkdirAll(filepath.Dir(filepath.Join(root, name)), 0o755))
		require.NoError(t, os.WriteFile(filepath.Join(root, name), []byte(content), 0o600))
	}

	s := NewSwagger(filepath.Join(root, "api"))
	genWithin(t, s, 60*time.Second)

	defs := s.swagger.Definitions
	require.Contains(t, defs, "model.User")
	require.Contains(t, defs, "model.Profile")
	assert.Equal(t, "#/definitions/model.Profile", refOf(defs["model.User"].Properties["profile"]))
}

// encoding/json 的编码口径：RawMessage 是任意 JSON，[]byte 是 base64 字符串，byte / rune 是整数。
func TestGenJSONByteTypes(t *testing.T) {
	root := t.TempDir()
	files := map[string]string{
		"go.mod": "module example.com/bytes\n\ngo 1.24\n",
		"router.go": `package api

// Router 路由
// @title    字节
// @version  1.0
func Router() error { return nil }
`,
		"raw.go": `package api

import "encoding/json"

type RawResponse struct {
	Spec  json.RawMessage ` + "`json:\"spec\"`" + `
	Blob  []byte          ` + "`json:\"blob\"`" + `
	One   byte            ` + "`json:\"one\"`" + `
	Glyph rune            ` + "`json:\"glyph\"`" + `
}
`,
	}
	for name, content := range files {
		require.NoError(t, os.WriteFile(filepath.Join(root, name), []byte(content), 0o600))
	}

	s := NewSwagger(root)
	genWithin(t, s, 60*time.Second)

	props := s.swagger.Definitions["api.RawResponse"].Properties
	assert.Equal(t, "#/definitions/json.RawMessage", refOf(props["spec"]))
	assert.Equal(t, []string{"object"}, []string(s.swagger.Definitions["json.RawMessage"].Type))
	assert.Equal(t, []string{"string"}, []string(props["blob"].Type))
	assert.Equal(t, "byte", props["blob"].Format)
	assert.Equal(t, []string{"integer"}, []string(props["one"].Type))
	assert.Equal(t, []string{"integer"}, []string(props["glyph"].Type))
}

// PATCH 路由进文档；字段名去掉 omitempty 等选项，json:"-" 的字段不出现。
func TestGenPatchRouteAndJSONTagOptions(t *testing.T) {
	root := t.TempDir()
	files := map[string]string{
		"go.mod": "module example.com/patch\n\ngo 1.24\n",
		"router.go": `package api

// Router 路由
// @title    补丁
// @version  1.0
func Router() error { return nil }
`,
		"item.go": `package api

import "github.com/cago-frame/cago/server/mux"

// PatchItemRequest 改一项
type PatchItemRequest struct {
	mux.Meta ` + "`path:\"/items/:id\" method:\"PATCH\"`" + `
	ID   int64  ` + "`uri:\"id\"`" + `
	Name string ` + "`json:\"name\"`" + `
}

type PatchItemResponse struct {
	Note   string ` + "`json:\"note,omitempty\"`" + `
	Secret string ` + "`json:\"-\"`" + `
}
`,
	}
	for name, content := range files {
		require.NoError(t, os.WriteFile(filepath.Join(root, name), []byte(content), 0o600))
	}

	s := NewSwagger(root)
	genWithin(t, s, 60*time.Second)

	item, ok := s.swagger.Paths.Paths["/items/{id}"]
	require.True(t, ok, "PATCH 路由要进文档")
	require.NotNil(t, item.Patch)

	props := s.swagger.Definitions["api.PatchItemResponse"].Properties
	assert.Contains(t, props, "note")
	assert.NotContains(t, props, "note,omitempty")
	assert.NotContains(t, props, "-")
	assert.NotContains(t, props, "Secret")
}
