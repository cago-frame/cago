package memory

import (
	"context"
	"time"

	"github.com/cago-frame/cago/database/cache/cache"
	gocache "github.com/patrickmn/go-cache"
)

type memoryCache struct {
	cache *gocache.Cache
}

func NewMemoryCache() (cache.Cache, error) {
	c := gocache.New(5*time.Minute, 10*time.Minute)
	return &memoryCache{
		cache: c,
	}, nil
}

func (m *memoryCache) GetOrSet(ctx context.Context, key string, set func() (interface{}, error), opts ...cache.Option) cache.Value {
	ret := m.Get(ctx, key, opts...)
	if ret.Err() != nil {
		val, err := set()
		if err != nil {
			return cache.NewValue(ctx, "", cache.NewOptions(opts...), err)
		}
		return m.Set(ctx, key, val, opts...)
	}
	return &cache.GetOrSetValue{Value: ret, Set: func() cache.Value {
		val, err := set()
		if err != nil {
			return cache.NewValue(ctx, "", cache.NewOptions(opts...), err)
		}
		return m.Set(ctx, key, val, opts...)
	}}
}

func (m *memoryCache) Set(ctx context.Context, key string, val interface{}, opts ...cache.Option) cache.Value {
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
	m.cache.Set(key, s, ttl)
	if options.Depend != nil {
		// 移除掉依赖
		options.Depend = &cache.NilDep{}
	}
	return cache.NewValue(ctx, s, options, err)
}

func (m *memoryCache) Get(ctx context.Context, key string, opts ...cache.Option) cache.Value {
	data, ok := m.cache.Get(key)
	options := cache.NewOptions(opts...)
	if !ok {
		return cache.NewValue(ctx, "", options, cache.ErrNil)
	}
	return cache.NewValue(ctx, data.(string), options, nil)
}

func (m *memoryCache) Has(ctx context.Context, key string) (bool, error) {
	_, ok := m.cache.Get(key)
	return ok, nil
}

func (m *memoryCache) Del(ctx context.Context, key string) error {
	m.cache.Delete(key)
	return nil
}

func (m *memoryCache) Close() error {
	return nil
}
