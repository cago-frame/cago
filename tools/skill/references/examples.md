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
- [gRPC Server](#grpc-server)
- [Unit Testing](#unit-testing)
- [Advanced Patterns](#advanced-patterns)

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
    "github.com/cago-frame/cago/server/cron"
    "github.com/cago-frame/cago/server/grpc"
    "github.com/cago-frame/cago/server/mux"
)

func main() {
    ctx := context.Background()
    cfg, err := configs.NewConfig("appname")
    if err != nil {
        log.Fatalf("load config err: %v", err)
    }

    // Register repository instances (dependency injection)
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
        Registry(cago.FuncComponent(task.Task)).
        RegistryCancel(mux.HTTP(api.Router)).
        RegistryCancel(grpc.GRPC(rpc.Register)).
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
- `binding`: Gin validation tags (e.g., `binding:"required"`, `binding:"oneof=0 1 2"`)
- `label`: i18n label for validation error messages
- `form:"key,default=value"`: Default value support

### Custom Validation

Request structs can implement `Validate(ctx context.Context) error` for complex validation:

```go
type CreateScriptRequest struct {
    mux.Meta    `path:"/scripts" method:"POST"`
    Content     string `form:"content" binding:"required,max=102400" label:"script description"`
    Code        string `form:"code" binding:"required,max=10485760" label:"script code"`
    Name        string `form:"name" binding:"max=128" label:"library name"`
    Type        int    `form:"type" binding:"required"`
}

// Custom validation logic, executed after binding validation
func (s *CreateScriptRequest) Validate(ctx context.Context) error {
    if s.Type == entity.LibraryType {
        s.Name = strings.TrimSpace(s.Name)
        if s.Name == "" {
            return i18n.NewError(ctx, code.ScriptNameIsEmpty)
        }
    }
    return nil
}
```

### Pagination Request

```go
type ListRequest struct {
    mux.Meta              `path:"/scripts" method:"GET"`
    httputils.PageRequest `form:",inline"`  // Embedded pagination params (page, size)
    Keyword               string `form:"keyword"`
    Sort                  string `form:"sort,default=today_download" binding:"oneof=today_update today_download total_download score createtime"`
}
```

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

## Entity (Rich Domain Model)

Entities follow the Rich Domain Model pattern, containing both data fields and business logic methods:

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

// Business validation method on the entity
func (u *User) Check(ctx context.Context) error {
    if u == nil {
        return i18n.NewError(ctx, code.UserNotFound)
    }
    if u.Status != consts.ACTIVE {
        return i18n.NewError(ctx, code.UserIsBanned)
    }
    return nil
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

Error with HTTP status: `i18n.NewErrorWithStatus(ctx, http.StatusNotFound, code.ScriptNotFound)`

Internal error (500): `i18n.NewInternalError(ctx, code.ScriptCreateFailed)`

Forbidden error (403): `i18n.NewForbiddenError(ctx, code.UserNotPermission)`

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

Environment-based cron:

```go
func Crontab(ctx context.Context, cfg *configs.Config) error {
    // Skip cron jobs in PRE environment to avoid conflicts with production
    if configs.Default().Env == configs.PRE {
        return nil
    }
    // register cron jobs...
    return nil
}
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

### Avoid Large Data in Messages

```go
// Only pass IDs in messages, not full data (avoid oversized MQ payloads)
type ScriptCreateMsg struct {
    Script *script_entity.Script
    CodeID int64  // Only pass ID, consumers query full data themselves
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
    return run(db, T20230611, T20250107)
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

## gRPC Server

```go
// internal/api/grpc_router.go
package api

import (
    "context"

    pb "yourapp/proto/user"
    "yourapp/internal/service/user_svc"
    "google.golang.org/grpc"
)

func GRPCRouter(ctx context.Context, s *grpc.Server) error {
    pb.RegisterUserServiceServer(s, &userGRPCService{})
    return nil
}

type userGRPCService struct {
    pb.UnimplementedUserServiceServer
}

func (s *userGRPCService) GetUser(ctx context.Context, req *pb.GetUserRequest) (*pb.GetUserResponse, error) {
    user, err := user_svc.User().Find(ctx, req.Id)
    if err != nil {
        return nil, err
    }
    return &pb.GetUserResponse{
        Id:       user.ID,
        Username: user.Username,
    }, nil
}
```

### With Custom Interceptors

```go
import (
    grpcServer "github.com/cago-frame/cago/server/grpc"
    "google.golang.org/grpc"
)

// Register in main.go
grpcServer.GRPC(api.GRPCRouter,
    grpc.ChainUnaryInterceptor(authInterceptor),
)
```

## Advanced Patterns

### Transaction with Context Propagation

Use transactions in Service, propagated to all repository calls via context:

```go
func (s *scriptSvc) Create(ctx context.Context, req *api.CreateRequest) (*api.CreateResponse, error) {
    script := &script_entity.Script{
        UserID:     req.UserID,
        Content:    req.Content,
        Status:     consts.ACTIVE,
        Createtime: time.Now().Unix(),
    }
    scriptCode := &script_entity.Code{
        Code:       req.Code,
        Changelog:  req.Changelog,
        Status:     consts.ACTIVE,
        Createtime: time.Now().Unix(),
    }

    err := db.Ctx(ctx).Transaction(func(tx *gorm.DB) error {
        // Key: put tx into context
        ctx = db.WithContextDB(ctx, tx)

        // All repositories using db.Ctx(ctx) automatically use the transaction
        if err := script_repo.Script().Create(ctx, script); err != nil {
            logger.Ctx(ctx).Error("failed to create script", zap.Error(err))
            return i18n.NewInternalError(ctx, code.ScriptCreateFailed)
        }

        scriptCode.ScriptID = script.ID
        if err := script_repo.ScriptCode().Create(ctx, scriptCode); err != nil {
            logger.Ctx(ctx).Error("failed to create code", zap.Error(err))
            return i18n.NewInternalError(ctx, code.ScriptCreateFailed)
        }

        // Link categories, tags, etc.
        if err := Category().LinkScriptCategory(ctx, script.ID, req.CategoryID); err != nil {
            return err
        }

        return nil
    })
    if err != nil {
        return nil, err
    }

    // Publish async messages after transaction commits
    if err := producer.PublishScriptCreate(ctx, script, scriptCode); err != nil {
        logger.Ctx(ctx).Error("failed to publish create message", zap.Error(err))
    }

    return &api.CreateResponse{ID: script.ID}, nil
}
```

### Repository with Cache-Aside and KeyDepend

Repository with cache-aside + dependency invalidation pattern:

```go
type scriptRepo struct{}

func (u *scriptRepo) key(id int64) string {
    return "script:" + strconv.FormatInt(id, 10)
}

func (u *scriptRepo) KeyDepend(id int64) *cache2.KeyDepend {
    return cache2.NewKeyDepend(cache.Default(), u.key(id)+":dep")
}

// Read with cache
func (u *scriptRepo) Find(ctx context.Context, id int64) (*entity.Script, error) {
    ret := &entity.Script{}
    err := cache.Ctx(ctx).GetOrSet(u.key(id), func() (interface{}, error) {
        if err := db.Ctx(ctx).Where("id=?", id).First(ret).Error; err != nil {
            if db.RecordNotFound(err) {
                return nil, nil
            }
            return nil, err
        }
        return ret, nil
    }, cache.WithDepend(u.KeyDepend(id)), cache.Expiration(time.Hour)).Scan(ret)
    if err != nil {
        return nil, err
    }
    return ret, nil
}

// Invalidate cache on create
func (u *scriptRepo) Create(ctx context.Context, script *entity.Script) error {
    if err := db.Ctx(ctx).Create(script).Error; err != nil {
        return err
    }
    return u.KeyDepend(script.ID).InvalidKey(ctx)
}

// Invalidate cache on update
func (u *scriptRepo) Update(ctx context.Context, script *entity.Script) error {
    if err := db.Ctx(ctx).Updates(script).Error; err != nil {
        return err
    }
    return u.KeyDepend(script.ID).InvalidKey(ctx)
}
```

### Memory Cache for Large Data

Use in-memory cache for large data (e.g., code content) to avoid frequent Redis access:

```go
import "github.com/cago-frame/cago/database/cache/cache/memory"

type codeRepo struct {
    memoryCache cache2.Cache
}

func NewCodeRepo() *codeRepo {
    c, _ := memory.NewMemoryCache()
    return &codeRepo{memoryCache: c}
}

func (u *codeRepo) FindLatest(ctx context.Context, scriptId int64, withCode bool) (*entity.Code, error) {
    ret := &entity.Code{}
    cacheKey := fmt.Sprintf("code:%d:%v", scriptId, withCode)

    err := u.memoryCache.GetOrSet(ctx, cacheKey, func() (interface{}, error) {
        q := db.Ctx(ctx)
        if !withCode {
            q = q.Select(ret.Fields())  // Exclude large fields when code is not needed
        }
        if err := q.Order("createtime desc").
            First(ret, "script_id=? and status=?", scriptId, consts.ACTIVE).Error; err != nil {
            if db.RecordNotFound(err) {
                return nil, nil
            }
            return nil, err
        }
        return ret, nil
    }, cache2.Expiration(time.Hour), cache2.WithDepend(u.KeyDepend(scriptId))).Scan(&ret)

    return ret, err
}
```

### RouterTree (Nested Middleware)

Use `muxutils.BindTree` to build nested middleware trees:

```go
import "github.com/cago-frame/cago/server/mux/muxutils"

func (s *Script) Router(root *mux.Router, r *mux.Router) {
    muxutils.BindTree(r, []*muxutils.RouterTree{
        {
            // Public routes
            Middleware: []gin.HandlerFunc{optionalAuthMiddleware},
            Handler: []interface{}{
                s.List,
                s.Detail,
            },
        },
        {
            // Requires authentication
            Middleware: []gin.HandlerFunc{requireAuthMiddleware},
            Handler: []interface{}{
                s.Create,
                // Nested: requires auth + resource permission check
                &muxutils.RouterTree{
                    Middleware: []gin.HandlerFunc{requireResourceMiddleware},
                    Handler: []interface{}{
                        s.Watch,
                        // Deeper nesting: requires auth + resource permission + write permission
                        &muxutils.RouterTree{
                            Middleware: []gin.HandlerFunc{
                                writePermissionMiddleware,
                                notArchivedMiddleware,
                            },
                            Handler: []interface{}{
                                s.UpdateCode,
                                s.Delete,
                            },
                        },
                    },
                },
            },
        },
    })
}
```

### Rate Limiting with Redis

Use combination rate limiting in Controller:

```go
import "github.com/cago-frame/cago/pkg/limit"

type Script struct {
    limit limit.Limit
}

func NewScript() *Script {
    return &Script{
        limit: limit.NewCombinationLimit(
            limit.NewPeriodLimit(300, 6, redis.Default(), "limit:create:script:minute"),   // 6 per 5 minutes
            limit.NewPeriodLimit(3600, 8, redis.Default(), "limit:create:script:hour"),    // 8 per hour
        ),
    }
}

func (s *Script) Create(ctx context.Context, req *api.CreateRequest) (*api.CreateResponse, error) {
    uid := strconv.FormatInt(getCurrentUserID(ctx), 10)
    resp, err := s.limit.FuncTake(ctx, uid, func() (interface{}, error) {
        return script_svc.Script().Create(ctx, req)
    })
    if err != nil {
        return nil, err
    }
    return resp.(*api.CreateResponse), nil
}
```

### Context Enrichment (Logger + Trace)

Enrich context in middleware to automatically add logger and trace fields for subsequent code:

```go
// Enrich logger and trace in middleware
func enrichContext(ctx context.Context, userID int64) context.Context {
    // Add to trace span
    trace.SpanFromContext(ctx).SetAttributes(
        attribute.Int64("user_id", userID),
    )

    // Add to logger context (subsequent logger.Ctx(ctx) calls automatically include user_id)
    ctx = logger.WithContextLogger(ctx, logger.Ctx(ctx).With(
        zap.Int64("user_id", userID),
    ))

    return ctx
}

// Modify request context in Gin middleware
func MyMiddleware() gin.HandlerFunc {
    return func(ctx *gin.Context) {
        newCtx := enrichContext(ctx.Request.Context(), getUserID(ctx))
        ctx.Request = ctx.Request.WithContext(newCtx)
    }
}
```

### Entity Validation Methods

Define validation methods on Entity, reusable in the service layer:

```go
type Script struct {
    ID     int64        `gorm:"column:id;primary_key"`
    UserID int64        `gorm:"column:user_id"`
    Status int64        `gorm:"column:status"`
    Archive ScriptArchive `gorm:"column:archive"`
    // ...
}

// Check if operable (exists and not deleted)
func (s *Script) CheckOperate(ctx context.Context) error {
    if s == nil {
        return i18n.NewErrorWithStatus(ctx, http.StatusNotFound, code.ScriptNotFound)
    }
    if s.Status != consts.ACTIVE {
        return i18n.NewErrorWithStatus(ctx, http.StatusNotFound, code.ScriptIsDelete)
    }
    return nil
}

// Check permission (is the author)
func (s *Script) CheckPermission(ctx context.Context, currentUserID int64) error {
    if err := s.CheckOperate(ctx); err != nil {
        return err
    }
    if s.UserID != currentUserID {
        return i18n.NewErrorWithStatus(ctx, http.StatusForbidden, code.UserNotPermission)
    }
    return nil
}

// Check if archived
func (s *Script) IsArchive(ctx context.Context) error {
    if err := s.CheckOperate(ctx); err != nil {
        return err
    }
    if s.Archive == IsArchive {
        return i18n.NewError(ctx, code.ScriptIsArchive)
    }
    return nil
}
```

### Consumer/Producer Pattern

Separate MQ producers and consumers into different modules:

```go
// internal/task/producer/script.go - Producer
package producer

const ScriptCreateTopic = "script.create"

type ScriptCreateMsg struct {
    Script *script_entity.Script
    CodeID int64
}

func PublishScriptCreate(ctx context.Context, script *script_entity.Script, code *script_entity.Code) error {
    body, _ := json.Marshal(&ScriptCreateMsg{
        Script: script,
        CodeID: code.ID,  // Only pass ID to avoid oversized payloads
    })
    return broker.Default().Publish(ctx, ScriptCreateTopic, &broker2.Message{Body: body})
}

func SubscribeScriptCreate(ctx context.Context, fn func(ctx context.Context, script *script_entity.Script, codeID int64) error, opts ...broker2.SubscribeOption) error {
    _, err := broker.Default().Subscribe(ctx, ScriptCreateTopic, func(ctx context.Context, ev broker2.Event) error {
        m := &ScriptCreateMsg{}
        if err := json.Unmarshal(ev.Message().Body, m); err != nil {
            return err
        }
        return fn(ctx, m.Script, m.CodeID)
    }, opts...)
    return err
}
```

```go
// internal/task/consumer/subscribe/es_sync.go - Consumer
type EsSync struct{}

func (e *EsSync) Subscribe(ctx context.Context) error {
    return producer.SubscribeScriptCreate(ctx, e.onScriptCreate, broker2.Retry())
}

func (e *EsSync) onScriptCreate(ctx context.Context, script *script_entity.Script, codeID int64) error {
    // Sync to Elasticsearch
    return script_repo.Migrate().Index(ctx, script)
}
```

## Unit Testing

Test files are placed in the corresponding controller directory, one test file per module.

### File Organization

```
internal/controller/
  user_ctr/
    user.go
    user_test.go           # User module tests
  example_ctr/
    example.go
    example_test.go        # Example module tests
```

### Test Setup

Each test file has a `setupXxxTest` function that initializes mock dependencies and registers routes.

```go
// internal/controller/user_ctr/user_test.go
package user_ctr

import (
    "context"
    "testing"

    api "yourapp/internal/api/user"
    "yourapp/internal/model/entity/user_entity"
    "yourapp/internal/repository/user_repo"
    mock_user_repo "yourapp/internal/repository/user_repo/mock"
    "github.com/cago-frame/cago/pkg/consts"
    "github.com/cago-frame/cago/pkg/utils/testutils"
    "github.com/cago-frame/cago/server/mux/muxtest"
    "github.com/smartystreets/goconvey/convey"
    "github.com/stretchr/testify/assert"
    "go.uber.org/mock/gomock"
)

func setupUserTest(t *testing.T) (context.Context, *mock_user_repo.MockUserRepo, *muxtest.TestMux) {
    testutils.Cache(t)
    mockCtrl := gomock.NewController(t)
    t.Cleanup(func() { mockCtrl.Finish() })

    ctx := context.Background()
    mockUserRepo := mock_user_repo.NewMockUserRepo(mockCtrl)
    user_repo.RegisterUser(mockUserRepo)

    // Register only the routes needed for this controller module
    testMux := muxtest.NewTestMux()
    r := testMux.Group("/api/v1")
    userCtr := NewUser()
    r.Group("/").Bind(
        userCtr.Create,
        userCtr.List,
    )

    return ctx, mockUserRepo, testMux
}
```

### Test Examples — CRUD Module

```go
func TestUserCreate(t *testing.T) {
    ctx, mockUserRepo, testMux := setupUserTest(t)

    convey.Convey("创建用户", t, func() {
        convey.Convey("创建成功", func() {
            mockUserRepo.EXPECT().FindByUsername(gomock.Any(), "newuser").Return(nil, nil)
            mockUserRepo.EXPECT().Create(gomock.Any(), gomock.Any()).Return(nil)
            resp := &api.CreateResponse{}
            err := testMux.Do(ctx, &api.CreateRequest{
                Username: "newuser",
                Password: "password123",
            }, resp)
            assert.NoError(t, err)
            assert.NotZero(t, resp.ID)
        })
        convey.Convey("用户名已存在", func() {
            mockUserRepo.EXPECT().FindByUsername(gomock.Any(), "existuser").Return(&user_entity.User{
                ID: 1, Username: "existuser", Status: consts.ACTIVE,
            }, nil)
            resp := &api.CreateResponse{}
            err := testMux.Do(ctx, &api.CreateRequest{
                Username: "existuser",
                Password: "password123",
            }, resp)
            assert.Error(t, err)
        })
    })
}

func TestUserFind(t *testing.T) {
    ctx, mockUserRepo, testMux := setupUserTest(t)

    convey.Convey("查询用户", t, func() {
        convey.Convey("查询成功", func() {
            mockUserRepo.EXPECT().Find(gomock.Any(), int64(1)).Return(&user_entity.User{
                ID: 1, Username: "test", Status: consts.ACTIVE,
            }, nil)
            resp := &api.FindResponse{}
            err := testMux.Do(ctx, &api.FindRequest{ID: 1}, resp)
            assert.NoError(t, err)
            assert.Equal(t, "test", resp.Username)
        })
        convey.Convey("用户不存在", func() {
            mockUserRepo.EXPECT().Find(gomock.Any(), int64(999)).Return(nil, nil)
            resp := &api.FindResponse{}
            err := testMux.Do(ctx, &api.FindRequest{ID: 999}, resp)
            assert.Error(t, err)
        })
    })
}
```

### Test Example — Module with Broker

When testing modules that depend on the broker (message queue), set up an in-memory broker:

```go
// internal/controller/example_ctr/example_test.go
package example_ctr

import (
    "github.com/cago-frame/cago/pkg/broker"
    "github.com/cago-frame/cago/pkg/broker/event_bus"
    // ... other imports
)

func setupExampleTest(t *testing.T) (context.Context, *mock_user_repo.MockUserRepo, *muxtest.TestMux) {
    testutils.Cache(t)
    mockCtrl := gomock.NewController(t)
    t.Cleanup(func() { mockCtrl.Finish() })

    ctx := context.Background()
    mockUserRepo := mock_user_repo.NewMockUserRepo(mockCtrl)
    user_repo.RegisterUser(mockUserRepo)

    broker.SetBroker(event_bus.NewEvBusBroker())  // In-memory broker for testing

    testMux := muxtest.NewTestMux()
    r := testMux.Group("/api/v1")
    exampleCtr := NewExample()
    r.Group("/").Bind(exampleCtr.Ping)

    return ctx, mockUserRepo, testMux
}

func TestExamplePing(t *testing.T) {
    ctx, _, testMux := setupExampleTest(t)

    convey.Convey("Ping", t, func() {
        resp := &api.PingResponse{}
        err := testMux.Do(ctx, &api.PingRequest{}, resp)
        assert.NoError(t, err)
        assert.NotEmpty(t, resp.Pong)
    })
}
```

### TestMux Options

`muxtest.TestMux` embeds `muxclient.Client`. Use `Do(ctx, req, resp, opts...)` for testing:

```go
// Basic usage
err := testMux.Do(ctx, &api.CreateRequest{Username: "test"}, resp)

// With custom headers (e.g., authentication)
err := testMux.Do(ctx, req, resp, muxclient.WithHeader(http.Header{
    "Authorization": []string{"Bearer token"},
}))

// Override path (useful for path params)
err := testMux.Do(ctx, req, resp, muxclient.WithPath("/api/v1/user/123"))

// Capture raw HTTP response
var httpResp *http.Response
err := testMux.Do(ctx, req, resp, muxclient.WithResponse(&httpResp))
```

### Test Utilities (testutils)

```go
import "github.com/cago-frame/cago/pkg/utils/testutils"

testutils.Cache(t)                           // In-memory cache (once per test suite)
testutils.Redis(t)                           // Miniredis (once per test suite)
testutils.IAM(t, database, opts...)          // IAM component
broker.SetBroker(event_bus.NewEvBusBroker()) // In-memory broker
```

### Testing Key Points

| Item | Convention |
|------|-----------|
| Test file location | `internal/controller/<module>_ctr/<module>_test.go` |
| Broker for tests | `broker.SetBroker(event_bus.NewEvBusBroker())` (in-memory) |
| Mock generation | `//go:generate mockgen -source user.go -destination mock/user.go` |
| Test framework | GoConvey (`convey.Convey`) + testify (`assert`) + gomock |
| One TestXxx per feature | `TestUserCreate`, `TestUserFind`, `TestExamplePing`, etc. |
| Error assertions | `assert.Error(t, err)` or `assert.Equal(t, expectedErr, err)` |
