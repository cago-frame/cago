package analyzer

import (
	"fmt"
	"go/ast"
	"go/token"
	"go/types"
	"net/http"
	"path"
	"reflect"
	"strconv"
	"strings"

	"golang.org/x/tools/go/analysis"
)

const (
	muxMetaPath  = "github.com/cago-frame/cago/server/mux"
	contextPath  = "context"
	ginPath      = "github.com/gin-gonic/gin"
	databasePath = "github.com/cago-frame/cago/database/db"
	gormPath     = "gorm.io/gorm"
)

type role int

const (
	roleOther role = iota
	roleAPIRouter
	roleAPI
	roleController
	roleService
	roleRepository
	roleEntity
	roleMigrations
)

// New builds the Cago framework analyzer.
func New(settings Settings) *analysis.Analyzer {
	settings = settings.withDefaults()
	return &analysis.Analyzer{
		Name: "cagolint",
		Doc:  "checks Cago project layout, routing metadata and layer conventions",
		Run: func(pass *analysis.Pass) (any, error) {
			run(pass, settings)
			return nil, nil
		},
	}
}

func run(pass *analysis.Pass, settings Settings) {
	packageRole := classify(pass.Pkg.Path(), settings)
	checkDependencies(pass, packageRole, settings)

	switch packageRole {
	case roleAPI:
		checkAPI(pass)
	case roleController:
		checkPackageSuffix(pass, "_ctr", "CAGO4001")
		checkController(pass)
	case roleService:
		checkPackageSuffix(pass, "_svc", "CAGO5001")
		checkService(pass)
	case roleRepository:
		checkPackageSuffix(pass, "_repo", "CAGO6001")
		checkRepository(pass)
	case roleMigrations:
		checkMigrations(pass)
	}
}

func classify(pkgPath string, settings Settings) role {
	if strings.HasSuffix(pkgPath, ".test") || strings.Contains(pkgPath, "/mock") {
		return roleOther
	}
	switch {
	case pkgPath == strings.TrimSuffix(pkgPath, "/") && strings.HasSuffix(pkgPath, settings.APIDir):
		return roleAPIRouter
	case strings.Contains(pkgPath, settings.APIDir+"/"):
		return roleAPI
	case strings.Contains(pkgPath, settings.ControllerDir):
		return roleController
	case strings.Contains(pkgPath, settings.ServiceDir):
		return roleService
	case strings.Contains(pkgPath, settings.RepositoryDir):
		return roleRepository
	case strings.Contains(pkgPath, settings.EntityDir):
		return roleEntity
	case packageInDir(pkgPath, settings.MigrationsDir):
		return roleMigrations
	default:
		return roleOther
	}
}

func packageInDir(pkgPath, directory string) bool {
	directory = strings.TrimSuffix(directory, "/")
	return strings.HasSuffix(pkgPath, directory) || strings.Contains(pkgPath, directory+"/")
}

func checkDependencies(pass *analysis.Pass, packageRole role, settings Settings) {
	for _, file := range pass.Files {
		if generated(file) {
			continue
		}
		for _, spec := range file.Imports {
			importPath, err := strconv.Unquote(spec.Path.Value)
			if err != nil {
				continue
			}

			var rule, message string
			switch packageRole {
			case roleAPI:
				switch {
				case strings.Contains(importPath, settings.ControllerDir):
					rule, message = "CAGO2001", "业务 API 包不能依赖 controller"
				case strings.Contains(importPath, settings.ServiceDir):
					rule, message = "CAGO2002", "业务 API 包不能依赖 service"
				case strings.Contains(importPath, settings.RepositoryDir):
					rule, message = "CAGO2003", "业务 API 包不能依赖 repository"
				}
			case roleRepository:
				switch {
				case strings.Contains(importPath, settings.ControllerDir):
					rule, message = "CAGO2004", "repository 不能反向依赖 controller"
				case strings.Contains(importPath, settings.ServiceDir):
					rule, message = "CAGO2005", "repository 不能反向依赖 service"
				}
			case roleEntity:
				if strings.Contains(importPath, settings.ControllerDir) || strings.Contains(importPath, settings.ServiceDir) || strings.Contains(importPath, settings.RepositoryDir) {
					rule, message = "CAGO2006", "entity 不能依赖 controller、service 或 repository"
				}
			case roleController:
				if importPath == databasePath || strings.HasPrefix(importPath, "gorm.io/") {
					rule, message = "CAGO2009", "controller 不应直接访问数据库，请通过 service/repository"
				}
			}
			if rule != "" {
				pass.Reportf(spec.Pos(), "%s: %s", rule, message)
			}
		}
	}
}

