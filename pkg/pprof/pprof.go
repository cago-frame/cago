package pprof

import (
	"context"
	"net/http"
	_ "net/http/pprof" //nolint:gosec // G108

	"github.com/cago-frame/cago"
	"github.com/cago-frame/cago/configs"
)

type Options struct {
	Address string
}

type Option func(*Options)

func Address(address string) Option {
	return func(o *Options) {
		o.Address = address
	}
}

func Pprof(opts ...Option) cago.FuncComponent {
	options := &Options{
		Address: "0.0.0.0:6060",
	}
	for _, o := range opts {
		o(options)
	}
	return func(ctx context.Context, cfg *configs.Config) error {
		go func() {
			err := http.ListenAndServe(options.Address, nil) //nolint:gosec // G114
			if err != nil {
				panic(err)
			}
		}()
		return nil
	}
}
