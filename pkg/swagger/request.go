package swagger

import (
	"go/ast"
	"go/parser"
	"go/token"
	"log"
	"net/http"
	"path"
	"strings"

	"github.com/cago-frame/cago/internal/cmd/gen/utils"
	"github.com/go-openapi/spec"
)

// 解析文件生成swagger
func (s *Swagger) parseFile(filename string) error {
	f, err := parser.ParseFile(token.NewFileSet(), filename, nil, parser.ParseComments)
	if err != nil {
		return err
	}
	for _, v := range f.Decls {
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
		// 解析结尾为Request的struct
		if strings.HasSuffix(typeSpec.Name.Name, "Response") {
			if err := newParseStruct(filename, s, f).parseStruct(typeSpec); err != nil {
				return err
			}
			continue
		}
		// 解析http.Meta
		for _, field := range structSpec.Fields.List {
			expr, ok := field.Type.(*ast.SelectorExpr)
			if !ok {
				continue
			}
			if expr.Sel.Name != "Meta" || expr.X.(*ast.Ident).Name != "mux" {
				continue
			}
			// 处理swagger route
			if err := s.parseRoute(filename, f, decl, typeSpec, structSpec, field); err != nil {
				return err
			}
			// 处理swagger response
			break
		}
	}
	return nil
}

