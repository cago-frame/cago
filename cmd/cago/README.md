# Cago 脚手架

Cago 提供代码生成命令，用于生成 Controller、Service、Swagger 文档以及数据库相关代码。

## 安装

```bash
go install github.com/cago-frame/cago/cmd/cago@latest
```

## 使用

可以使用 `cago -h` 或 `cago <command> -h` 查看命令帮助。

当前命令概览：

| 命令 | 状态 | 说明 |
| --- | --- | --- |
| `cago gen` | 可用 | 扫描 API 和 Service，生成代码及 Swagger 文档 |
| `cago gen gorm <table>` | 可用 | 根据已配置的数据库表生成 GORM Entity 和 Repository |
| `cago gen mongo <table>` | 可用 | 生成 MongoDB Entity 和 Repository |
| `cago init <name>` | 预留 | 命令已注册，但当前尚未实现项目初始化逻辑 |

### API

在`internal/api`目录下,定义好 api 请求结构,使用下面命令和自动生成`controller`代码和`swagger`文档

在`internal/service`目录下,定义好 service 接口,使用下面命令和自动生成`service`代码

```bash
cago gen
```

默认扫描 `./internal/api`。扫描其他目录时使用：

```bash
cago gen --dir ./path/to/api
```

默认使用 Cago 内置的 Swagger 文档生成器，并且不会在 controller 中生成 Swagger 注释。如需生成兼容 Swaggo 的接口注释，并使用 Swaggo 生成 `docs/docs.go`、`docs/swagger.json` 和 `docs/swagger.yaml`，可使用：

```bash
cago gen --swagger
```

也可以使用短参数：`cago gen -s`。对已有 Controller 再次执行时，会补充缺失的 Swagger 注释，不会重复添加。启用该参数后不会再执行 Cago 内置的 Swagger 文档生成器。

需要查看逐文件扫描、方法生成以及 Swaggo 的详细日志时，可增加 `--verbose`（或 `-v`）。默认输出只保留生成阶段、创建结果和最终状态，便于在本地及 CI 中阅读。

`cago gen` 参数汇总：

| 参数 | 短参数 | 默认值 | 说明 |
| --- | --- | --- | --- |
| `--dir` | `-d` | `./internal/api` | API 目录 |
| `--swagger` | `-s` | `false` | 生成 Swaggo 注释和 `docs/` 下的文档文件 |
| `--verbose` | `-v` | `false` | 输出详细生成日志 |

### 数据库模型

#### GORM

定义好表结构和`configs`文件后,使用下面命令和自动生成`model`代码和`repository`代码

```bash
cago gen gorm table_name
```

#### MongoDB

mongo数据库无需先定义数据库接口,使用下面命令即可直接生成`model`代码和`repository`代码

```bash
cago gen mongo table_name
```
