package sync

import (
	"context"
	"errors"
	"fmt"
	"time"

	redis2 "github.com/cago-frame/cago/database/redis"
	"github.com/redis/go-redis/v9"
)

type redisLocker struct {
	prefix string
	redis  *redis.Client
}

func newRedis(prefix string) *redisLocker {
	return &redisLocker{
		prefix: prefix,
		redis:  redis2.Default(),
	}
}

func (r *redisLocker) genKey(key string) string {
	return fmt.Sprintf("%s:%s", r.prefix, key)
}

func (r *redisLocker) lockOptions(opts ...LockOption) *LockOptions {
	options := &LockOptions{
		timeout: time.Second * 5,
	}
	for _, o := range opts {
		o(options)
	}
	return options
}

// LockKey implements Locker
func (r *redisLocker) LockKey(ctx context.Context, key string, opts ...LockOption) error {
	options := r.lockOptions(opts...)
	var cancel context.CancelFunc
	ctx, cancel = context.WithTimeout(ctx, options.timeout)
	defer cancel()
	key = r.genKey(key)
	for {
		select {
		case <-ctx.Done():
			return ErrTryLockTimeout
		default:
			if err := r.tryLockKey(ctx, key, options); err != nil {
				if !errors.Is(err, ErrLockOccurred) {
					return err
				}
			} else {
				return nil
			}
			// 延迟100ms再请求
			time.Sleep(time.Millisecond * 100)
		}
	}
}

// TryLockKey 尝试获取锁
func (r *redisLocker) TryLockKey(ctx context.Context, key string, opts ...LockOption) error {
	options := r.lockOptions(opts...)
	return r.tryLockKey(ctx, r.genKey(key), options)
}

func (r *redisLocker) TryLock(ctx context.Context, opts ...LockOption) error {
	return r.TryLockKey(ctx, "", opts...)
}

func (r *redisLocker) tryLockKey(ctx context.Context, key string, options *LockOptions) error {
	if ok, err := r.redis.SetNX(ctx, key, 1, options.timeout).Result(); err != nil {
		return err
	} else if !ok {
		return ErrLockOccurred
	}
	return nil
}

// UnlockKey implements Locker
func (r *redisLocker) UnlockKey(ctx context.Context, key string) error {
	cnt, err := r.redis.Del(ctx, r.genKey(key)).Result()
	if err != nil {
		return err
	} else if cnt == 0 {
		return ErrLockNotExists
	}
	return nil
}

// Lock implements Locker
func (r *redisLocker) Lock(ctx context.Context, opts ...LockOption) error {
	return r.LockKey(ctx, "", opts...)
}

// Unlock implements Locker
func (r *redisLocker) Unlock(ctx context.Context) error {
	return r.UnlockKey(ctx, "")
}
