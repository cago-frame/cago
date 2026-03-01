---
name: cago
description: "Cago Go framework skill. Provides project layout, API patterns (mux.Meta), controller/service/repository layer conventions, component usage (database, etcd, redis, cache, broker, cron, grpc), database migrations, message queue patterns, TDD workflow, and complete code examples for the cago framework (github.com/cago-frame/cago).\nTRIGGER when: code imports `github.com/cago-frame/cago`, go.mod contains cago dependency, user mentions cago framework, or user explicitly invokes /cago.\nDO NOT TRIGGER when: project does not use cago framework, general Go questions unrelated to cago."
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
    user_ctr/
      user.go
      user_test.go           # Unit tests per module
  model/
    entity/                  # GORM entities (Rich Domain Model, with business logic methods)
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

### Entity (Rich Domain Model)

Entities follow the Rich Domain Model pattern — they contain not only data fields but also encapsulate business logic methods related to the entity (e.g., validation, status checks), avoiding piling all logic into the Service layer.

```go
// model/entity/user_entity/user.go
type User struct {
    ID             int64  `gorm:"column:id;type:bigint(20);not null;primary_key"`
    Username       string `gorm:"column:username;type:varchar(255);index:username,unique;not null"`
    HashedPassword string `gorm:"column:hashed_password;type:varchar(255);not null"`
    Status         int    `gorm:"column:status;type:int(11);not null"`
    Createtime     int64  `gorm:"column:createtime;type:bigint(20)"`
    Updatetime     int64  `gorm:"column:updatetime;type:bigint(20)"`
}

// Entity method: encapsulates business logic related to User
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

Suitable for Entity: existence checks, status validation, field formatting, simple business rules.
Not suitable for Entity: cross-entity coordination, dependencies on external services (e.g., calling Repository or third-party APIs).

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
    ctx = db.WithContextDB(ctx, tx)       // Key: subsequent db.Ctx(ctx) calls automatically use the transaction
    if err := repo1.Create(ctx, e1); err != nil { return err }
    if err := repo2.Create(ctx, e2); err != nil { return err }
    return nil
})
// Publish messages after transaction commits
```

### Redis & Cache

```go
redis.Ctx(ctx).Set("key", "value", time.Hour)   // Context-aware Redis
redis.Nil(err)                                    // Check key not found

cache.Ctx(ctx).GetOrSet("key", func() (interface{}, error) {  // Cache-aside
    return db.Ctx(ctx).First(&entity).Error
}, cache.WithDepend(keyDep), cache.Expiration(time.Hour)).Scan(&result)

keyDep.InvalidKey(ctx)  // Invalidate dependency, all associated caches auto-refresh
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

// Register gRPC service (RegistryCancel, can stop the app)
grpcServer.GRPC(func(ctx context.Context, s *grpc.Server) error {
    pb.RegisterUserServiceServer(s, &userServiceImpl{})
    return nil
})

// With custom interceptors
grpcServer.GRPC(registerServices,
    grpc.ChainUnaryInterceptor(authInterceptor, logInterceptor),
)
```

Config key: `grpc.address` (default `127.0.0.1:9090`). Auto-integrates OpenTelemetry tracing and metrics when `component.Core()` is registered.

## TDD Development Workflow (Recommended)

When developing with Cago, follow TDD (Test-Driven Development) to ensure code quality and design clarity:

### Workflow

1. **Write API definition** — Define request/response structs in `internal/api/`
2. **Write tests first** — Create test file in controller directory, write test cases covering expected behavior (success + error scenarios)
3. **Run tests → verify they fail** — `go test -v -run TestXxx ./internal/controller/xxx_ctr/...`
4. **Implement code** — Write controller → service → repository layer code to make tests pass
5. **Run tests → verify they pass** — All test cases should be green
6. **Refactor** — Clean up code while keeping tests passing, then run `make lint`

### TDD Step-by-Step Example

**Step 1: Define API**

```go
// internal/api/user/user.go
type CreateUserRequest struct {
    mux.Meta `path:"/user" method:"POST"`
    Username string `form:"username" binding:"required"`
}
type CreateUserResponse struct {
    ID int64 `json:"id"`
}
```

**Step 2: Write test first (tests will fail — service/repo not yet implemented)**

