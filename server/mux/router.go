package mux

import (
	"context"
	"errors"
	"net/http"
	"reflect"
	"strings"

	"github.com/cago-frame/cago/pkg/utils/httputils"
	"github.com/gin-gonic/gin"
)

type Router struct {
	*Routes
	gin.IRouter
}

type Routes struct {
	gin.IRoutes
}

func (r *Router) Group(path string, handler ...gin.HandlerFunc) *Router {
	g := r.IRouter.Group(path, handler...)
	return &Router{
		Routes:  &Routes{IRoutes: g},
		IRouter: g,
	}
}

func (r *Router) Use(handler ...gin.HandlerFunc) *Routes {
	return &Routes{IRoutes: r.IRouter.Use(handler...)}
}

// Bind 绑定控制器
// 传入的只能是下面格式的方法，且定义的结构体中必须有Meta字段，定义好路由信息
// 定义好Request结构体，可以使用脚手架自动生成相关方法
// func A(ctx context.Context, req *api.ARequest) (*api.AResponse, error)
// func B(ctx context.Context, req *api.BRequest) error
// func C(ctx *gin.Context, req *api.CRequest) (*api.CResponse, error)
// func D(ctx *gin.Context, req *api.DRequest) error
func (r *Routes) Bind(handler ...interface{}) {
	// 反射解析控制器方法
	for _, c := range handler {
		el := reflect.TypeOf(c)
		if el.Kind() == reflect.Func {
			if err := r.bindFunc(reflect.Zero(el), reflect.ValueOf(c), true); err != nil {
				panic(err)
			}
		} else {
			panic("invalid controller")
		}
	}
}

// RequestHandle 绑定请求处理
// 输入一个请求结构体和一个处理函数
func (r *Routes) RequestHandle(request interface{}, handlers ...gin.HandlerFunc) *Routes {
	requestEl := reflect.TypeOf(request)
	return r.requestHandle(requestEl, handlers...)
}

// Metadata 获取路由元数据
// 输入一个
func Metadata(request any) (string, string) {
	requestEl := reflect.TypeOf(request)
	route, ok := requestEl.FieldByName("Meta")
	// 必须有Route字段
	if !ok || route.Type != reflect.TypeOf(Meta{}) {
		panic("invalid method, second parameter must have Meta field")
	}
	return route.Tag.Get("path"), route.Tag.Get("method")
}

func (r *Routes) requestHandle(requestEl reflect.Type, handlers ...gin.HandlerFunc) *Routes {
	route, ok := requestEl.FieldByName("Meta")
	// 必须有Route字段
	if !ok || route.Type != reflect.TypeOf(Meta{}) {
		panic("invalid method, second parameter must have Meta field")
	}
	paths := strings.Split(route.Tag.Get("path"), ",")
	methods := strings.Split(route.Tag.Get("method"), ",")
	for _, path := range paths {
		for _, method := range methods {
			if method == "" {
				method = http.MethodGet
			}
			r.IRoutes.Handle(method, path, handlers...)
		}
	}
	return r
}

// 根据方法去绑定路由
func (r *Routes) bindFunc(controller reflect.Value, method reflect.Value, isFunc bool) error {
	methodType := method.Type()
	pos := 0
	if isFunc {
		pos = -1
	}
	// 获取路由参数
	request := methodType.In(2 + pos).Elem()
	route, ok := request.FieldByName("Meta")
	// 必须有Route字段
	if !ok || route.Type != reflect.TypeOf(Meta{}) {
		return errors.New("invalid method, second parameter must have Meta field")
	}
	ginContext := false
	// 判断方法的第一个参数是否是context.Context或者*gin.Context
	param1 := methodType.In(1 + pos)
	if methodType.NumIn() != 3+pos {
		return errors.New("invalid method, first parameter must be context.Context or *gin.Context")
	}
	if param1 != reflect.TypeOf((*context.Context)(nil)).Elem() {
		if param1 != reflect.TypeOf((*gin.Context)(nil)) {
			return errors.New("invalid method, first parameter must be context.Context or *gin.Context")
		}
		ginContext = true
	}
	var call func(a reflect.Value, b interface{}) []reflect.Value
	if !isFunc {
		call = func(a reflect.Value, b interface{}) []reflect.Value {
			return method.Call([]reflect.Value{controller, a, reflect.ValueOf(b)})
		}
	} else {
		call = func(a reflect.Value, b interface{}) []reflect.Value {
			return method.Call([]reflect.Value{a, reflect.ValueOf(b)})
		}
	}

	r.requestHandle(request, r.bindHandler(request, call, ginContext))
	return nil
}

func (r *Routes) bindHandler(request reflect.Type,
	call func(a reflect.Value, b interface{}) []reflect.Value,
	ginContext bool) gin.HandlerFunc {
	return func(c *gin.Context) {
		// 创建请求参数
		req := reflect.New(request)
		// 绑定请求参数
		i := req.Interface()
		if err := ShouldBindWith(c, i); err != nil {
			httputils.HandleResp(c, err)
			return
		}
		// 调用控制器方法
		var ctx reflect.Value
		if ginContext {
			ctx = reflect.ValueOf(c)
		} else {
			ctx = reflect.ValueOf(c.Request.Context())
		}
		resp := call(ctx, req.Interface())
		switch len(resp) {
		case 0:
			return
		case 1:
			// 一个参数只能接受error
			if resp[0].IsNil() {
				return
			}
			_ = httputils.HandleError(c, resp[0].Interface().(error))
		case 2:
			// 两个参数，第一个是返回值，第二个是error
			if resp[1].IsNil() {
				httputils.HandleResp(c, resp[0].Interface())
				return
			}
			httputils.HandleResp(c, resp[1].Interface())
		}
	}
}