func checkAPI(pass *analysis.Pass) {
	responses := make(map[string]bool)
	requests := make([]requestInfo, 0)
	for _, file := range pass.Files {
		if generated(file) {
			continue
		}
		for _, decl := range file.Decls {
			gen, ok := decl.(*ast.GenDecl)
			if !ok || gen.Tok != token.TYPE {
				continue
			}
			for _, rawSpec := range gen.Specs {
				spec, ok := rawSpec.(*ast.TypeSpec)
				if !ok {
					continue
				}
				if strings.HasSuffix(spec.Name.Name, "Response") {
					responses[spec.Name.Name] = true
				}
				structure, ok := spec.Type.(*ast.StructType)
				if !ok {
					if strings.HasSuffix(spec.Name.Name, "Request") {
						pass.Reportf(spec.Name.Pos(), "CAGO3001: 路由请求类型 %s 必须是 struct", spec.Name.Name)
					}
					continue
				}
				meta := findMeta(pass, structure)
				if meta == nil {
					if strings.HasSuffix(spec.Name.Name, "Request") {
						pass.Reportf(spec.Name.Pos(), "CAGO3002: 路由请求类型 %s 必须嵌入 mux.Meta", spec.Name.Name)
					}
					continue
				}
				info := requestInfo{name: spec.Name.Name, spec: spec, structure: structure, meta: meta, file: file}
				requests = append(requests, info)
				checkRouteTag(pass, info)
			}
		}
	}

	for _, request := range requests {
		if !strings.HasSuffix(request.name, "Request") {
			pass.Reportf(request.spec.Name.Pos(), "CAGO3003: 路由请求类型 %s 应以 Request 结尾", request.name)
			continue
		}
		response := strings.TrimSuffix(request.name, "Request") + "Response"
		if !responses[response] {
			message := fmt.Sprintf("CAGO3004: %s 缺少对应的 %s", request.name, response)
			fix := responseFix(pass, request.file, response)
			diagnostic := analysis.Diagnostic{Pos: request.spec.Name.Pos(), Message: message}
			if fix != nil {
				diagnostic.SuggestedFixes = []analysis.SuggestedFix{*fix}
			}
			pass.Report(diagnostic)
		}
	}
}

type requestInfo struct {
	name      string
	spec      *ast.TypeSpec
	structure *ast.StructType
	meta      *ast.Field
	file      *ast.File
}

func findMeta(pass *analysis.Pass, structure *ast.StructType) *ast.Field {
	for _, field := range structure.Fields.List {
		objectType := pass.TypesInfo.TypeOf(field.Type)
		named, ok := types.Unalias(objectType).(*types.Named)
		if !ok || named.Obj().Pkg() == nil {
			continue
		}
		if named.Obj().Pkg().Path() == muxMetaPath && named.Obj().Name() == "Meta" {
			return field
		}
	}
	return nil
}

func checkRouteTag(pass *analysis.Pass, request requestInfo) {
	if request.meta.Tag == nil {
		pass.Reportf(request.meta.Pos(), "CAGO3005: %s 的 mux.Meta 缺少 path", request.name)
		return
	}
	raw, err := strconv.Unquote(request.meta.Tag.Value)
	if err != nil {
		return
	}
	tag := reflect.StructTag(raw)
	pathValue := tag.Get("path")
	methodValue := tag.Get("method")

	normalizedPaths, pathProblems := normalizePaths(pathValue)
	normalizedMethods, methodProblems := normalizeMethods(methodValue)
	updatedRaw := raw
	if pathValue != normalizedPaths && normalizedPaths != "" {
		updatedRaw = strings.Replace(updatedRaw, "path:\""+pathValue+"\"", "path:\""+normalizedPaths+"\"", 1)
	}
	if methodValue != normalizedMethods && normalizedMethods != "" {
		updatedRaw = strings.Replace(updatedRaw, "method:\""+methodValue+"\"", "method:\""+normalizedMethods+"\"", 1)
	}
	if len(pathProblems) > 0 {
		reportTagProblem(pass, request.meta.Tag, "CAGO3005", strings.Join(pathProblems, "; "), raw, updatedRaw, "规范化路由标签")
	}
	if len(methodProblems) > 0 {
		fixedRaw := raw
		if len(pathProblems) == 0 {
			fixedRaw = updatedRaw
		}
		reportTagProblem(pass, request.meta.Tag, "CAGO3006", strings.Join(methodProblems, "; "), raw, fixedRaw, "规范化 method")
	}
	checkURIParams(pass, request, normalizedPaths)
}

