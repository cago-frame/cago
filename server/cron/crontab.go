package cron

import (
	"context"

	"github.com/cago-frame/cago/pkg/logger"
	"github.com/robfig/cron/v3"
	"go.opentelemetry.io/otel/attribute"
	trace2 "go.opentelemetry.io/otel/trace"
	"go.uber.org/zap"
)

type Crontab interface {
	AddFunc(spec string, cmd func(ctx context.Context) error) (cron.EntryID, error)
}

type crontab struct {
	tracer trace2.Tracer
	cron   *cron.Cron
}

func (c *crontab) AddFunc(spec string, cmd func(ctx context.Context) error) (cron.EntryID, error) {
	return c.cron.AddFunc(spec, func() {
		ctx := context.Background()
		if c.tracer != nil {
			var span trace2.Span
			ctx, span = c.tracer.Start(ctx, "cron",
				trace2.WithSpanKind(trace2.SpanKindServer),
				trace2.WithAttributes(
					attribute.String("spec", spec),
				),
			)
			defer span.End()
		}
		logger.Ctx(ctx).Info("cron start", zap.String("spec", spec))
		if err := cmd(ctx); err != nil {
			logger.Ctx(ctx).Error("cron error", zap.Error(err))
		}
	})
}
