package testutils

import (
	"context"
	"github.com/cago-frame/cago/pkg/logger"
	"github.com/cago-frame/cago/pkg/opentelemetry/trace"
	"testing"
)

func TestMain(m *testing.M) {
	RunTestEnv()
}

func TestComponent(t *testing.T) {
	trace.SpanFromContext(context.Background()).SpanContext().IsValid()
	logger.Ctx(context.Background()).Info("TestComponent")
}
