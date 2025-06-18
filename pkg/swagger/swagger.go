package swagger

import (
	"go/parser"
	"go/token"
	"path"
	"strings"

	"github.com/cago-frame/cago/internal/cmd/gen/utils"
	"github.com/go-openapi/spec"
)

const (
	JSONBodyType       = "json"
	FormDataBodyType   = "form-data"
	XWWWFormURLEncoded = "x-www-form-urlencoded"
)

type Swagger struct {
	apiDir             string
	defaultContentType string
	swagger            *spec.Swagger
	//parseStruct        *parseStruct
	InnerDefinitions map[string]spec.Schema

	rootPkgPath string
	rootPkgName string
}

func NewSwagger(apiDir string) *Swagger {
	return &Swagger{
		apiDir:           apiDir,
		InnerDefinitions: make(map[string]spec.Schema),
	}
}

func (s *Swagger) Gen() error {
	return s.gen()
}

func (s *Swagger) gen() error {
	var err error
	s.rootPkgPath, s.rootPkgName, err = utils.FindRootPkgName(s.apiDir)
	if err != nil {
		return err
	}
	// 先读取router文件,解析出全局信息
	if err := s.parseSwagger(path.Join(s.apiDir, "router.go")); err != nil {
		return err
	}
	// 读取目录下的文件生成swagger
	return utils.ReadDir(s.apiDir, func(path string) error {
		return s.parseFile(path)
	})
}

// 解析获取基础swagger信息
func (s *Swagger) parseSwagger(file string) error {
	// 解析main.go
	f, err := parser.ParseFile(token.NewFileSet(), file, nil, parser.ParseComments)
	if err != nil {
		return err
	}
	info := &spec.Info{}
	ret := &spec.Swagger{
		SwaggerProps: spec.SwaggerProps{
			Swagger: "2.0",
			Definitions: spec.Definitions{
				"BadRequest": {
					SchemaProps: spec.SchemaProps{
						Type: []string{"object"},
						Properties: map[string]spec.Schema{
							"code": {
								SchemaProps: spec.SchemaProps{
									Type:        []string{"integer"},
									Description: "错误码",
									Format:      "int32",
								},
							},
							"msg": {
								SchemaProps: spec.SchemaProps{
									Type:        []string{"string"},
									Description: "错误信息",
								},
							},
						},
					},
				},
			},
			Info: info,
			Paths: &spec.Paths{
				Paths: make(map[string]spec.PathItem),
			},
		},
	}
	for _, comment := range f.Comments {
		flag := false
		for _, v := range comment.List {
			text := strings.TrimPrefix(v.Text, "// @")
			// 证明是swagger的注释
			if text == v.Text {
				continue
			}
			flag = true
			// 解析注释
			key := strings.Split(text, " ")[0]
			value := strings.TrimPrefix(text, key+" ")
			value = strings.TrimSpace(value)
			switch strings.ToLower(key) {
			case "title":
				info.Title = value
			case "description":
				info.Description = value
			case "version":
				info.Version = value
			case "basepath":
				ret.BasePath = value
			case "defaultcontenttype":
				s.defaultContentType = value
			}
		}
		if flag {
			break
		}
	}
	s.defaultContentType = JSONBodyType
	s.swagger = ret
	return err
}
