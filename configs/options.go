package configs

import (
	"github.com/cago-frame/cago/configs/file"
	"github.com/cago-frame/cago/configs/source"
)

type Option func(*Options)

type Options struct {
	file          string
	serialization file.Serialization
	source        source.Source
}

func WithConfigFile(file string) Option {
	return func(options *Options) {
		options.file = file
	}
}

func WithSerialization(serialization file.Serialization) Option {
	return func(options *Options) {
		options.serialization = serialization
	}
}

func WithSource(source source.Source) Option {
	return func(options *Options) {
		options.source = source
	}
}
