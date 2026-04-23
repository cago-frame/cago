package kafka

import (
	"context"

	"github.com/cago-frame/cago/configs"
	"github.com/cago-frame/cago/pkg/broker"
	broker2 "github.com/cago-frame/cago/pkg/broker/broker"
)

func init() {
	broker.RegisterBroker("kafka", func(ctx context.Context, cfg *configs.Config) (broker2.Broker, error) {
		c := &Config{}
		if err := cfg.Scan(ctx, "broker.kafka", c); err != nil {
			return nil, err
		}
		return NewBroker(*c)
	})
}
