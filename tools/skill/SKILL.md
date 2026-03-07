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

Entities contain data fields and business logic methods (validation, status checks), avoiding piling all logic into Service.

```go
type User struct {
    ID             int64  `gorm:"column:id;type:bigint(20);not null;primary_key"`
    Username       string `gorm:"column:username;type:varchar(255);index:username,unique;not null"`
    HashedPassword string `gorm:"column:hashed_password;type:varchar(255);not null"`
    Status         int    `gorm:"column:status;type:int(11);not null"`
    Createtime     int64  `gorm:"column:createtime;type:bigint(20)"`
    Updatetime     int64  `gorm:"column:updatetime;type:bigint(20)"`
}

func (u *User) Check(ctx context.Context) error {
    if u == nil { return i18n.NewError(ctx, code.UserNotFound) }
    if u.Status != consts.ACTIVE { return i18n.NewError(ctx, code.UserIsBanned) }
    return nil
}
```

Suitable for Entity: existence checks, status validation, field formatting, simple business rules.
Not suitable: cross-entity coordination, dependencies on external services.

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
return i18n.NewForbiddenError(ctx, code.UserNotPermission)             // 403
```

### Goroutines

Always use `gogo.Go(ctx, func(ctx context.Context) error { ... })` for async work.

### Configuration Source

Default: read from `configs/config.yaml`. Set `source: etcd` to use etcd as config center.

When using etcd, the config file still needs basic etcd connection info. The framework reads etcd config from the file first, then switches to etcd as the source. etcd key prefix rule: `{etcd.prefix}/{env}/{appName}`.

```yaml
# configs/config.yaml (etcd config source)
source: etcd
env: dev
etcd:
  endpoints:
    - 127.0.0.1:2379
  prefix: /config     # Final key: /config/dev/appname/db, /config/dev/appname/redis, etc.
```

Each top-level config key (db, redis, cache, etc.) is stored as a separate etcd key. If a key doesn't exist in etcd, it's auto-initialized with the default value and returns an error on first run — set the value in etcd and restart.

Config API:

```go
cfg.Scan(ctx, "db", &dbConfig)         // Scan into struct
cfg.String(ctx, "http.address")        // Dot notation, returns string
cfg.Bool(ctx, "debug")                 // Returns bool
cfg.Has(ctx, "key")                    // Check existence
cfg.Watch(ctx, "key", callback)        // Watch changes
cfg.Env                                // "dev", "test", "pre", "prod"
cfg.Debug                              // bool
```

## BDD/TDD Development Workflow (Recommended)

### When to Use BDD/TDD

**Ask the user first** — Before starting implementation, ask the user whether they want to use BDD/TDD workflow. It's the recommended approach but not mandatory.

**Well suited for:**
- New API endpoints (Controller/Service layer) with clear behavior expectations
- Business logic with multiple scenarios (success, error, edge cases)
- Features involving authentication, authorization, state transitions
- **Bug fixes** — prioritize writing a test that reproduces the bug first, then fix it (test-first ensures the bug won't regress)
- Any feature where requirements can be expressed as "Given...When...Then..."

**Less suited for (suggest other test approaches):**
- Pure utility functions — write regular unit tests after implementation
- Infrastructure/config changes — manual verification may suffice
- Database migrations — test via migration tool
- Simple CRUD with no special logic — basic integration test after implementation

### Bug Fix Workflow (Test-First)

For bug fixes, prioritize reproducing the bug with a test before writing the fix:

1. **Analyze the bug** — Understand the root cause and affected code path
2. **Write a failing test** — Create a test case that reproduces the exact bug scenario
3. **Run test → confirm it fails** — Verify the test correctly captures the bug
4. **Fix the bug** — Modify the minimum code needed to fix the issue
5. **Run test → confirm it passes** — The previously failing test should now pass
6. **Run full test suite** — Ensure no regressions: `go test -v ./internal/controller/...`

### BDD/TDD Steps

1. **Write API definition** — Define request/response structs in `internal/api/`
2. **Design test scenarios** — From requirements, derive Convey nesting structure:
   - Top-level `Convey`: feature name (e.g., "登录")
   - Nested `Convey`: each scenario (e.g., "登录成功", "用户名不存在", "密码错误")
   - Deeply nested `Convey`: sequential behavior (e.g., "退出后再访问接口失败")
3. **Write tests first** — Create test file, implement setup function and test cases
4. **Run tests → verify they fail** — `go test -v -run TestXxx ./internal/controller/xxx_ctr/...`
5. **Implement code** — Write controller → service → repository to make tests pass
6. **Run tests → verify they pass** — All test cases should be green
7. **Refactor** — Clean up code while keeping tests passing, then run `make lint`

### Key Principles

- **Tests define behavior** — Write tests based on requirements before writing implementation
- **Mock external dependencies** — Use `go.uber.org/mock` for repository interfaces
- **Cover edge cases** — Each `Convey` block represents a scenario (success, validation error, not found, etc.)
- **Small iterations** — One feature at a time: write test → implement → pass → next feature
- **Helper functions** — Extract repeated test operations (e.g., login) into `t.Helper()` functions for reuse across tests

## Unit Testing

### Test Setup Pattern

Each test file has a `setupXxxTest` function that initializes mock dependencies and registers routes:

```go
func setupUserTest(t *testing.T) (context.Context, *mock_user_repo.MockUserRepo, *muxtest.TestMux) {
    testutils.Cache(t)
    mockCtrl := gomock.NewController(t)
    t.Cleanup(func() { mockCtrl.Finish() })

    ctx := context.Background()
    mockUserRepo := mock_user_repo.NewMockUserRepo(mockCtrl)
    user_repo.RegisterUser(mockUserRepo)

    // Initialize other components as needed (IAM, broker, etc.)
    // e.g. iam.SetDefault(iam.New(user_repo.User()))
    // e.g. broker.SetBroker(event_bus.NewEvBusBroker())

    testMux := muxtest.NewTestMux()
    r := testMux.Group("/api/v1")
    ctr := NewUser()
    r.Group("/").Bind(ctr.Create, ctr.List)
    // Register middleware routes as needed
    // e.g. r.Group("/", user_svc.User().Middleware(true)).Bind(ctr.CurrentUser)

    return ctx, mockUserRepo, testMux
}
```

### TestMux Usage

`muxtest.TestMux` embeds `muxclient.Client` — use `Do(ctx, req, resp)` to test handlers:

```go
resp := &api.CreateResponse{}
err := testMux.Do(ctx, &api.CreateRequest{Username: "newuser"}, resp)
assert.NoError(t, err)
```

Options: `muxclient.WithHeader(h)` (set headers), `muxclient.WithPath(p)` (override path), `muxclient.WithResponse(&httpResp)` (capture raw response).

### Test Utilities

```go
testutils.Cache(t)                           // In-memory cache (memory.NewMemoryCache)
testutils.Redis(t)                           // Miniredis
testutils.IAM(t, database, opts...)          // IAM
broker.SetBroker(event_bus.NewEvBusBroker()) // In-memory broker
```

### Helper Functions

Extract repeated test operations into helper functions with `t.Helper()`, so failures report the caller's line:

```go
func createUser(t *testing.T, ctx context.Context, testMux *muxtest.TestMux, mockUserRepo *mock_user_repo.MockUserRepo) *api.CreateResponse {
    t.Helper()
    mockUserRepo.EXPECT().FindByUsername(gomock.Any(), "newuser").Return(nil, nil)
    mockUserRepo.EXPECT().Create(gomock.Any(), gomock.Any()).Return(nil)
    resp := &api.CreateResponse{}
    err := testMux.Do(ctx, &api.CreateRequest{Username: "newuser", Password: "password123"}, resp)
    assert.NoError(t, err)
    return resp
}
```

### Test Structure

Use GoConvey for BDD-style nested tests. Nested `Convey` blocks can express sequential behavior:

```go
func TestUserCreate(t *testing.T) {
    ctx, mockUserRepo, testMux := setupUserTest(t)
    convey.Convey("创建用户", t, func() {
        convey.Convey("创建成功", func() {
            mockUserRepo.EXPECT().FindByUsername(gomock.Any(), "newuser").Return(nil, nil)
            mockUserRepo.EXPECT().Create(gomock.Any(), gomock.Any()).Return(nil)
            resp := &api.CreateResponse{}
            err := testMux.Do(ctx, &api.CreateRequest{Username: "newuser", Password: "password123"}, resp)
            assert.NoError(t, err)
        })
        convey.Convey("用户名已存在", func() {
            mockUserRepo.EXPECT().FindByUsername(gomock.Any(), "existuser").Return(&user_entity.User{
                ID: 1, Username: "existuser", Status: consts.ACTIVE,
            }, nil)
            err := testMux.Do(ctx, &api.CreateRequest{Username: "existuser"}, &api.CreateResponse{})
            assert.Error(t, err)
        })
    })
}

