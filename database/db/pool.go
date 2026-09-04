package db

import (
	"fmt"

	"gorm.io/gorm"
)

// applyPool 把 Config 里的连接池参数写进底层的 database/sql。
//
// gorm 只负责建连接,连接数与连接寿命它一概不管,所以必须自己 orm.DB() 取出
// *sql.DB 再设一遍。每项都是零值即跳过:不配的应用行为与加这组字段之前完全一致,
// 保持 database/sql 自己的默认值。各项的含义与推荐做法见 Config 上的注释。
func applyPool(orm *gorm.DB, cfg *Config) error {
	if cfg.MaxOpenConns == 0 && cfg.MaxIdleConns == 0 &&
		cfg.ConnMaxLifetime == 0 && cfg.ConnMaxIdleTime == 0 {
		return nil
	}
	sqlDB, err := orm.DB()
	if err != nil {
		return fmt.Errorf("resolve sql db for pool settings: %w", err)
	}
	// MaxOpenConns 要先设:database/sql 在 SetMaxIdleConns 里会把空闲上限压到
	// 当前的连接上限,顺序反了空闲上限就压不住。
	if cfg.MaxOpenConns != 0 {
		sqlDB.SetMaxOpenConns(cfg.MaxOpenConns)
	}
	if cfg.MaxIdleConns != 0 {
		sqlDB.SetMaxIdleConns(cfg.MaxIdleConns)
	}
	if cfg.ConnMaxLifetime != 0 {
		sqlDB.SetConnMaxLifetime(cfg.ConnMaxLifetime)
	}
	if cfg.ConnMaxIdleTime != 0 {
		sqlDB.SetConnMaxIdleTime(cfg.ConnMaxIdleTime)
	}
	return nil
}
