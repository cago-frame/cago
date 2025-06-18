package cron

import (
	"context"

	"github.com/cago-frame/cago"
	"github.com/cago-frame/cago/configs"
	"github.com/cago-frame/cago/pkg/opentelemetry/trace"
	"github.com/robfig/cron/v3"
	trace2 "go.opentelemetry.io/otel/trace"
)

const (
	tracerName = "github.com/cago-frame/cago/server/cron"
)

type server struct {
	cron *cron.Cron
}

var defaultCrontab Crontab

// Cron 定时任务组件,需要先注册logger和redis组件
func Cron() cago.Component {
	return &server{
		cron: cron.New(),
	}
}

func (s *server) Start(ctx context.Context, cfg *configs.Config) error {
	var tracer trace2.Tracer
	if trace.Default() != nil {
		tracer = trace.Default().Tracer(
			tracerName,
			trace2.WithInstrumentationVersion("0.1.0"),
		)
	}
	defaultCrontab = &crontab{tracer: tracer, cron: s.cron}
	s.cron.Start()
	return nil
}

func (s *server) CloseHandle() {
	s.cron.Stop()
}

func Default() Crontab {
	return defaultCrontab
}
