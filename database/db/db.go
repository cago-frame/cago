package db

import (
	"context"
	"errors"
	"time"

	"github.com/cago-frame/cago/configs"
	"github.com/cago-frame/cago/pkg/opentelemetry/metric"
	"github.com/cago-frame/cago/pkg/opentelemetry/trace"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
	"gorm.io/gorm/schema"
	"gorm.io/plugin/opentelemetry/tracing"
)

type contextKey int

const (
	dbKey contextKey = iota
)

var defaultDB *DB

// Driver 数据库驱动
type Driver string

const (
	MySQL      Driver = "mysql"
	Clickhouse Driver = "clickhouse"
	SQLite     Driver = "sqlite"
	Postgres   Driver = "postgres"
)

type Config struct {
	Driver Driver `yaml:"driver"`
	Dsn    string `yaml:"dsn"`
	Prefix string `yaml:"prefix"`
	// 读写分离，以后再说吧
	//WriterDsn []string `yaml:"writerDsn,omitempty"` // 写入数据源
	//ReaderDsn []string `yaml:"readerDsn,omitempty"` // 读取数据源
}

type GroupConfig map[string]*Config

type DB struct {
	defaultDb *gorm.DB
	dbs       map[string]*gorm.DB
}

// Database gorm数据库封装，支持多数据库，如果你配置了 trace 的话会自动开启链路追踪
// 默认注册和使用 mysql 驱动，如果你需要其他驱动，可以使用 RegisterDriver 注册
// 或者导入对应的数据库驱动，例如：
// import _ "clickhouse"
func Database() *DB {
	return &DB{}
}

func (d *DB) newDB(cfg *Config, debug bool) (*gorm.DB, error) {
	if cfg.Driver == "" {
		cfg.Driver = MySQL
	}
	logCfg := logger.Config{
		SlowThreshold:             200 * time.Millisecond,
		LogLevel:                  logger.Warn,
		IgnoreRecordNotFoundError: true,
		Colorful:                  false,
	}
	if debug {
		logCfg.IgnoreRecordNotFoundError = false
		logCfg.Colorful = true
	}
	orm, err := gorm.Open(driver[cfg.Driver](cfg), &gorm.Config{
		DisableForeignKeyConstraintWhenMigrating: true,
		NamingStrategy: schema.NamingStrategy{
			TablePrefix:   cfg.Prefix,
			SingularTable: true,
		},
		Logger: NewLogger(cfg.Driver, logCfg),
	})
	if err != nil {
		return nil, err
	}
	tracingPlugin := make([]tracing.Option, 0)
	if tp := trace.Default(); tp != nil {
		tracingPlugin = append(tracingPlugin,
			tracing.WithTracerProvider(tp),
		)
		if metric.Default() == nil {
			tracingPlugin = append(tracingPlugin,
				tracing.WithoutMetrics(),
			)
		}
	}
	if len(tracingPlugin) != 0 {
		if err := orm.Use(tracing.NewPlugin(
			tracingPlugin...,
		)); err != nil {
			return nil, err
		}
	}
	return orm, nil
}

func (d *DB) Start(ctx context.Context, config *configs.Config) error {
	cfg := &Config{}
	// 数据库group配置
	cfgGroup := make(GroupConfig)
	if ok, err := config.Has(ctx, "dbs"); err != nil {
		return err
	} else if ok {
		if err := config.Scan(ctx, "dbs", &cfgGroup); err != nil {
			return err
		}
	} else {
		if err := config.Scan(ctx, "db", cfg); err != nil {
			return err
		}
		cfgGroup["default"] = cfg
	}
	if _, ok := cfgGroup["default"]; !ok {
		return errors.New("no default db config")
	}
	dbs := make(map[string]*gorm.DB)
	orm, err := d.newDB(cfgGroup["default"], config.Debug)
	if err != nil {
		return err
	}
	delete(cfgGroup, "default")
	if len(cfgGroup) > 0 {
		for name, v := range cfgGroup {
			db, err := d.newDB(v, config.Debug)
			if err != nil {
				return err
			}
			dbs[name] = db
		}
	}
	d.defaultDb = orm
	d.dbs = dbs
	defaultDB = d
	return nil
}

func (d *DB) CloseHandle() {
	db, _ := d.defaultDb.DB()
	_ = db.Close()
	for _, v := range d.dbs {
		db, _ := v.DB()
		_ = db.Close()
	}
}

// SetDefault 设置默认数据库
func (d *DB) SetDefault(db *gorm.DB) {
	d.defaultDb = db
}

// Default 默认数据库
func Default() *gorm.DB {
	return defaultDB.defaultDb
}

// Use 根据 key 参数指定数据库
func Use(key string) *gorm.DB {
	if key == "default" {
		return defaultDB.defaultDb
	}
	return defaultDB.dbs[key]
}

// WithContext 根据 key 参数指定数据库，并将数据库实例存入 context
// 如果你想修改后续的数据库实例，你可以使用 db.WithContext
// ctx:=db.WithContext(ctx, "db2")
// SomeMethod(ctx)
func WithContext(ctx context.Context, key string) context.Context {
	if key == "default" {
		return WithContextDB(ctx, defaultDB.defaultDb)
	}
	return WithContextDB(ctx, defaultDB.dbs[key])
}

// WithContextDB 将数据库实例存入 context
// 如果你有事务的需求，你可以使用 db.WithContextDB，然后使用带Ctx的方法来获取数据库实例，这样会自动传递数据库实例
//
//	db.Default().Transaction(func(tx *gorm.DB) error {
//	  return SomeMethod(db.WithContextDB(ctx, tx))
//	})
func WithContextDB(ctx context.Context, db *gorm.DB) context.Context {
	return context.WithValue(ctx, dbKey, db)
}

// Ctx 从 context 中获取数据库实例，如果不存在则返回默认数据库
func Ctx(ctx context.Context) *gorm.DB {
	return CtxWith(ctx, "default")
}

// CtxWith 从 context 中获取数据库实例，如果不存在则返回指定数据库
func CtxWith(ctx context.Context, key string) *gorm.DB {
	if db, ok := ctx.Value(dbKey).(*gorm.DB); ok {
		return db.WithContext(ctx)
	}
	if key == "default" {
		return defaultDB.defaultDb.WithContext(ctx)
	}
	return defaultDB.dbs[key].WithContext(ctx)
}
