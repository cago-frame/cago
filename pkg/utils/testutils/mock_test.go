package testutils

import (
	"context"
	"testing"

	"github.com/cago-frame/cago/pkg/logger"
	"github.com/cago-frame/cago/pkg/opentelemetry/trace"
)

func TestMain(m *testing.M) {
	RunTestEnv()
}

func TestComponent(t *testing.T) {
	trace.SpanFromContext(context.Background()).SpanContext().IsValid()
	logger.Ctx(context.Background()).Info("TestComponent")
}
