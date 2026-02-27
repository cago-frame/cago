package redis

import (
	"context"
	"time"

	"github.com/redis/go-redis/v9"
)

// CtxRedis 简化操作,慢慢封装
type CtxRedis struct {
	*redis.Client
	ctx context.Context
}

func (c *CtxRedis) Get(key string) *redis.StringCmd {
	return c.Client.Get(c.ctx, key)
}

func (c *CtxRedis) Del(key string) *redis.IntCmd {
	return c.Client.Del(c.ctx, key)
}

func (c *CtxRedis) Set(key string, value interface{}, expiration time.Duration) *redis.StatusCmd {
	return c.Client.Set(c.ctx, key, value, expiration)
}

func (c *CtxRedis) SetNX(key string, value interface{}, expiration time.Duration) *redis.BoolCmd {
	return c.Client.SetNX(c.ctx, key, value, expiration)
}

func (c *CtxRedis) SetXX(key string, value interface{}, expiration time.Duration) *redis.BoolCmd {
	return c.Client.SetXX(c.ctx, key, value, expiration)
}

func (c *CtxRedis) SetEx(key string, value interface{}, expiration time.Duration) *redis.StatusCmd {
	return c.Client.SetEx(c.ctx, key, value, expiration)
}

func (c *CtxRedis) SetRange(key string, offset int64, value string) *redis.IntCmd {
	return c.Client.SetRange(c.ctx, key, offset, value)
}

func (c *CtxRedis) GetRange(key string, start, end int64) *redis.StringCmd {
	return c.Client.GetRange(c.ctx, key, start, end)
}

func (c *CtxRedis) GetSet(key string, value interface{}) *redis.StringCmd {
	return c.Client.GetSet(c.ctx, key, value)
}

func (c *CtxRedis) Incr(key string) *redis.IntCmd {
	return c.Client.Incr(c.ctx, key)
}

func (c *CtxRedis) IncrBy(key string, value int64) *redis.IntCmd {
	return c.Client.IncrBy(c.ctx, key, value)
}

func (c *CtxRedis) IncrByFloat(key string, value float64) *redis.FloatCmd {
	return c.Client.IncrByFloat(c.ctx, key, value)
}

func (c *CtxRedis) Decr(key string) *redis.IntCmd {
	return c.Client.Decr(c.ctx, key)
}

func (c *CtxRedis) DecrBy(key string, decrement int64) *redis.IntCmd {
	return c.Client.DecrBy(c.ctx, key, decrement)
}

func (c *CtxRedis) Exists(keys ...string) *redis.IntCmd {
	return c.Client.Exists(c.ctx, keys...)
}

func (c *CtxRedis) Expire(key string, expiration time.Duration) *redis.BoolCmd {
	return c.Client.Expire(c.ctx, key, expiration)
}

func (c *CtxRedis) ExpireAt(key string, tm time.Time) *redis.BoolCmd {
	return c.Client.ExpireAt(c.ctx, key, tm)
}

func (c *CtxRedis) TTL(key string) *redis.DurationCmd {
	return c.Client.TTL(c.ctx, key)
}

func (c *CtxRedis) LPush(key string, values ...interface{}) *redis.IntCmd {
	return c.Client.LPush(c.ctx, key, values...)
}

func (c *CtxRedis) RPush(key string, values ...interface{}) *redis.IntCmd {
	return c.Client.RPush(c.ctx, key, values...)
}

func (c *CtxRedis) LPop(key string) *redis.StringCmd {
	return c.Client.LPop(c.ctx, key)
}

func (c *CtxRedis) RPop(key string) *redis.StringCmd {
	return c.Client.RPop(c.ctx, key)
}

func (c *CtxRedis) LLen(key string) *redis.IntCmd {
	return c.Client.LLen(c.ctx, key)
}

func (c *CtxRedis) LRange(key string, start, stop int64) *redis.StringSliceCmd {
	return c.Client.LRange(c.ctx, key, start, stop)
}

func (c *CtxRedis) LTrim(key string, start, stop int64) *redis.StatusCmd {
	return c.Client.LTrim(c.ctx, key, start, stop)
}

func (c *CtxRedis) LRem(key string, count int64, value interface{}) *redis.IntCmd {
	return c.Client.LRem(c.ctx, key, count, value)
}

func (c *CtxRedis) HGet(key, field string) *redis.StringCmd {
	return c.Client.HGet(c.ctx, key, field)
}

func (c *CtxRedis) HSet(key string, value ...interface{}) *redis.IntCmd {
	return c.Client.HSet(c.ctx, key, value...)
}

func (c *CtxRedis) HDel(key string, fields ...string) *redis.IntCmd {
	return c.Client.HDel(c.ctx, key, fields...)
}

func (c *CtxRedis) HGetAll(key string) *redis.MapStringStringCmd {
	return c.Client.HGetAll(c.ctx, key)
}

func (c *CtxRedis) HIncrBy(key, field string, incr int64) *redis.IntCmd {
	return c.Client.HIncrBy(c.ctx, key, field, incr)
}

