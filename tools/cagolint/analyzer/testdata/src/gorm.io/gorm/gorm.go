package gorm

type DB struct{}

func (db *DB) AutoMigrate(dst ...any) error {
	return nil
}

func (db *DB) Exec(sql string, values ...any) *DB {
	return db
}
