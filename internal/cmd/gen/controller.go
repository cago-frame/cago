package gen

import (
	"go/ast"
	"log"
	"os"
	"path"
	"path/filepath"
	"strings"

	"github.com/cago-frame/cago/internal/cmd/gen/utils"
)

const controllerHeaderTpl = `package {PkgName}

import (
	{ContextPkg}

	api "{ApiPkg}"
	"{ServicePkg}"
)

type {ControllerName} struct {
}

func New{ControllerName}() *{ControllerName} {
	return &{ControllerName}{}
}
`

const controllerFuncTpl = `
// {FuncName} {FuncDesc}
func ({SimpleName} *{ControllerName}) {FuncName}(ctx {Context}, req *api.{ApiRequest}) (*api.{ApiResponse}, error) {
	return {ServiceName}.{ControllerName}().{FuncName}({ContextParam}, req)
}
`

// 生成controller
func (c *Cmd) genController(apiFile string, f *ast.File, decl *ast.GenDecl, specs *ast.TypeSpec, routeField *ast.Field) (bool, error) {
	// 获取controller目录
	filename := strings.TrimPrefix(apiFile, c.apiPath)
	dir := path.Dir(filename)
	base := path.Base(filename)
	ctrlFile := path.Join(path.Dir(c.apiPath), "controller", dir+"_ctr", base)
	if err := os.MkdirAll(path.Dir(ctrlFile), 0755); err != nil {
		return false, err
	}
	// 生成controller
	_, err := os.Stat(ctrlFile)
	if err != nil {
		// 不存在重新生成
		if os.IsNotExist(err) {
			return false, c.regenController(ctrlFile, f, decl, apiFile, specs)
		}
		return false, err
	}
	// 存在则判断是否需要添加新方法
	data, err := os.ReadFile(ctrlFile)
	if err != nil {
		return false, err
	}
	if strings.Contains(string(data), " "+strings.TrimSuffix(specs.Name.Name, "Request")+"(") {
		return true, nil
	}
	// 生成函数
	funcTpl := c.genCtrlFunc(ctrlFile, decl, specs)
	data = append(data, []byte(funcTpl)...)
	return false, os.WriteFile(ctrlFile, data, 0644)
}

// 重新生成controller
func (c *Cmd) regenController(ctrlFile string, f *ast.File, decl *ast.GenDecl,
	apiFile string, specs *ast.TypeSpec) error {
	// 生成controller头部
	data := controllerHeaderTpl
	ctrlName := utils.FileNameToCamel(ctrlFile)
	data = strings.ReplaceAll(data, "{ControllerName}", ctrlName)
	data = strings.ReplaceAll(data, "{PkgName}", f.Name.Name+"_ctr")
	abs, err := filepath.Abs(apiFile)
	if err != nil {
		return err
	}
	data = strings.ReplaceAll(data, "{ContextPkg}", `"context"`)
	data = strings.ReplaceAll(data, "{ApiPkg}", strings.ReplaceAll(c.pkgName+strings.TrimPrefix(path.Dir(abs), c.pkgPath), "\\", "/"))
	// 获取service包名
	abs, err = filepath.Abs(c.apiPath)
	if err != nil {
		return err
	}
	separator := string(os.PathSeparator)
	servicePkg := c.pkgName + strings.TrimPrefix(path.Dir(abs), c.pkgPath) + "/service/" + strings.Split(path.Dir(apiFile), "internal"+separator+"api"+separator)[1] + "_svc"
	data = strings.ReplaceAll(data, "{ServicePkg}", strings.ReplaceAll(servicePkg, "\\", "/"))

	log.Printf("生成controller: %s", ctrlName)

	data += c.genCtrlFunc(ctrlFile, decl, specs)

	return os.WriteFile(ctrlFile, []byte(data), 0644)
}

func (c *Cmd) genCtrlFunc(ctrlFile string, decl *ast.GenDecl, specs *ast.TypeSpec) string {
	// 生成函数
	funcTpl := controllerFuncTpl
	pkgName := strings.TrimSuffix(path.Base(path.Dir(ctrlFile)), "_ctr")
	ctrlName := utils.FileNameToCamel(ctrlFile)
	funcTpl = strings.ReplaceAll(funcTpl, "{ControllerName}", ctrlName)
	funcTpl = strings.ReplaceAll(funcTpl, "{SimpleName}", strings.ToLower(ctrlName[0:1]))
	funcName := strings.TrimSuffix(specs.Name.Name, "Request")
	funcTpl = strings.ReplaceAll(funcTpl, "{FuncName}", funcName)
	funcTpl = strings.ReplaceAll(funcTpl, "{ApiRequest}", specs.Name.Name)
	funcTpl = strings.ReplaceAll(funcTpl, "{ApiResponse}", funcName+"Response")
	funcTpl = strings.ReplaceAll(funcTpl, "{ServiceName}", utils.LowerFirstChar(utils.ToCamel(pkgName))+"_svc")
	desc := utils.GetTypeComment(decl, specs)
	if desc == "" {
		desc = "TODO"
	}
	funcTpl = strings.ReplaceAll(funcTpl, "{Context}", "context.Context")
	funcTpl = strings.ReplaceAll(funcTpl, "{ContextParam}", "ctx")
	funcTpl = strings.ReplaceAll(funcTpl, "{FuncDesc}", desc)
	log.Printf("生成controller函数: %s", funcName)
	return funcTpl
}
