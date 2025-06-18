package component

import (
	"context"

	"github.com/cago-frame/cago"
	"github.com/cago-frame/cago/configs"
	_ "github.com/cago-frame/cago/configs/etcd"
	"github.com/cago-frame/cago/database/cache"
	"github.com/cago-frame/cago/database/db"
	"github.com/cago-frame/cago/database/elasticsearch"
	"github.com/cago-frame/cago/database/mongo"
	"github.com/cago-frame/cago/database/redis"
	"github.com/cago-frame/cago/pkg/broker"
	"github.com/cago-frame/cago/pkg/logger"
	_ "github.com/cago-frame/cago/pkg/logger/loki"
	"github.com/cago-frame/cago/pkg/opentelemetry/metric"
	"github.com/cago-frame/cago/pkg/opentelemetry/trace"
	"github.com/cago-frame/cago/server/mux"
	"github.com/gin-gonic/gin"
	swaggerFiles "github.com/swaggo/files"
	ginSwagger "github.com/swaggo/gin-swagger"
)

// Core 核心组件,包括日志组件、链路追踪、指标
// 日志组件必须注册，链路追踪和指标注册了后，某些组件会根据它们自动开启相关功能
func Core() cago.FuncComponent {
	mux.RegisterMiddleware(func(cfg *configs.Config, r *gin.Engine) error {
		if cfg.Env != configs.PROD {
			url := ginSwagger.URL("/swagger/doc.json")
			r.GET("/swagger/*any", ginSwagger.WrapHandler(swaggerFiles.Handler, url))
		}
		return nil
	})
	return func(ctx context.Context, cfg *configs.Config) error {
		// 日志组件必须注册
		if err := logger.Logger(ctx, cfg); err != nil {
			return err
		}
		// 判断是否有trace配置
		if ok, err := cfg.Has(ctx, "trace"); err != nil {
			return err
		} else if ok {
			if err := trace.Trace(ctx, cfg); err != nil {
				return err
			}
		}
		// metrics组件
		if err := metric.Metrics(ctx, cfg); err != nil {
			return err
		}
		return nil
	}
}

// Logger 日志组件
func Logger() cago.FuncComponent {
	return logger.Logger
}

// Trace 链路追踪组件
func Trace() cago.FuncComponent {
	return trace.Trace
}

// Metrics 指标组件
func Metrics() cago.FuncComponent {
	return metric.Metrics
}

// Database 数据库组件
func Database() cago.Component {
	return db.Database()
}

// Broker 消息队列组件
func Broker() cago.FuncComponent {
	return broker.Broker
}

// Mongo mongodb组件
func Mongo() cago.FuncComponent {
	return mongo.Mongo
}

// Redis redis组件
func Redis() cago.FuncComponent {
	return redis.Redis
}

// Cache 缓存组件
func Cache() cago.Component {
	return cache.Cache()
}

// Elasticsearch elasticsearch组件
func Elasticsearch() cago.FuncComponent {
	return elasticsearch.Elasticsearch
}
