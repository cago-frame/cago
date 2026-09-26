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
	// 以下是 lumberjack 的轮转参数，Filename 与 ErrorFilename 共用。
	// 零值沿用默认值，不配的应用行为与改动前一致。
	// MaxSize 单个文件的大小上限（MB），默认 2
	MaxSize int `yaml:"maxSize"`
	// MaxBackups 保留的已轮转文件个数，默认 10；负数表示不按个数清理
	MaxBackups int `yaml:"maxBackups"`
	// MaxAge 已轮转文件的保留天数，默认 30；负数表示不按天数清理
	MaxAge int `yaml:"maxAge"`
	// Compress 是否把已轮转的文件 gzip 压缩
	Compress bool `yaml:"compress"`
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

// NewFileCore 按默认轮转参数创建写文件的日志 core。
func NewFileCore(level zapcore.Level, filename string) zapcore.Core {
	return NewRotateFileCore(level, filename, LogFileConfig{})
}

// NewRotateFileCore 按 cfg 里的轮转参数创建写文件的日志 core，cfg 中的文件名字段不参与。
func NewRotateFileCore(level zapcore.Level, filename string, cfg LogFileConfig) zapcore.Core {
	var w io.Writer = newRotateWriter(filename, cfg)
	encodeConfig := zap.NewProductionEncoderConfig()
	encodeConfig.EncodeTime = zapcore.ISO8601TimeEncoder
	encode := zapcore.NewJSONEncoder(encodeConfig)
	return zapcore.NewCore(
		encode,
		zapcore.AddSync(w),
		level,
	)
}

func newRotateWriter(filename string, cfg LogFileConfig) *lumberjack.Logger {
	return &lumberjack.Logger{
		Filename:   filename,
		MaxSize:    orDefault(cfg.MaxSize, 2),
		MaxBackups: orDefault(cfg.MaxBackups, 10),
		MaxAge:     orDefault(cfg.MaxAge, 30),
		LocalTime:  true,
		Compress:   cfg.Compress,
	}
}

// orDefault 零值取默认值；负数映射成 lumberjack 的 0（不限）。
func orDefault(v, def int) int {
	switch {
	case v == 0:
		return def
	case v < 0:
		return 0
	}
	return v
}
