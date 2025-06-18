package configs

import (
	"github.com/cago-frame/cago/configs/file"
	"github.com/cago-frame/cago/configs/source"
)

type NewSource func(cfg *Config, serialization file.Serialization) (source.Source, error)

var (
	defaultConfig *Config
	sources       = make(map[string]NewSource)
)

func Default() *Config {
	return defaultConfig
}

func RegistrySource(name string, f NewSource) {
	sources[name] = f
}