func normalizePaths(value string) (string, []string) {
	if strings.TrimSpace(value) == "" {
		return value, []string{"path 不能为空"}
	}
	parts := strings.Split(value, ",")
	problems := make([]string, 0)
	for index, item := range parts {
		original := item
		item = strings.TrimSpace(item)
		if original != item {
			problems = append(problems, fmt.Sprintf("path %q 不应包含首尾空格", original))
		}
		if item == "" {
			problems = append(problems, "path 列表包含空项")
			parts[index] = item
			continue
		}
		if !strings.HasPrefix(item, "/") {
			problems = append(problems, fmt.Sprintf("path %q 必须以 / 开头", item))
			item = "/" + item
		}
		if strings.Contains(item, "?") {
			problems = append(problems, fmt.Sprintf("path %q 不应包含 query string", item))
		}
		if item != "/" && strings.Contains(item, "//") {
			problems = append(problems, fmt.Sprintf("path %q 包含重复 /", item))
		}
		parts[index] = item
	}
	return strings.Join(parts, ","), problems
}

func normalizeMethods(value string) (string, []string) {
	if strings.TrimSpace(value) == "" {
		return "", nil
	}
	valid := map[string]bool{
		http.MethodGet: true, http.MethodPost: true, http.MethodPut: true,
		http.MethodPatch: true, http.MethodDelete: true, http.MethodHead: true,
		http.MethodOptions: true, http.MethodConnect: true, http.MethodTrace: true,
	}
	parts := strings.Split(value, ",")
	problems := make([]string, 0)
	for index, item := range parts {
		original := strings.TrimSpace(item)
		upper := strings.ToUpper(original)
		if original == "" {
			parts[index] = original
			problems = append(problems, "method 列表包含空项")
			continue
		}
		if !valid[upper] {
			parts[index] = original
			problems = append(problems, fmt.Sprintf("method %q 不是合法 HTTP 方法", original))
		} else if original != upper || item != original {
			parts[index] = upper
			problems = append(problems, fmt.Sprintf("method %q 应规范为 %s", item, upper))
		} else {
			parts[index] = upper
		}
	}
	return strings.Join(parts, ","), problems
}

func reportTagProblem(pass *analysis.Pass, literal *ast.BasicLit, rule, message, raw, updatedRaw, fixMessage string) {
	diagnostic := analysis.Diagnostic{Pos: literal.Pos(), End: literal.End(), Message: rule + ": " + message}
	if raw != updatedRaw {
		diagnostic.SuggestedFixes = []analysis.SuggestedFix{{
			Message:   fixMessage,
			TextEdits: []analysis.TextEdit{{Pos: literal.Pos(), End: literal.End(), NewText: []byte("`" + updatedRaw + "`")}},
		}}
	}
	pass.Report(diagnostic)
}

func checkURIParams(pass *analysis.Pass, request requestInfo, paths string) {
	parameters := make(map[string]bool)
	for _, routePath := range strings.Split(paths, ",") {
		for _, segment := range strings.Split(routePath, "/") {
			if strings.HasPrefix(segment, ":") && len(segment) > 1 {
				parameters[strings.TrimPrefix(segment, ":")] = true
			}
		}
	}
	uriFields := make(map[string]*ast.Field)
	for _, field := range request.structure.Fields.List {
		if field.Tag == nil {
			continue
		}
		raw, err := strconv.Unquote(field.Tag.Value)
		if err != nil {
			continue
		}
		if uri := reflect.StructTag(raw).Get("uri"); uri != "" && uri != "-" {
			uriFields[uri] = field
		}
	}
	for parameter := range parameters {
		if uriFields[parameter] == nil {
			pass.Reportf(request.meta.Pos(), "CAGO3007: path 参数 :%s 缺少对应的 uri:%q 字段", parameter, parameter)
		}
	}
	for uri, field := range uriFields {
		if !parameters[uri] {
			pass.Reportf(field.Pos(), "CAGO3007: uri:%q 未出现在任何 route path 中", uri)
		}
	}
}

