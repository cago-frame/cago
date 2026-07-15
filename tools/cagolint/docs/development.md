# 插件开发

## 代码结构

```text
tools/cagolint/
├── go.mod
├── plugin.go                 # golangci-lint module plugin 注册
├── analyzer/
│   ├── analyzer.go           # 规则和 Suggested Fix
│   ├── settings.go           # 插件配置
│   ├── analyzer_test.go      # analysistest 入口
│   └── testdata/             # bad case 和 golden fix
```

## Analyzer

入口：

```go
func New(settings Settings) *analysis.Analyzer
```

插件使用 `typesinfo` load mode，因此规则可以使用：

- AST；
- `go/types`；
- import path；
- interface implementation；
- `analysis.Diagnostic`；
- `analysis.SuggestedFix`。

类型判断应优先使用 `go/types`。例如识别 `mux.Meta` 时，应确认其 package path 和类型名，而不是只
检查源码中是否出现字符串 `mux.Meta`。

## 包角色识别

当前 Analyzer 将 package 分为：

```text
API Router
Business API
Controller
Service
Repository
Entity
Other
```

测试 main、repository mock 和标准 generated file 会被排除。新增目录角色时，需要同时考虑：

- 正常 package；
- external test package；
- Go 测试生成的 `.test` main；
- mockgen 输出；
- protobuf 输出；
- migration 和 task 等合法特殊层。

## 测试

运行：

```bash
cd tools/cagolint
go test ./...
go vet ./...
```

测试使用 `golang.org/x/tools/go/analysis/analysistest`。

诊断样本通过 `// want` 声明：

```go
type GetUserRequest struct { // want "CAGO3004"
    mux.Meta `path:"users" method:"post"` // want "CAGO3005" "CAGO3006"
}
```

支持自动修复的规则必须使用：

```go
analysistest.RunWithSuggestedFixes(...)
```

并提供 `.golden` 文件：

```text
api.go
api.go.golden
```

`api.go.golden` 表示同时应用该文件全部 Suggested Fix 后的预期内容。

## 新增规则要求

新增规则至少需要：

1. 稳定的 `CAGOxxxx` 规则 ID；
2. 一段清晰、可执行的诊断信息；
3. 一个不会触发规则的 Good case；
4. 一个触发规则的 Bad case；
5. 如果支持修复，提供 golden test；
6. 在 `docs/rules.md` 中补充 Bad、Good、诊断和修复示例；
7. 运行自定义 `bin/golangci-lint run ./examples/simple/...` 验证兼容性和插件集成。

## 自动修复原则

只为结果唯一、不改变业务语义的修改提供 Suggested Fix。

适合自动修复：

- HTTP method 规范化为大写；
- route path 补前导 `/`；
- 清理逗号列表空格；
- 生成空 Response；
- 在明确 context 参数存在时，将 `db.Default()` 改为 `db.Ctx(ctx)`。

不适合自动修复：

- 移动目录；
- 重命名导出类型；
- 修改 Controller 返回语义；
- 猜测非法 HTTP method 的真实意图；
- 自动拆分跨层业务逻辑；
- 覆盖已有函数体；
- 在无法唯一匹配字段时补 `uri` tag。

同一段源码不能由多个 Suggested Fix 产生重叠 TextEdit。需要同时规范化同一个 struct tag 的 path 和
method 时，应生成一个合并后的编辑，避免 golangci-lint 或 analysistest 报告修复冲突。

## 真实项目兼容性

`examples/simple` 是第一版的合法基准。以下写法必须保持无误报：

- `internal/api/router.go` 导入 controller 和 service；
- Controller 使用 `*gin.Context`；
- Handler 无返回值；
- Handler 只返回 `error`；
- Service 返回 `gin.HandlerFunc`；
- Repository 实现 Cago IAM interface；
- repository mock 使用 `mock_user_repo` package；
- protobuf 生成文件包含 `GetUserRequest`；
- migration 使用显式 `*gorm.DB`。

验证：

```bash
make lint-plugin-test
bin/golangci-lint run ./examples/simple/...
```

预期：

```text
0 issues.
```

## 后续能力

以下规则需要项目级索引，目前尚未实现：

- 跨 package 的重复 method/path；
- Request → Controller → Service → Router Bind 完整映射；
- 未绑定的 handler；
- Controller 与 Service 的跨包签名一致性；
- 跨文件脚手架修复。

实现这些能力时应使用可靠的 package graph 或 analysis facts，不能仅通过全仓库字符串搜索建立关系。
