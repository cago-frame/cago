package utils

import (
	"errors"
	"fmt"
	"go/ast"
	"io"
	"os"
	"path"
	"path/filepath"
	"strings"
	"text/template"
	"unicode"
)

// UpperFirstChar 首字母大写
func UpperFirstChar(str string) string {
	if len(str) == 0 {
		return str
	}
	return strings.ToUpper(str[0:1]) + str[1:]
}

// LowerFirstChar 首字母小写
func LowerFirstChar(str string) string {
	if len(str) == 0 {
		return str
	}
	return strings.ToLower(str[:1]) + str[1:]
}

// SwaggerName 获取swagger name
func SwaggerName(field *ast.Field) string {
	if field.Tag != nil {
		tag := field.Tag.Value
		name := ParseTag(tag, "json")
		if name != "" {
			return name
		}
		name = ParseTag(tag, "form")
		if name != "" {
			return name
		}
		name = ParseTag(tag, "uri")
		if name != "" {
			return name
		}
	}
	return field.Names[0].Name
}

// ToCamel 下划线转驼峰
func ToCamel(str string) string {
	if str == "id" {
		return "ID"
	}
	var result string
	for _, v := range strings.Split(str, "_") {
		if v == "id" {
			result += "ID"
		} else {
			result += strings.ToUpper(v[:1]) + v[1:]
		}
	}
	return result
}

func GetTypeComment(decl *ast.GenDecl, typeSpec *ast.TypeSpec) string {
	comment := ""
	if decl.Doc != nil {
		comment = decl.Doc.Text()
		comment = strings.TrimSpace(strings.TrimPrefix(comment, typeSpec.Name.Name))
	}
	return comment
}

func GetFieldComment(field *ast.Field) string {
	comment := ""
	if field.Doc != nil {
		comment = field.Doc.Text()
		comment = strings.TrimSpace(strings.TrimPrefix(comment, field.Names[0].Name))
	} else if field.Comment != nil {
		comment = field.Comment.Text()
		comment = strings.TrimSpace(strings.TrimPrefix(comment, "//"))
	}
	return comment
}

func GetMethodParams(field []*ast.Field) string {
	params := ""
	for _, param := range field {
		params += param.Names[0].Name + " " + GetType(param.Type) + ", "
	}
	return strings.TrimSuffix(params, ", ")
}

func GetMethodResult(field []*ast.Field) string {
	result := ""
	for _, param := range field {
		result += GetType(param.Type) + ", "
	}
	if len(field) > 1 {
		return "(" + strings.TrimSuffix(result, ", ") + ")"
	}
	return strings.TrimSuffix(result, ", ")
}

func GetMethodResultValues(field []*ast.Field) string {
	result := ""
	for _, param := range field {
		result += GetTypeDefaultVal(param.Type) + ", "
	}
	return strings.TrimSuffix(result, ", ")
}

func GetTypeDefaultVal(expr ast.Expr) string {
	switch expr := expr.(type) {
	case *ast.Ident:
		switch expr.Name {
		case "string":
			return `""`
		case "int", "int8", "int16", "int32", "int64", "uint", "uint8", "uint16", "uint32", "uint64",
			"float32", "float64", "complex64", "complex128":
			return "0"
		case "bool":
			return "false"
		}
		return "nil"
	case *ast.SelectorExpr:
		return "nil"
	case *ast.StarExpr:
		return "nil"
	case *ast.ArrayType:
		return "nil"
	case *ast.MapType:
		return "nil"
	default:
		return ""
	}
}

func GetType(expr ast.Expr) string {
	switch expr := expr.(type) {
	case *ast.Ident:
		return expr.Name
	case *ast.SelectorExpr:
		return expr.X.(*ast.Ident).Name + "." + expr.Sel.Name
	case *ast.StarExpr:
		return "*" + GetType(expr.X)
	case *ast.ArrayType:
		return "[]" + GetType(expr.Elt)
	case *ast.MapType:
		return "map[" + GetType(expr.Key) + "]" + GetType(expr.Value)
	default:
		return ""
	}
}

func ReadDir(path string, gen func(path string) error) error {
	dir, err := os.ReadDir(path)
	if err != nil {
		return err
	}
	for _, v := range dir {
		path := fmt.Sprintf("%s/%s", path, v.Name())
		if v.IsDir() {
			// 目录继续遍历
			if err := ReadDir(path, gen); err != nil {
				return err
			}
		} else {
			// 文件
			if !strings.HasSuffix(v.Name(), ".go") {
				continue
			}
			if err := gen(path); err != nil {
				return err
			}
		}
	}
	return nil
}

