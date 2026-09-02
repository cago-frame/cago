package redis

import (
	"context"
	"testing"

	"github.com/alicebob/miniredis/v2"
	redislib "github.com/redis/go-redis/v9"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestCtxRedisModernArgs(t *testing.T) {
	server := miniredis.RunT(t)
	client := redislib.NewClient(&redislib.Options{Addr: server.Addr()})
	t.Cleanup(func() { require.NoError(t, client.Close()) })
	c := &CtxRedis{Client: client, ctx: context.Background()}

	require.NoError(t, client.Set(context.Background(), "value", "old", 0).Err())
	old, err := c.SetArgs("value", "new", redislib.SetArgs{Get: true}).Result()
	require.NoError(t, err)
	assert.Equal(t, "old", old)

	require.NoError(t, client.ZAdd(context.Background(), "scores",
		redislib.Z{Score: 1, Member: "one"},
		redislib.Z{Score: 2, Member: "two"},
	).Err())
	values, err := c.ZRangeArgs(redislib.ZRangeArgs{
		Key:     "scores",
		Start:   "2",
		Stop:    "1",
		ByScore: true,
		Rev:     true,
	}).Result()
	require.NoError(t, err)
	assert.Equal(t, []string{"two", "one"}, values)
}

func TestCtxRedisCommonCommands(t *testing.T) {
	server := miniredis.RunT(t)
	client := redislib.NewClient(&redislib.Options{Addr: server.Addr()})
	t.Cleanup(func() { require.NoError(t, client.Close()) })
	c := &CtxRedis{Client: client, ctx: context.Background()}

	require.NoError(t, c.MSet("first", "1", "second", "2").Err())
	values, err := c.MGet("first", "second", "missing").Result()
	require.NoError(t, err)
	assert.Equal(t, []interface{}{"1", "2", nil}, values)

	require.NoError(t, c.HSet("hash", "first", "1", "second", "2").Err())
	hashValues, err := c.HMGet("hash", "second", "missing").Result()
	require.NoError(t, err)
	assert.Equal(t, []interface{}{"2", nil}, hashValues)
}

// key 不存在时应原样透出 redis.Nil，而不是被包装成其他错误
func TestCtxRedisGetNil(t *testing.T) {
	server := miniredis.RunT(t)
	client := redislib.NewClient(&redislib.Options{Addr: server.Addr()})
	t.Cleanup(func() { require.NoError(t, client.Close()) })
	c := &CtxRedis{Client: client, ctx: context.Background()}

	_, err := c.Get("missing").Result()
	assert.ErrorIs(t, err, redislib.Nil)
}

// 底层报错时应原样透出，方便上层判断
func TestCtxRedisPropagatesError(t *testing.T) {
	server := miniredis.RunT(t)
	client := redislib.NewClient(&redislib.Options{Addr: server.Addr()})
	t.Cleanup(func() { require.NoError(t, client.Close()) })
	c := &CtxRedis{Client: client, ctx: context.Background()}

	server.SetError("LOADING Redis is loading the dataset in memory")
	err := c.Del("key").Err()
	require.Error(t, err)
	assert.Contains(t, err.Error(), "LOADING")
}
