package grpc

import (
	"context"
	"errors"
	"net"

	"github.com/cago-frame/cago"
	"github.com/cago-frame/cago/configs"
	"github.com/cago-frame/cago/pkg/gogo"
	"github.com/cago-frame/cago/pkg/logger"
	"github.com/cago-frame/cago/pkg/opentelemetry/metric"
	"github.com/cago-frame/cago/pkg/opentelemetry/trace"
	"go.opentelemetry.io/contrib/instrumentation/google.golang.org/grpc/otelgrpc"
	"go.uber.org/zap"
	"google.golang.org/grpc"
)

// Config grpc服务配置
type Config struct {
	Address string `yaml:"address"`
}

// Callback 注册grpc服务的回调函数
type Callback func(ctx context.Context, s *grpc.Server) error

// RegisterServerOptionFunc 注册grpc server option的函数
type RegisterServerOptionFunc func(cfg *configs.Config) ([]grpc.ServerOption, error)

var registerServerOptions []RegisterServerOptionFunc

// RegisterServerOption 注册全局grpc server option，类似于mux.RegisterMiddleware
func RegisterServerOption(f RegisterServerOptionFunc) {
	registerServerOptions = append(registerServerOptions, f)
}

type server struct {
	callback Callback
	opts     []grpc.ServerOption
}

// GRPC grpc服务组件,需要先注册logger组件
// 可以通过opts传入自定义的grpc.ServerOption，例如拦截器
func GRPC(callback Callback, opts ...grpc.ServerOption) cago.ComponentCancel {
	return &server{
		callback: callback,
		opts:     opts,
	}
}

func (s *server) Start(ctx context.Context, cfg *configs.Config) error {
	return s.StartCancel(ctx, nil, cfg)
}

func (s *server) StartCancel(
	ctx context.Context,
	cancel context.CancelFunc,
	cfg *configs.Config,
) error {
	config := &Config{}
	err := cfg.Scan(ctx, "grpc", config)
	if err != nil {
		return err
	}
	if config.Address == "" {
		config.Address = "127.0.0.1:9090"
	}
	l := logger.Default()

	// 收集server options
	serverOpts := make([]grpc.ServerOption, 0)

	// 自动接入链路追踪和metrics
	otelOpts := make([]otelgrpc.Option, 0)
	if tp := trace.Default(); tp != nil {
		otelOpts = append(otelOpts, otelgrpc.WithTracerProvider(tp))
	}
	if mp := metric.Default(); mp != nil {
		otelOpts = append(otelOpts, otelgrpc.WithMeterProvider(mp))
	}
	if len(otelOpts) > 0 {
		serverOpts = append(serverOpts,
			grpc.StatsHandler(otelgrpc.NewServerHandler(otelOpts...)),
		)
	}

	// 全局注册的server options
	for _, f := range registerServerOptions {
		opts, err := f(cfg)
		if err != nil {
			return err
		}
		serverOpts = append(serverOpts, opts...)
	}

	// 用户自定义的server options
	serverOpts = append(serverOpts, s.opts...)

	srv := grpc.NewServer(serverOpts...)

	// 注册grpc服务
	if err := s.callback(ctx, srv); err != nil {
		return errors.New("failed to register grpc server: " + err.Error())
	}

	lis, err := net.Listen("tcp", config.Address)
	if err != nil {
		return errors.New("failed to listen grpc: " + err.Error())
	}

	// 优雅关闭
	gogo.Go(func() error {
		<-ctx.Done()
		l.Info("grpc server closing...")
		srv.GracefulStop()
		l.Info("grpc server closed")
		return nil
	})

	// 启动grpc服务
	gogo.Go(func() error {
		defer func() {
			if cancel != nil {
				cancel()
			}
		}()
		l.Info("grpc server started", zap.String("address", config.Address))
		if err := srv.Serve(lis); err != nil {
			l.Error("failed to start grpc server", zap.Error(err))
			return err
		}
		return nil
	})

	return nil
}

func (s *server) CloseHandle() {
}
