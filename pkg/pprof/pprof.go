package pprof

import (
	"context"
	"github.com/cago-frame/cago"
	"github.com/cago-frame/cago/configs"
	"net/http"
	_ "net/http/pprof"
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
			err := http.ListenAndServe(options.Address, nil)
			if err != nil {
				panic(err)
			}
		}()
		return nil
	}
}
