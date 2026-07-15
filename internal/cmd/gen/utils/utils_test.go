package utils

import (
	"go/ast"
	"go/parser"
	"go/token"
	"os"
	"path/filepath"
	"sort"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestCaseHelpers(t *testing.T) {
	tests := []struct {
		input string
		upper string
		lower string
		camel string
	}{
		{input: "", upper: "", lower: "", camel: ""},
		{input: "user", upper: "User", lower: "user", camel: "User"},
		{input: "User", upper: "User", lower: "user", camel: "User"},
		{input: "id", upper: "Id", lower: "id", camel: "ID"},
		{input: "user_id", upper: "User_id", lower: "user_id", camel: "UserID"},
		{input: "user__profile", upper: "User__profile", lower: "user__profile", camel: "UserProfile"},
	}
	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			require.Equal(t, tt.upper, UpperFirstChar(tt.input))
			require.Equal(t, tt.lower, LowerFirstChar(tt.input))
			require.Equal(t, tt.camel, ToCamel(tt.input))
		})
	}
}

func TestGetType(t *testing.T) {
	tests := map[string]string{
		"string":                                 "string",
		"*example.Value":                         "*example.Value",
		"map[string][]*example.Value":            "map[string][]*example.Value",
		"chan<- int":                             "chan<- int",
		"<-chan error":                           "<-chan error",
		"interface{}":                            "interface{}",
		"interface{ Value() string }":            "interface{ Value() string }",
		"func(context.Context, ...string) error": "func(context.Context, ...string) error",
	}
	for source, expected := range tests {
		t.Run(source, func(t *testing.T) {
			expr, err := parser.ParseExpr(source)
			require.NoError(t, err)
			require.Equal(t, expected, GetType(expr))
		})
	}
}

func TestMethodFormatting(t *testing.T) {
	field := parseInterfaceMethod(t, "Execute(ctx context.Context, values ...string) (*Result, int, error)")
	funcType := field.Type.(*ast.FuncType)
	require.Equal(t, "ctx context.Context, values ...string", GetMethodParams(funcType.Params.List))
	require.Equal(t, "(*Result, int, error)", GetMethodResult(funcType.Results.List))
	require.Equal(t, "nil, 0, nil", GetMethodResultValues(funcType.Results.List))

	unnamed := parseInterfaceMethod(t, "Handle(context.Context, string) error")
	require.Equal(t, "context.Context, string", GetMethodParams(unnamed.Type.(*ast.FuncType).Params.List))
}

func TestCommentsAndTags(t *testing.T) {
	file, err := parser.ParseFile(token.NewFileSet(), "test.go", `package test
// Request creates a resource.
type Request struct {
	// Identifier is the resource ID.
	Identifier string `+"`json:\"id,omitempty\" form:\"identifier\"`"+`
	Name string // display name
}`, parser.ParseComments)
	require.NoError(t, err)
	decl := file.Decls[0].(*ast.GenDecl)
	typeSpec := decl.Specs[0].(*ast.TypeSpec)
	fields := typeSpec.Type.(*ast.StructType).Fields.List

	require.Equal(t, "creates a resource.", GetTypeComment(decl, typeSpec))
	require.Equal(t, "is the resource ID.", GetFieldComment(fields[0]))
	require.Equal(t, "display name", GetFieldComment(fields[1]))
	require.Equal(t, "id,omitempty", SwaggerName(fields[0]))
	require.Equal(t, "id,omitempty", ParseTag(fields[0].Tag.Value, "json"))
	require.Empty(t, ParseTag(fields[0].Tag.Value, "uri"))
}

func TestGetTypeCommentFromGroupedDeclaration(t *testing.T) {
	file, err := parser.ParseFile(token.NewFileSet(), "test.go", `package test
type (
	// CreateRequest creates a resource.
	CreateRequest struct{}
	// DeleteRequest deletes a resource.
	DeleteRequest struct{}
)`, parser.ParseComments)
	require.NoError(t, err)
	decl := file.Decls[0].(*ast.GenDecl)
	require.Equal(t, "creates a resource.", GetTypeComment(decl, decl.Specs[0].(*ast.TypeSpec)))
	require.Equal(t, "deletes a resource.", GetTypeComment(decl, decl.Specs[1].(*ast.TypeSpec)))
}

func TestFindRootPkgName(t *testing.T) {
	t.Run("finds parent module", func(t *testing.T) {
		root := t.TempDir()
		nested := filepath.Join(root, "internal", "api")
		require.NoError(t, os.MkdirAll(nested, 0755))
		require.NoError(t, os.WriteFile(filepath.Join(root, "go.mod"), []byte("// comment\nmodule\t\"example.com/project\"\n\ngo 1.24\n"), 0644))

		pkgPath, pkgName, err := FindRootPkgName(nested)
		require.NoError(t, err)
		require.Equal(t, root, pkgPath)
		require.Equal(t, "example.com/project", pkgName)
	})

	t.Run("rejects missing module declaration", func(t *testing.T) {
		root := t.TempDir()
		require.NoError(t, os.WriteFile(filepath.Join(root, "go.mod"), []byte("go 1.24\n"), 0644))
		_, _, err := FindRootPkgName(root)
		require.ErrorContains(t, err, "未找到 module 声明")
	})
}

func TestReadDir(t *testing.T) {
	root := t.TempDir()
	require.NoError(t, os.MkdirAll(filepath.Join(root, "nested"), 0755))
	for _, file := range []string{"a.go", "nested/b.go", "nested/readme.md"} {
		require.NoError(t, os.WriteFile(filepath.Join(root, file), []byte("test"), 0644))
	}

	var files []string
	require.NoError(t, ReadDir(root, func(file string) error {
		files = append(files, filepath.Base(file))
		return nil
	}))
	sort.Strings(files)
	require.Equal(t, []string{"a.go", "b.go"}, files)
}

func TestGeneratedFileWriting(t *testing.T) {
	t.Run("formats valid source", func(t *testing.T) {
		file := filepath.Join(t.TempDir(), "generated.go")
		require.NoError(t, WriteGoFile(file, []byte("package generated\nfunc value( )int{return 1}")))
		require.Equal(t, "package generated\n\nfunc value() int { return 1 }\n", readFile(t, file))
	})

	t.Run("does not overwrite existing file", func(t *testing.T) {
		file := filepath.Join(t.TempDir(), "generated.go")
		require.NoError(t, WriteFile(file, "package generated"))
		err := WriteFile(file, "package replaced")
		require.ErrorContains(t, err, "已经存在")
		require.Contains(t, readFile(t, file), "package generated")
	})

	t.Run("rejects invalid source without writing it", func(t *testing.T) {
		file := filepath.Join(t.TempDir(), "generated.go")
		err := WriteGoFile(file, []byte("package generated\nfunc"))
		require.ErrorContains(t, err, "格式化生成文件")
		_, statErr := os.Stat(file)
		require.ErrorIs(t, statErr, os.ErrNotExist)
	})
}

func parseInterfaceMethod(t *testing.T, method string) *ast.Field {
	t.Helper()
	expr, err := parser.ParseExpr("interface { " + method + " }")
	require.NoError(t, err)
	return expr.(*ast.InterfaceType).Methods.List[0]
}

func readFile(t *testing.T, file string) string {
	t.Helper()
	data, err := os.ReadFile(file) //nolint:gosec // test-controlled path
	require.NoError(t, err)
	return string(data)
}