```go
// internal/controller/user_ctr/user_test.go
func TestUserCreate(t *testing.T) {
    ctx, mockUserRepo, testMux := setupUserTest(t)

    convey.Convey("创建用户", t, func() {
        convey.Convey("创建成功", func() {
            mockUserRepo.EXPECT().FindByUsername(gomock.Any(), "newuser").Return(nil, nil)
            mockUserRepo.EXPECT().Create(gomock.Any(), gomock.Any()).Return(nil)
            resp := &api.CreateUserResponse{}
            err := testMux.Do(ctx, &api.CreateUserRequest{Username: "newuser"}, resp)
            assert.NoError(t, err)
            assert.True(t, resp.ID > 0)
        })
        convey.Convey("用户名已存在", func() {
            mockUserRepo.EXPECT().FindByUsername(gomock.Any(), "existuser").Return(&user_entity.User{ID: 1}, nil)
            resp := &api.CreateUserResponse{}
            err := testMux.Do(ctx, &api.CreateUserRequest{Username: "existuser"}, resp)
            assert.Error(t, err)
        })
    })
}
```

**Step 3: Run tests → RED (fail)**

```bash
go test -v -run TestUserCreate ./internal/controller/user_ctr/...
```

**Step 4: Implement controller, service, repository to make tests pass**

**Step 5: Run tests → GREEN (pass)**

**Step 6: Refactor + lint**

```bash
go test -v -run TestUserCreate ./internal/controller/user_ctr/...
make lint
```

### Key Principles

- **Tests define behavior** — Write tests based on requirements before writing implementation
- **Mock external dependencies** — Use `go.uber.org/mock` for repository interfaces, making tests fast and isolated
- **Cover edge cases** — Each `Convey` block represents a scenario (success, validation error, not found, duplicate, etc.)
- **Small iterations** — Implement one feature at a time: write test → implement → pass → next feature

## Unit Testing

Test files are placed in the corresponding controller directory, one test file per module:

```
internal/controller/
  user_ctr/
    user.go
    user_test.go       # User module tests
  example_ctr/
    example.go
    example_test.go    # Example module tests
```

### Test Setup Pattern

Each test file has a `setupXxxTest` function that initializes dependencies, mock, and routes:

```go
func setupUserTest(t *testing.T) (context.Context, *mock_user_repo.MockUserRepo, *muxtest.TestMux) {
    testutils.Cache(t)
    mockCtrl := gomock.NewController(t)
    t.Cleanup(func() { mockCtrl.Finish() })

    ctx := context.Background()
    mockUserRepo := mock_user_repo.NewMockUserRepo(mockCtrl)
    user_repo.RegisterUser(mockUserRepo)

    // Register only the routes needed for this module
    testMux := muxtest.NewTestMux()
    r := testMux.Group("/api/v1")
    ctr := NewUser()
    r.Group("/").Bind(ctr.Create, ctr.List)

    return ctx, mockUserRepo, testMux
}
```

Key points:
- Use `broker.SetBroker(event_bus.NewEvBusBroker())` if broker is needed
- Register only the routes relevant to the module being tested

### Test Structure

Use GoConvey for BDD-style nested tests, one `TestXxx` per feature:

```go
func TestUserCreate(t *testing.T) {
    ctx, mockUserRepo, testMux := setupUserTest(t)

    convey.Convey("创建用户", t, func() {
        convey.Convey("创建成功", func() {
            mockUserRepo.EXPECT().FindByUsername(gomock.Any(), "newuser").Return(nil, nil)
            mockUserRepo.EXPECT().Create(gomock.Any(), gomock.Any()).Return(nil)
            resp := &api.CreateResponse{}
            err := testMux.Do(ctx, &api.CreateRequest{Username: "newuser"}, resp)
            assert.NoError(t, err)
        })
        convey.Convey("用户名已存在", func() {
            mockUserRepo.EXPECT().FindByUsername(gomock.Any(), "existuser").Return(&user_entity.User{
                ID: 1, Username: "existuser", Status: consts.ACTIVE,
            }, nil)
            resp := &api.CreateResponse{}
            err := testMux.Do(ctx, &api.CreateRequest{Username: "existuser"}, resp)
            assert.Error(t, err)
        })
    })
}
```

## Conventions

- Code comments and commit messages in Chinese
- Components panic on startup failure (fail-fast)
- Testing: GoConvey + testify + go.uber.org/mock + go-sqlmock + miniredis
- Linting: golangci-lint v2 (`make lint`)
- Mock generation: `//go:generate mockgen -source file.go -destination mock/file.go`

## References

- **Complete examples** (entry point, all layers, cron, queue, migration, unit testing, advanced patterns):
  See [references/examples.md](references/examples.md)
- **Components & configuration** (component system, config YAML, database, etcd, redis, cache, logger, trace, metrics, broker, gogo):
  See [references/components.md](references/components.md)
