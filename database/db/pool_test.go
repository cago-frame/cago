package db

import (
	"context"
	"database/sql"
	"reflect"
	"testing"
	"time"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/cago-frame/cago/configs"
	"github.com/cago-frame/cago/configs/memory"
	"github.com/stretchr/testify/assert"
	mysqlDriver "gorm.io/driver/mysql"
	"gorm.io/gorm"
)

// poolField 读取 *sql.DB 上未导出的连接池字段。database/sql 只把 MaxOpenConns
// 暴露在 Stats() 里，maxIdleCount/maxLifetime/maxIdleTime 没有任何公开读法，
// 只能靠反射断言。字段名一旦在 Go 版本间改名，这里会直接失败而不是静默放过。
func poolField(t *testing.T, sqlDB *sql.DB, name string) int64 {
	t.Helper()
	f := reflect.ValueOf(sqlDB).Elem().FieldByName(name)
	if !f.IsValid() {
		t.Fatalf("database/sql.DB 没有字段 %s，Go 版本可能改了实现", name)
	}
	return f.Int()
}

// newPoolConfig 造一个只跑连接池断言的配置，dsn 用哪个 mock 连接不重要。
func newPoolConfig(t *testing.T, cfg *Config) *configs.Config {
	t.Helper()
	sqlDB, _, err := sqlmock.New()
	assert.Nil(t, err)
	t.Cleanup(func() { _ = sqlDB.Close() })

	RegisterDriver(cfg.Driver, func(*Config) gorm.Dialector {
		return mysqlDriver.New(mysqlDriver.Config{SkipInitializeWithVersion: true, Conn: sqlDB})
	})
	c, err := configs.NewConfig("test", configs.WithSource(
		memory.NewSource(map[string]interface{}{
			"env": "dev",
			"db":  cfg,
		}),
	))
	assert.Nil(t, err)
	return c
}

// 配了连接池参数就要落到底层的 database/sql 上，而不是停在 gorm 那一层。
func TestDatabasePool(t *testing.T) {
	c := newPoolConfig(t, &Config{
		Driver:          "mock-pool",
		Dsn:             "mock",
		MaxOpenConns:    40,
		MaxIdleConns:    20,
		ConnMaxLifetime: 30 * time.Minute,
		ConnMaxIdleTime: 5 * time.Minute,
	})

	d := Database()
	assert.Nil(t, d.Start(context.Background(), c))

	sqlDB, err := Default().DB()
	assert.Nil(t, err)
	assert.Equal(t, 40, sqlDB.Stats().MaxOpenConnections)
	assert.Equal(t, int64(20), poolField(t, sqlDB, "maxIdleCount"))
	assert.Equal(t, int64(30*time.Minute), poolField(t, sqlDB, "maxLifetime"))
	assert.Equal(t, int64(5*time.Minute), poolField(t, sqlDB, "maxIdleTime"))
}

// 不配就一个 setter 都不能调：老应用的行为必须和加这组字段之前一模一样，
// 也就是保持 database/sql 自己的默认值（无上限 / 空闲 2 / 永不过期）。
func TestDatabasePoolUnsetKeepsStdDefaults(t *testing.T) {
	c := newPoolConfig(t, &Config{
		Driver: "mock-pool-unset",
		Dsn:    "mock",
	})

	d := Database()
	assert.Nil(t, d.Start(context.Background(), c))

	sqlDB, err := Default().DB()
	assert.Nil(t, err)
	assert.Equal(t, 0, sqlDB.Stats().MaxOpenConnections)
	// maxIdleCount 的 0 表示 defaultMaxIdleConns(2)，负数才表示 0。
	assert.Equal(t, int64(0), poolField(t, sqlDB, "maxIdleCount"))
	assert.Equal(t, int64(0), poolField(t, sqlDB, "maxLifetime"))
	assert.Equal(t, int64(0), poolField(t, sqlDB, "maxIdleTime"))
}

// 多库模式下每个库各配各的，不能让 default 的池子串到别的库上。
func TestDatabasePoolPerDatabase(t *testing.T) {
	db1, _, err := sqlmock.New()
	assert.Nil(t, err)
	defer db1.Close() //nolint:errcheck
	db2, _, err := sqlmock.New()
	assert.Nil(t, err)
	defer db2.Close() //nolint:errcheck

	RegisterDriver("mock-pool-group", func(config *Config) gorm.Dialector {
		conn := db2
		if config.Dsn == "mock1" {
			conn = db1
		}
		return mysqlDriver.New(mysqlDriver.Config{SkipInitializeWithVersion: true, Conn: conn})
	})
	c, err := configs.NewConfig("test", configs.WithSource(
		memory.NewSource(map[string]interface{}{
			"env": "dev",
			"dbs": GroupConfig{
				"default": &Config{
					Driver:       "mock-pool-group",
					Dsn:          "mock1",
					MaxOpenConns: 40,
					MaxIdleConns: 20,
				},
				"analytics": &Config{
					Driver:       "mock-pool-group",
					Dsn:          "mock2",
					MaxOpenConns: 4,
					MaxIdleConns: 1,
				},
			},
		}),
	))
	assert.Nil(t, err)

	d := Database()
	assert.Nil(t, d.Start(context.Background(), c))

	defaultSQL, err := Default().DB()
	assert.Nil(t, err)
	assert.Equal(t, 40, defaultSQL.Stats().MaxOpenConnections)
	assert.Equal(t, int64(20), poolField(t, defaultSQL, "maxIdleCount"))

	analyticsSQL, err := Use("analytics").DB()
	assert.Nil(t, err)
	assert.Equal(t, 4, analyticsSQL.Stats().MaxOpenConnections)
	assert.Equal(t, int64(1), poolField(t, analyticsSQL, "maxIdleCount"))
}
