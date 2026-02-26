# Cago Complete Examples

## Table of Contents

- [Application Entry Point](#application-entry-point)
- [API Definition](#api-definition)
- [Router](#router)
- [Controller](#controller)
- [Service](#service)
- [Repository](#repository)
- [Entity](#entity)
- [Error Codes (i18n)](#error-codes)
- [Cron Jobs](#cron-jobs)
- [Message Queue](#message-queue)
- [Database Migration](#database-migration)

## Application Entry Point

```go
// cmd/app/main.go
package main

import (
    "context"
    "log"

    "yourapp/internal/api"
    "yourapp/internal/repository/user_repo"
    "yourapp/internal/task"
    "yourapp/migrations"

    "github.com/cago-frame/cago"
    "github.com/cago-frame/cago/configs"
    "github.com/cago-frame/cago/database/db"
    "github.com/cago-frame/cago/pkg/component"
    "github.com/cago-frame/cago/pkg/iam"
    "github.com/cago-frame/cago/pkg/iam/audit"
    "github.com/cago-frame/cago/pkg/iam/audit/audit_db"
    "github.com/cago-frame/cago/server/cron"
    "github.com/cago-frame/cago/server/mux"
)

func main() {
    ctx := context.Background()
    cfg, err := configs.NewConfig("appname")
    if err != nil {
        log.Fatalf("load config err: %v", err)
    }

    // Register repository instances
    user_repo.RegisterUser(user_repo.NewUser())

    err = cago.New(ctx, cfg).
        Registry(component.Core()).        // Logger + trace + metrics
        Registry(component.Database()).    // GORM database
        Registry(component.Broker()).      // Message queue
        Registry(component.Redis()).       // Redis
        Registry(component.Cache()).       // Cache
        Registry(cron.Cron()).             // Cron scheduler
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
    }
}
```

## API Definition

```go
// internal/api/user/user.go
package user

import "github.com/cago-frame/cago/server/mux"

// RegisterRequest registration
type RegisterRequest struct {
    mux.Meta `path:"/user/register" method:"POST"`
    Username string `form:"username" binding:"required"`
    Password string `form:"password" binding:"required"`
}

type RegisterResponse struct{}

// LoginRequest login
type LoginRequest struct {
    mux.Meta `path:"/user/login" method:"POST"`
    Username string `form:"username" binding:"required"`
    Password string `form:"password" binding:"required"`
}

// CurrentUserRequest get current user
type CurrentUserRequest struct {
    mux.Meta `path:"/user/current" method:"GET"`
}

type CurrentUserResponse struct {
    Username string `json:"username"`
}
```

### Tag Reference

- `mux.Meta`: Route metadata. `path` = URL path (comma-separated for multiple), `method` = HTTP method (defaults GET)
- `form`: Query param (GET/DELETE) or form/JSON field (POST/PUT)
- `uri`: URL path parameter (e.g., `uri:"id"` with `path:"/user/:id"`)
- `header`: HTTP header value
- `json`: JSON response field name
- `binding`: Gin validation tags (e.g., `binding:"required"`)
- `form:"key,default=value"`: Default value support

## Router

```go
// internal/api/router.go
package api

import (
    "context"

    "yourapp/internal/controller/user_ctr"
    "yourapp/internal/service/user_svc"
    "github.com/cago-frame/cago/server/mux"
)

// Router
// @title    api docs
// @version  1.0
// @BasePath /api/v1
func Router(ctx context.Context, root *mux.Router) error {
    r := root.Group("/api/v1")

    userCtr := user_ctr.NewUser()
    {
        // Public routes
        r.Group("/").Bind(
            userCtr.Register,
            userCtr.Login,
        )
        // Authenticated routes
        r.Group("/", user_svc.User().Middleware(true)).Bind(
            userCtr.CurrentUser,
            userCtr.Logout,
        )
    }
    return nil
}
```

## Controller

```go
// internal/controller/user_ctr/user.go
package user_ctr

import (
    "context"

    api "yourapp/internal/api/user"
    "yourapp/internal/service/user_svc"
    "github.com/gin-gonic/gin"
)

type User struct{}

func NewUser() *User {
    return &User{}
}

// Register - context.Context + request -> response + error
func (l *User) Register(ctx context.Context, req *api.RegisterRequest) (*api.RegisterResponse, error) {
    return user_svc.User().Register(ctx, req)
}

// Login - *gin.Context + request -> error (for cookie/session ops)
func (l *User) Login(ctx *gin.Context, req *api.LoginRequest) error {
    return user_svc.User().Login(ctx, req)
}

// CurrentUser - context.Context + request -> response + error
func (l *User) CurrentUser(ctx context.Context, req *api.CurrentUserRequest) (*api.CurrentUserResponse, error) {
    return user_svc.User().CurrentUser(ctx, req)
}
```

## Service

```go
// internal/service/user_svc/user.go
package user_svc

import (
    "context"

    api "yourapp/internal/api/user"
    "yourapp/internal/pkg/code"
    "yourapp/internal/repository/user_repo"
    "github.com/cago-frame/cago/pkg/i18n"
)

type UserSvc interface {
    Register(ctx context.Context, req *api.RegisterRequest) (*api.RegisterResponse, error)
    CurrentUser(ctx context.Context, req *api.CurrentUserRequest) (*api.CurrentUserResponse, error)
}

type userSvc struct{}

var defaultUser = &userSvc{}

func User() UserSvc {
    return defaultUser
}

func (l *userSvc) Register(ctx context.Context, req *api.RegisterRequest) (*api.RegisterResponse, error) {
    user, err := user_repo.User().FindByUsername(ctx, req.Username)
    if err != nil {
        return nil, err
    }
    if user != nil {
        return nil, i18n.NewError(ctx, code.UsernameAlreadyExists)
    }
    // create user...
    return &api.RegisterResponse{}, nil
}
```

## Repository

```go
// internal/repository/user_repo/user.go
package user_repo

import (
    "context"

    "github.com/cago-frame/cago/database/db"
    "yourapp/internal/model/entity/user_entity"
    "github.com/cago-frame/cago/pkg/consts"
    "github.com/cago-frame/cago/pkg/utils/httputils"
)

//go:generate mockgen -source user.go -destination mock/user.go
type UserRepo interface {
    Find(ctx context.Context, id int64) (*user_entity.User, error)
    FindPage(ctx context.Context, page httputils.PageRequest) ([]*user_entity.User, int64, error)
    Create(ctx context.Context, user *user_entity.User) error
    Update(ctx context.Context, user *user_entity.User) error
    Delete(ctx context.Context, id int64) error
    FindByUsername(ctx context.Context, username string) (*user_entity.User, error)
}

var defaultUser UserRepo

func User() UserRepo { return defaultUser }

func RegisterUser(i UserRepo) { defaultUser = i }

type userRepo struct{}

func NewUser() UserRepo { return &userRepo{} }

func (u *userRepo) Find(ctx context.Context, id int64) (*user_entity.User, error) {
    ret := &user_entity.User{}
    if err := db.Ctx(ctx).Where("id=? and status=?", id, consts.ACTIVE).First(ret).Error; err != nil {
        if db.RecordNotFound(err) {
            return nil, nil
        }
        return nil, err
    }
    return ret, nil
}

func (u *userRepo) FindPage(ctx context.Context, page httputils.PageRequest) ([]*user_entity.User, int64, error) {
    var list []*user_entity.User
    var count int64
    find := db.Ctx(ctx).Model(&user_entity.User{}).Where("status=?", consts.ACTIVE)
    if err := find.Count(&count).Error; err != nil {
        return nil, 0, err
    }
    if err := find.Order("createtime desc").Offset(page.GetOffset()).Limit(page.GetLimit()).Find(&list).Error; err != nil {
        return nil, 0, err
    }
    return list, count, nil
}

func (u *userRepo) Create(ctx context.Context, user *user_entity.User) error {
    return db.Ctx(ctx).Create(user).Error
}

func (u *userRepo) Update(ctx context.Context, user *user_entity.User) error {
    return db.Ctx(ctx).Updates(user).Error
}

func (u *userRepo) Delete(ctx context.Context, id int64) error {
    return db.Ctx(ctx).Model(&user_entity.User{}).Where("id=?", id).Update("status", consts.DELETE).Error
}
```

## Entity

```go
// internal/model/entity/user_entity/user.go
package user_entity

type User struct {
    ID             int64  `gorm:"column:id;type:bigint(20);not null;primary_key"`
    Username       string `gorm:"column:username;type:varchar(255);index:username,unique;not null"`
    HashedPassword string `gorm:"column:hashed_password;type:varchar(255);not null"`
    Status         int    `gorm:"column:status;type:int(11);not null"`
    Createtime     int64  `gorm:"column:createtime;type:bigint(20)"`
    Updatetime     int64  `gorm:"column:updatetime;type:bigint(20)"`
}
```

## Error Codes

```go
// internal/pkg/code/code.go
package code

const (
    UserIsBanned = iota + 10000
    UserNotFound
    UserNotLogin
    UsernameAlreadyExists
)

// internal/pkg/code/zh_cn.go
package code

import "github.com/cago-frame/cago/pkg/i18n"

func init() {
    i18n.Register(i18n.DefaultLang, zhCN)
}

var zhCN = map[int]string{
    UserIsBanned:          "用户已被禁用",
    UserNotFound:          "用户不存在",
    UserNotLogin:          "用户未登录",
    UsernameAlreadyExists: "用户名已存在",
}
```

Usage: `i18n.NewError(ctx, code.UserNotFound)`

## Cron Jobs

```go
// internal/task/crontab/example.go
package crontab

import (
    "context"
    "github.com/cago-frame/cago/pkg/logger"
)

func Example(ctx context.Context) error {
    logger.Ctx(ctx).Info("cron job executed")
    return nil
}
```

Register in task.go:

```go
cron.Default().AddFunc("*/5 * * * *", crontab.Example)
```

## Message Queue

```go
// internal/task/queue/message/example.go
package message

import "encoding/json"

type ExampleMsg struct {
    Time int64 `json:"time"`
}

func (e *ExampleMsg) Marshal() []byte {
    b, _ := json.Marshal(e)
    return b
}

func (e *ExampleMsg) Unmarshal(data []byte) error {
    return json.Unmarshal(data, e)
}
```

```go
// internal/task/queue/example.go - Publish/Subscribe helpers
package queue

import (
    "context"
    "yourapp/internal/task/queue/message"
    "github.com/cago-frame/cago/pkg/broker"
    broker2 "github.com/cago-frame/cago/pkg/broker/broker"
)

const ExampleTopic = "example"

func PublishExample(ctx context.Context, msg *message.ExampleMsg) error {
    return broker.Default().Publish(ctx, ExampleTopic, &broker2.Message{
        Body: msg.Marshal(),
    })
}

func SubscribeExample(ctx context.Context, fn func(ctx context.Context, msg *message.ExampleMsg) error) error {
    _, err := broker.Default().Subscribe(ctx, ExampleTopic, func(ctx context.Context, event broker2.Event) error {
        msg := &message.ExampleMsg{}
        if err := msg.Unmarshal(event.Message().Body); err != nil {
            return err
        }
        return fn(ctx, msg)
    }, broker2.Retry())
    return err
}
```

```go
// internal/task/queue/handler/example.go - Handler
package handler

import (
    "context"
    "yourapp/internal/task/queue"
    "yourapp/internal/task/queue/message"
    "github.com/cago-frame/cago/pkg/logger"
    "go.uber.org/zap"
)

type Example struct{}

func (u *Example) Register(ctx context.Context) error {
    return queue.SubscribeExample(ctx, u.example)
}

func (u *Example) example(ctx context.Context, msg *message.ExampleMsg) error {
    logger.Ctx(ctx).Info("received message", zap.Int64("time", msg.Time))
    return nil
}
```

## Database Migration

```go
// migrations/init.go
package migrations

import (
    "github.com/go-gormigrate/gormigrate/v2"
    "gorm.io/gorm"
)

func RunMigrations(db *gorm.DB) error {
    return run(db, T20230611)
}

func run(db *gorm.DB, fs ...func() *gormigrate.Migration) error {
    ms := make([]*gormigrate.Migration, 0)
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
    return m.Migrate()
}
```

```go
// migrations/20230611.go
package migrations

import (
    "context"
    "yourapp/internal/model/entity/user_entity"
    "github.com/cago-frame/cago/database/db"
    "github.com/go-gormigrate/gormigrate/v2"
    "gorm.io/gorm"
)

func T20230611() *gormigrate.Migration {
    return &gormigrate.Migration{
        ID: "20230611",
        Migrate: func(tx *gorm.DB) error {
            ctx := db.WithContextDB(context.Background(), tx)
            if err := tx.Migrator().AutoMigrate(&user_entity.User{}); err != nil {
                return err
            }
            // Seed data using ctx with transaction
            _ = ctx
            return nil
        },
        Rollback: func(tx *gorm.DB) error {
            return nil
        },
    }
}
```
