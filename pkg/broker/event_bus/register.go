package event_bus

import (
	"context"

	"github.com/cago-frame/cago/configs"
	"github.com/cago-frame/cago/pkg/broker"
	broker2 "github.com/cago-frame/cago/pkg/broker/broker"
)

func init() {
	broker.RegisterBroker("event_bus", func(ctx context.Context, cfg *configs.Config) (broker2.Broker, error) {
		return NewEvBusBroker(), nil
	})
}
