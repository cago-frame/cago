package mongomigrate

import (
	"context"

	"github.com/cago-frame/cago/database/mongo"
)

// MigrateFunc 数据库迁移函数
type MigrateFunc func(ctx context.Context, db *mongo.Client) error

// RollbackFunc 数据库回滚函数
type RollbackFunc func(ctx context.Context, db *mongo.Client) error

type Migration struct {
	ID       string
	Migrate  MigrateFunc
	Rollback RollbackFunc
}
