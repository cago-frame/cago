# cagolint 规则手册

本文档描述当前版本已经实现的规则。每条规则包含错误示例、正确示例、诊断形式以及是否支持
`golangci-lint run --fix`。示例中的 module path 使用 `example.com/project`，实际项目无需采用该名称。

## 规则总览

| 规则 | 类型 | 自动修复 |
| --- | --- | --- |
| `CAGO2001` | 业务 API 依赖 controller | 否 |
| `CAGO2002` | 业务 API 依赖 service | 否 |
| `CAGO2003` | 业务 API 依赖 repository | 否 |
| `CAGO2004` | repository 依赖 controller | 否 |
| `CAGO2005` | repository 依赖 service | 否 |
| `CAGO2006` | entity 依赖上层包 | 否 |
| `CAGO2009` | controller 直接访问数据库 | 否 |
| `CAGO3001` | Request 不是 struct | 否 |
| `CAGO3002` | Request 缺少 Cago `mux.Meta` | 否 |
| `CAGO3003` | 路由类型未以 `Request` 结尾 | 否 |
| `CAGO3004` | Request 缺少对应 Response | 是 |
| `CAGO3005` | route path 非法 | 是，限确定性格式修复 |
| `CAGO3006` | HTTP method 非法或不规范 | 是，限大小写和空格 |
| `CAGO3007` | path 参数与 `uri` tag 不对应 | 否 |
| `CAGO4001` | controller 包命名不符合约定 | 否 |
| `CAGO4002` | handler context 参数非法 | 否 |
| `CAGO4003` | handler 返回签名非法 | 否 |
| `CAGO4004` | handler 与 Request 命名不对应 | 否 |
| `CAGO5001` | service 包命名不符合约定 | 否 |
| `CAGO5002` | service 缺少标准接口或私有实现 | 否 |
| `CAGO5003` | service 实现未完整实现接口 | 否 |
| `CAGO6001` | repository 包命名不符合约定 | 否 |
| `CAGO6003` | repository 使用 `db.Default()` | 是 |
| `CAGO7001` | migration 使用 GORM `AutoMigrate` | 否 |

## 分层依赖

### CAGO2001—CAGO2003：业务 API 包依赖上层

`internal/api/<module>` 只负责 Request、Response、校验和路由元数据，不应依赖 controller、service
或 repository。

Bad：

```go
// internal/api/user/user.go
package user

import "example.com/project/internal/service/user_svc"

type CreateRequest struct {
    Service user_svc.UserSvc
}
```

诊断：

```text
CAGO2002: 业务 API 包不能依赖 service
```

Good：

```go
package user

import "github.com/cago-frame/cago/server/mux"

type CreateRequest struct {
    mux.Meta `path:"/users" method:"POST"`
    Username string `form:"username" binding:"required"`
}

type CreateResponse struct {
    ID int64 `json:"id"`
}
```

路由聚合包是明确例外。下面的代码合法，因为 `internal/api/router.go` 的职责就是装配 Controller 和
middleware：

```go
// internal/api/router.go
package api

import (
    "example.com/project/internal/controller/user_ctr"
    "example.com/project/internal/service/user_svc"
)
```

### CAGO2004—CAGO2005：Repository 反向依赖上层

Bad：

```go
package user_repo

import "example.com/project/internal/service/user_svc"
```

Good：

```go
package user_repo

import (
    "github.com/cago-frame/cago/database/db"
    "example.com/project/internal/model/entity/user_entity"
)
```

Repository 可以依赖 entity、Cago database、公共组件和外部存储 SDK，但不能调用 controller 或
service。架构依赖错误无法可靠自动重构，因此不提供自动修复。

### CAGO2006：Entity 依赖上层

Bad：

```go
package user_entity

import "example.com/project/internal/repository/user_repo"
```

Good：

```go
package user_entity

import (
    "context"

    "github.com/cago-frame/cago/pkg/consts"
    "example.com/project/internal/pkg/code"
)

type User struct {
    ID     int64
    Status int
}

func (u *User) Check(ctx context.Context) error {
    // Entity 可以包含自身状态校验等领域逻辑。
    return nil
}
```

### CAGO2009：Controller 直接访问数据库

Bad：

