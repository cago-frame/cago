package cron

import (
	"context"
	"errors"
	"strings"
	"testing"

	"github.com/cago-frame/cago/pkg/logger"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.opentelemetry.io/otel/codes"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	"go.opentelemetry.io/otel/sdk/trace/tracetest"
	"go.uber.org/zap"
	"go.uber.org/zap/zaptest/observer"
)

type jobs struct{}

func (jobs) syncScripts(context.Context) error { return errors.New("sync failed") }
func (jobs) cleanup(context.Context) error     { return nil }

func newTracedCrontab(t *testing.T) (*crontab, *tracetest.SpanRecorder, *observer.ObservedLogs) {
	t.Helper()
	rec := tracetest.NewSpanRecorder()
	tp := sdktrace.NewTracerProvider(sdktrace.WithSpanProcessor(rec))

	core, logs := observer.New(zap.InfoLevel)
	prev := logger.Default()
	logger.SetLogger(zap.New(core))
	t.Cleanup(func() { logger.SetLogger(prev) })

	return &crontab{tracer: tp.Tracer("test")}, rec, logs
}

func TestCrontab_FailedJobMarksSpanError(t *testing.T) {
	c, rec, logs := newTracedCrontab(t)

	c.run("0 */6 * * *", jobs{}.syncScripts)

	spans := rec.Ended()
	require.Len(t, spans, 1)
	span := spans[0]
	assert.Equal(t, codes.Error, span.Status().Code, "任务失败要标 Error，尾部采样才会保留")
	require.NotEmpty(t, span.Events())
	assert.Equal(t, "exception", span.Events()[0].Name)
	assert.True(t, strings.HasPrefix(span.Name(), "cron "), "span 名：%q", span.Name())
	assert.Contains(t, span.Name(), "syncScripts", "span 名应能区分是哪个任务")

	// 日志带上 trace_id，才能从日志跳到链路
	errLogs := logs.FilterMessage("cron error").All()
	require.Len(t, errLogs, 1)
	assert.Equal(t, span.SpanContext().TraceID().String(), errLogs[0].ContextMap()["trace_id"])
}

func TestCrontab_SuccessfulJobKeepsStatusUnset(t *testing.T) {
	c, rec, _ := newTracedCrontab(t)

	c.run("@every 60s", jobs{}.cleanup)

	spans := rec.Ended()
	require.Len(t, spans, 1)
	assert.Equal(t, codes.Unset, spans[0].Status().Code)
	assert.Contains(t, spans[0].Name(), "cleanup")
}

func TestCrontab_NoTracerStillRuns(t *testing.T) {
	c := &crontab{}
	var called bool
	c.run("@every 60s", func(context.Context) error {
		called = true
		return nil
	})
	assert.True(t, called)
}