func (c *CtxRedis) HIncrByFloat(key, field string, incr float64) *redis.FloatCmd {
	return c.Client.HIncrByFloat(c.ctx, key, field, incr)
}

func (c *CtxRedis) HExists(key, field string) *redis.BoolCmd {
	return c.Client.HExists(c.ctx, key, field)
}

func (c *CtxRedis) HKeys(key string) *redis.StringSliceCmd {
	return c.Client.HKeys(c.ctx, key)
}

func (c *CtxRedis) HLen(key string) *redis.IntCmd {
	return c.Client.HLen(c.ctx, key)
}

func (c *CtxRedis) HSetNX(key, field string, value interface{}) *redis.BoolCmd {
	return c.Client.HSetNX(c.ctx, key, field, value)
}

func (c *CtxRedis) HVals(key string) *redis.StringSliceCmd {
	return c.Client.HVals(c.ctx, key)
}

func (c *CtxRedis) HScan(key string, cursor uint64, match string, count int64) *redis.ScanCmd {
	return c.Client.HScan(c.ctx, key, cursor, match, count)
}

func (c *CtxRedis) PFCount(keys ...string) *redis.IntCmd {
	return c.Client.PFCount(c.ctx, keys...)
}

func (c *CtxRedis) PFAdd(key string, els ...interface{}) *redis.IntCmd {
	return c.Client.PFAdd(c.ctx, key, els...)
}

func (c *CtxRedis) PFMerge(dest string, keys ...string) *redis.StatusCmd {
	return c.Client.PFMerge(c.ctx, dest, keys...)
}

func (c *CtxRedis) ScanType(cursor uint64, match string, count int64, keyType string) *redis.ScanCmd {
	return c.Client.ScanType(c.ctx, cursor, match, count, keyType)
}

func (c *CtxRedis) ZAdd(key string, members ...redis.Z) *redis.IntCmd {
	return c.Client.ZAdd(c.ctx, key, members...)
}

func (c *CtxRedis) ZAddNX(key string, members ...redis.Z) *redis.IntCmd {
	return c.Client.ZAddNX(c.ctx, key, members...)
}

func (c *CtxRedis) ZAddXX(key string, members ...redis.Z) *redis.IntCmd {
	return c.Client.ZAddXX(c.ctx, key, members...)
}

func (c *CtxRedis) ZAddArgs(key string, args redis.ZAddArgs) *redis.IntCmd {
	return c.Client.ZAddArgs(c.ctx, key, args)
}

func (c *CtxRedis) ZAddArgsIncr(key string, args redis.ZAddArgs) *redis.FloatCmd {
	return c.Client.ZAddArgsIncr(c.ctx, key, args)
}

func (c *CtxRedis) ZRemRangeByScore(key, min, max string) *redis.IntCmd { //nolint:predeclared
	return c.Client.ZRemRangeByScore(c.ctx, key, min, max)
}

func (c *CtxRedis) ZRemRangeByLex(key, min, max string) *redis.IntCmd { //nolint:predeclared
	return c.Client.ZRemRangeByLex(c.ctx, key, min, max)
}

func (c *CtxRedis) ZRemRangeByRank(key string, start, stop int64) *redis.IntCmd {
	return c.Client.ZRemRangeByRank(c.ctx, key, start, stop)
}

func (c *CtxRedis) ZRevRangeByScore(key string, opt *redis.ZRangeBy) *redis.StringSliceCmd {
	return c.Client.ZRevRangeByScore(c.ctx, key, opt)
}

func (c *CtxRedis) ZRevRangeByScoreWithScores(key string, opt *redis.ZRangeBy) *redis.ZSliceCmd {
	return c.Client.ZRevRangeByScoreWithScores(c.ctx, key, opt)
}

func (c *CtxRedis) ZRevRangeByLex(key string, opt *redis.ZRangeBy) *redis.StringSliceCmd {
	return c.Client.ZRevRangeByLex(c.ctx, key, opt)
}

func (c *CtxRedis) ZRangeWithScores(key string, start, stop int64) *redis.ZSliceCmd {
	return c.Client.ZRangeWithScores(c.ctx, key, start, stop)
}

func (c *CtxRedis) ZRevRangeWithScores(key string, start, stop int64) *redis.ZSliceCmd {
	return c.Client.ZRevRangeWithScores(c.ctx, key, start, stop)
}

func (c *CtxRedis) ZRangeByScoreWithScores(key string, opt *redis.ZRangeBy) *redis.ZSliceCmd {
	return c.Client.ZRangeByScoreWithScores(c.ctx, key, opt)
}

func (c *CtxRedis) ZRangeByScore(key string, opt *redis.ZRangeBy) *redis.StringSliceCmd {
	return c.Client.ZRangeByScore(c.ctx, key, opt)
}

func (c *CtxRedis) ZRangeByLex(key string, opt *redis.ZRangeBy) *redis.StringSliceCmd {
	return c.Client.ZRangeByLex(c.ctx, key, opt)
}