```go
package user_ctr

import "github.com/cago-frame/cago/database/db"

func (u *User) Get(
    ctx context.Context,
    req *api.GetRequest,
) (*api.GetResponse, error) {
    var user user_entity.User
    if err := db.Ctx(ctx).First(&user, req.ID).Error; err != nil {
        return nil, err
    }
    return &api.GetResponse{ID: user.ID}, nil
}
```

Good：

```go
func (u *User) Get(
    ctx context.Context,
    req *api.GetRequest,
) (*api.GetResponse, error) {
    return user_svc.User().Get(ctx, req)
}
```

数据库访问应放在 Repository。插件会检查 Controller 对 Cago `database/db` 和 `gorm.io/*` 的直接依赖。

## API 路由声明

### CAGO3001：Request 必须是 struct

Bad：

```go
type GetUserRequest string
```

Good：

```go
type GetUserRequest struct {
    mux.Meta `path:"/users/:id" method:"GET"`
    ID       int64 `uri:"id"`
}
```

### CAGO3002：Request 必须嵌入真实的 mux.Meta

Bad：

```go
type CreateRequest struct {
    Username string `form:"username"`
}
```

下面的同名自定义类型也不合法：

```go
type Meta struct{}

type CreateRequest struct {
    Meta
}
```

Good：

```go
import "github.com/cago-frame/cago/server/mux"

type CreateRequest struct {
    mux.Meta `path:"/users" method:"POST"`
    Username string `form:"username"`
}
```

插件使用 Go 类型信息确认字段确实是 `github.com/cago-frame/cago/server/mux.Meta`，不是简单检查字段名。

### CAGO3003：路由类型命名

Bad：

```go
type CreateUser struct {
    mux.Meta `path:"/users" method:"POST"`
}
```

Good：

```go
type CreateUserRequest struct {
    mux.Meta `path:"/users" method:"POST"`
}
```

该规则只检查包含真实 `mux.Meta` 的路由类型，不会因为 protobuf 中存在 `GetUserRequest` 而误判。

### CAGO3004：Request 缺少对应 Response

Bad：

```go
type CreateRequest struct {
    mux.Meta `path:"/users" method:"POST"`
}
```

诊断：

```text
CAGO3004: CreateRequest 缺少对应的 CreateResponse
```

运行：

```bash
make lint-fix
```

自动修复后：

```go
type CreateRequest struct {
    mux.Meta `path:"/users" method:"POST"`
}

type CreateResponse struct {
}
```

插件只生成空结构，不猜测业务响应字段。

### CAGO3005：Route path 格式

Bad：

```go
mux.Meta `path:"users" method:"POST"`
```

诊断：

```text
CAGO3005: path "users" 必须以 / 开头
```

自动修复后：

```go
mux.Meta `path:"/users" method:"POST"`
```

多路径声明也会清理空格：

```go
// Bad
mux.Meta `path:"/users, /members" method:"GET"`

// Good / 自动修复结果
mux.Meta `path:"/users,/members" method:"GET"`
```

以下情况会报告，但不会猜测业务路径：

```go
mux.Meta `path:"" method:"GET"`
mux.Meta `path:"/users?id=1" method:"GET"`
mux.Meta `path:"/users//current" method:"GET"`
```

### CAGO3006：HTTP method

Bad：

```go
mux.Meta `path:"/users" method:"post"`
```

自动修复后：

```go
mux.Meta `path:"/users" method:"POST"`
```

多 method：

```go
// Bad
mux.Meta `path:"/users" method:"GET, POST"`

// Good / 自动修复结果
mux.Meta `path:"/users" method:"GET,POST"`
```

支持的标准方法：

```text
GET POST PUT PATCH DELETE HEAD OPTIONS CONNECT TRACE
```

非法方法只报告，不猜测修复：

```go
mux.Meta `path:"/users" method:"FETCH"`
```

不写 method 是合法的，Cago runtime 会使用默认 GET：

```go
mux.Meta `path:"/users"`
```

### CAGO3007：Path 参数与 uri tag 对应

Bad：

```go
type GetUserRequest struct {
    mux.Meta `path:"/users/:id" method:"GET"`
    ID       int64 `form:"id"`
}
```

诊断：

```text
CAGO3007: path 参数 :id 缺少对应的 uri:"id" 字段
```

