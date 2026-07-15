package gen

import (
	"bytes"
	"go/ast"
	"go/parser"
	"go/token"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestGenCommandSwaggerFlag(t *testing.T) {
	cmd := NewGenCmd()
	command := cmd.Commands()[0]
	flag := command.Flags().Lookup("swagger")
	require.NotNil(t, flag)
	require.Equal(t, "s", flag.Shorthand)
	require.Equal(t, "false", flag.DefValue)
}

func TestInitLoggerDefaultsToInfo(t *testing.T) {
	var output bytes.Buffer
	cmd := NewGenCmd()
	command := cmd.Commands()[0]
	command.SetErr(&output)

	cmd.initLogger(command)
	cmd.log.Debug("debug message")
	cmd.log.Info("info message")

	require.NotContains(t, output.String(), "debug message")
	require.Contains(t, output.String(), "info message")
}

func TestInitLoggerVerboseEnablesDebug(t *testing.T) {
	var output bytes.Buffer
	cmd := NewGenCmd()
	command := cmd.Commands()[0]
	cmd.verbose = true
	command.SetErr(&output)

	cmd.initLogger(command)
	cmd.log.Debug("debug message")

	require.Contains(t, output.String(), "debug message")
}

func TestSwaggerConfig(t *testing.T) {
	root := filepath.Join(string(filepath.Separator), "tmp", "project")
	cmd := &Cmd{
		apiPath:     filepath.Join("internal", "api"),
		pkgPath:     filepath.Dir(root),
		projectPath: root,
	}
	config := cmd.swaggerConfig()
	require.Equal(t, root, config.SearchDir)
	require.Equal(t, filepath.Join("internal", "api", "router.go"), config.MainAPIFile)
	require.Equal(t, filepath.Join(root, "docs"), config.OutputDir)
	require.Equal(t, []string{"go", "json", "yaml"}, config.OutputTypes)
	require.Equal(t, 1, config.ParseDependency)
	require.True(t, config.ParseInternal)
	require.True(t, config.ParseGoList)
	require.False(t, config.GeneratedTime)
}

func TestIsRouteType(t *testing.T) {
	tests := map[string]bool{
		"struct{}":                   false,
		"struct{ mux.Meta }":         true,
		"struct{ other.Meta }":       false,
		"struct{ *mux.Meta }":        false,
		"interface{ Execute() }":     false,
		"struct{ Value mux.Meta }":   true,
		"struct{ Value other.Meta }": false,
	}
	for source, expected := range tests {
		t.Run(source, func(t *testing.T) {
			expr, err := parser.ParseExpr(source)
			require.NoError(t, err)
			typeSpec := &ast.TypeSpec{Name: ast.NewIdent("Request"), Type: expr}
			require.Equal(t, expected, isRouteType(typeSpec))
		})
	}
}

func TestGenFileGeneratesControllerAndServiceIdempotently(t *testing.T) {
	root := t.TempDir()
	apiDir := filepath.Join(root, "internal", "api")
	apiFile := filepath.Join(apiDir, "user", "user.go")
	require.NoError(t, os.MkdirAll(filepath.Dir(apiFile), 0755))
	require.NoError(t, os.WriteFile(filepath.Join(root, "go.mod"), []byte("module example.com/project\n\ngo 1.24\n"), 0644))
	require.NoError(t, os.WriteFile(apiFile, []byte(`package user

import "github.com/cago-frame/cago/server/mux"

type (
	// CreateRequest creates a user.
	CreateRequest struct {
		mux.Meta `+"`path:\"/users\" method:\"POST\"`"+`
	}
	CreateResponse struct{}

	// DeleteRequest deletes a user.
	DeleteRequest struct {
		mux.Meta `+"`path:\"/users/:id\" method:\"DELETE\"`"+`
		// ID is the user identifier.
		ID int64 `+"`uri:\"id\"`"+`
	}
	DeleteResponse struct{}
)
`), 0644))

	cmd := &Cmd{
		apiPath: apiDir,
		pkgPath: root,
		pkgName: "example.com/project",
	}
	require.NoError(t, cmd.genFile(apiFile))
	require.NoError(t, cmd.genFile(apiFile), "generator should be safe to run repeatedly")

	controller := readTestFile(t, filepath.Join(root, "internal", "controller", "user_ctr", "user.go"))
	require.Contains(t, controller, "func (u *User) Create(")
	require.Contains(t, controller, "func (u *User) Delete(")
	require.Equal(t, 1, strings.Count(controller, "func (u *User) Create("))
	require.Equal(t, 1, strings.Count(controller, "func (u *User) Delete("))
	require.Contains(t, controller, `api "example.com/project/internal/api/user"`)
	require.Contains(t, controller, `"example.com/project/internal/service/user_svc"`)
	require.NotContains(t, controller, "@Router", "Swagger comments must be opt-in")
	requireValidGo(t, controller)

	service := readTestFile(t, filepath.Join(root, "internal", "service", "user_svc", "user.go"))
	require.Contains(t, service, "type UserSvc interface")
	require.Contains(t, service, "Create(ctx context.Context, req *api.CreateRequest) (*api.CreateResponse, error)")
	require.Contains(t, service, "Delete(ctx context.Context, req *api.DeleteRequest) (*api.DeleteResponse, error)")
	require.Equal(t, 1, strings.Count(service, "func (u *userSvc) Create("))
	require.Equal(t, 1, strings.Count(service, "func (u *userSvc) Delete("))
	requireValidGo(t, service)

	cmd.swagger = true
	require.NoError(t, cmd.genFile(apiFile), "Swagger comments should be added to existing controllers")
	require.NoError(t, cmd.genFile(apiFile), "Swagger comments should not be duplicated")

	controller = readTestFile(t, filepath.Join(root, "internal", "controller", "user_ctr", "user.go"))
	require.Contains(t, controller, "// @Summary creates a user.")
	require.Contains(t, controller, "// @Tags user")
	require.Contains(t, controller, "// @Param request body api.CreateRequest true \"请求参数\"")
	require.Contains(t, controller, "// @Success 200 {object} api.CreateResponse")
	require.Contains(t, controller, "// @Router /users [post]")
	require.Contains(t, controller, "// @Param id path integer true \"is the user identifier.\"")
	require.Contains(t, controller, "// @Router /users/{id} [delete]")
	require.Equal(t, 1, strings.Count(controller, "// @Router /users [post]"))
	require.Equal(t, 1, strings.Count(controller, "// @Router /users/{id} [delete]"))
	requireValidGo(t, controller)
}

func TestSwaggerHelpers(t *testing.T) {
	require.Equal(t, "/users/{id}/orders/{orderID}", routePathForSwagger("/users/:id/orders/:orderID"))
	require.Equal(t, "json", swaggerMimeType(JSONBodyType))
	require.Equal(t, "mpfd", swaggerMimeType(FormDataBodyType))
	require.Equal(t, "x-www-form-urlencoded", swaggerMimeType(XWWWFormURLEncoded))

	types := map[string]string{
		"string":    "string",
		"bool":      "boolean",
		"int64":     "integer",
		"float64":   "number",
		"[]string":  "array",
		"time.Time": "string",
	}
	for source, expected := range types {
		expr, err := parser.ParseExpr(source)
		require.NoError(t, err)
		require.Equal(t, expected, swaggerParamType(expr))
	}
}

func readTestFile(t *testing.T, file string) string {
	t.Helper()
	data, err := os.ReadFile(file) //nolint:gosec // test-controlled path
	require.NoError(t, err)
	return string(data)
}

func requireValidGo(t *testing.T, source string) {
	t.Helper()
	_, err := parser.ParseFile(token.NewFileSet(), "generated.go", source, parser.AllErrors)
	require.NoError(t, err)
}
