# Cago 脚手架

Cago有一个简单的脚手架，你可以使用脚手架快速开发你的项目

## 安装

```bash
go install github.com/cago-frame/cago/cmd/cago@latest
```

## 使用

你可以使用`cago -h`来查看脚手架支持的命令

### API

在`internal/api`目录下,定义好 api 请求结构,使用下面命令和自动生成`controller`代码和`swagger`文档

在`internal/service`目录下,定义好 service 接口,使用下面命令和自动生成`service`代码

```bash
cago gen
```

默认使用 Cago 内置的 Swagger 文档生成器，并且不会在 controller 中生成 Swagger 注释。如需生成兼容 Swaggo 的接口注释，并使用 Swaggo 生成 `docs/docs.go`、`docs/swagger.json` 和 `docs/swagger.yaml`，可使用：

```bash
cago gen --swagger
```

也可以使用短参数：`cago gen -s`。对已有 controller 再次执行时，会补充缺失的 Swagger 注释，不会重复添加。启用该参数后不会再执行 Cago 内置的 Swagger 文档生成器。

需要查看逐文件扫描、方法生成以及 Swaggo 的详细日志时，可增加 `--verbose`（或 `-v`）。默认输出只保留生成阶段、创建结果和最终状态，便于在本地及 CI 中阅读。
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