Good：

```go
type GetUserRequest struct {
    mux.Meta `path:"/users/:id" method:"GET"`
    ID       int64 `uri:"id"`
}
```

反向多余声明也会报告：

```go
type ListUsersRequest struct {
    mux.Meta `path:"/users" method:"GET"`
    ID       int64 `uri:"id"`
}
```

```text
CAGO3007: uri:"id" 未出现在任何 route path 中
```

## Controller

### CAGO4001：Controller 包命名

标准结构：

```text
internal/controller/user_ctr/user.go
```

```go
package user_ctr
```

目录和 package 都应使用 `_ctr` 后缀。目录移动和 package rename 涉及跨文件引用，不提供自动修复。

### CAGO4002：Handler Context 参数

Bad：

```go
func (u *User) Get(
    ctx string,
    req *api.GetRequest,
) (*api.GetResponse, error) {
    return nil, nil
}
```

Good，普通业务接口：

```go
func (u *User) Get(
    ctx context.Context,
    req *api.GetRequest,
) (*api.GetResponse, error) {
    return user_svc.User().Get(ctx, req)
}
```

Good，需要 Cookie、Session、Header 或直接 HTTP 操作：

```go
func (u *User) Login(
    ctx *gin.Context,
    req *api.LoginRequest,
) error {
    return user_svc.User().Login(ctx, req)
}
```

`*gin.Context` 是 Cago 正式支持的 handler 参数，不是违规行为。

### CAGO4003：Handler 返回签名

Cago 支持三类返回形式，并且 Context 可以是 `context.Context` 或 `*gin.Context`。

Good，无返回值：

```go
func (u *User) Raw(ctx *gin.Context, req *api.RawRequest) {
    ctx.String(http.StatusOK, "ok")
}
```

Good，仅返回 error：

```go
func (u *User) Login(ctx *gin.Context, req *api.LoginRequest) error {
    return user_svc.User().Login(ctx, req)
}
```

Good，返回 Response 和 error：

```go
func (u *User) Get(
    ctx context.Context,
    req *api.GetRequest,
) (*api.GetResponse, error) {
    return user_svc.User().Get(ctx, req)
}
```

Bad：

```go
func (u *User) Get(ctx context.Context, req *api.GetRequest) string {
    return "invalid"
}
```

Bad：

```go
func (u *User) Get(
    ctx context.Context,
    req *api.GetRequest,
) (error, *api.GetResponse) {
    return nil, nil
}
```

### CAGO4004：Handler 名与 Request 对应

Bad：

```go
func (u *User) HandleCreate(
    ctx context.Context,
    req *api.CreateRequest,
) (*api.CreateResponse, error) {
    return user_svc.User().Create(ctx, req)
}
```

Good：

```go
func (u *User) Create(
    ctx context.Context,
    req *api.CreateRequest,
) (*api.CreateResponse, error) {
    return user_svc.User().Create(ctx, req)
}
```

这是 Cago 工程和脚手架约定；mux runtime 本身不依赖方法名完成绑定。

## Service

### CAGO5001：Service 包命名

标准结构：

```text
internal/service/user_svc/user.go
```

```go
package user_svc
```

### CAGO5002—CAGO5003：Service 接口和实现

Bad：

```go
type UserService struct{}

func NewUserService() *UserService {
    return &UserService{}
}
```

Good：

```go
type UserSvc interface {
    Create(
        ctx context.Context,
        req *api.CreateRequest,
    ) (*api.CreateResponse, error)
}

type userSvc struct{}

var defaultUser = &userSvc{}

func User() UserSvc {
    return defaultUser
}

func (u *userSvc) Create(
    ctx context.Context,
    req *api.CreateRequest,
) (*api.CreateResponse, error) {
    return &api.CreateResponse{}, nil
}
```

接口声明了方法但私有实现缺失时：

```text
CAGO5003: *userSvc 未完整实现 UserSvc
```

Service 可以合法提供 Gin middleware：

```go
func (u *userSvc) Middleware(force bool) gin.HandlerFunc {
    return authn.Default().Middleware(force, callback)
}
```

## Repository

### CAGO6001：Repository 包命名

标准结构：

```text
internal/repository/user_repo/user.go
```

