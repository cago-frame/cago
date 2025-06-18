package muxutils

import (
	"net/http"

	"github.com/cago-frame/cago/server/mux"
	"github.com/gin-gonic/gin"
)

// Router 路由定义，你可以直接定义一个Handler方法来使用
type Router struct {
	Method       string
	RelativePath string
	Handler      gin.HandlerFunc
}

// RouterTree 路由树
// 可以用来构建复杂的路由，你也可以使用 Use 方法来简化使用，例如
//
//	muxutils.RouterTree{
//	 Middleware: []gin.HandlerFunc{middleware1, middleware2},
//	 Handler: []interface{}{
//	   &muxutils.RouterTree{
//	     Middleware: []gin.HandlerFunc{middleware3},
//	     Handler: []interface{}{
//	        Route1,
//	        Route2,
//		  },
//	   },
//	   Route3,
//	 },
type RouterTree struct {
	Middleware []gin.HandlerFunc
	Handler    []interface{}
}

// Use 用来构建路由树
// 相比直接使用 RouterTree，Use 方法可以更加简洁的构建路由树，例如：
//
//	muxutils.Use(middleware1, middleware2).Append(
//		Route1,
//		Route2,
//		muxutils.Use(middleware3).Append(
//			Route3,
//		),
//	)
func Use(handler ...gin.HandlerFunc) *RouterTree {
	return &RouterTree{
		Middleware: handler,
		Handler:    make([]interface{}, 0),
	}
}

func (r *RouterTree) Use(handler ...gin.HandlerFunc) *RouterTree {
	r.Middleware = append(r.Middleware, handler...)
	return r
}

func (r *RouterTree) Append(handler ...interface{}) *RouterTree {
	r.Handler = append(r.Handler, handler...)
	return r
}

// BindTree 将路由树绑定到gin的路由上
func BindTree(r *mux.Router, tree []*RouterTree) {
	for _, v := range tree {
		if len(v.Handler) > 0 {
			rg := r.Group("/")
			rg.Use(v.Middleware...)
			for _, handler := range v.Handler {
				switch h := handler.(type) {
				case *RouterTree:
					BindTree(rg, []*RouterTree{h})
				case *Router:
					switch h.Method {
					case http.MethodGet:
						rg.GET(h.RelativePath, h.Handler)
					case http.MethodPost:
						rg.POST(h.RelativePath, h.Handler)
					case http.MethodPut:
						rg.PUT(h.RelativePath, h.Handler)
					case http.MethodDelete:
						rg.DELETE(h.RelativePath, h.Handler)
					}
				default:
					rg.Bind(h)
				}
			}
		}
	}
}
