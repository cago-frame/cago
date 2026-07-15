package main

import (
	"os"

	"github.com/cago-frame/cago/internal/cmd"
	"github.com/cago-frame/cago/internal/cmd/gen"
	_ "github.com/cago-frame/cago/pkg/component"
	"github.com/cago-frame/cago/pkg/logger"
	"github.com/spf13/cobra"
	"go.uber.org/zap"
)

func main() {
	os.Exit(run())
}

func run() int {
	rootCmd := &cobra.Command{
		Use:           "cago",
		Short:         "Cago 项目代码生成工具",
		SilenceErrors: true,
		SilenceUsage:  true,
	}

	init := cmd.NewInitCmd()
	rootCmd.AddCommand(init.Commands()...)

	genCmd := gen.NewGenCmd()
	rootCmd.AddCommand(genCmd.Commands()...)
	logger.SetLogger(logger.NewConsole(rootCmd.ErrOrStderr(), "info"))

	if err := rootCmd.Execute(); err != nil {
		logger.Default().Error("命令执行失败", zap.Error(err))
		return 1
	}
	return 0
}
