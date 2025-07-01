package gogo

type Option func(*Options)

type Options struct {
	ignorePanic bool // 是否忽略panic
}

// WithIgnorePanic 设置是否忽略panic
func WithIgnorePanic(ignore bool) Option {
	return func(o *Options) {
		o.ignorePanic = ignore
	}
}
