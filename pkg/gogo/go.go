package gogo

import (
	"context"
	"sync"

	"github.com/cago-frame/cago/pkg/logger"
	"go.uber.org/zap"
)

var wg sync.WaitGroup

// Go 框架处理协程
// 可以处理协程的panic，但是不会返回错误
// 也可以处理安全退出，当还有协程在运行时，gogo.Wait()会一直阻塞
func Go(fun func(ctx context.Context) error, opts ...Option) error {
	wg.Add(1)
	options := &Options{}
	for _, o := range opts {
		o(options)
	}
	if options.ctx == nil {
		options.ctx = context.Background()
	}
	go func() {
		defer func() {
			wg.Done()
			// 错误处理
			if err := recover(); err != nil {
				logger.Default().Error("goroutine panic", zap.Any("err", err), zap.Stack("stack"))
			}
		}()
		_ = fun(options.ctx)
	}()
	return nil
}

// Wait 等待所有协程结束
func Wait() {
	wg.Wait()
}
