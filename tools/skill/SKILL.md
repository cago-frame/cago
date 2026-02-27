---
name: cago
description: "User-invocable skill for the Cago Go framework. ONLY use when the user explicitly invokes /cago. Do NOT auto-trigger. Provides project layout, API patterns (mux.Meta), controller/service/repository layer conventions, component usage (database, etcd, redis, cache, broker, cron, grpc), database migrations, message queue patterns, and complete code examples for the cago framework (github.com/cago-frame/cago)."
---

# Cago Framework

Cago is a modular Go framework for rapid API development. It integrates Gin, GORM, zap, OpenTelemetry, etc. Uses
three-tier architecture with DDD principles.

## Project Layout

```
cmd/app/main.go              # Entry point
configs/config.yaml          # Configuration
internal/
  api/                       # API request/response structs + router.go
    user/user.go             # Request/Response per domain
    router.go                # Route registration
  controller/                # Thin layer: validate + forward to service
    user_ctr/user.go
  model/
    entity/                  # GORM entities
      user_entity/user.go
  repository/                # Data access (GORM queries)
    user_repo/user.go
  service/                   # Business logic interfaces + impl
    user_svc/user.go
  task/
    crontab/                 # Cron job handlers
    queue/                   # Message queue (basic)
      handler/               # Subscription handlers
      message/               # Message structs
    producer/                # Message publishers (advanced)
    consumer/                # Message subscribers (advanced)
      subscribe/
  pkg/code/                  # Error codes + i18n
migrations/                  # go-gormigrate migrations
```

## Core Patterns

### API Definition

Define request/response structs with `mux.Meta` for route metadata:

```go
type CreateUserRequest struct {
    mux.Meta `path:"/user" method:"POST"`
    Username string `form:"username" binding:"required"`
    Age      int    `form:"age"`
}
type CreateUserResponse struct {
    ID int64 `json:"id"`
}
```

Tags: `form` (query/body), `uri` (path param), `header`, `json` (response), `binding` (validation),
`label` (i18n validation error), `form:"key,default=value"`.

Custom validation: implement `Validate(ctx context.Context) error` on request struct.

Pagination: embed `httputils.PageRequest` with `form:",inline"`.

### Handler Signatures

Controllers must use one of these signatures:

```go
func (c *Ctr) Method(ctx context.Context, req *Request) (*Response, error)
func (c *Ctr) Method(ctx *gin.Context, req *Request) (*Response, error)
func (c *Ctr) Method(ctx context.Context, req *Request) error
func (c *Ctr) Method(ctx *gin.Context, req *Request) error
```

Use `*gin.Context` when direct HTTP access is needed (cookies, headers, sessions).

### Route Binding

```go
func Router(ctx context.Context, root *mux.Router) error {
    r := root.Group("/api/v1")
    ctr := user_ctr.NewUser()
    r.Group("/").Bind(ctr.Create, ctr.List)                              // Public
    r.Group("/", authMiddleware).Bind(ctr.Update, ctr.Delete)            // With middleware
    return nil
}
```

Nested middleware via `muxutils.BindTree` for complex route hierarchies.

### Service Pattern

Interface + singleton accessor + private impl:

```go
type UserSvc interface {
    Create(ctx context.Context, req *api.CreateRequest) (*api.CreateResponse, error)
}
type userSvc struct{}
var defaultUser = &userSvc{}
func User() UserSvc { return defaultUser }
```

### Repository Pattern

Interface + Register/accessor + `db.Ctx(ctx)` for queries:

```go
type UserRepo interface { Find(ctx context.Context, id int64) (*Entity, error) }
var defaultUser UserRepo
func User() UserRepo { return defaultUser }
func RegisterUser(i UserRepo) { defaultUser = i }
```

Register in main.go: `user_repo.RegisterUser(user_repo.NewUser())`

### Database Access

```go
db.Default()                    // Default *gorm.DB
db.Ctx(ctx)                     // Context-aware (transaction support)
db.CtxWith(ctx, "secondary")   // Context-aware with named DB fallback
db.WithContextDB(ctx, tx)       // Set transaction in context
db.RecordNotFound(err)          // Check not found
```

### Transaction Pattern

```go
err := db.Ctx(ctx).Transaction(func(tx *gorm.DB) error {
    ctx = db.WithContextDB(ctx, tx)       // 关键：后续 db.Ctx(ctx) 自动使用事务
    if err := repo1.Create(ctx, e1); err != nil { return err }
    if err := repo2.Create(ctx, e2); err != nil { return err }
    return nil
})
// 事务提交后再发布消息
```

### Redis & Cache

```go
redis.Ctx(ctx).Set("key", "value", time.Hour)   // Context-aware Redis
redis.Nil(err)                                    // Check key not found

cache.Ctx(ctx).GetOrSet("key", func() (interface{}, error) {  // Cache-aside
    return db.Ctx(ctx).First(&entity).Error
}, cache.WithDepend(keyDep), cache.Expiration(time.Hour)).Scan(&result)

keyDep.InvalidKey(ctx)  // 使依赖失效，所有关联缓存自动刷新
```

### Error Codes (i18n)

```go
// pkg/code/code.go
const ( UserNotFound = iota + 10000; ... )
// pkg/code/zh_cn.go
var zhCN = map[int]string{ UserNotFound: "用户不存在" }
// Usage
return i18n.NewError(ctx, code.UserNotFound)
return i18n.NewInternalError(ctx, code.ServerError)       // 500
return i18n.NewErrorWithStatus(ctx, http.StatusNotFound, code.NotFound) // custom status
```

### Goroutines

Always use `gogo.Go(ctx, func(ctx context.Context) error { ... })` for async work.

### gRPC Server

```go
import grpcServer "github.com/cago-frame/cago/server/grpc"

// 注册 gRPC 服务 (RegistryCancel，可以停止应用)
grpcServer.GRPC(func(ctx context.Context, s *grpc.Server) error {
    pb.RegisterUserServiceServer(s, &userServiceImpl{})
    return nil
})

// 带自定义拦截器
grpcServer.GRPC(registerServices,
    grpc.ChainUnaryInterceptor(authInterceptor, logInterceptor),
)
```

Config key: `grpc.address` (default `127.0.0.1:9090`). Auto-integrates OpenTelemetry tracing and metrics when `component.Core()` is registered.

## Conventions

- Code comments and commit messages in Chinese
- Components panic on startup failure (fail-fast)
- Testing: GoConvey + testify + go.uber.org/mock + go-sqlmock + miniredis
- Linting: golangci-lint v2 (`make lint`)
- Mock generation: `//go:generate mockgen -source file.go -destination mock/file.go`

## References

- **Complete examples** (entry point, all layers, cron, queue, migration, advanced patterns):
  See [references/examples.md](references/examples.md)
- **Components & configuration** (component system, config YAML, database, etcd, redis, cache, logger, trace, metrics, broker, gogo):
  See [references/components.md](references/components.md)