```go
package user_repo
```

`internal/repository/user_repo/mock` 是生成 Mock 的子包，不需要使用 `_repo` 后缀。

### CAGO6003：使用 db.Ctx(ctx)

Bad：

```go
func (u *userRepo) Find(
    ctx context.Context,
    id int64,
) (*user_entity.User, error) {
    var user user_entity.User
    if err := db.Default().First(&user, id).Error; err != nil {
        return nil, err
    }
    return &user, nil
}
```

自动修复后：

```go
func (u *userRepo) Find(
    ctx context.Context,
    id int64,
) (*user_entity.User, error) {
    var user user_entity.User
    if err := db.Ctx(ctx).First(&user, id).Error; err != nil {
        return nil, err
    }
    return &user, nil
}
```

`db.Ctx(ctx)` 会保留 request context，并配合 `db.WithContextDB(ctx, tx)` 传播事务。只有当前函数存在唯一、
明确的 Context 参数时，插件才提供自动修复。

## Migration

### CAGO7001：禁止使用 AutoMigrate

`migrations` 目录中的迁移必须是确定性的。`AutoMigrate` 会读取当前版本的 entity 定义；当 entity 随
业务演进增加或修改字段后，重新执行一条历史 migration 可能产生与当时不同的数据库结构，甚至与后续
migration 冲突。

Bad：

```go
package migrations

func createUsers(tx *gorm.DB) error {
    return tx.AutoMigrate(&entity.User{})
}
```

诊断：

```text
CAGO7001: migration 不能使用 AutoMigrate，请使用确定性的 DDL 语句或 Migrator 具体方法
```

Good：

```go
package migrations

func createUsers(tx *gorm.DB) error {
    return tx.Exec(`CREATE TABLE users (
        id BIGINT PRIMARY KEY,
        username VARCHAR(64) NOT NULL
    )`).Error
}
```

也可以使用语义明确、结果固定的 `tx.Migrator().CreateTable`、`AddColumn` 等具体方法，但不要让历史
migration 依赖会持续变化的当前 entity。该规则通过类型信息确认调用目标确实是
`gorm.io/gorm.(*DB).AutoMigrate`，其他类型上的同名方法不会误报。由于无法唯一推导目标 DDL，本规则
不提供自动修复。

## 合法但容易误解的写法

以下写法不会被 cagolint 报告：

```go
// Router 聚合层可以依赖 Controller 和 Service middleware。
package api

import (
    "example.com/project/internal/controller/user_ctr"
    "example.com/project/internal/service/user_svc"
)
```

```go
// 需要直接操作 HTTP 时允许使用 *gin.Context。
func (u *User) Login(ctx *gin.Context, req *api.LoginRequest) error
```

```go
// Cago runtime 支持无返回值 handler。
func (u *User) Raw(ctx *gin.Context, req *api.RawRequest)
```

```go
// Service 可以提供 middleware。
func (u *userSvc) Middleware(force bool) gin.HandlerFunc
```

```go
// method 为空时默认为 GET。
mux.Meta `path:"/users"`
```

此外：

- `_test.go` 可以进行测试装配和跨层依赖；
- `repository/mock` 不执行 `_repo` package 约束；
- 标准 Go generated 文件会跳过；
- protobuf 等不在 `internal/api/<module>` 中的 Request 不会作为 Cago HTTP Request 检查；
- migration 可以接收和使用显式的 `*gorm.DB`，但不能调用其 `AutoMigrate` 方法。

## 局部忽略

cagolint 作为 golangci-lint 插件运行时，可以使用标准 `nolint` 指令：

```go
//nolint:cagolint // 历史接口，待 issue #123 完成后迁移
```

建议始终填写原因。当前 cagolint 的多个规则由同一个 Analyzer 提供，因此 `nolint` 的粒度是整个
`cagolint`，不能只关闭单个 `CAGOxxxx`。需要长期例外时，优先通过目录配置或代码结构调整解决。

## 运行和修复

检查整个仓库：

```bash
make lint
```

应用安全修复：

```bash
make lint-fix
```

只检查示例项目：

```bash
bin/golangci-lint run ./examples/simple/...
```

当前 `examples/simple/...` 是合法基准项目，预期输出：

```text
0 issues.
```
