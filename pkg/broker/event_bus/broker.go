package event_bus

import (
	"context"

	evbus "github.com/asaskevich/EventBus"
	"github.com/cago-frame/cago/pkg/broker/broker"
)

type eventBusBroker struct {
	bus evbus.Bus
}

func NewEvBusBroker() broker.Broker {
	ret := &eventBusBroker{
		bus: evbus.New(),
	}
	return ret
}

func (e *eventBusBroker) Publish(ctx context.Context, topic string, data *broker.Message, opt ...broker.PublishOption) error {
	e.bus.Publish(topic, data)
	return nil
}

func (e *eventBusBroker) Subscribe(ctx context.Context, topic string, h broker.Handler, opts ...broker.SubscribeOption) (broker.Subscriber, error) {
	options := broker.NewSubscribeOptions(opts...)
	return newSubscriber(e, topic, h, options)
}

func (e *eventBusBroker) Close() error {
	return nil
}

func (e *eventBusBroker) String() string {
	return "event_bus"
}
