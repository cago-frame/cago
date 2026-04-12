package gogo

type Option func(*Options)

type Options struct {
	ignorePanic bool // 是否忽略panic
}

// WithIgnorePanic 忽略panic，不再继续抛出
func WithIgnorePanic() Option {
	return func(o *Options) {
		o.ignorePanic = true
	}
}