// Deep nesting for sequential behavior verification
func TestUserDelete(t *testing.T) {
    ctx, mockUserRepo, testMux := setupUserTest(t)
    convey.Convey("删除用户", t, func() {
        convey.Convey("删除成功", func() {
            mockUserRepo.EXPECT().Find(gomock.Any(), int64(1)).Return(&user_entity.User{ID: 1, Status: consts.ACTIVE}, nil)
            mockUserRepo.EXPECT().Delete(gomock.Any(), int64(1)).Return(nil)
            err := testMux.Do(ctx, &api.DeleteRequest{ID: 1}, &api.DeleteResponse{})
            assert.NoError(t, err)

            convey.Convey("删除后查询不到", func() {
                mockUserRepo.EXPECT().Find(gomock.Any(), int64(1)).Return(nil, nil)
                err := testMux.Do(ctx, &api.FindRequest{ID: 1}, &api.FindResponse{})
                assert.Error(t, err)
            })
        })
    })
}
```

For complete test examples with IAM authentication, middleware, and token refresh, see [references/examples.md](references/examples.md).

## Conventions

- Code comments and commit messages in Chinese
- Components panic on startup failure (fail-fast)
- Testing: GoConvey + testify + go.uber.org/mock + go-sqlmock + miniredis
- Linting: golangci-lint v2 (`make lint`)
- Mock generation: `//go:generate mockgen -source file.go -destination mock/file.go`
- **Common constants**: Use `github.com/cago-frame/cago/pkg/consts` for universal constants:

```go
import "github.com/cago-frame/cago/pkg/consts"
// Status: consts.UNKNOWN(0), consts.ACTIVE(1), consts.DELETE(2), consts.AUDIT(3), consts.BAN(4)
// Boolean: consts.YES(1), consts.NO(2)
```

## References

- **Complete examples** (entry point, all layers, cron, queue, migration, unit testing, advanced patterns):
  See [references/examples.md](references/examples.md)
- **Components & configuration** (component system, config YAML, database, etcd, redis, cache, logger, trace, metrics, broker, gRPC, gogo):
  See [references/components.md](references/components.md)
