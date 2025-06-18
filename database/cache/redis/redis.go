package redis

import (
	"context"
	"encoding/json"
	"time"

	"github.com/cago-frame/cago/database/cache/cache"
	"github.com/cago-frame/cago/pkg/opentelemetry/metric"
	"github.com/cago-frame/cago/pkg/opentelemetry/trace"
	"github.com/redis/go-redis/extra/redisotel/v9"
	"github.com/redis/go-redis/v9"
)

type redisCache struct {
	redis *redis.Client
}

func NewRedisCache(config *redis.Options) (cache.Cache, error) {
	client := redis.NewClient(config)
	err := client.Ping(context.Background()).Err()
	if err != nil {
		return nil, err
	}
	if tp := trace.Default(); tp != nil {
		if err := redisotel.InstrumentTracing(client,
			redisotel.WithTracerProvider(tp),
			redisotel.WithDBSystem("cache"),
		); err != nil {
			return nil, err
		}
	}
	if metric.Default() != nil {
		if err := redisotel.InstrumentMetrics(client,
			redisotel.WithMeterProvider(metric.Default()),
		); err != nil {
			return nil, err
		}
	}
	return &redisCache{
		redis: client,
	}, nil
}

func (r *redisCache) GetOrSet(ctx context.Context, key string, set func() (interface{}, error), opts ...cache.Option) cache.Value {
	ret := r.Get(ctx, key, opts...)
	if ret.Err() != nil {
		val, err := set()
		if err != nil {
			return cache.NewValue(ctx, "", cache.NewOptions(opts...), err)
		}
		return r.Set(ctx, key, val, opts...)
	}
	return &cache.GetOrSetValue{Value: ret, Set: func() cache.Value {
		val, err := set()
		if err != nil {
			return cache.NewValue(ctx, "", cache.NewOptions(opts...), err)
		}
		return r.Set(ctx, key, val, opts...)
	}}
}

func (r *redisCache) Unmarshal(data []byte, v interface{}) error {
	return json.Unmarshal(data, v)
}

func (r *redisCache) Marshal(v interface{}) ([]byte, error) {
	return json.Marshal(v)
}

func (r *redisCache) Get(ctx context.Context, key string, opts ...cache.Option) cache.Value {
	data, err := r.redis.Get(ctx, key).Result()
	if err == redis.Nil {
		err = cache.ErrNil
	}
	options := cache.NewOptions(opts...)
	return cache.NewValue(ctx, data, options, err)
}

func (r *redisCache) Set(ctx context.Context, key string, val interface{}, opts ...cache.Option) cache.Value {
	options := cache.NewOptions(opts...)
	ttl := time.Duration(0)
	if options.Expiration > 0 {
		ttl = options.Expiration
	}
	data, err := cache.Marshal(ctx, val, options)
	if err != nil {
		return cache.NewValue(ctx, "", options, err)
	}
	s := string(data)
	if err := r.redis.Set(ctx, key, s, ttl).Err(); err != nil {
		return cache.NewValue(ctx, "", options, err)
	}
	if options.Depend != nil {
		// 移除掉依赖
		options.Depend = &cache.NilDep{}
	}
	return cache.NewValue(ctx, s, options, err)
}

func (r *redisCache) Has(ctx context.Context, key string) (bool, error) {
	ok, err := r.redis.Exists(ctx, key).Result()
	if err != nil {
		return false, err
	}
	if ok == 1 {
		return true, nil
	}
	return false, nil
}

func (r *redisCache) Del(ctx context.Context, key string) error {
	return r.redis.Del(ctx, key).Err()
}

func (r *redisCache) Close() error {
	return r.redis.Close()
}
