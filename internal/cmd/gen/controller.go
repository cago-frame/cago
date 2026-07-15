package gen

import (
	"go/ast"
	"go/parser"
	"go/token"
	"net/http"
	"os"
	"path"
	"path/filepath"
	"strings"

	"github.com/cago-frame/cago/internal/cmd/gen/utils"
	"go.uber.org/zap"
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
			return false, c.regenController(ctrlFile, f, decl, apiFile, specs, routeField)
		}
		return false, err
	}
	// 存在则判断是否需要添加新方法
	data, err := os.ReadFile(ctrlFile) //nolint:gosec // G304
	if err != nil {
		return false, err
	}
	if strings.Contains(string(data), " "+strings.TrimSuffix(specs.Name.Name, "Request")+"(") {
		if c.swagger {
			return true, c.ensureSwaggerComment(ctrlFile, data, f, decl, specs, routeField)
		}
		return true, nil
	}
	// 生成函数
	funcTpl := c.genCtrlFunc(ctrlFile, f, decl, specs, routeField)
	data = append(data, []byte(funcTpl)...)
	return false, utils.WriteGoFile(ctrlFile, data)
}

// 重新生成controller
func (c *Cmd) regenController(ctrlFile string, f *ast.File, decl *ast.GenDecl,
	apiFile string, specs *ast.TypeSpec, routeField *ast.Field) error {
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

	c.logger().Info("创建 controller", zap.String("file", ctrlFile), zap.String("controller", ctrlName))

	data += c.genCtrlFunc(ctrlFile, f, decl, specs, routeField)

	return utils.WriteGoFile(ctrlFile, []byte(data))
}

func (c *Cmd) genCtrlFunc(ctrlFile string, f *ast.File, decl *ast.GenDecl, specs *ast.TypeSpec, routeField *ast.Field) string {
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
	if c.swagger {
		funcTpl = "\n" + c.swaggerComment(f, decl, specs, routeField) + strings.TrimPrefix(funcTpl, "\n")
	}
	c.logger().Debug("生成 controller 方法", zap.String("controller", ctrlName), zap.String("method", funcName))
	return funcTpl
}

func (c *Cmd) ensureSwaggerComment(ctrlFile string, data []byte, apiFile *ast.File, decl *ast.GenDecl,
	typeSpec *ast.TypeSpec, routeField *ast.Field) error {
	fileSet := token.NewFileSet()
	controller, err := parser.ParseFile(fileSet, ctrlFile, data, parser.ParseComments)
	if err != nil {
		return err
	}
	methodName := strings.TrimSuffix(typeSpec.Name.Name, "Request")
	for _, controllerDecl := range controller.Decls {
		function, ok := controllerDecl.(*ast.FuncDecl)
		if !ok || function.Recv == nil || function.Name.Name != methodName {
			continue
		}
		if function.Doc != nil && strings.Contains(function.Doc.Text(), "@Router ") {
			return nil
		}
		offset := fileSet.Position(function.Pos()).Offset
		comment := c.swaggerComment(apiFile, decl, typeSpec, routeField)
		updated := make([]byte, 0, len(data)+len(comment))
		updated = append(updated, data[:offset]...)
		updated = append(updated, comment...)
		updated = append(updated, data[offset:]...)
		return utils.WriteGoFile(ctrlFile, updated)
	}
	return nil
}

func (c *Cmd) swaggerComment(file *ast.File, decl *ast.GenDecl, typeSpec *ast.TypeSpec, routeField *ast.Field) string {
	description := utils.GetTypeComment(decl, typeSpec)
	if description == "" {
		description = strings.TrimSuffix(typeSpec.Name.Name, "Request")
	}
	tag := ""
	if routeField.Tag != nil {
		tag = strings.Trim(routeField.Tag.Value, "`")
	}
	paths := strings.Split(utils.ParseTag(tag, "path"), ",")
	methods := strings.Split(utils.ParseTag(tag, "method"), ",")
	contentType := utils.ParseTag(tag, "contentType")
	if contentType == "" {
		contentType = JSONBodyType
	}

	requestName := "api." + typeSpec.Name.Name
	responseName := "api." + strings.TrimSuffix(typeSpec.Name.Name, "Request") + "Response"
	paramType := "body"
	for _, method := range methods {
		if strings.EqualFold(method, http.MethodGet) {
			paramType = "query"
			break
		}
	}

	var result strings.Builder
	result.WriteString("// @Summary " + description + "\n")
	result.WriteString("// @Description " + description + "\n")
	result.WriteString("// @Tags " + file.Name.Name + "\n")
	result.WriteString("// @Accept " + swaggerMimeType(contentType) + "\n")
	result.WriteString("// @Produce json\n")
	if structType, ok := typeSpec.Type.(*ast.StructType); ok {
		for _, field := range structType.Fields.List {
			if field.Tag == nil {
				continue
			}
			fieldTag := strings.Trim(field.Tag.Value, "`")
			name := strings.Split(utils.ParseTag(fieldTag, "uri"), ",")[0]
			if name == "" || name == "-" {
				continue
			}
			description := utils.GetFieldComment(field)
			if description == "" {
				description = name
			}
			description = strings.ReplaceAll(description, `"`, `'`)
			result.WriteString("// @Param " + name + " path " + swaggerParamType(field.Type) + " true \"" + description + "\"\n")
		}
	}
	result.WriteString("// @Param request " + paramType + " " + requestName + " true \"请求参数\"\n")
	result.WriteString("// @Success 200 {object} " + responseName + "\n")
	for _, routePath := range paths {
		routePath = strings.TrimSpace(routePath)
		if routePath == "" {
			continue
		}
		routePath = routePathForSwagger(routePath)
		for _, method := range methods {
			method = strings.ToLower(strings.TrimSpace(method))
			if method == "" {
				method = strings.ToLower(http.MethodGet)
			}
			result.WriteString("// @Router " + routePath + " [" + method + "]\n")
		}
	}
	return result.String()
}

func swaggerParamType(expr ast.Expr) string {
	switch value := expr.(type) {
	case *ast.Ident:
		switch value.Name {
		case "string":
			return "string"
		case "bool":
			return "boolean"
		case "float32", "float64":
			return "number"
		case "int", "int8", "int16", "int32", "int64", "uint", "uint8", "uint16", "uint32", "uint64":
			return "integer"
		}
	case *ast.ArrayType:
		return "array"
	}
	return "string"
}

func swaggerMimeType(contentType string) string {
	switch contentType {
	case FormDataBodyType:
		return "mpfd"
	case XWWWFormURLEncoded:
		return "x-www-form-urlencoded"
	default:
		return "json"
	}
}

func routePathForSwagger(routePath string) string {
	parts := strings.Split(routePath, "/")
	for i, part := range parts {
		if strings.HasPrefix(part, ":") {
			parts[i] = "{" + strings.TrimPrefix(part, ":") + "}"
		}
	}
	return strings.Join(parts, "/")
}