func responseFix(pass *analysis.Pass, file *ast.File, response string) *analysis.SuggestedFix {
	end := file.End()
	if end == token.NoPos {
		return nil
	}
	return &analysis.SuggestedFix{
		Message:   "生成空的 " + response,
		TextEdits: []analysis.TextEdit{{Pos: end, End: end, NewText: []byte("\n\ntype " + response + " struct {\n}\n")}},
	}
}

func checkPackageSuffix(pass *analysis.Pass, suffix, rule string) {
	expected := path.Base(pass.Pkg.Path())
	if !strings.HasSuffix(expected, suffix) || pass.Pkg.Name() != expected {
		pass.Reportf(pass.Files[0].Package, "%s: 包目录和 package 名应使用 %s 后缀，当前为 %s", rule, suffix, pass.Pkg.Name())
	}
}

func checkController(pass *analysis.Pass) {
	for _, file := range pass.Files {
		if generated(file) {
			continue
		}
		for _, decl := range file.Decls {
			function, ok := decl.(*ast.FuncDecl)
			if !ok || function.Recv == nil || function.Type.Params == nil || len(function.Type.Params.List) != 2 {
				continue
			}
			requestPointer, ok := pass.TypesInfo.TypeOf(function.Type.Params.List[1].Type).(*types.Pointer)
			if !ok {
				continue
			}
			requestNamed, ok := types.Unalias(requestPointer.Elem()).(*types.Named)
			if !ok || !strings.HasSuffix(requestNamed.Obj().Name(), "Request") {
				continue
			}
			if !validContext(pass.TypesInfo.TypeOf(function.Type.Params.List[0].Type)) {
				pass.Reportf(function.Type.Params.List[0].Pos(), "CAGO4002: handler %s 的第一个参数必须是 context.Context 或 *gin.Context", function.Name.Name)
			}
			checkHandlerResults(pass, function, requestNamed.Obj().Name())
			methodName := strings.TrimSuffix(requestNamed.Obj().Name(), "Request")
			if function.Name.Name != methodName {
				pass.Reportf(function.Name.Pos(), "CAGO4004: handler %s 应与请求类型匹配并命名为 %s", function.Name.Name, methodName)
			}
		}
	}
}

func validContext(value types.Type) bool {
	if named, ok := types.Unalias(value).(*types.Named); ok {
		return named.Obj().Pkg() != nil && named.Obj().Pkg().Path() == contextPath && named.Obj().Name() == "Context"
	}
	pointer, ok := types.Unalias(value).(*types.Pointer)
	if !ok {
		return false
	}
	named, ok := types.Unalias(pointer.Elem()).(*types.Named)
	return ok && named.Obj().Pkg() != nil && named.Obj().Pkg().Path() == ginPath && named.Obj().Name() == "Context"
}

func checkHandlerResults(pass *analysis.Pass, function *ast.FuncDecl, requestName string) {
	if function.Type.Results == nil || len(function.Type.Results.List) == 0 {
		return
	}
	results := flattenFields(function.Type.Results.List)
	if len(results) == 1 && isError(pass.TypesInfo.TypeOf(results[0].Type)) {
		return
	}
	if len(results) == 2 && isResponsePointer(pass.TypesInfo.TypeOf(results[0].Type), strings.TrimSuffix(requestName, "Request")+"Response") && isError(pass.TypesInfo.TypeOf(results[1].Type)) {
		return
	}
	pass.Reportf(function.Type.Results.Pos(), "CAGO4003: handler %s 只能返回空、error 或 (*%s, error)", function.Name.Name, strings.TrimSuffix(requestName, "Request")+"Response")
}

func flattenFields(fields []*ast.Field) []*ast.Field {
	result := make([]*ast.Field, 0, len(fields))
	for _, field := range fields {
		count := len(field.Names)
		if count == 0 {
			count = 1
		}
		for range count {
			result = append(result, field)
		}
	}
	return result
}

func isError(value types.Type) bool {
	return value != nil && types.Identical(value, types.Universe.Lookup("error").Type())
}

func isResponsePointer(value types.Type, expected string) bool {
	pointer, ok := types.Unalias(value).(*types.Pointer)
	if !ok {
		return false
	}
	named, ok := types.Unalias(pointer.Elem()).(*types.Named)
	return ok && named.Obj().Name() == expected
}

