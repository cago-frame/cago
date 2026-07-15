# Cago 文档导航

本文档按“使用框架、开发框架、AI 辅助”三个场景整理。代码与公开 API 是最终事实来源；文档与代码不一致时，应以当前代码和测试为准，并同步修正文档。

## 使用 Cago

- [项目首页与快速开始](../README.md)
- [简单示例](../examples/simple/README.md)
- [配置系统](../configs/README.md)
- [Cago 脚手架](../cmd/cago/README.md)
- [部署资源](../deploy/README.md)
- [中间件](../middleware/README.md)

常用组件的就近文档：

- [数据库](../database/db/README.md)
- [日志](../pkg/logger/README.md)
- [链路追踪](../pkg/opentelemetry/trace/README.md)
- [指标](../pkg/opentelemetry/metric/README.md)
- [IAM](../pkg/iam/README.md)
- [公共组件](../pkg/component/README.md)
- [工具函数](../pkg/utils/README.md)

## 开发与维护 Cago

仓库常用命令：

```bash
make test       # 运行主模块测试
make lint       # 测试 cagolint、校验配置并运行完整 lint
make lint-fix   # 应用 golangci-lint 和 cagolint 的安全修复
make cover      # 生成 coverage.out 并输出覆盖率
make install    # 安装本仓库的 cago 命令
```

结构检查器文档：

- [cagolint 总览](../tools/cagolint/README.md)
- [规则手册](../tools/cagolint/docs/rules.md)
- [配置与下游集成](../tools/cagolint/docs/integration.md)
- [插件开发](../tools/cagolint/docs/development.md)

`docs/README.md` 是仓库统一文档入口。`docs/` 下的其他内容仍用于 Swagger 生成物和本地设计记录，并由 `.gitignore` 忽略。已经实现的功能应在本入口、根 README、相关包 README 和 Skill 中维护当前用法。

## Tool 与 Skill

三者职责不同：

| 目录 | 面向对象 | 作用 | 是否参与构建/检查 |
| --- | --- | --- | --- |
| `cmd/cago/` | Cago 使用者 | 项目代码生成命令 | 是，可通过 `make install` 安装 |
| `tools/cagolint/` | Cago 项目与维护者 | 框架结构静态检查器 | 是，由 `make lint` 集成 |
| `tools/skill/` | AI 编程助手 | 提供框架约定、组件用法和代码示例 | 否，作为上下文文档使用 |

Skill 的入口是 [tools/skill/SKILL.md](../tools/skill/SKILL.md)，安装与内容索引见 [tools/skill/README.md](../tools/skill/README.md)。新增组件、修改公开接口、目录约定或生成规则时，应同时检查根 README、相关包 README、cagolint 规则文档和 Skill 是否需要更新。
