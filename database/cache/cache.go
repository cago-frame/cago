package cache

import (
	"context"
	"errors"

	"github.com/cago-frame/cago"
	"github.com/cago-frame/cago/configs"
	cache2 "github.com/cago-frame/cago/database/cache/cache"
	"github.com/cago-frame/cago/database/cache/memory"
	"github.com/cago-frame/cago/database/cache/redis"
	redis2 "github.com/redis/go-redis/v9"
)

const (
	Redis  Type = "redis"
	Memory Type = "memory"
)

type Type string

type Config struct {
	Type
	Addr     string
	Password string
	DB       int
}

var defaultCache cache2.Cache

type cache struct {
	cache2.Cache
}

func Cache() cago.Component {
	return &cache{}
}

func (c *cache) Start(ctx context.Context, config *configs.Config) error {
	cfg := &Config{}
	if err := config.Scan(ctx, "cache", cfg); err != nil {
		return err
	}
	cache, err := NewWithConfig(ctx, cfg)
	if err != nil {
		return err
	}
	c.Cache = cache
	defaultCache = cache
	return nil
}

func (c *cache) CloseHandle() {
	_ = c.Close()
}

func NewWithConfig(ctx context.Context, cfg *Config, opts ...cache2.Option) (cache2.Cache, error) {
	switch cfg.Type {
	case Redis:
		return redis.NewRedisCache(&redis2.Options{
			Addr:     cfg.Addr,
			Password: cfg.Password,
			DB:       cfg.DB,
		})
	case Memory:
		return memory.NewMemoryCache()
	default:
		return nil, errors.New("not support cache type")
	}
}

func Default() cache2.Cache {
	return defaultCache
}

func SetDefault(cache cache2.Cache) {
	defaultCache = cache
}
