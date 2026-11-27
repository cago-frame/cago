package limit

import (
	"context"
	"errors"
	"net/http"
	"strconv"
	"sync/atomic"
	"testing"
	"time"

	"github.com/alicebob/miniredis/v2"
	"github.com/cago-frame/cago/pkg/utils/httputils"
	"github.com/redis/go-redis/v9"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// countHook 统计发到 Redis 的往返次数（单条命令或一次 pipeline 各算一次）
type countHook struct{ n atomic.Int64 }

func (h *countHook) DialHook(next redis.DialHook) redis.DialHook { return next }
func (h *countHook) ProcessHook(next redis.ProcessHook) redis.ProcessHook {
	return func(ctx context.Context, cmd redis.Cmder) error {
		switch cmd.Name() {
		case "hello", "client", "auth", "select", "ping":
		default:
			h.n.Add(1)
		}
		return next(ctx, cmd)
	}
}
func (h *countHook) ProcessPipelineHook(next redis.ProcessPipelineHook) redis.ProcessPipelineHook {
	return func(ctx context.Context, cmds []redis.Cmder) error {
		h.n.Add(1)
		return next(ctx, cmds)
	}
}

func setupPeriodLimit(t *testing.T, period, quota int64) (*PeriodLimit, *miniredis.Miniredis, *countHook) {
	t.Helper()
	mr := miniredis.RunT(t)
	client := redis.NewClient(&redis.Options{Addr: mr.Addr(), DisableIdentity: true})
	t.Cleanup(func() { _ = client.Close() })
	hook := &countHook{}
	client.AddHook(hook)
	return NewPeriodLimit(period, quota, client, "limit:test"), mr, hook
}

func isTooMany(err error) bool {
	var he *httputils.Error
	return errors.As(err, &he) && he.Status == http.StatusTooManyRequests
}

func TestPeriodLimit_Take(t *testing.T) {
	ctx := context.Background()

	t.Run("配额内放行，超额返回 429", func(t *testing.T) {
		l, _, _ := setupPeriodLimit(t, 60, 2)
		_, err := l.Take(ctx, "ip")
		require.NoError(t, err)
		_, err = l.Take(ctx, "ip")
		require.NoError(t, err)
		_, err = l.Take(ctx, "ip")
		assert.True(t, isTooMany(err), "got %v", err)
	})

	t.Run("记录按当前时间打分，key 过期时间 period+60", func(t *testing.T) {
		l, mr, _ := setupPeriodLimit(t, 60, 5)
		before := time.Now().Unix()
		_, err := l.Take(ctx, "ip")
		require.NoError(t, err)
		members, err := mr.ZMembers("limit:test:ip")
		require.NoError(t, err)
		require.Len(t, members, 1)
		score, err := mr.ZScore("limit:test:ip", members[0])
		require.NoError(t, err)
		assert.GreaterOrEqual(t, int64(score), before)
		assert.LessOrEqual(t, int64(score), time.Now().Unix())
		assert.Equal(t, 120*time.Second, mr.TTL("limit:test:ip"))
	})

	t.Run("取消函数删掉本次记录，配额回来", func(t *testing.T) {
		l, mr, _ := setupPeriodLimit(t, 60, 1)
		cancel, err := l.Take(ctx, "ip")
		require.NoError(t, err)
		require.NoError(t, cancel())
		members, _ := mr.ZMembers("limit:test:ip")
		assert.Empty(t, members)
		_, err = l.Take(ctx, "ip")
		assert.NoError(t, err)
	})

	t.Run("窗口外的旧记录不计数", func(t *testing.T) {
		l, mr, _ := setupPeriodLimit(t, 60, 1)
		_, err := mr.ZAdd("limit:test:ip", float64(time.Now().Unix()-61), "old")
		require.NoError(t, err)
		_, err = l.Take(ctx, "ip")
		assert.NoError(t, err)
	})

	t.Run("总数到 500 的倍数（>500）时清理两个周期前的记录", func(t *testing.T) {
		l, mr, _ := setupPeriodLimit(t, 60, 10)
		now := time.Now().Unix()
		stale := float64(now - 60*2 + 60 - 10) // 早于清理阈值
		for i := 0; i < 1000; i++ {
			_, err := mr.ZAdd("limit:test:ip", stale, "s"+strconv.Itoa(i))
			require.NoError(t, err)
		}
		_, err := l.Take(ctx, "ip")
		require.NoError(t, err)
		members, _ := mr.ZMembers("limit:test:ip")
		assert.Len(t, members, 1, "清理应在本次调用内同步完成，只剩新记录")
	})

	t.Run("一次往返：脚本已缓存时只发 1 条命令", func(t *testing.T) {
		l, _, hook := setupPeriodLimit(t, 60, 10)
		_, err := l.Take(ctx, "ip") // 首次可能 EVALSHA 未命中再 EVAL
		require.NoError(t, err)
		hook.n.Store(0)
		_, err = l.Take(ctx, "ip")
		require.NoError(t, err)
		assert.Equal(t, int64(1), hook.n.Load())
	})
}
