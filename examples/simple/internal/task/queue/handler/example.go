package handler

import (
	"context"

	"github.com/cago-frame/cago/examples/simple/internal/task/queue"
	"github.com/cago-frame/cago/examples/simple/internal/task/queue/message"

	"github.com/cago-frame/cago/pkg/logger"
	"go.uber.org/zap"
)

type Example struct {
}

func (u *Example) Register(ctx context.Context) error {
	if err := queue.SubscribeExample(ctx, u.example); err != nil {
		return err
	}
	return nil
}

func (u *Example) example(ctx context.Context, msg *message.ExampleMsg) error {
	logger.Ctx(ctx).Info("收到消息", zap.Int64("time", msg.Time))
	return nil
}
