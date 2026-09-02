package migrations

import "gorm.io/gorm"

type fakeDB struct{}

func (fakeDB) AutoMigrate(...any) error {
	return nil
}

func migrate(db *gorm.DB) error {
	return db.AutoMigrate(&struct{}{}) // want "CAGO7001"
}

func deterministicMigration(db *gorm.DB) *gorm.DB {
	return db.Exec("CREATE TABLE users (id BIGINT PRIMARY KEY)")
}

func sameNameIsAllowed(db fakeDB) error {
	return db.AutoMigrate(&struct{}{})
}
