package gen

import (
	"go/ast"
	"go/parser"
	"go/token"

	"github.com/cago-frame/cago/internal/cmd/gen/utils"
	"github.com/cago-frame/cago/pkg/swagger"
	"github.com/spf13/cobra"
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
	return []*cobra.Command{ret}
}

func (c *Cmd) gen(cmd *cobra.Command, args []string) error {
	c.defaultBody = JSONBodyType
	var err error
	c.pkgPath, c.pkgName, err = utils.FindRootPkgName(c.apiPath)
	if err != nil {
		return err
	}
	if err := utils.ReadDir(c.apiPath, func(path string) error {
		return c.genFile(path)
	}); err != nil {
		return err
	}
	// 生成swagger
	swagger := swagger.NewSwagger(c.apiPath)
	if err := swagger.Gen(); err != nil {
		return err
	}
	return swagger.Write()
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
		typeSpec := decl.Specs[0].(*ast.TypeSpec)
		structSpec, ok := typeSpec.Type.(*ast.StructType)
		if !ok {
			continue
		}
		// 解析http.Meta
		var routeField *ast.Field
		for _, field := range structSpec.Fields.List {
			expr, ok := field.Type.(*ast.SelectorExpr)
			if !ok {
				continue
			}
			if expr.Sel.Name != "Meta" || expr.X.(*ast.Ident).Name != "mux" {
				continue
			}
			routeField = field
			break
		}
		if routeField == nil {
			continue
		}
		// 生成controller
		exist, err := c.genController(filepath, f, decl, typeSpec, routeField)
		if err != nil {
			return err
		}
		// 存在controller,跳过service生成
		if exist {
			continue
		}
		// 生成service接口
		if err := c.genService(filepath, f, decl, typeSpec); err != nil {
			return err
		}
	}
	// 读取service目录根据接口生成service
	return c.findService()
}
