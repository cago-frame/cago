// 一些用于测试的工具函数
package testutils

import (
	"context"
	"sync"
	"testing"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/alicebob/miniredis/v2"
	"github.com/cago-frame/cago/database/cache"
	"github.com/cago-frame/cago/database/cache/memory"
	"github.com/cago-frame/cago/database/db"
	redis2 "github.com/cago-frame/cago/database/redis"
	"github.com/cago-frame/cago/pkg/iam"
	"github.com/cago-frame/cago/pkg/iam/authn"
	"github.com/redis/go-redis/v9"
	"gorm.io/driver/mysql"
	"gorm.io/gorm"
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
func Cache() {
	onceDo("cache", func() {
		// 初始化组件
		m, _ := memory.NewMemoryCache()
		cache.SetDefault(m)
	})
}

// Redis 注册Redis组件
func Redis() {
	onceDo("redis", func() {
		m, err := miniredis.Run()
		if err != nil {
			panic(err)
		}
		client := redis.NewClient(&redis.Options{
			Addr: m.Addr(),
		})
		redis2.SetDefault(client)
	})
}

// Database 创建基于 sqlmock 的数据库测试环境
// 返回带有 mock 数据库的 context，业务代码通过 db.Ctx(ctx) 自动使用 mock 实例
func Database(t *testing.T) (context.Context, *gorm.DB, sqlmock.Sqlmock) {
	sqlDB, mock, err := sqlmock.New()
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = sqlDB.Close() })
	gormDB, err := gorm.Open(mysql.New(mysql.Config{
		SkipInitializeWithVersion: true,
		Conn:                      sqlDB,
	}), &gorm.Config{})
	if err != nil {
		t.Fatal(err)
	}
	ctx := db.WithContextDB(context.Background(), gormDB)
	return ctx, gormDB, mock
}

// IAM 注册IAM组件
func IAM(t *testing.T, database authn.Database, opts ...iam.Option) {
	onceDo("iam", func() {
		iam.SetDefault(iam.New(database, opts...))
	})
}
