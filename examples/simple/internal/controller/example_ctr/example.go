package example_ctr

import (
	"context"
	"net/http"

	"github.com/gin-gonic/gin"

	api "github.com/cago-frame/cago/examples/simple/internal/api/example"
	"github.com/cago-frame/cago/examples/simple/internal/service/example_svc"
)

type Example struct {
}

func NewExample() *Example {
	return &Example{}
}

// Ping ping
func (e *Example) Ping(ctx context.Context, req *api.PingRequest) (*api.PingResponse, error) {
	return example_svc.Example().Ping(ctx, req)
}

// GinFun gin function
func (e *Example) GinFun(ctx *gin.Context, req *api.GinFunRequest) {
	ctx.String(http.StatusOK, "ok")
}

// Audit 审计操作
func (e *Example) Audit(ctx context.Context, req *api.AuditRequest) (*api.AuditResponse, error) {
	return example_svc.Example().Audit(ctx, req)
}
