package cagolint

import (
	"github.com/cago-frame/cagolint/analyzer"
	"github.com/golangci/plugin-module-register/register"
	"golang.org/x/tools/go/analysis"
)

func init() {
	register.Plugin("cagolint", New)
}

type plugin struct {
	settings analyzer.Settings
}

// New creates the golangci-lint module plugin.
func New(rawSettings any) (register.LinterPlugin, error) {
	settings, err := register.DecodeSettings[analyzer.Settings](rawSettings)
	if err != nil {
		return nil, err
	}
	return &plugin{settings: settings}, nil
}

func (p *plugin) BuildAnalyzers() ([]*analysis.Analyzer, error) {
	return []*analysis.Analyzer{analyzer.New(p.settings)}, nil
}

func (p *plugin) GetLoadMode() string {
	return register.LoadModeTypesInfo
}
