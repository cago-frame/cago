# Cago 组件包

> Cago 组件包,提供框架常用的一些组件

## Core

`component.Core`,核心组件包,提供了框架所需核心组件的初始化

- logger 日志组件,使用zap进行封装
- trace 链路追踪,支持jaeger和uptrace
- metrics 指标监控

## Database

`component.Database`,GORM数据库组件包

- 使用gorm进行封装,支持常见sql数据库

## Mongo

`component.Mongo`,MongoDB数据库组件包

## Redis

`component.Redis`,Redis组件包

## Cache

`component.Cache`,缓存组件包

支持下面的缓存

- redis

## Broker

`component.Broker`,消息队列组件包

支持下面的消息队列后端（对齐 `database/db` 的按需 import 模式）:

| 后端 | 是否需要显式 import | 说明 |
|------|-------------------|------|
| `nsq` | 否（默认内置） | 生产级消息队列 |
| `event_bus` | 是：`import _ "github.com/cago-frame/cago/pkg/broker/event_bus"` | 进程内内存队列，用于开发/测试 |
| `kafka` | 是：`import _ "github.com/cago-frame/cago/pkg/broker/kafka"` | 生产级高吞吐/有序流 |

Kafka 特有：

- 通过 `kafkabroker.WithKey("user-123")` 指定分区 Key（保证同 key 消息进入同一 partition）
- 支持 SASL (PLAIN / SCRAM-SHA-256 / SCRAM-SHA-512) 和 TLS
- `Requeue` 不支持，返回 `kafka.ErrRequeueUnsupported`
- `SubscribeOption.Retry=true` 在 Kafka 下会阻塞分区（不 commit offset 重投递）
