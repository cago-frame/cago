package audit_logger

import (
	"context"

	"github.com/cago-frame/cago/pkg/logger"
	"go.uber.org/zap"
)

type Logger struct {
	logger *zap.Logger
}

func NewLoggerStorage() *Logger {
	return &Logger{
		logger: logger.Default(),
	}
}

func (l *Logger) Record(ctx context.Context, module, eventName string, fields ...zap.Field) error {
	l.logger.Info("审计日志",
		append(fields, zap.String("module", module), zap.String("event", eventName))...)
	return nil
}
