package migrations

import (
	"context"

	"github.com/cago-frame/cago/database/db"
	"github.com/cago-frame/cago/examples/simple/internal/api/user"
	"github.com/cago-frame/cago/examples/simple/internal/service/user_svc"
	"github.com/go-gormigrate/gormigrate/v2"
	"gorm.io/gorm"
)

func T20230611() *gormigrate.Migration {
	return &gormigrate.Migration{
		ID: "20230611",
		Migrate: func(tx *gorm.DB) error {
			if err := tx.Exec(`CREATE TABLE IF NOT EXISTS sm_users (
				id BIGINT(20) NOT NULL AUTO_INCREMENT,
				username VARCHAR(255) NOT NULL,
				hashed_password VARCHAR(255) NOT NULL,
				status INT(11) NOT NULL,
				createtime BIGINT(20) DEFAULT NULL,
				updatetime BIGINT(20) DEFAULT NULL,
				PRIMARY KEY (id),
				UNIQUE KEY username (username)
			) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci`).Error; err != nil {
				return err
			}

			// 初始化用户
			ctx := context.Background()
			ctx = db.WithContextDB(ctx, tx)
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
			return tx.Exec("DROP TABLE IF EXISTS sm_users").Error
		},
	}
}
