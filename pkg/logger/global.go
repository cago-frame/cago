package logger

import (
	"context"
	"os"

	"github.com/cago-frame/cago/configs"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

type loggerContextKeyType int

const loggerKey loggerContextKeyType = iota

type InitLogger func(ctx context.Context, config *configs.Config, loggerConfig *Config) ([]Option, error)

var (
	logger     = zap.L()
	initLogger = make([]InitLogger, 0)
)

func RegistryInitLogger(f InitLogger) {
	initLogger = append(initLogger, f)
}

// Logger 日志组件,核心组件,必须提前注册
func Logger(ctx context.Context, config *configs.Config) error {
	cfg := &Config{}
	if err := config.Scan(ctx, "logger", cfg); err != nil {
		return err
	}
	opts := make([]Option, 0)
	if cfg.Level != "" {
		opts = append(opts, Level(cfg.Level))
	}
	level := ToLevel(cfg.Level)
	if cfg.LogFile.Enable {
		if cfg.LogFile.Filename != "" {
			opts = append(opts, AppendCore(NewFileCore(level, cfg.LogFile.Filename)))
		}
		if cfg.LogFile.ErrorFilename != "" {
			opts = append(opts, AppendCore(NewFileCore(zap.ErrorLevel, cfg.LogFile.ErrorFilename)))
		}
	}
	if config.Debug {
		opts = append(opts, AppendCore(zapcore.NewCore(
			zapcore.NewConsoleEncoder(zap.NewDevelopmentEncoderConfig()),
			zapcore.Lock(os.Stdout),
			zapcore.DebugLevel,
		)))
	} else {
		if !cfg.DisableConsole {
			opts = append(opts, AppendCore(zapcore.NewCore(
				zapcore.NewJSONEncoder(zap.NewProductionEncoderConfig()),
				zapcore.Lock(os.Stdout),
				level,
			)))
		}
	}
	for _, f := range initLogger {
		o, err := f(ctx, config, cfg)
		if err != nil {
			return err
		}
		opts = append(opts, o...)
	}
	l, err := New(opts...)
	if err != nil {
		return err
	}
	logger = l
	return nil
}

// SetLogger 设置全局日志实例
func SetLogger(l *zap.Logger) {
	logger = l
}

// Default 默认日志，尽量不要使用，会丢失上下文信息
func Default() *zap.Logger {
	return logger
}

// With 默认日志添加字段，尽量不要使用，会丢失上下文信息
func With(fields ...zap.Field) *zap.Logger {
	return logger.With(fields...)
}

// Ctx 从上下文中获取日志实例
func Ctx(ctx context.Context) *zap.Logger {
	log, ok := ctx.Value(loggerKey).(*zap.Logger)
	if !ok {
		return logger
	}
	return log
}

// WithContextLogger 将日志实例存入上下文
// 在想为后续操作指定日志实例时使用
// logger.WithContextLogger(ctx, logger.With(zap.String("key", "value")))
func WithContextLogger(ctx context.Context, logger *zap.Logger) context.Context {
	return context.WithValue(ctx, loggerKey, logger)
}

// WithContextField 取出context中的日志实例并添加字段存入上下文
// 在想为后续操作添加字段时使用
// logger.WithContextField(ctx, zap.String("key", "value"))
func WithContextField(ctx context.Context, fields ...zap.Field) context.Context {
	return context.WithValue(ctx, loggerKey, Ctx(ctx).With(fields...))
}
