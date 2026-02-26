---
name: cago
description: "User-invocable skill for the Cago Go framework. ONLY use when the user explicitly invokes /cago. Do NOT auto-trigger. Provides project layout, API patterns (mux.Meta), controller/service/repository layer conventions, component usage (database, redis, cache, broker, cron), database migrations, message queue patterns, and complete code examples for the cago framework (github.com/cago-frame/cago)."
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
    queue/                   # Message queue
      handler/               # Subscription handlers
      message/               # Message structs
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
`form:"key,default=value"`.

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
db.WithContextDB(ctx, tx)       // Set transaction in context
db.RecordNotFound(err)          // Check not found
```

### Error Codes (i18n)

```go
// pkg/code/code.go
const ( UserNotFound = iota + 10000; ... )
// pkg/code/zh_cn.go
var zhCN = map[int]string{ UserNotFound: "用户不存在" }
// Usage
return i18n.NewError(ctx, code.UserNotFound)
```

### Goroutines

Always use `gogo.Go(ctx, func(ctx context.Context) error { ... })` for async work.

## Conventions

- Code comments and commit messages in Chinese
- Components panic on startup failure (fail-fast)
- Testing: GoConvey + testify + go.uber.org/mock + go-sqlmock + miniredis
- Linting: golangci-lint v2 (`make lint`)
- Mock generation: `//go:generate mockgen -source file.go -destination mock/file.go`

## References

- **Complete examples** (entry point, all layers, cron, queue, migration):
  See [references/examples.md](references/examples.md)
- **Components & configuration** (component system, config YAML, database, redis, cache, logger, broker, IAM, gogo):
  See [references/components.md](references/components.md)
