package mux

import (
	"context"
	"errors"
	"net/http"
	"time"

	"github.com/cago-frame/cago"
	"github.com/cago-frame/cago/configs"
	"github.com/cago-frame/cago/middleware"
	"github.com/cago-frame/cago/pkg/gogo"
	"github.com/cago-frame/cago/pkg/logger"
	"github.com/cago-frame/cago/pkg/utils/validator"
	"github.com/gin-gonic/gin"
	"github.com/gin-gonic/gin/binding"
	"go.uber.org/zap"
)

type Config struct {
	Address []string `yaml:"address"`
}

type Callback func(ctx context.Context, r *Router) error

type server struct {
	//ctx context.Context
	//cancel   context.CancelFunc
	callback Callback
}

// HTTP http服务组件,需要先注册logger组件
func HTTP(callback Callback) cago.ComponentCancel {
	return &server{
		callback: callback,
	}
}

func (h *server) Start(ctx context.Context, cfg *configs.Config) error {
	return h.StartCancel(ctx, nil, cfg)
}

func (h *server) StartCancel(
	ctx context.Context,
	cancel context.CancelFunc,
	cfg *configs.Config,
) error {
	config := &Config{}
	err := cfg.Scan(ctx, "http", config)
	if err != nil {
		return err
	}
	l := logger.Default()
	var r *gin.Engine
	if cfg.Debug {
		gin.SetMode(gin.DebugMode)
		r = gin.New()
		r.Use(gin.Logger(), Recover())
	} else {
		gin.SetMode(gin.ReleaseMode)
		r = gin.New()
		r.Use(Recover())
	}
	binding.Validator, err = validator.NewValidator()
	if err != nil {
		return err
	}
	// ginContext支持fallback
	r.ContextWithFallback = true
	// 加入日志中间件
	r.Use(middleware.Logger(logger.Default()))
	// 加入健康检查
	r.GET("/health", func(c *gin.Context) {
		c.String(http.StatusOK, "ok")
	})
	for _, f := range registerMiddleware {
		if err := f(cfg, r); err != nil {
			return err
		}
	}
	if err := h.callback(ctx, &Router{
		Routes:  &Routes{IRoutes: r},
		IRouter: r,
	}); err != nil {
		return errors.New("failed to register http server: " + err.Error())
	}
	if len(config.Address) == 0 {
		config.Address = []string{"127.0.0.1:80"}
	}
	srv := &http.Server{
		Addr:              config.Address[0],
		Handler:           r.Handler(),
		ReadHeaderTimeout: 5 * time.Second,
	}
	_ = gogo.Go(func(ctx context.Context) error {
		<-ctx.Done()
		l.Info("http server closing...")
		if err := srv.Shutdown(context.Background()); err != nil {
			l.Error("failed to close http server", zap.Error(err))
			return err
		}
		l.Info("http server closed")
		return nil
	}, gogo.WithContext(ctx))
	// 启动http服务
	_ = gogo.Go(func(ctx context.Context) error {
		defer cancel()
		if err := srv.ListenAndServe(); err != nil {
			if errors.Is(err, http.ErrServerClosed) {
				return nil
			}
			l.Error("failed to start http server", zap.Error(err))
			return err
		}
		return nil
	})
	return nil
}

func (h *server) CloseHandle() {

}
