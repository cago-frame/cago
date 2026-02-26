# Cago AI Skill

为 AI 编程助手提供 Cago 框架的专业知识，让 AI 在开发 Cago 项目时能够生成符合框架规范的代码。

## 包含内容

```
tools/skill/
├── SKILL.md                      # 核心知识：项目结构、API 定义、各层代码模式、框架约定
└── references/
    ├── examples.md               # 完整代码示例：入口文件、各层实现、定时任务、消息队列、数据库迁移
    └── components.md             # 组件与配置：组件系统、配置文件格式、数据库、Redis、缓存、日志、IAM 等
```

### SKILL.md

Skill 的主文件，包含 AI 需要的核心知识：

- **项目目录结构** — `cmd/`、`internal/api/`、`controller/`、`service/`、`repository/` 等各目录职责
- **API 定义模式** — 使用 `mux.Meta` 声明路由，请求/响应结构体的 tag 用法（`form`、`uri`、`header`、`binding`）
- **Handler 签名** — Controller 方法的四种合法签名格式
- **路由绑定** — `Router()` 函数、`Group()`、`Bind()`、中间件用法
- **Service 模式** — 接口 + 单例访问器 + 私有实现
- **Repository 模式** — 接口 + Register/访问器 + `db.Ctx(ctx)` 查询
- **数据库访问** — `db.Default()`、`db.Ctx(ctx)`、事务模式
- **错误码 i18n** — 错误码定义 + 多语言注册
- **框架约定** — 中文注释、测试工具、lint 规则、mock 生成

### references/examples.md

每一层的完整代码示例，包括：

- 应用入口（`main.go`）— 组件注册链
- API 定义 — 请求/响应结构体 + tag 参考
- Router — 路由注册、分组、中间件
- Controller — 三种 handler 签名的实际用法
- Service — 接口定义 + 业务逻辑实现
- Repository — CRUD 操作、分页查询、mock 生成
- Entity — GORM 实体定义
- 错误码 — 定义 + i18n 注册
- 定时任务 — Cron 注册 + handler
- 消息队列 — Message 定义、Publish/Subscribe、Handler
- 数据库迁移 — gormigrate 用法

### references/components.md

组件系统与配置的详细说明：

- Component 接口 — `Component`、`ComponentCancel`、`FuncComponent`
- 预置组件 — Core、Database、Redis、Cache、Broker、Mongo、Elasticsearch、Cron、HTTP
- 配置文件 — 完整的 YAML 配置示例（http、logger、db、redis、cache、broker、trace）
- 数据库 — 多数据库支持、事务模式、上下文感知查询
- Redis & Cache — 访问方式
- Logger — zap 日志、上下文日志
- OpenTelemetry — 自动埋点 + 手动 span
- Broker — 消息发布/订阅
- IAM — 认证、会话管理、审计中间件
- Goroutines — `gogo.Go` 安全协程

## 安装

### Claude Code

```bash
ln -s $(pwd)/tools/skill ~/.claude/skills/cago
```

安装后在 Claude Code 中使用 `/cago` 命令调用：

```
/cago 帮我新增一个用户管理的 CRUD 接口
/cago 添加一个每小时执行的定时任务
/cago 为 order 表创建数据库迁移
```

### 其他 AI 工具

将 `SKILL.md` 及 `references/` 目录下的文件作为上下文提供给 AI 即可。
