package trace

import (
	"context"

	"github.com/cago-frame/cago/configs"
	"github.com/cago-frame/cago/server/mux"
	"github.com/gin-gonic/gin"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/propagation"
	semconv "go.opentelemetry.io/otel/semconv/v1.12.0"
	"go.opentelemetry.io/otel/trace"
)

func init() {
	mux.RegisterMiddleware(func(cfg *configs.Config, r *gin.Engine) error {
		if tp := Default(); tp != nil {
			// 加入链路追踪中间件
			r.Use(Middleware(cfg.AppName, tp))
		}
		return nil
	})
}

var tracerProvider trace.TracerProvider

// Trace 链路追踪组件,尽早注册,其他组件会判断是否存在,再进行使用
func Trace(ctx context.Context, config *configs.Config) error {
	cfg := &Config{}
	if err := config.Scan(ctx, "trace", cfg); err != nil {
		return err
	}
	tp, err := NewWithConfig(ctx, cfg, AppendAttributes(
		semconv.ServiceNameKey.String(config.AppName),
		semconv.ServiceVersionKey.String(config.Version),
		semconv.DeploymentEnvironmentKey.String(string(config.Env)),
	))
	if err != nil {
		return err
	}
	tracerProvider = tp
	otel.SetTracerProvider(tp)
	otel.SetTextMapPropagator(propagation.NewCompositeTextMapPropagator(
		propagation.TraceContext{},
		//propagation.Baggage{},
	))
	return nil
}

// Default 获取默认链路追踪组件
// 获取到后，其它组件可以判断是否存在，再进行使用
func Default() trace.TracerProvider {
	return tracerProvider
}
