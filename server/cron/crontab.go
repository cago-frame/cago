package cron

import (
	"context"
	"reflect"
	"runtime"
	"strings"

	"github.com/cago-frame/cago/pkg/logger"
	"github.com/cago-frame/cago/pkg/opentelemetry/trace"
	"github.com/robfig/cron/v3"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
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
		c.run(spec, cmd)
	})
}

// run 执行一次定时任务：每次执行是一条独立的 trace，失败标 Error，日志带 trace_id
func (c *crontab) run(spec string, cmd func(ctx context.Context) error) {
	ctx := context.Background()
	name := funcName(cmd)
	var span trace2.Span
	if c.tracer != nil {
		ctx, span = c.tracer.Start(ctx, "cron "+name,
			trace2.WithSpanKind(trace2.SpanKindServer),
			trace2.WithAttributes(
				attribute.String("spec", spec),
				attribute.String("cron.job", name),
			),
		)
		defer span.End()
		ctx = logger.WithContextLogger(ctx, logger.Ctx(ctx).With(trace.LoggerLabel(ctx)...))
	}
	logger.Ctx(ctx).Info("cron start", zap.String("spec", spec), zap.String("job", name))
	if err := cmd(ctx); err != nil {
		logger.Ctx(ctx).Error("cron error", zap.String("job", name), zap.Error(err))
		if span != nil {
			span.RecordError(err)
			span.SetStatus(codes.Error, err.Error())
		}
	}
}

// funcName 取任务函数名（去掉包路径和方法值的 -fm 后缀），用来区分不同的定时任务
func funcName(f any) string {
	fn := runtime.FuncForPC(reflect.ValueOf(f).Pointer())
	if fn == nil {
		return "unknown"
	}
	name := fn.Name()
	if i := strings.LastIndex(name, "/"); i >= 0 {
		name = name[i+1:]
	}
	return strings.TrimSuffix(name, "-fm")
}