func checkService(pass *analysis.Pass) {
	module := strings.TrimSuffix(pass.Pkg.Name(), "_svc")
	exported := upperFirst(module) + "Svc"
	private := module + "Svc"
	scope := pass.Pkg.Scope()
	interfaceObject := scope.Lookup(exported)
	implementationObject := scope.Lookup(private)
	if interfaceObject == nil || implementationObject == nil {
		pass.Reportf(pass.Files[0].Package, "CAGO5002: service 应声明 %s 接口和私有实现 %s", exported, private)
		return
	}
	interfaceNamed, interfaceOK := types.Unalias(interfaceObject.Type()).(*types.Named)
	implementationNamed, implementationOK := types.Unalias(implementationObject.Type()).(*types.Named)
	if !interfaceOK || !implementationOK {
		return
	}
	iface, ok := interfaceNamed.Underlying().(*types.Interface)
	if !ok {
		pass.Reportf(interfaceObject.Pos(), "CAGO5002: %s 必须是 interface", exported)
		return
	}
	if !types.Implements(types.NewPointer(implementationNamed), iface) {
		pass.Reportf(implementationObject.Pos(), "CAGO5003: *%s 未完整实现 %s", private, exported)
	}
}

func checkRepository(pass *analysis.Pass) {
	for _, file := range pass.Files {
		if generated(file) {
			continue
		}
		ast.Inspect(file, func(node ast.Node) bool {
			call, ok := node.(*ast.CallExpr)
			if !ok {
				return true
			}
			selector, ok := call.Fun.(*ast.SelectorExpr)
			if !ok || selector.Sel.Name != "Default" || len(call.Args) != 0 {
				return true
			}
			object := pass.TypesInfo.Uses[selector.Sel]
			function, ok := object.(*types.Func)
			if !ok || function.Pkg() == nil || function.Pkg().Path() != databasePath {
				return true
			}
			contextName := enclosingContextName(pass, file, call.Pos())
			diagnostic := analysis.Diagnostic{Pos: call.Pos(), End: call.End(), Message: "CAGO6003: repository 数据库操作应使用 db.Ctx(ctx) 传播 context 和事务"}
			if contextName != "" {
				diagnostic.SuggestedFixes = []analysis.SuggestedFix{{
					Message:   "改用 db.Ctx(" + contextName + ")",
					TextEdits: []analysis.TextEdit{{Pos: selector.Sel.Pos(), End: call.End(), NewText: []byte("Ctx(" + contextName + ")")}},
				}}
			}
			pass.Report(diagnostic)
			return true
		})
	}
}

func checkMigrations(pass *analysis.Pass) {
	for _, file := range pass.Files {
		if generated(file) {
			continue
		}
		ast.Inspect(file, func(node ast.Node) bool {
			call, ok := node.(*ast.CallExpr)
			if !ok {
				return true
			}
			selector, ok := call.Fun.(*ast.SelectorExpr)
			if !ok || selector.Sel.Name != "AutoMigrate" {
				return true
			}
			function, ok := pass.TypesInfo.Uses[selector.Sel].(*types.Func)
			if !ok || function.Pkg() == nil || function.Pkg().Path() != gormPath {
				return true
			}
			pass.Reportf(call.Pos(), "CAGO7001: migration 不能使用 AutoMigrate，请使用确定性的 DDL 语句或 Migrator 具体方法")
			return true
		})
	}
}

func enclosingContextName(pass *analysis.Pass, file *ast.File, position token.Pos) string {
	for _, decl := range file.Decls {
		function, ok := decl.(*ast.FuncDecl)
		if !ok || function.Body == nil || position < function.Body.Pos() || position > function.Body.End() {
			continue
		}
		for _, field := range function.Type.Params.List {
			if validContext(pass.TypesInfo.TypeOf(field.Type)) && len(field.Names) == 1 {
				return field.Names[0].Name
			}
		}
	}
	return ""
}

func generated(file *ast.File) bool {
	for _, group := range file.Comments {
		if group.Pos() > file.Package {
			break
		}
		if strings.Contains(group.Text(), "Code generated") && strings.Contains(group.Text(), "DO NOT EDIT") {
			return true
		}
	}
	return false
}

func upperFirst(value string) string {
	if value == "" {
		return value
	}
	return strings.ToUpper(value[:1]) + value[1:]
}
