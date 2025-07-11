package testutils

import (
	"context"
	"github.com/cago-frame/cago"
	"github.com/cago-frame/cago/configs"
	"github.com/cago-frame/cago/configs/memory"
	"github.com/cago-frame/cago/pkg/component"
	"github.com/cago-frame/cago/pkg/logger"
	"github.com/cago-frame/cago/pkg/opentelemetry/trace"
	"testing"
)

func RunTest(m *testing.M) int {
	cfg, err := configs.NewConfig("arb-bot", configs.WithSource(
		memory.NewSource(map[string]interface{}{
			"logger": logger.Config{
				Level:          "info",
				DisableConsole: true,
				LogFile:        logger.LogFileConfig{Enable: false, Filename: "robot.log"},
			},
			"trace": trace.Config{
				Type: "noop",
			},
		}),
	))
	if err != nil {
		panic(err.Error())
	}
	cago.New(context.Background(), cfg).
		Registry(component.Core())

	return m.Run()
}
