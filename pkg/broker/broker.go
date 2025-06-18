package broker

import (
	"context"
	"errors"
	"time"

	"github.com/cago-frame/cago/pkg/logger"
	trace2 "github.com/cago-frame/cago/pkg/opentelemetry/trace"
	wrap2 "github.com/cago-frame/cago/pkg/utils/wrap"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/propagation"
	"go.opentelemetry.io/otel/trace"
	"go.uber.org/zap"

	broker2 "github.com/cago-frame/cago/pkg/broker/broker"
	"github.com/cago-frame/cago/pkg/broker/event_bus"
	"github.com/cago-frame/cago/pkg/broker/nsq"
)

type Type string

const (
	NSQ      Type = "nsq"
	EventBus Type = "event_bus"
)

type Config struct {
	Type Type
	NSQ  nsq.Config
}

func NewWithConfig(ctx context.Context, cfg *Config, opts ...Option) (broker2.Broker, error) {
	var ret broker2.Broker
	var err error
	switch cfg.Type {
	case NSQ:
		ret, err = nsq.NewBroker(cfg.NSQ)
	case EventBus:
		ret = event_bus.NewEvBusBroker()
	default:
		return nil, errors.New("type not found")
	}
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
