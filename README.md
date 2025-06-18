# Cago

[![godoc](https://godoc.org/github.com/cago-frame/cago?status.svg)](https://pkg.go.dev/github.com/cago-frame/cago)
[![Go Report Card](https://goreportcard.com/badge/github.com/cago-frame/cago)](https://goreportcard.com/report/github.com/cago-frame/cago)
[![License](https://img.shields.io/github/license/go-eagle/eagle?style=flat-square)](/LICENSE)

Cago 一个快速开发的集成式框架.使用模块化的开发模式,每一个组件都可以单独的调用.

Cago 只对社区工具进行集成,大大减少迁移难度和学习成本,我们不生产代码,我们只是方案的搬运工.

使用 go 的`struct`来声明 API 和 swagger 文档,可以通过脚手架来帮助你生成相关内容.

## 快速开始

你可以通过简单示例来快速的了解 Cago 的使用。你也可以复制这个示例来创建你的项目。

[简单示例](./examples/simple)

## 脚手架

你可以安装脚手架来帮助你生成代码。

[脚手架文档](./cmd/cago)

## CI/CD

[部署资源](./deploy)

cago 提供了[`gitlab-ci`](deploy/gitlab/.gitlab-ci.yml)、[`golanglint-ci`](./deploy/.golangci.yml)和
[`Kubernetes helm`](./deploy)的 CI/CD 配置文件,可以快速帮你实现 CI/CD.

当本地调试时也可以使用[`docker-compose.yaml`](./deploy/docker-compose.yaml)启动调试环境.

可以使用`etcd`作为配置中心，也支持文件作为配置启动。

## 组件

大多数组件都是基于社区工具进行封装，方便使用。

- [常用组件包](./pkg/component)
- [数据库组件包](./database)
- [服务组件包](./server)
- [中间件](./middleware)
- [工具包/杂项](./pkg)

## 目录结构

Cago使用三层架构，并混合了DDD的思想，推荐使用下面的目录结构。你也可以根据自己的需求来调整目录结构。

- `cmd/app/main.go` 项目的入口
- `configs`
  - `config.yaml` 配置文件
  - `...` 其他配置文件，也可以写配置读取方法，其它包调用：`configs.GetConfig().XXX`
- `docs` 文档目录，包括 swagger api 文档、设计文档等
- `deploy` 部署资源文件
- `internal` 内部包
  - `api` API 请求结构体
    - `example/example.go` api请求与返回结构
    - `router.go` 路由定义
  - `controller` 控制器目录，API请求会调用控制器，做一些数据处理校验逻辑
  - `model` 数据模型
    - `entity` 数据库实体模型，推荐使用充血模型
    - `xxx.go` 一些数据模型的定义
  - `pkg` 项目内通用的模块
    - `code` 错误码定义
    - `utils` 工具包
  - `repository` 数据库操作
  - `service` 服务接口
  - `task` 任务模块
    - `crontab` 定时任务
    - `queue` 消息队列
      - `handler` 消息队列处理器
      - `message` 消息定义
      - `xxx.go` 消息队列定义
    - `task.go` 任务定义
- `migrations` 数据库迁移文件
- `pkg` 公共的模块，可以被其它项目引用
- `.golangci.yml` golangci-lint 配置文件
- `Makefile` 项目的 Makefile 文件

## 参考项目

- GoFrame
- GoMicro

## Who use Cago？

- [脚本猫](https://github.com/scriptscat/scriptlist)
- [DSP2B](https://github.com/dsp2b/dsp2b)

## License

[MIT](./LICENSE)
