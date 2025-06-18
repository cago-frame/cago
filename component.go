package cago

import (
	"context"

	"github.com/cago-frame/cago/configs"
)

// Component 组件接口
type Component interface {
	// Start 启动组件 会传入框架的context和配置文件
	Start(ctx context.Context, cfg *configs.Config) error
	// CloseHandle 关闭处理 当应用停止时，会调用该方法，停止组件
	CloseHandle()
}

// ComponentCancel 带cancel方法的组件 可以停止整个应用
type ComponentCancel interface {
	Component
	// StartCancel 启动组件，会传入框架的context和配置文件，以及cancel方法
	// 可以通过cancel方法停止整个应用
	StartCancel(ctx context.Context, cancel context.CancelFunc, cfg *configs.Config) error
}

// FuncComponent 函数式组件 适用于简单的组件
// 当你的组件不需要释放资源时，你可以只写一个函数，然后通过 FuncComponent 转换成组件
// 例如：
//
//	cago.FuncComponent(func(ctx context.Context, cfg *configs.Config) error {
//		return nil
//	})
type FuncComponent func(ctx context.Context, cfg *configs.Config) error

func (f FuncComponent) Start(ctx context.Context, cfg *configs.Config) error {
	return f(ctx, cfg)
}

func (f FuncComponent) CloseHandle() {
}

// FuncComponentCancel 函数式带cancel方法的组件 适用于简单的组件，与 FuncComponent 的区别是多了一个cancel方法
type FuncComponentCancel func(ctx context.Context, cancel context.CancelFunc, cfg *configs.Config) error

func (f FuncComponentCancel) Start(ctx context.Context, cfg *configs.Config) error {
	return f.StartCancel(ctx, nil, cfg)
}

func (f FuncComponentCancel) StartCancel(ctx context.Context, cancel context.CancelFunc, cfg *configs.Config) error {
	return f(ctx, cancel, cfg)
}

func (f FuncComponentCancel) CloseHandle() {
}
