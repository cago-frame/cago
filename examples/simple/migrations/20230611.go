package migrations

import (
	"context"

	"github.com/cago-frame/cago/database/db"
	"github.com/cago-frame/cago/examples/simple/internal/api/user"
	"github.com/cago-frame/cago/examples/simple/internal/model/entity/user_entity"
	"github.com/cago-frame/cago/examples/simple/internal/service/user_svc"
	"github.com/go-gormigrate/gormigrate/v2"
	"gorm.io/gorm"
)

func T20230611() *gormigrate.Migration {
	return &gormigrate.Migration{
		ID: "20230611",
		Migrate: func(tx *gorm.DB) error {
			// 初始化用户
			ctx := context.Background()
			ctx = db.WithContextDB(ctx, tx)
			if err := tx.Migrator().AutoMigrate(&user_entity.User{}); err != nil {
				return err
			}
			// 添加admin用户
			_, err := user_svc.User().Register(ctx, &user.RegisterRequest{
				Username: "admin",
				Password: "123456",
			})
			if err != nil {
				return err
			}
			return nil
		},
		Rollback: func(tx *gorm.DB) error {
			return nil
		},
	}
}
