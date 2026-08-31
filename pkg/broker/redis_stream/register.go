package redis_stream

import (
	"context"

	"github.com/cago-frame/cago/configs"
	"github.com/cago-frame/cago/pkg/broker"
	broker2 "github.com/cago-frame/cago/pkg/broker/broker"
)

func init() {
	broker.RegisterBroker("redis_stream", func(ctx context.Context, cfg *configs.Config) (broker2.Broker, error) {
		c := &Config{}
		if err := cfg.Scan(ctx, "broker.redis_stream", c); err != nil {
			return nil, err
		}
		return NewBroker(*c)
	})
}
