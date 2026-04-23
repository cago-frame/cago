package broker

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/cago-frame/cago/configs"
	"github.com/cago-frame/cago/pkg/logger"
	trace2 "github.com/cago-frame/cago/pkg/opentelemetry/trace"
	wrap2 "github.com/cago-frame/cago/pkg/utils/wrap"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/propagation"
	"go.opentelemetry.io/otel/trace"
	"go.uber.org/zap"

	broker2 "github.com/cago-frame/cago/pkg/broker/broker"
)

type Type string

const (
	NSQ      Type = "nsq"
	EventBus Type = "event_bus"
	Kafka    Type = "kafka"
)

// Config broker 基础配置。具体 broker 的配置（如 broker.nsq、broker.kafka）
// 由各 broker 子包在自己的 init() factory 里从 *configs.Config 中 Scan。
type Config struct {
	Type Type `yaml:"type"`
}

// NewWithConfig 根据配置构建 broker。要求用户已经通过
// `import _ "github.com/cago-frame/cago/pkg/broker/<type>"` 完成自注册。
// nsq 作为默认 broker 已由主包 default_nsq.go 内联注册。
func NewWithConfig(ctx context.Context, config *configs.Config, opts ...Option) (broker2.Broker, error) {
	cfg := &Config{}
	if err := config.Scan(ctx, "broker", cfg); err != nil {
		return nil, err
	}
	if cfg.Type == "" {
		return nil, errors.New("broker.type is empty")
	}
	f := GetFactory(string(cfg.Type))
	if f == nil {
		return nil, fmt.Errorf(
			"broker type %q not registered; please import the corresponding package, e.g. _ \"github.com/cago-frame/cago/pkg/broker/kafka\" or _ \"github.com/cago-frame/cago/pkg/broker/event_bus\" (nsq 默认已注册)",
			cfg.Type,
		)
	}
	ret, err := f(ctx, config)
	if err != nil {
		return nil, err
	}
	opts = append(opts, WithBroker(ret))
	return New(opts...)
}

func New(opts ...Option) (broker2.Broker, error) {
	options := &Options{}
	for _, o := range opts {
		o(options)
	}
	ret := options.broker
	// logger
	wrapHandler := wrap2.New()
	wrapHandler.Wrap(func(ctx *wrap2.Context) {
		sctx := ctx.Context
		switch ctx.Name() {
		case "Subscribe":
			topic := ctx.Args(0).(string)
			options := ctx.Args(2).(broker2.SubscribeOptions)
			sctx = logger.WithContextLogger(sctx, logger.Ctx(sctx).With(
				zap.String("topic", topic), zap.String("group", options.Group),
				// 请求开始时间
				zap.Time("start_time", time.Now()),
			))

			defer func() {
				if r := recover(); r != nil {
					logger.Ctx(ctx).Error("broker subscribe panic",
						zap.String("topic", topic), zap.String("group", options.Group),
						zap.Any("recover", r), zap.StackSkip("stack", 3))
				}
			}()
			ctx = ctx.WithContext(sctx)
		}
		ctx.Next()
	})
	if options.tracer != nil {
		wrapHandler.Wrap(func(ctx *wrap2.Context) {
			sctx := ctx.Context
			switch ctx.Name() {
			case "Publish":
				topic := ctx.Args(0).(string)
				data := ctx.Args(1).(*broker2.Message)
				sctx, span := options.tracer.Start(sctx, "Broker."+ctx.Name(),
					trace.WithAttributes(
						attribute.String("messaging.system", ret.String()),
						attribute.String("messaging.destination", topic),
						attribute.String("messaging.destination_kind", "queue"),
					),
					trace.WithSpanKind(trace.SpanKindProducer),
				)
				defer span.End()
				if data.Header == nil {
					data.Header = make(map[string]string)
				}
				otel.GetTextMapPropagator().Inject(sctx, propagation.MapCarrier(data.Header))
				ctx = ctx.WithContext(sctx)
			case "Subscribe":
				event := ctx.Args(1).(broker2.Event)
				soptions := ctx.Args(2).(broker2.SubscribeOptions)
				sctx = otel.GetTextMapPropagator().Extract(sctx, propagation.MapCarrier(event.Message().Header))
				sctx, span := options.tracer.Start(sctx, "Broker."+ctx.Name(),
					trace.WithAttributes(
						attribute.String("messaging.system", ret.String()),
						attribute.String("messaging.operation", "process"),
						attribute.String("messaging.destination", event.Topic()),
						attribute.String("messaging.destination_kind", "queue"),
						attribute.String("messaging.group", soptions.Group),
					),
					trace.WithSpanKind(trace.SpanKindConsumer),
				)
				defer span.End()
				sctx = logger.WithContextLogger(sctx, logger.Ctx(sctx).With(
					trace2.LoggerLabel(sctx)...,
				))
				ctx = ctx.WithContext(sctx)
			}
			ctx.Next()
		})
	}
	return newWrap(ret, wrapHandler, options), nil
}