func (s *Swagger) parseRoute(filename string, file *ast.File, decl *ast.GenDecl,
	typeSpec *ast.TypeSpec, structSpec *ast.StructType, field *ast.Field) error {
	// 解析tag,取出route和method等参数
	tag := strings.TrimPrefix(field.Tag.Value, "`")
	tag = strings.TrimSuffix(tag, "`")
	// 取出path值
	urlPath := utils.ParseTag(tag, "path")
	// 取出method值
	methods := strings.Split(utils.ParseTag(tag, "method"), ",")
	for _, method := range methods {
		operation := &spec.Operation{
			OperationProps: spec.OperationProps{
				Produces:  []string{"application/json"},
				Responses: &spec.Responses{},
			},
		}
		// 解析注释
		text := utils.GetTypeComment(decl, typeSpec)
		operation.Summary = text
		operation.Description = text
		// 解析参数
		operation.Parameters = []spec.Parameter{}
		// GET请求参数放在query中
		contentType := utils.ParseTag(tag, "contentType")
		if contentType == "" {
			contentType = s.defaultContentType
		}
		operation.OperationProps.Consumes = []string{"application/" + contentType}
		// get请求,或者是非json请求,将参数单独解析出来
		if method == http.MethodGet || contentType != JSONBodyType {
			for _, field := range structSpec.Fields.List {
				// 解析参数
				if field.Tag == nil {
					continue
				}
				tag := strings.TrimPrefix(field.Tag.Value, "`")
				form := utils.ParseTag(tag, "form")
				if field.Names == nil {
					//inline类型
					if form == ",inline" {
						schema, err := newParseStruct(filename, s, file).parseFieldSwagger(field)
						if err != nil {
							log.Printf("%v,%v", tag, err)
							return err
						}
						if schema.Type != nil {
							for k, v := range schema.SchemaProps.Properties {
								paramProps := spec.ParamProps{
									Description: v.Description,
									Name:        k,
									In:          "query",
								}
								paramProps.Name = k
								operation.Parameters = append(operation.Parameters, spec.Parameter{
									ParamProps:   paramProps,
									SimpleSchema: spec.SimpleSchema{Type: v.Type[0]},
								})
							}
						}
					}
					continue
				}
				in := ""
				uri := utils.ParseTag(tag, "uri")
				if uri != "" {
					in = "path"
				} else {
					if method == http.MethodGet {
						in = "query"
					} else {
						in = "formData"
					}
				}
				validate := utils.ParseTag(tag, "validate")
				required := false
				if strings.Contains(validate, "required") {
					required = true
				}
				if in == "path" {
					required = true
					urlPath = strings.ReplaceAll(urlPath, ":"+uri, "{"+uri+"}")
				}
				schema, err := newParseStruct(filename, s, file).parseFieldSwagger(field)
				if err != nil {
					return err
				}
				paramProps := spec.ParamProps{
					Description:     schema.Description,
					Name:            utils.SwaggerName(field),
					In:              in,
					Required:        required,
					AllowEmptyValue: false,
				}
				if schema.Type != nil {
					operation.Parameters = append(operation.Parameters, spec.Parameter{
						ParamProps: paramProps,
						SimpleSchema: spec.SimpleSchema{
							Type: schema.Type[0],
						},
					})
				} else {
					// 获取ref
					schema = s.swagger.Definitions[strings.Split(schema.Ref.GetURL().Fragment, "/")[2]]
					paramProps.Description += "\n" + schema.Description
					paramProps.Description = strings.TrimSpace(paramProps.Description)
					operation.Parameters = append(operation.Parameters, spec.Parameter{
						ParamProps: paramProps,
						CommonValidations: spec.CommonValidations{
							Enum: schema.Enum,
						},
						SimpleSchema: spec.SimpleSchema{
							Type: schema.Type[0],
						},
					})
				}
			}
		} else {
			// 解析uri参数
			ignoreField := make(map[string]struct{})
			for _, field := range structSpec.Fields.List {
				// 解析参数
				if field.Names == nil || field.Tag == nil {
					continue
				}
				tag := strings.TrimPrefix(field.Tag.Value, "`")
				in := "path"
				uri := utils.ParseTag(tag, "uri")
				if uri == "" {
					continue
				}
				ignoreField[uri] = struct{}{}
				urlPath = strings.ReplaceAll(urlPath, ":"+uri, "{"+uri+"}")
				schema, err := newParseStruct(filename, s, file).parseFieldSwagger(field)
				if err != nil {
					return err
				}
				paramProps := spec.ParamProps{
					Description:     schema.Description,
					Name:            utils.SwaggerName(field),
					In:              in,
					Required:        true,
					AllowEmptyValue: false,
				}
				if schema.Type != nil {
					operation.Parameters = append(operation.Parameters, spec.Parameter{
						ParamProps: paramProps,
						SimpleSchema: spec.SimpleSchema{
							Type: schema.Type[0],
						},
					})
				}
			}
			// json请求,将参数放在body中
			schema, err := newParseStruct(filename, s, file).parseFieldType(typeSpec.Type)
			if err != nil {
				return err
			}
			for k := range ignoreField {
				delete(schema.SchemaProps.Properties, k)
			}
			ref := spec.MustCreateRef("#/definitions/" + file.Name.Name + "." + typeSpec.Name.Name)
			s.swagger.Definitions[file.Name.Name+"."+typeSpec.Name.Name] = schema
			operation.Parameters = append(operation.Parameters, spec.Parameter{
				ParamProps: spec.ParamProps{
					Description: schema.Description,
					Name:        "body",
					In:          "body",
					Schema: &spec.Schema{
						SchemaProps: spec.SchemaProps{
							Ref: ref,
						},
					},
				},
				//SimpleSchema: spec.SimpleSchema{
				//	Type: schema.Type[0],
				//},
			})
			//operation.Parameters = append(operation.Parameters, spec.Parameter{
			//	ParamProps: spec.ParamProps{
			//		Name: "body",
			//		In:   "body",
			//		Schema: &spec.Schema{
			//			SchemaProps: spec.SchemaProps{
			//				Ref: spec.MustCreateRef("#/definitions/" + file.Name.Name + "." + typeSpec.Name.Name),
			//			},
			//		},
			//	},
			//})
		}

		// 解析返回值
		operation.Responses.StatusCodeResponses = map[int]spec.Response{http.StatusOK: {
			ResponseProps: spec.ResponseProps{
				Description: "OK",
				Schema: &spec.Schema{
					SchemaProps: spec.SchemaProps{
						Properties: map[string]spec.Schema{
							"code": {
								SchemaProps: spec.SchemaProps{
									Type: spec.StringOrArray{"integer"},
								},
							},
							"msg": {
								SchemaProps: spec.SchemaProps{
									Type: spec.StringOrArray{"string"},
								},
							},
							"data": {
								SchemaProps: spec.SchemaProps{
									Ref: spec.MustCreateRef("#/definitions/" + file.Name.Name + "." +
										strings.Replace(typeSpec.Name.Name, "Request", "Response", 1)),
								},
							},
						},
					},
				},
			},
		}, http.StatusBadRequest: {
			ResponseProps: spec.ResponseProps{
				Description: "Bad Request",
				Schema: &spec.Schema{
					SchemaProps: spec.SchemaProps{
						Ref: spec.MustCreateRef("#/definitions/BadRequest"),
					},
				},
			},
		}}

		// 添加tag
		base := path.Base(filename)
		// 去除后缀
		base = base[:len(base)-len(path.Ext(base))]
		operation.Tags = []string{file.Name.Name}
		if base != file.Name.Name {
			operation.Tags = []string{file.Name.Name + "/" + base}
		}

		pathItem, ok := s.swagger.Paths.Paths[urlPath]
		if !ok {
			pathItem = spec.PathItem{
				PathItemProps: spec.PathItemProps{},
			}
		}
		switch method {
		case http.MethodGet:
			pathItem.PathItemProps.Get = operation
		case http.MethodPost:
			pathItem.PathItemProps.Post = operation
		case http.MethodPut:
			pathItem.PathItemProps.Put = operation
		case http.MethodDelete:
			pathItem.PathItemProps.Delete = operation
		}
		s.swagger.Paths.Paths[urlPath] = pathItem
	}
	return nil
}
