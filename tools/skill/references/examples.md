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
    Content     string `form:"content" binding:"required,max=102400" label:"脚本详细描述"`
    Code        string `form:"code" binding:"required,max=10485760" label:"脚本代码"`
    Name        string `form:"name" binding:"max=128" label:"库名"`
    Type        int    `form:"type" binding:"required"`
}

// 自定义校验逻辑，在 binding 校验之后执行
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
    httputils.PageRequest `form:",inline"`  // 内嵌分页参数 (page, size)
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

// Entity 上的业务校验方法
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
    // PRE 环境不执行定时任务，避免与生产冲突
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

### 消息中避免大数据

```go
// 消息中只传 ID，不传完整数据 (避免 MQ payload 过大)
type ScriptCreateMsg struct {
    Script *script_entity.Script
    CodeID int64  // 只传 ID，消费者自己查询完整数据
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

### 带自定义拦截器

```go
import (
    grpcServer "github.com/cago-frame/cago/server/grpc"
    "google.golang.org/grpc"
)

// main.go 中注册
grpcServer.GRPC(api.GRPCRouter,
    grpc.ChainUnaryInterceptor(authInterceptor),
)
```

## Advanced Patterns

### Transaction with Context Propagation

Service 中使用事务，通过 context 传播到所有 repository 调用：

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
        // 关键：将 tx 放入 context
        ctx = db.WithContextDB(ctx, tx)

        // 所有使用 db.Ctx(ctx) 的 repository 自动使用事务
        if err := script_repo.Script().Create(ctx, script); err != nil {
            logger.Ctx(ctx).Error("创建脚本失败", zap.Error(err))
            return i18n.NewInternalError(ctx, code.ScriptCreateFailed)
        }

        scriptCode.ScriptID = script.ID
        if err := script_repo.ScriptCode().Create(ctx, scriptCode); err != nil {
            logger.Ctx(ctx).Error("创建代码失败", zap.Error(err))
            return i18n.NewInternalError(ctx, code.ScriptCreateFailed)
        }

        // 关联分类、标签等
        if err := Category().LinkScriptCategory(ctx, script.ID, req.CategoryID); err != nil {
            return err
        }

        return nil
    })
    if err != nil {
        return nil, err
    }

    // 事务提交后再发布异步消息
    if err := producer.PublishScriptCreate(ctx, script, scriptCode); err != nil {
        logger.Ctx(ctx).Error("发布创建消息失败", zap.Error(err))
    }

    return &api.CreateResponse{ID: script.ID}, nil
}
```

### Repository with Cache-Aside and KeyDepend

Repository 使用缓存 + 依赖失效模式：

```go
type scriptRepo struct{}

func (u *scriptRepo) key(id int64) string {
    return "script:" + strconv.FormatInt(id, 10)
}

func (u *scriptRepo) KeyDepend(id int64) *cache2.KeyDepend {
    return cache2.NewKeyDepend(cache.Default(), u.key(id)+":dep")
}

// 读取带缓存
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

// 创建时使缓存失效
func (u *scriptRepo) Create(ctx context.Context, script *entity.Script) error {
    if err := db.Ctx(ctx).Create(script).Error; err != nil {
        return err
    }
    return u.KeyDepend(script.ID).InvalidKey(ctx)
}

// 更新时使缓存失效
func (u *scriptRepo) Update(ctx context.Context, script *entity.Script) error {
    if err := db.Ctx(ctx).Updates(script).Error; err != nil {
        return err
    }
    return u.KeyDepend(script.ID).InvalidKey(ctx)
}
```

### Memory Cache for Large Data

对大数据 (如代码内容) 使用内存缓存，避免频繁访问 Redis：

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
            q = q.Select(ret.Fields())  // 不需要代码时排除大字段
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

使用 `muxutils.BindTree` 构建嵌套中间件树：

```go
import "github.com/cago-frame/cago/server/mux/muxutils"

func (s *Script) Router(root *mux.Router, r *mux.Router) {
    muxutils.BindTree(r, []*muxutils.RouterTree{
        {
            // 公开接口
            Middleware: []gin.HandlerFunc{optionalAuthMiddleware},
            Handler: []interface{}{
                s.List,
                s.Detail,
            },
        },
        {
            // 需要登录
            Middleware: []gin.HandlerFunc{requireAuthMiddleware},
            Handler: []interface{}{
                s.Create,
                // 嵌套：需要登录 + 资源权限检查
                &muxutils.RouterTree{
                    Middleware: []gin.HandlerFunc{requireResourceMiddleware},
                    Handler: []interface{}{
                        s.Watch,
                        // 更深层嵌套：需要登录 + 资源权限 + 写权限
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

Controller 中使用组合限流：

```go
import "github.com/cago-frame/cago/pkg/limit"

type Script struct {
    limit limit.Limit
}

