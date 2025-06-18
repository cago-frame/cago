package api

import (
	"context"

	_ "github.com/cago-frame/cago/examples/simple/docs"
	"github.com/cago-frame/cago/examples/simple/internal/controller/example_ctr"
	"github.com/cago-frame/cago/examples/simple/internal/controller/user_ctr"
	"github.com/cago-frame/cago/examples/simple/internal/service/user_svc"
	"github.com/cago-frame/cago/server/mux"
)

// Router 路由
// @title    api文档
// @version  1.0
// @BasePath /api/v1
func Router(ctx context.Context, root *mux.Router) error {
	r := root.Group("/api/v1")

	userCtr := user_ctr.NewUser()
	{
		// 绑定路由
		r.Group("/").Bind(
			userCtr.Register,
			userCtr.Login,
		)

		r.Group("/", user_svc.User().Middleware(true)).Bind(
			userCtr.CurrentUser,
			userCtr.Logout,
			userCtr.RefreshToken,
		)
	}

	exampleCtl := example_ctr.NewExample()
	{
		r.Group("/").Bind(
			exampleCtl.Ping,
			exampleCtl.GinFun,
		)

		r.Group("/",
			user_svc.User().Middleware(true),
			user_svc.User().AuditMiddleware("example")).Bind(
			exampleCtl.Audit,
		)
	}

	return nil
}
