package audit

import (
	"context"

	"github.com/cago-frame/cago/pkg/iam/audit/audit_logger"
	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
)

type Audit struct {
	storage Storage
	// 模块
	module string
	// 字段
	fields []zap.Field
}

type Options struct {
	storage Storage
}

type Option func(*Options)

func newOptions(opts ...Option) *Options {
	options := &Options{
		storage: audit_logger.NewLoggerStorage(),
	}
	for _, o := range opts {
		o(options)
	}
	return options
}

func WithStorage(storage Storage) Option {
	return func(options *Options) {
		options.storage = storage
	}
}

// NewAudit 创建审计
// 你可以实现 Storage 接口来自定义审计存储
// 默认的话可以使用日志组件来存储审计日志
func NewAudit(opts ...Option) *Audit {
	options := newOptions(opts...)
	return &Audit{
		storage: options.storage,
	}
}

// Module 设置模块
func (a *Audit) Module(module string) *Audit {
	return &Audit{
		storage: a.storage,
		module:  module,
		fields:  a.fields,
	}
}

// With 添加字段
func (a *Audit) With(fields ...zap.Field) *Audit {
	return &Audit{
		storage: a.storage,
		module:  a.module,
		fields:  fields,
	}
}

// Record 记录审计日志
func (a *Audit) Record(ctx context.Context, eventName string, fields ...zap.Field) error {
	fields = append(fields, a.fields...)
	return a.storage.Record(ctx, a.module, eventName, fields...)
}

// Middleware 中间件 可以添加自定义的字段
func (a *Audit) Middleware(module string, getFields func(ctx *gin.Context) []zap.Field) gin.HandlerFunc {
	return func(ctx *gin.Context) {
		// 添加会话相关信息, 例如ip、user-agent等
		audit := Ctx(ctx).Module(module)
		fields := []zap.Field{
			zap.String("ip", ctx.ClientIP()),
			zap.String("user-agent", ctx.GetHeader("User-Agent")),
		}
		if getFields != nil {
			fields = append(fields, getFields(ctx)...)
		}
		audit = audit.With(fields...)
		ctx.Request = ctx.Request.WithContext(WithAudit(ctx.Request.Context(), audit))
	}
}

type CtxAudit struct {
	context.Context
	*Audit
}

func NewCtxAudit(ctx context.Context, audit *Audit) *CtxAudit {
	return &CtxAudit{
		Context: ctx, Audit: audit,
	}
}

func (c *CtxAudit) Record(eventName string, fields ...zap.Field) error {
	fields = append(fields, c.Audit.fields...)
	return c.Audit.storage.Record(c.Context, c.Audit.module, eventName, fields...)
}
