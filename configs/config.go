package configs

import (
	"context"
	"reflect"
	"strings"

	"github.com/mitchellh/mapstructure"

	"github.com/cago-frame/cago/configs/file"
	"github.com/cago-frame/cago/configs/source"
)

type Env string

// Version 编译时注入 -w -s -X github.com/cago-frame/cago/configs.Version=1.0.0
var Version = "1.0.0"

const (
	DEV  Env = "dev"
	TEST Env = "test"
	PRE  Env = "pre"
	PROD Env = "prod"
)

type Config struct {
	AppName       string
	Version       string
	Env           Env
	Debug         bool
	source        source.Source
	serialization file.Serialization
}

// NewConfig 创建配置
func NewConfig(appName string, opt ...Option) (*Config, error) {
	ctx := context.Background()
	options := &Options{
		file:          "./configs/config.yaml",
		serialization: file.Yaml(),
	}
	for _, o := range opt {
		o(options)
	}
	var s source.Source
	if options.source == nil {
		var err error
		s, err = file.NewSource(options.file, options.serialization)
		if err != nil {
			return nil, err
		}
	} else {
		s = options.source
	}
	var env Env
	if err := s.Scan(ctx, "env", &env); err != nil {
		return nil, err
	}
	var debug bool
	if err := s.Scan(ctx, "debug", &debug); err != nil {
		return nil, err
	}
	c := &Config{
		AppName:       appName,
		Debug:         debug,
		Env:           env,
		Version:       Version,
		source:        s,
		serialization: options.serialization,
	}
	if err := c.init(); err != nil {
		return nil, err
	}
	defaultConfig = c
	return c, nil
}

func (c *Config) init() error {
	configSource := ""
	err := c.source.Scan(context.Background(), "source", &configSource)
	if err != nil {
		return err
	}
	if configSource == "" || configSource == "file" {
		return nil
	}
	c.source, err = sources[configSource](c, c.serialization)
	if err != nil {
		return err
	}
	return nil
}

// Scan 读取配置，可以将配置读取到结构体中
func (c *Config) Scan(ctx context.Context, key string, value interface{}) error {
	keys := strings.Split(key, ".")
	if len(keys) == 1 {
		return c.source.Scan(ctx, key, value)
	}
	var i interface{}
	if err := c.findKey(ctx, key, &i); err != nil {
		return err
	}
	return mapstructure.Decode(i, value)
}

func (c *Config) findKey(ctx context.Context, key string, value interface{}) error {
	keys := strings.Split(key, ".")
	if len(keys) == 1 {
		return c.source.Scan(ctx, key, value)
	}
	valueMap := make(map[string]interface{})
	if err := c.source.Scan(ctx, keys[0], &valueMap); err != nil {
		return err
	}
	for i := 1; i < len(keys); i++ {
		if v, ok := valueMap[keys[i]]; ok {
			if i == len(keys)-1 {
				reflect.ValueOf(value).Elem().Set(reflect.ValueOf(v))
				return nil
			}
			valueMap = v.(map[string]interface{})
		} else {
			return nil
		}
	}
	return nil
}

// String 获取配置，返回字符串
func (c *Config) String(ctx context.Context, key string) string {
	var str string
	if err := c.findKey(ctx, key, &str); err != nil {
		return ""
	}
	return str
}

// Bool 获取配置，返回bool
func (c *Config) Bool(ctx context.Context, key string) bool {
	var b bool
	if err := c.findKey(ctx, key, &b); err != nil {
		return false
	}
	return b
}

// Has 判断配置是否存在
func (c *Config) Has(ctx context.Context, key string) (bool, error) {
	return c.source.Has(ctx, key)
}

// Watch 监听配置变化
func (c *Config) Watch(ctx context.Context, key string, callback func(event source.Event)) error {
	return c.source.Watch(ctx, key, callback)
}
