package wrap

import (
	stdContext "context"

	"github.com/cago-frame/cago/pkg/logger"
	"go.opentelemetry.io/otel/trace"
)

type Wrap struct {
	wraps []Handler
}

// New 创建一个包装器
func New() *Wrap {
	return &Wrap{}
}

func (w *Wrap) Wrap(f Handler) *Wrap {
	w.wraps = append(w.wraps, f)
	return w
}

func (w *Wrap) Run(sctx stdContext.Context, name string, args []interface{}, handler Handler) error {
	ctx := &Context{
		Context: sctx,
		name:    name,
		handler: append(w.wraps, handler),
		pos:     -1,
	}
	ctx.args = args
	ctx.Next()
	return ctx.IsAbort()
}

// UnWarContext 解包基础context
func UnWarContext(parentCtx stdContext.Context) stdContext.Context {
	ctx := stdContext.Background()
	ctx = logger.WithContextLogger(ctx, logger.Ctx(parentCtx))
	span := trace.SpanFromContext(ctx)
	if span.SpanContext().IsValid() {
		ctx = trace.ContextWithSpan(ctx, span)
	}
	return ctx
}
