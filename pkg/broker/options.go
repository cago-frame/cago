package broker

import (
	broker2 "github.com/cago-frame/cago/pkg/broker/broker"
	"go.opentelemetry.io/otel/trace"
)

type Option func(options *Options)

type Options struct {
	tracer       trace.Tracer
	broker       broker2.Broker
	defaultGroup string
	topicPrefix  string
}

func WithTracer(t trace.Tracer) Option {
	return func(options *Options) {
		options.tracer = t
	}
}

func WithBroker(b broker2.Broker) Option {
	return func(options *Options) {
		options.broker = b
	}
}

func WithDefaultGroup(group string) Option {
	return func(options *Options) {
		options.defaultGroup = group
	}
}

func WithTopicPrefix(prefix string) Option {
	return func(options *Options) {
		options.topicPrefix = prefix
	}
}
