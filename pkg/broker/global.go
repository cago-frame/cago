package broker

import (
	"context"

	"github.com/cago-frame/cago"
	"github.com/cago-frame/cago/configs"
	broker2 "github.com/cago-frame/cago/pkg/broker/broker"
	"github.com/cago-frame/cago/pkg/opentelemetry/trace"
	trace2 "go.opentelemetry.io/otel/trace"
)

var broker broker2.Broker

const instrumName = "github.com/cago-frame/cago/pkg/broker"

// Broker 消息队列组件
func Broker(ctx context.Context, config *configs.Config) error {
	cfg := &Config{}
	if err := config.Scan(ctx, "broker", cfg); err != nil {
		return err
	}
	options := make([]Option, 0)
	if tp := trace.Default(); tp != nil {
		options = append(options, WithTracer(tp.Tracer(
			instrumName,
			trace2.WithInstrumentationVersion("semver:"+cago.Version()),
		)))
	}
	options = append(options, WithDefaultGroup(config.AppName),
		WithTopicPrefix(config.AppName+"."+string(config.Env)))
	b, err := NewWithConfig(ctx, cfg, options...)
	if err != nil {
		return err
	}
	broker = b
	return nil
}

// SetBroker 设置消息队列组件
func SetBroker(b broker2.Broker) {
	broker = b
}

// Default 获取默认消息队列组件
func Default() broker2.Broker {
	return broker
}
