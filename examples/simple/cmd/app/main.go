package main

import (
	"context"
	"log"

	"github.com/cago-frame/cago/examples/simple/internal/repository/user_repo"
	"github.com/cago-frame/cago/examples/simple/internal/task"
	"github.com/cago-frame/cago/examples/simple/migrations"
	"github.com/cago-frame/cago/pkg/iam"
	"github.com/cago-frame/cago/pkg/iam/audit"
	"github.com/cago-frame/cago/pkg/iam/audit/audit_db"
	"github.com/cago-frame/cago/server/cron"

	"github.com/cago-frame/cago/database/db"
	"github.com/cago-frame/cago/pkg/component"

	"github.com/cago-frame/cago"
	"github.com/cago-frame/cago/configs"
	"github.com/cago-frame/cago/examples/simple/internal/api"
	"github.com/cago-frame/cago/server/mux"
)

func main() {
	ctx := context.Background()
	cfg, err := configs.NewConfig("simple")
	if err != nil {
		log.Fatalf("load config err: %v", err)
	}

	// 注册储存实例
	user_repo.RegisterUser(user_repo.NewUser())

	err = cago.New(ctx, cfg).
		Registry(component.Core()).
		Registry(component.Database()).
		Registry(component.Broker()).
		Registry(component.Redis()).
		Registry(component.Cache()).
		Registry(cron.Cron()).
		Registry(cago.FuncComponent(func(ctx context.Context, cfg *configs.Config) error {
			return migrations.RunMigrations(db.Default())
		})).
		Registry(cago.FuncComponent(func(ctx context.Context, cfg *configs.Config) error {
			storage, err := audit_db.NewDatabaseStorage(db.Default())
			if err != nil {
				return err
			}
			return iam.IAM(user_repo.User(),
				iam.WithAuthnOptions(),
				iam.WithAuditOptions(audit.WithStorage(storage)))(ctx, cfg)
		})).
		Registry(cago.FuncComponent(task.Task)).
		RegistryCancel(mux.HTTP(api.Router)).
		Start()
	if err != nil {
		log.Fatalf("start err: %v", err)
		return
	}
}