// FindRootPkgName 根据go.mod搜寻根包名
func FindRootPkgName(dir string) (string, string, error) {
	f, err := os.OpenFile(path.Join(dir, "./go.mod"), os.O_RDONLY, 0644) //nolint:gosec // G304
	if err != nil {
		if os.IsNotExist(err) {
			// 向上层继续搜索
			absDir, err := filepath.Abs(dir)
			if err != nil {
				return "", "", err
			}
			return FindRootPkgName(path.Dir(absDir))
		}
		return "", "", err
	}
	defer f.Close() //nolint:errcheck
	// 解析go.mod
	b, err := io.ReadAll(f)
	if err != nil {
		return "", "", err
	}
	// 解析出包名
	moduleName := strings.Split(string(b), "module ")[1]
	moduleName = strings.Split(moduleName, "\n")[0]
	pkgPath, err := filepath.Abs(dir)
	if err != nil {
		return "", "", err
	}
	return pkgPath, moduleName, nil
}

func PkgToPath(rootPkg, rootPkgName, pkgName string) (string, error) {
	// 去掉根包名
	pkgPath := strings.TrimPrefix(pkgName, rootPkgName)
	if pkgPath != pkgName {
		// 拼接路径
		return path.Join(rootPkg, pkgPath), nil
	}
	// 读取go.mod然后去GOPATH中寻找
	f, err := os.OpenFile(path.Join(rootPkg, "./go.mod"), os.O_RDONLY, 0644) //nolint:gosec // G304
	if err != nil {
		return "", err
	}
	defer f.Close() //nolint:errcheck
	// 解析go.mod
	b, err := io.ReadAll(f)
	if err != nil {
		return "", err
	}
	s := string(b)
	// 解析出包名
	findRootPkgName := pkgName
	pkgDir := ""
	for {
		index := strings.Index(s, findRootPkgName)
		if index == -1 {
			splits := strings.Split(findRootPkgName, "/")
			if len(splits) == 1 {
				return "", errors.New("找不到包名")
			}
			findRootPkgName = strings.Join(splits[:len(splits)-1], "/")
			pkgDir = splits[len(splits)-1] + "/" + pkgDir
		} else {
			// 获取包版本
			pkgVersion := s[index+len(findRootPkgName)+1:]
			pkgVersion = strings.Split(pkgVersion, "\n")[0]
			// 获取GOPATH
			gopath := os.Getenv("GOPATH")
			if gopath == "" {
				return "", errors.New("GOPATH为空")
			}
			// 大写转!小写
			for i, v := range findRootPkgName {
				if unicode.IsUpper(v) {
					findRootPkgName = findRootPkgName[:i] + "!" + string(v) + findRootPkgName[i+1:]
				}
			}
			// 拼接路径
			return path.Join(gopath, "pkg/mod", findRootPkgName+"@"+pkgVersion+"/", pkgDir), nil
		}
	}
}

func ParseTag(tag string, key string) string {
	keys := strings.Split(tag, key+":\"")
	if len(keys) != 2 {
		return ""
	}
	value := keys[1]
	value = strings.Split(value, "\"")[0]
	return value
}

func FileNameToCamel(filename string) string {
	return UpperFirstChar(ToCamel(strings.TrimSuffix(path.Base(filename), ".go")))
}

func ParseTemplate(tpl string, params ...map[string]interface{}) (string, error) {
	// 合并参数
	param := make(map[string]interface{})
	for _, p := range params {
		for k, v := range p {
			param[k] = v
		}
	}
	t := template.New("template")
	// 执行模板
	t = t.Funcs(template.FuncMap{
		"UpperFirstChar": UpperFirstChar,
		"ToCamel":        ToCamel,
		"LowerFirstChar": LowerFirstChar,
	})
	t, err := t.Parse(tpl)
	if err != nil {
		return "", err
	}

	var result strings.Builder
	err = t.Execute(&result, param)
	if err != nil {
		return "", err
	}
	return result.String(), nil
}

func WriteFile(file string, data string) error {
	// 存在不创建
	if _, err := os.Stat(file); err == nil {
		return errors.New(file + " 已经存在")
	} else if !os.IsNotExist(err) {
		return err
	}
	if err := os.MkdirAll(path.Dir(file), 0755); err != nil {
		return err
	}
	return os.WriteFile(file, []byte(data), 0644)
}
