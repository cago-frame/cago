package postgres

import (
	"github.com/cago-frame/cago/database/db"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
)

func init() {
	db.RegisterDriver(db.Postgres, func(config *db.Config) gorm.Dialector {
		return postgres.Open(config.Dsn)
	})
}
