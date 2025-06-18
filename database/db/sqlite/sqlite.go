package sqlite

import (
	"github.com/cago-frame/cago/database/db"
	"github.com/glebarez/sqlite"
	"gorm.io/gorm"
)

// 默认不引用, 需要自行注册

func init() {
	db.RegisterDriver(db.SQLite, func(config *db.Config) gorm.Dialector {
		return sqlite.Open(config.Dsn)
	})
}
