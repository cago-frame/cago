package example_svc

import (
	"context"
	"time"

	"github.com/cago-frame/cago/examples/simple/internal/task/queue"
	"github.com/cago-frame/cago/examples/simple/internal/task/queue/message"

	api "github.com/cago-frame/cago/examples/simple/internal/api/example"
	"github.com/cago-frame/cago/pkg/iam/audit"
	"github.com/cago-frame/cago/pkg/logger"
	"github.com/cago-frame/cago/pkg/utils"
	"go.uber.org/zap"
)

type ExampleSvc interface {
	// Ping ping
	Ping(ctx context.Context, req *api.PingRequest) (*api.PingResponse, error)
	// Audit 审计操作
	Audit(ctx context.Context, req *api.AuditRequest) (*api.AuditResponse, error)
}

type exampleSvc struct {
}

var defaultExample = &exampleSvc{}

func Example() ExampleSvc {
	return defaultExample
}

// Ping ping
func (e *exampleSvc) Ping(ctx context.Context, req *api.PingRequest) (*api.PingResponse, error) {
	if err := queue.PublishExample(ctx, &message.ExampleMsg{Time: time.Now().Unix()}); err != nil {
		logger.Ctx(ctx).Error("发布消息失败", zap.Error(err))
		return nil, err
	}
	return &api.PingResponse{Pong: utils.RandString(6, utils.Mix)}, nil
}

// Audit 审计操作
func (e *exampleSvc) Audit(ctx context.Context, req *api.AuditRequest) (*api.AuditResponse, error) {
	_ = audit.Ctx(ctx).Record("audit", zap.String("key", "value"))
	return nil, nil
}
