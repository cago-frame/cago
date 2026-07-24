# 配置与集成

## 两份配置文件的职责

cagolint 使用 golangci-lint v2 module plugin。构建配置和运行配置是两个独立阶段：

| 文件 | 职责 |
| --- | --- |
| `.custom-gcl.yml` | 构建包含 cagolint 的自定义 golangci-lint 二进制 |
| `.golangci.yml` | 启用 cagolint、传递插件设置并配置其他 linters |

`.custom-gcl.yml`：

```yaml
version: v2.12.2
name: golangci-lint
destination: ./bin

plugins:
  - module: github.com/cago-frame/cago/tools/cagolint
    path: ./tools/cagolint
```

它不会定义任何 lint 规则，只声明自定义二进制的组成。

`.golangci.yml`：

```yaml
version: "2"

linters:
  enable:
    - cagolint

  settings:
    custom:
      cagolint:
        type: module
        description: Checks Cago framework structure and conventions.
        settings:
          api-dir: /internal/api
          controller-dir: /internal/controller/
          service-dir: /internal/service/
          repository-dir: /internal/repository/
          entity-dir: /internal/model/entity/
```

## 构建流程

```text
官方 golangci-lint v2.12.2 builder
                 │
                 ├── 读取 .custom-gcl.yml
                 │
                 ├── 编译 tools/cagolint module
                 │
                 └── 生成 bin/golangci-lint
                              │
                              ├── 加载 .golangci.yml
                              └── 运行 cagolint 和其他 linters
```

`make lint` 会自动完成这条链路，不要求开发者全局安装特定版本的 golangci-lint。

## Makefile 入口

```bash
# 插件测试、示例兼容性、配置校验和全量 lint
make lint

# 应用 golangci-lint 和 cagolint 的安全 Suggested Fix
make lint-fix

# 只验证自定义 golangci-lint 配置
make lint-config

# 只测试插件并检查 examples/simple
make lint-plugin-test
```

构建产物位于 `bin/`：

```text
bin/golangci-lint-builder
bin/golangci-lint
```

`bin/` 已由仓库 `.gitignore` 忽略。

## CI

GitHub Actions 工作流位于：

```text
.github/workflows/lint.yml
```

CI 直接运行：

```bash
make lint
```

没有使用官方 `golangci-lint-action` 的预编译执行模式，因为官方二进制不包含本地 cagolint module。
CI 和开发机均根据同一份 `.custom-gcl.yml` 构建，因此插件版本与行为一致。

## 目录配置

默认值：

| 配置 | 默认值 |
| --- | --- |
| `api-dir` | `/internal/api` |
| `controller-dir` | `/internal/controller/` |
| `service-dir` | `/internal/service/` |
| `repository-dir` | `/internal/repository/` |
| `entity-dir` | `/internal/model/entity/` |

如果项目采用不同布局，可以修改设置：

```yaml
settings:
  api-dir: /app/api
  controller-dir: /app/controller/
  service-dir: /app/service/
  repository-dir: /app/repository/
  entity-dir: /app/domain/entity/
```

目录值使用 import path 片段，而不是本机文件系统绝对路径。前后 `/` 用于避免模块名中恰好包含
`api`、`service` 等单词时被误判。

未知配置字段会导致插件初始化失败。例如：

```yaml
settings:
  api-directory: /internal/api
```

不会被静默忽略，而会报告配置解码错误。

## 运行指定项目

cagolint 不提供独立 CLI，所有检查统一由包含插件的 golangci-lint 执行：

```bash
bin/golangci-lint run ./examples/simple/...
```

这样可以确保 cagolint 与其他 linter 使用相同的 package loading、排除规则、`nolint` 处理和修复流程。

## 下游项目接入

当前 cagolint 源码位于 Cago 仓库的嵌套 module 中，仓库内通过本地 `path` 集成。若要让其他项目直接
集成，建议后续将其发布为独立 module/tag，例如：

```yaml
plugins:
  - module: github.com/cago-frame/cago/tools/cagolint
    version: v0.1.0
```

在正式发布版本之前，下游项目可以使用本地路径或固定 Git revision，但不建议使用浮动分支作为 CI
依赖，以免 lint 结果不可复现。
