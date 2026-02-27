package migrations

import (
	"github.com/go-gormigrate/gormigrate/v2"
	"gorm.io/gorm"
)

// RunMigrations 数据库迁移操作
func RunMigrations(db *gorm.DB) error {
	return run(db,
		T20230611,
	)
}

func run(db *gorm.DB, fs ...func() *gormigrate.Migration) error {
	ms := make([]*gormigrate.Migration, 0, len(fs))
	for _, f := range fs {
		ms = append(ms, f())
	}
	m := gormigrate.New(db, &gormigrate.Options{
		TableName:                 "migrations",
		IDColumnName:              "id",
		IDColumnSize:              200,
		UseTransaction:            true,
		ValidateUnknownMigrations: true,
	}, ms)
	if err := m.Migrate(); err != nil {
		return err
	}
	return nil
}
