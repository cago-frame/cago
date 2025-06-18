package metric

import (
	"context"

	"github.com/cago-frame/cago/configs"
	"github.com/cago-frame/cago/server/mux"
	"github.com/gin-gonic/gin"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/exporters/prometheus"
	"go.opentelemetry.io/otel/sdk/metric"
)

func init() {
	mux.RegisterMiddleware(func(cfg *configs.Config, r *gin.Engine) error {
		// 加入metrics中间件
		if Default() != nil {
			m, err := Middleware(Default())
			if err != nil {
				return err
			}
			r.Use(m)
			r.GET("/metrics", gin.WrapH(promhttp.Handler()))
		}
		return nil
	})
}

var provider *metric.MeterProvider

func Metrics(ctx context.Context, cfg *configs.Config) error {
	// 初始化全局Meter实例并绑定Prometheus Exporter
	exporter, err := prometheus.New()
	if err != nil {
		return err
	}
	provider = metric.NewMeterProvider(metric.WithReader(exporter))
	otel.SetMeterProvider(provider)
	return nil
}

func Default() *metric.MeterProvider {
	return provider
}
