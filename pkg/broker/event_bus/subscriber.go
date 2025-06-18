package event_bus

import (
	"context"

	"github.com/cago-frame/cago/pkg/broker/broker"
	"github.com/cago-frame/cago/pkg/logger"
	"go.uber.org/zap"
)

type subscriber struct {
	e      *eventBusBroker
	topic  string
	handle func(data *broker.Message)
}

func newSubscriber(e *eventBusBroker, topic string, handler broker.Handler, options broker.SubscribeOptions) (broker.Subscriber, error) {
	ret := &subscriber{
		e:     e,
		topic: topic,
	}
	logger := logger.Default().With(
		zap.String("topic", topic), zap.String("group", options.Group),
	)
	ret.handle = func(data *broker.Message) {
		go func() {
			var ctx context.Context
			if options.Context != nil {
				ctx = options.Context
			} else {
				ctx = context.Background()
			}
			err := handler(ctx, &event{
				topic: topic,
				data:  data,
			})
			if err != nil {
				logger.Error("event bus subscriber handle error", zap.Error(err))
			}
		}()
	}
	if err := e.bus.SubscribeAsync(topic, ret.handle, true); err != nil {
		return nil, err
	}
	return ret, nil
}

func (n *subscriber) Topic() string {
	return n.topic
}

func (n *subscriber) Unsubscribe() error {
	return n.e.bus.Unsubscribe(n.topic, n.handle)
}
