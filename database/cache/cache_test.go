package cache

import (
	"context"
	cache2 "github.com/cago-frame/cago/database/cache/cache"
	"github.com/cago-frame/cago/database/cache/memory"
	"github.com/stretchr/testify/assert"
	"testing"
)

func TestDepend(t *testing.T) {
	c, _ := memory.NewMemoryCache()
	dep := cache2.NewKeyDepend(c, "test:dep")
	c.Set(context.Background(), "test", 1, WithDepend(dep))
	result, err := c.Get(context.Background(), "test", WithDepend(cache2.NewKeyDepend(c, "test:dep"))).Int64()
	assert.NoError(t, err)
	assert.Equal(t, int64(1), result)
	err = dep.InvalidKey(context.Background())
	assert.NoError(t, err)
	result, err = c.Get(context.Background(), "test", WithDepend(cache2.NewKeyDepend(c, "test:dep"))).Int64()
	assert.Error(t, err)

	// 错误的depend格式
	c.Set(context.Background(), "test", 1, WithDepend(cache2.NewKeyDepend(c, "test:dep")))
	c.Set(context.Background(), "test:dep", "123456")
	result, err = c.Get(context.Background(), "test", WithDepend(cache2.NewKeyDepend(c, "test:dep"))).Int64()
	assert.Error(t, err)
	assert.Equal(t, int64(0), result)

	// getOrSet
	dep = cache2.NewKeyDepend(c, "getOrSet:dep")
	result, err = c.GetOrSet(context.Background(), "getOrSet", func() (interface{}, error) {
		return 1, nil
	}, WithDepend(cache2.NewKeyDepend(c, "getOrSet:dep"))).Int64()
	assert.NoError(t, err)
	assert.Equal(t, int64(1), result)
	_ = dep.InvalidKey(context.Background())
	result, err = c.GetOrSet(context.Background(), "getOrSet", func() (interface{}, error) {
		return 2, nil
	}, WithDepend(cache2.NewKeyDepend(c, "getOrSet:dep"))).Int64()
	assert.NoError(t, err)
	assert.Equal(t, int64(2), result)
	_ = dep.InvalidKey(context.Background())
	result, err = c.GetOrSet(context.Background(), "getOrSet", func() (interface{}, error) {
		return 3, nil
	}, WithDepend(cache2.NewKeyDepend(c, "getOrSet:dep"))).Int64()
	assert.NoError(t, err)
	assert.Equal(t, int64(3), result)
	result, err = c.GetOrSet(context.Background(), "getOrSet", func() (interface{}, error) {
		return 4, nil
	}, WithDepend(cache2.NewKeyDepend(c, "getOrSet:dep"))).Int64()
	assert.NoError(t, err)
	assert.Equal(t, int64(3), result)
	result, err = c.GetOrSet(context.Background(), "getOrSet", func() (interface{}, error) {
		return 5, nil
	}, WithDepend(cache2.NewKeyDepend(c, "getOrSet:dep"))).Int64()
	assert.NoError(t, err)
	assert.Equal(t, int64(3), result)
	_ = dep.InvalidKey(context.Background())
	result, err = c.GetOrSet(context.Background(), "getOrSet", func() (interface{}, error) {
		return 6, nil
	}, WithDepend(cache2.NewKeyDepend(c, "getOrSet:dep"))).Int64()
	assert.NoError(t, err)
	assert.Equal(t, int64(6), result)
}

func TestCache(t *testing.T) {
	c, _ := memory.NewMemoryCache()
	// 字符串
	c.Set(context.Background(), "test", 1)
	result, err := c.Get(context.Background(), "test").Int64()
	assert.NoError(t, err)
	assert.Equal(t, int64(1), result)

	// byte类型
	c.Set(context.Background(), "test", []byte("test"), WithDepend(cache2.NewKeyDepend(c, "test:dep")))
	resultByte, err := c.Get(context.Background(), "test", WithDepend(cache2.NewKeyDepend(c, "test:dep"))).Bytes()
	assert.NoError(t, err)
	assert.Equal(t, []byte("test"), resultByte)
	c.Set(context.Background(), "test", []byte("test"))
	resultByte, err = c.Get(context.Background(), "test").Bytes()
	assert.NoError(t, err)
	assert.Equal(t, []byte("test"), resultByte)
}
