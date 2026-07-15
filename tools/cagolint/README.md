# cagolint

`cagolint` 是 Cago 框架的结构检查器。它基于 `go/analysis` 实现，并通过
golangci-lint v2 的 module plugin 机制集成到仓库的 lint harness 中。

## 文档导航

- [规则手册](./docs/rules.md)：每条规则的 Bad、Good、诊断和自动修复示例；
- [配置与集成](./docs/integration.md)：golangci-lint、Makefile、CI 和目录配置；
- [插件开发](./docs/development.md)：代码结构、测试方式和新增规则要求。

## 检查范围

第一版以 `examples/simple` 和 Cago 的实际路由绑定逻辑为兼容基准：

- `internal/api/<module>` 是业务 API 声明包，不能依赖 controller、service 或 repository；
- `internal/api/router.go` 是路由聚合入口，允许装配 controller 和 service middleware；
- controller 允许使用 `context.Context` 或 `*gin.Context`；
- handler 允许无返回值、仅返回 `error`，或返回 `(*Response, error)`；
- service 允许提供 Gin middleware；
- test、生成代码和 repository mock 不执行目录结构约束。

### 分层规则

| 规则 | 说明 |
| --- | --- |
| `CAGO2001` | 业务 API 包依赖 controller |
| `CAGO2002` | 业务 API 包依赖 service |
| `CAGO2003` | 业务 API 包依赖 repository |
| `CAGO2004` | repository 反向依赖 controller |
| `CAGO2005` | repository 反向依赖 service |
| `CAGO2006` | entity 依赖 controller、service 或 repository |
| `CAGO2009` | controller 直接依赖 Cago DB 或 GORM |

### API 规则

| 规则 | 说明 | 自动修复 |
| --- | --- | --- |
| `CAGO3001` | 路由 Request 必须是 struct | 否 |
| `CAGO3002` | 路由 Request 必须嵌入 Cago `mux.Meta` | 否 |
| `CAGO3003` | 包含 `mux.Meta` 的路由类型应以 `Request` 结尾 | 否 |
| `CAGO3004` | `XxxRequest` 应有对应的 `XxxResponse` | 生成空 Response |
| `CAGO3005` | `path` 不能为空、必须以 `/` 开头且格式有效 | 补 `/`、清理列表空格 |
| `CAGO3006` | `method` 必须是合法的 HTTP 方法 | 转为大写、清理列表空格 |
| `CAGO3007` | `:param` 与字段的 `uri:"param"` 必须对应 | 暂不自动修改字段 |

`method` 为空时遵循 Cago 的运行时语义，默认为 `GET`，不会强制补写。
`path` 和 `method` 均支持逗号分隔的多值声明。

### Controller、Service 和 Repository 规则

| 规则 | 说明 | 自动修复 |
| --- | --- | --- |
| `CAGO4001` | controller 目录和 package 使用 `_ctr` 后缀 | 否 |
| `CAGO4002` | handler 首参必须为 `context.Context` 或 `*gin.Context` | 否 |
| `CAGO4003` | handler 返回签名必须是框架支持的三种形式之一 | 否 |
| `CAGO4004` | handler 名应与 Request 去掉后缀后的名称一致 | 否 |
| `CAGO5001` | service 目录和 package 使用 `_svc` 后缀 | 否 |
| `CAGO5002` | service 使用 `XxxSvc` 接口和私有 `xxxSvc` 实现 | 否 |
| `CAGO5003` | 私有 service 实现必须完整实现接口 | 否 |
| `CAGO6001` | repository 目录和 package 使用 `_repo` 后缀 | 否 |
| `CAGO6003` | repository 数据库访问使用 `db.Ctx(ctx)` | 将明确的 `db.Default()` 改为 `db.Ctx(ctx)` |

完整代码示例见[规则手册](./docs/rules.md)。

## 仓库内使用

仓库使用 `.custom-gcl.yml` 将插件编译进固定版本的 golangci-lint：

```bash
make lint
```

该命令依次执行：

1. 运行 cagolint 单元测试；
2. 在需要时构建 `bin/golangci-lint-builder`；
3. 根据 `.custom-gcl.yml` 构建包含 cagolint 的 `bin/golangci-lint`；
4. 校验 `.golangci.yml`；
5. 执行全部 golangci-lint 检查。

应用安全修复：

```bash
make lint-fix
```

只检查 Cago 示例项目：

```bash
bin/golangci-lint run ./examples/simple/...
```

## 配置

插件配置位于 `.golangci.yml`：

```yaml
linters:
  enable:
    - cagolint
  settings:
    custom:
      cagolint:
        type: module
        settings:
          api-dir: /internal/api
          controller-dir: /internal/controller/
          service-dir: /internal/service/
          repository-dir: /internal/repository/
          entity-dir: /internal/model/entity/
```

配置使用完整目录片段，避免模块名中恰好包含 `api`、`service` 等单词时被误判。
未知配置字段会导致插件初始化失败，防止拼写错误被静默忽略。

`.custom-gcl.yml` 和 `.golangci.yml` 的职责区别、CI 构建过程以及下游项目接入方式见
[配置与集成](./docs/integration.md)。

## 开发与测试

```bash
cd tools/cagolint
go test ./...
```

测试使用 `analysistest`，同时验证诊断和 Suggested Fix。插件不提供独立 CLI，正式和本地检查统一通过
集成后的 golangci-lint 执行。新增规则时至少应包含：

- 一个合法样本；
- 一个触发诊断的非法样本；
- 如果规则支持修复，对应的 `.golden` 文件；
- 对 `examples/simple/...` 的兼容性验证。

目前跨 package 的 Router Bind 覆盖检查、全项目重复路由索引和跨文件脚手架生成尚未加入。
这些能力需要项目级索引，将作为后续版本实现，避免第一版通过不稳定的字符串匹配产生误报。
