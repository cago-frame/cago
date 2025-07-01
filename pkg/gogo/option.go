package gogo

import "context"

type Option func(*Options)

type Options struct {
	ctx         context.Context
	ignorePanic bool // 是否忽略panic
}

// WithContext 设置上下文
func WithContext(ctx context.Context) Option {
	return func(o *Options) {
		o.ctx = ctx
	}
}

// WithIgnorePanic 设置是否忽略panic
func WithIgnorePanic(ignore bool) Option {
	return func(o *Options) {
		o.ignorePanic = ignore
	}
}
