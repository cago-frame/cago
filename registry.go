package cago

import (
	"context"
	"errors"
	"os"
	"os/signal"
	"reflect"
	"syscall"
	"time"

	"github.com/cago-frame/cago/configs"
	"github.com/cago-frame/cago/pkg/gogo"
	"github.com/cago-frame/cago/pkg/logger"
	"go.uber.org/zap"
)

type Cago struct {
	ctx        context.Context
	cancel     context.CancelFunc
	cfg        *configs.Config
	components []Component
	disableLog bool
}

type CloseHandle func()

// New create a new cago instance
// ctx 可以管理整个应用的生命周期，当ctx.Done()时，会传递到每一个组件，安全退出
// cfg 配置文件，每一个组件都可以使用，通过 configs.NewConfig 去构建
// 应用启动时，会调用每一个组件的 Component.Start 方法，启动组件
// 应用停止时，会调用每一个组件的 Component.CloseHandle 方法，关闭组件
// 推荐链式调用的方式去使用
// cago.New(ctx, cfg).Registry(component.Core()).RegistryCancel(mux.HTTP(api.Router)).Start()
func New(ctx context.Context, cfg *configs.Config) *Cago {
	ctx, cancel := context.WithCancel(ctx)
	cago := &Cago{
		ctx:    ctx,
		cancel: cancel,
		cfg:    cfg,
	}
	return cago
}

// Registry 注册组件
func (r *Cago) Registry(component Component) *Cago {
	err := component.Start(r.ctx, r.cfg)
	if err != nil {
		panic(err)
	}
	r.components = append(r.components, component)
	return r
}

// RegistryCancel 注册cancel组件，cancel组件可以停止整个应用
func (r *Cago) RegistryCancel(component ComponentCancel) *Cago {
	err := component.StartCancel(r.ctx, r.cancel, r.cfg)
	if err != nil {
		panic(errors.New("start component error: " + reflect.TypeOf(component).String() + " " + err.Error()))
	}
	r.components = append(r.components, component)
	return r
}

// Start 启动框架 在此之前组件已全部执行 Component.Start 方法启动，此处只做停止等待
// 可以通过ctx、cancelFunc和进程信号量来控制整个应用的生命周期
// 停止时会调用 Component.CloseHandle 方法关闭组件，会等待所有组件关闭完成，最终关闭整个应用
func (r *Cago) Start() error {
	r.info(r.cfg.AppName + " is starting...")
	quitSignal := make(chan os.Signal, 1)
	// 优雅启停
	signal.Notify(
		quitSignal,
		syscall.SIGINT, syscall.SIGTERM,
	)
	select {
	case <-quitSignal:
		r.cancel()
	case <-r.ctx.Done():
	}
	r.info(r.cfg.AppName + " is stopping...")
	for _, v := range r.components {
		v.CloseHandle()
	}
	// 等待所有组件退出
	stopCh := make(chan struct{})
	go func() {
		gogo.Wait()
		close(stopCh)
	}()
	select {
	case <-stopCh:
	case <-time.After(time.Second * 10):
	}
	r.info(r.cfg.AppName + " is stopped")
	return nil
}

func (r *Cago) info(msg string, fields ...zap.Field) {
	if r.disableLog {
		return
	}
	logger.Default().Info(msg, fields...)
}

// DisableLogger 禁用框架日志
func (r *Cago) DisableLogger() *Cago {
	r.disableLog = true
	return r
}
