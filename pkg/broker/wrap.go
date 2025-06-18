package broker

import (
	"context"

	broker2 "github.com/cago-frame/cago/pkg/broker/broker"
	wrap2 "github.com/cago-frame/cago/pkg/utils/wrap"
)

type wrap struct {
	broker2.Broker
	wrap    *wrap2.Wrap
	options *Options
}

// newWrap 包装原有broker
func newWrap(broker broker2.Broker, w *wrap2.Wrap, options *Options) broker2.Broker {
	return &wrap{Broker: broker, wrap: w, options: options}
}

func (t *wrap) Publish(ctx context.Context, topic string, data *broker2.Message, opts ...broker2.PublishOption) error {
	if t.options.topicPrefix != "" {
		topic = t.options.topicPrefix + "." + topic
	}
	return t.wrap.Run(ctx, "Publish", []interface{}{topic, data}, func(ctx *wrap2.Context) {
		ctx.Abort(t.Broker.Publish(ctx, topic, data, opts...))
	})
}

func (t *wrap) Subscribe(ctx context.Context, topic string,
	h broker2.Handler, opts ...broker2.SubscribeOption) (sub broker2.Subscriber, err error) {
	options := broker2.NewSubscribeOptions(opts...)
	if t.options.topicPrefix != "" {
		topic = t.options.topicPrefix + "." + topic
	}
	if t.options.defaultGroup != "" && options.Group == "" {
		opts = append(opts, broker2.Group(t.options.defaultGroup))
	}
	return t.Broker.Subscribe(ctx, topic, func(ctx context.Context, event broker2.Event) error {
		return t.wrap.Run(ctx, "Subscribe", []interface{}{topic, event, options}, func(ctx *wrap2.Context) {
			ctx.Abort(h(ctx, event))
		})
	}, opts...)
}