func NewScript() *Script {
    return &Script{
        limit: limit.NewCombinationLimit(
            limit.NewPeriodLimit(300, 6, redis.Default(), "limit:create:script:minute"),   // 5分钟6次
            limit.NewPeriodLimit(3600, 8, redis.Default(), "limit:create:script:hour"),    // 1小时8次
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

在 middleware 中丰富 context，为后续代码自动添加日志和 trace 字段：

```go
// Middleware 中丰富 logger 和 trace
func enrichContext(ctx context.Context, userID int64) context.Context {
    // 添加到 trace span
    trace.SpanFromContext(ctx).SetAttributes(
        attribute.Int64("user_id", userID),
    )

    // 添加到 logger context (后续 logger.Ctx(ctx) 自动带上 user_id)
    ctx = logger.WithContextLogger(ctx, logger.Ctx(ctx).With(
        zap.Int64("user_id", userID),
    ))

    return ctx
}

// Gin middleware 中修改 request context
func MyMiddleware() gin.HandlerFunc {
    return func(ctx *gin.Context) {
        newCtx := enrichContext(ctx.Request.Context(), getUserID(ctx))
        ctx.Request = ctx.Request.WithContext(newCtx)
    }
}
```

### Entity Validation Methods

Entity 上定义校验方法，在 service 层复用：

```go
type Script struct {
    ID     int64        `gorm:"column:id;primary_key"`
    UserID int64        `gorm:"column:user_id"`
    Status int64        `gorm:"column:status"`
    Archive ScriptArchive `gorm:"column:archive"`
    // ...
}

// 检查是否可操作 (存在且未删除)
func (s *Script) CheckOperate(ctx context.Context) error {
    if s == nil {
        return i18n.NewErrorWithStatus(ctx, http.StatusNotFound, code.ScriptNotFound)
    }
    if s.Status != consts.ACTIVE {
        return i18n.NewErrorWithStatus(ctx, http.StatusNotFound, code.ScriptIsDelete)
    }
    return nil
}

// 检查权限 (是否是作者)
func (s *Script) CheckPermission(ctx context.Context, currentUserID int64) error {
    if err := s.CheckOperate(ctx); err != nil {
        return err
    }
    if s.UserID != currentUserID {
        return i18n.NewErrorWithStatus(ctx, http.StatusForbidden, code.UserNotPermission)
    }
    return nil
}

// 检查是否已归档
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

将 MQ 生产者和消费者分离到不同模块：

```go
// internal/task/producer/script.go - 生产者
package producer

const ScriptCreateTopic = "script.create"

type ScriptCreateMsg struct {
    Script *script_entity.Script
    CodeID int64
}

func PublishScriptCreate(ctx context.Context, script *script_entity.Script, code *script_entity.Code) error {
    body, _ := json.Marshal(&ScriptCreateMsg{
        Script: script,
        CodeID: code.ID,  // 只传 ID，避免 payload 过大
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
// internal/task/consumer/subscribe/es_sync.go - 消费者
type EsSync struct{}

func (e *EsSync) Subscribe(ctx context.Context) error {
    return producer.SubscribeScriptCreate(ctx, e.onScriptCreate, broker2.Retry())
}

func (e *EsSync) onScriptCreate(ctx context.Context, script *script_entity.Script, codeID int64) error {
    // 同步到 Elasticsearch
    return script_repo.Migrate().Index(ctx, script)
}
```

### Testing Patterns

```go
func TestRouter(t *testing.T) {
    // Setup mock infrastructure
    testutils.Cache(t)
    mockCtrl := gomock.NewController(t)
    defer mockCtrl.Finish()

    // Mock repository
    mockUserRepo := mock_user_repo.NewMockUserRepo(mockCtrl)
    user_repo.RegisterUser(mockUserRepo)

    // Setup test HTTP mux
    testMux := muxtest.NewTestMux()
    err := Router(context.Background(), testMux.Router)
    assert.Nil(t, err)

    convey.Convey("用户注册", t, func() {
        mockUserRepo.EXPECT().FindByUsername(gomock.Any(), "test").Return(nil, nil)
        mockUserRepo.EXPECT().Create(gomock.Any(), gomock.Any()).Return(nil)

        resp := &user.RegisterResponse{}
        err := testMux.Do(ctx, &user.RegisterRequest{
            Username: "test",
            Password: "qwe123",
        }, resp)
        assert.NoError(t, err)

        convey.Convey("重复注册失败", func() {
            mockUserRepo.EXPECT().FindByUsername(gomock.Any(), "test").Return(&user_entity.User{
                ID: 1, Username: "test", Status: consts.ACTIVE,
            }, nil)
            resp := &user.RegisterResponse{}
            err := testMux.Do(ctx, &user.RegisterRequest{
                Username: "test",
                Password: "qwe123",
            }, resp)
            assert.Error(t, err)
        })
    })
}
```
