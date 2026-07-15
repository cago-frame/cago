package logger

import (
	"io"

	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
	lumberjack "gopkg.in/natefinch/lumberjack.v2"
)

// NewConsole creates a logger suitable for short-lived CLI commands. Unlike
// the application logger, it does not require a config file and omits
// timestamps and caller information to keep terminal output concise.
func NewConsole(w io.Writer, level string) *zap.Logger {
	encoderConfig := zapcore.EncoderConfig{
		LevelKey:    "level",
		MessageKey:  "message",
		EncodeLevel: zapcore.CapitalLevelEncoder,
	}
	return zap.New(zapcore.NewCore(
		zapcore.NewConsoleEncoder(encoderConfig),
		zapcore.Lock(zapcore.AddSync(w)),
		ToLevel(level),
	))
}

type Config struct {
	Level          string
	DisableConsole bool          `yaml:"disableConsole"`
	LogFile        LogFileConfig `yaml:"logFile"`
}

type LogFileConfig struct {
	Enable        bool
	Filename      string
	ErrorFilename string `yaml:"errorFilename"`
}

func New(opt ...Option) (*zap.Logger, error) {
	options := &Options{}
	for _, o := range opt {
		o(options)
	}
	core := make([]zapcore.Core, 0, 1)
	level := ToLevel(options.level)
	levelEnable := zap.LevelEnablerFunc(func(lvl zapcore.Level) bool {
		return lvl >= level
	})
	if options.w != nil {
		encodeConfig := zap.NewProductionEncoderConfig()
		encodeConfig.EncodeTime = zapcore.ISO8601TimeEncoder
		encode := zapcore.NewJSONEncoder(encodeConfig)
		core = append(core, zapcore.NewCore(
			encode,
			zapcore.AddSync(options.w),
			levelEnable,
		))
	}
	if options.cores != nil {
		core = append(core, options.cores...)
	}
	logger := zap.New(zapcore.NewTee(core...), zap.AddCaller())
	return logger, nil
}

func ToLevel(level string) zapcore.Level {
	switch level {
	case "debug":
		return zap.DebugLevel
	case "info":
		return zap.InfoLevel
	case "warn":
		return zap.WarnLevel
	case "error":
		return zap.ErrorLevel
	}
	return zap.InfoLevel
}

func NewFileCore(level zapcore.Level, filename string) zapcore.Core {
	var w io.Writer = &lumberjack.Logger{
		Filename:   filename,
		MaxSize:    2,
		MaxBackups: 10,
		MaxAge:     30,
		LocalTime:  true,
		Compress:   false,
	}
	encodeConfig := zap.NewProductionEncoderConfig()
	encodeConfig.EncodeTime = zapcore.ISO8601TimeEncoder
	encode := zapcore.NewJSONEncoder(encodeConfig)
	return zapcore.NewCore(
		encode,
		zapcore.AddSync(w),
		level,
	)
}
