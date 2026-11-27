package limit

import (
	"context"
	"fmt"
	"net/http"
	"strconv"
	"time"

	"github.com/cago-frame/cago/pkg/utils"
	"github.com/cago-frame/cago/pkg/utils/httputils"
	"github.com/redis/go-redis/v9"
)

// takeScript 在 Redis 上一次执行完「计数 → 按需清理 → 未超额则记一笔并续期」：
// 一次往返、原子执行，并发请求不会在计数与写入之间插队而超出配额。
// KEYS[1]=key；ARGV: now, period, quota, member, expire(秒)。返回 1=放行，0=超额。
// 清理条件与阈值沿用原实现：总数 >500 且为 500 的倍数时，删掉 score <= now-period*2+60 的记录。
var takeScript = redis.NewScript(`
local key = KEYS[1]
local now = tonumber(ARGV[1])
local period = tonumber(ARGV[2])
local cnt = redis.call('ZCOUNT', key, now - period, '+inf')
local total = redis.call('ZCARD', key)
if total > 500 and total % 500 == 0 then
  redis.call('ZREMRANGEBYSCORE', key, '-inf', now - period * 2 + 60)
end
if cnt < tonumber(ARGV[3]) then
  redis.call('ZADD', key, now, ARGV[4])
  redis.call('EXPIRE', key, ARGV[5])
  return 1
end
return 0
`)

// PeriodLimit 周期限流器,redis zet实现滑动窗口
type PeriodLimit struct {
	period, quota int64
	limitStore    *redis.Client
	keyPrefix     string
}

// NewPeriodLimit 创建周期限流器
// period单位秒，quota限流数量，limitStore redis客户端，keyPrefix键前缀
func NewPeriodLimit(period, quota int64, limitStore *redis.Client, keyPrefix string) *PeriodLimit {
	return &PeriodLimit{
		period:     period,
		quota:      quota,
		limitStore: limitStore,
		keyPrefix:  keyPrefix,
	}
}

func (p *PeriodLimit) key(key string) string {
	return p.keyPrefix + ":" + key
}

// Take 进行限流
// 输入对应的key，返回一个取消函数和错误
func (p *PeriodLimit) Take(ctx context.Context, key string) (func() error, error) {
	key = p.key(key)
	now := time.Now().Unix()
	flag := utils.RandString(8, utils.Mix)
	ok, err := takeScript.Run(ctx, p.limitStore, []string{key},
		now, p.period, p.quota, flag, p.period+60).Int()
	if err != nil {
		return nil, err
	}
	if ok == 1 {
		// 删除本次记录
		return func() error {
			return p.limitStore.ZRem(ctx, key, flag).Err()
		}, nil
	}
	log := fmt.Sprintf("%d秒内产生了太多请求", p.period)
	return nil, httputils.NewError(http.StatusTooManyRequests, -1, log)
}

// FuncTake 进行限流
// 输入对应的key和一个函数，当函数返回error时，会自动执行取消，返回函数的返回值和错误
func (p *PeriodLimit) FuncTake(ctx context.Context, key string, f func() (interface{}, error)) (interface{}, error) {
	cancel, err := p.Take(ctx, key)
	if err != nil {
		return nil, err
	}
	resp, err := f()
	if err != nil {
		if err := cancel(); err != nil {
			return nil, err
		}
		return nil, err
	}
	return resp, nil
}

// Count 获取用量
// 输入对应的key和period(秒)，返回在period时间内的用量
func (p *PeriodLimit) Count(ctx context.Context, key string, period int64) (int64, error) {
	key = p.key(key)
	now := time.Now().Unix()
	cnt, err := p.limitStore.ZCount(ctx, key, strconv.FormatInt(now-period, 10), "+inf").Result()
	if err != nil {
		return 0, err
	}
	return cnt, nil
}
