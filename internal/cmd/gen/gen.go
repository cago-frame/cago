package gen

import (
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"path/filepath"

	"github.com/cago-frame/cago/internal/cmd/gen/utils"
	"github.com/cago-frame/cago/pkg/logger"
	"github.com/cago-frame/cago/pkg/swagger"
	"github.com/spf13/cobra"
	swaggen "github.com/swaggo/swag/gen"
	"go.uber.org/zap"
)

const (
	JSONBodyType       = "json"
	FormDataBodyType   = "form-data"
	XWWWFormURLEncoded = "x-www-form-urlencoded"
)

type Cmd struct {
	apiPath     string
	pkgName     string
	pkgPath     string
	defaultBody string
	swagger     bool
	projectPath string
	verbose     bool
	log         *zap.Logger
}

func NewGenCmd() *Cmd {
	return &Cmd{}
}

func (c *Cmd) Commands() []*cobra.Command {
	ret := &cobra.Command{
		Use:   "gen",
		Short: "读取api目录下的文件,生成controller、service和swagger文档",
		RunE:  c.gen,
	}
	ret.AddCommand(&cobra.Command{
		Use:   "gorm [table]",
		Short: "输入表名,生成对应的数据库操作,需要配置好数据库连接",
		RunE:  c.genDB,
		Args:  cobra.ExactArgs(1),
	})
	ret.AddCommand(&cobra.Command{
		Use:   "mongo [table]",
		Short: "输入表名,生成对应的数据库操作,mongodb无需配置数据库连接",
		RunE:  c.genMongo,
		Args:  cobra.ExactArgs(1),
	})
	ret.Flags().StringVarP(&c.apiPath, "dir", "d", "./internal/api", "api目录")
	ret.Flags().BoolVarP(&c.swagger, "swagger", "s", false, "在controller方法上生成Swagger注释")
	ret.Flags().BoolVarP(&c.verbose, "verbose", "v", false, "显示详细生成日志")
	return []*cobra.Command{ret}
}

func (c *Cmd) gen(cmd *cobra.Command, args []string) error {
	c.initLogger(cmd)
	c.defaultBody = JSONBodyType
	c.log.Info("开始生成代码", zap.String("api_dir", c.apiPath))
	var err error
	c.projectPath, err = filepath.Abs(".")
	if err != nil {
		return err
	}
	c.pkgPath, c.pkgName, err = utils.FindRootPkgName(c.apiPath)
	if err != nil {
		return err
	}
	if err := utils.ReadDir(c.apiPath, func(path string) error {
		c.log.Debug("扫描 API 文件", zap.String("file", path))
		return c.genFile(path)
	}); err != nil {
		return err
	}
	if c.swagger {
		c.log.Info("使用 Swaggo 生成 Swagger 文档")
		if err := swaggen.New().Build(c.swaggerConfig()); err != nil {
			return err
		}
		c.log.Info("代码与 Swagger 文档生成完成")
		return nil
	}
	// 生成swagger
	swagger := swagger.NewSwagger(c.apiPath)
	if err := swagger.Gen(); err != nil {
		return err
	}
	if err := swagger.Write(); err != nil {
		return err
	}
	c.log.Info("代码与 Swagger 文档生成完成")
	return nil
}

func (c *Cmd) swaggerConfig() *swaggen.Config {
	projectPath := c.projectPath
	if projectPath == "" {
		projectPath = c.pkgPath
	}
	apiPath := c.apiPath
	if !filepath.IsAbs(apiPath) {
		apiPath = filepath.Join(projectPath, apiPath)
	}
	mainAPIFile, err := filepath.Rel(projectPath, filepath.Join(apiPath, "router.go"))
	if err != nil {
		mainAPIFile = filepath.Join(apiPath, "router.go")
	}
	return &swaggen.Config{
		Debugger:           swagLogger{log: c.logger()},
		SearchDir:          projectPath,
		MainAPIFile:        mainAPIFile,
		OutputDir:          filepath.Join(projectPath, "docs"),
		OutputTypes:        []string{"go", "json", "yaml"},
		InstanceName:       "swagger",
		ParseDepth:         100,
		ParseDependency:    1,
		ParseInternal:      true,
		ParseGoList:        true,
		GeneratedTime:      false,
		OverridesFile:      swaggen.DefaultOverridesFile,
		CollectionFormat:   "csv",
		PropNamingStrategy: "camelcase",
	}
}

func (c *Cmd) initLogger(cmd *cobra.Command) {
	level := "info"
	if c.verbose {
		level = "debug"
	}
	c.log = logger.NewConsole(cmd.ErrOrStderr(), level)
	logger.SetLogger(c.log)
}

func (c *Cmd) logger() *zap.Logger {
	if c.log == nil {
		c.log = zap.NewNop()
	}
	return c.log
}

type swagLogger struct {
	log *zap.Logger
}

func (l swagLogger) Printf(format string, args ...interface{}) {
	l.log.Debug(fmt.Sprintf(format, args...))
}

// 解析生成文件
func (c *Cmd) genFile(filepath string) error {
	// ast解析并生成swagger文档
	f, err := parser.ParseFile(token.NewFileSet(), filepath, nil, parser.ParseComments)
	if err != nil {
		return err
	}
	for _, v := range f.Decls {
		// 解析带有mux.Meta的struct
		decl, ok := v.(*ast.GenDecl)
		if !ok {
			continue
		}
		if decl.Tok != token.TYPE {
			continue
		}
		for _, spec := range decl.Specs {
			typeSpec, ok := spec.(*ast.TypeSpec)
			if !ok || !isRouteType(typeSpec) {
				continue
			}
			routeField := findRouteField(typeSpec)
			if routeField == nil {
				continue
			}
			exist, err := c.genController(filepath, f, decl, typeSpec, routeField)
			if err != nil {
				return err
			}
			if !exist {
				if err := c.genService(filepath, f, decl, typeSpec); err != nil {
					return err
				}
			}
		}
	}
	// 读取service目录根据接口生成service
	return c.findService()
}

func isRouteType(typeSpec *ast.TypeSpec) bool {
	return findRouteField(typeSpec) != nil
}

func findRouteField(typeSpec *ast.TypeSpec) *ast.Field {
	structSpec, ok := typeSpec.Type.(*ast.StructType)
	if !ok {
		return nil
	}
	for _, field := range structSpec.Fields.List {
		expr, ok := field.Type.(*ast.SelectorExpr)
		if !ok || expr.Sel.Name != "Meta" {
			continue
		}
		pkg, ok := expr.X.(*ast.Ident)
		if ok && pkg.Name == "mux" {
			return field
		}
	}
	return nil
}
