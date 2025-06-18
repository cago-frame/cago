// 一些用于测试的工具函数
package testutils

import (
	"sync"
	"testing"

	"github.com/alicebob/miniredis/v2"
	"github.com/cago-frame/cago/database/cache"
	"github.com/cago-frame/cago/database/cache/memory"
	redis2 "github.com/cago-frame/cago/database/redis"
	"github.com/cago-frame/cago/pkg/iam"
	"github.com/cago-frame/cago/pkg/iam/authn"
	"github.com/redis/go-redis/v9"
)

var onceMap = make(map[string]*sync.Once)

func onceDo(key string, f func()) {
	once, ok := onceMap[key]
	if !ok {
		once = &sync.Once{}
		onceMap[key] = once
	}
	once.Do(f)
}

// Cache 注册缓存组件
func Cache(t *testing.T) {
	onceDo("cache", func() {
		// 初始化组件
		m, _ := memory.NewMemoryCache()
		cache.SetDefault(m)
	})
}

// Redis 注册Redis组件
func Redis(t *testing.T) {
	onceDo("redis", func() {
		m := miniredis.RunT(t)
		db := redis.NewClient(&redis.Options{
			Addr: m.Addr(),
		})
		redis2.SetDefault(db)
	})
}

// Database 注册数据库组件
//func Database(t *testing.T) {
//	onceDo("database", func() {
//		db.Default()
//	})
//}

// IAM 注册IAM组件
func IAM(t *testing.T, database authn.Database, opts ...iam.Option) {
	onceDo("iam", func() {
		iam.SetDefault(iam.New(database, opts...))
	})
}
