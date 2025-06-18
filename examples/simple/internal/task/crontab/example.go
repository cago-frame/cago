package crontab

import (
	"context"

	"github.com/cago-frame/cago/pkg/logger"
)

func Example(ctx context.Context) error {
	logger.Ctx(ctx).Info("定时任务")
	return nil
}
