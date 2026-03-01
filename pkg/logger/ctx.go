package logger

import (
	"context"

	"go.uber.org/zap"
)

type CtxLogger struct {
	*zap.Logger
	labels []zap.Field
}

func NewCtxLogger(logger *zap.Logger) *CtxLogger {
	return &CtxLogger{
		Logger: logger,
		labels: make([]zap.Field, 0),
	}
}

func (c *CtxLogger) Ctx(ctx context.Context) *CtxLogger {
	return &CtxLogger{
		Logger: Ctx(ctx).With(c.labels...),
	}
}

func (c *CtxLogger) With(fields ...zap.Field) *CtxLogger {
	return &CtxLogger{
		Logger: c.Logger.With(fields...),
		labels: append(c.labels, fields...),
	}
}
