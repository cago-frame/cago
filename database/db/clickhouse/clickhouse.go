package clickhouse

import (
	"github.com/cago-frame/cago/database/db"
	"gorm.io/driver/clickhouse"
	"gorm.io/gorm"
)

// 默认不引用, 需要自行注册

func init() {
	db.RegisterDriver(db.Clickhouse, func(config *db.Config) gorm.Dialector {
		return clickhouse.Open(config.Dsn)
	})
}
