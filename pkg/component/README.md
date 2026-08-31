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
| `redis_stream` | 是：`import _ "github.com/cago-frame/cago/pkg/broker/redis_stream"` | 基于 Redis Stream，适合已有 Redis、不想额外部署 MQ 的场景 |

Kafka 特有：

- 通过 `kafkabroker.WithKey("user-123")` 指定分区 Key（保证同 key 消息进入同一 partition）
- 支持 SASL (PLAIN / SCRAM-SHA-256 / SCRAM-SHA-512) 和 TLS
- `Requeue` 不支持，返回 `kafka.ErrRequeueUnsupported`
- `SubscribeOption.Retry=true` 在 Kafka 下会阻塞分区（不 commit offset 重投递）

Redis Stream 特有：

- 消费依赖 Consumer Group，`SubscribeOption.Group` 必填（组件默认会填 `AppName`）
- 客户端有两种来源：配置 `broker.redis_stream.addr` 由 broker 自建，或留空并在启动前调用
  `redis_stream.SetClient(redis.Default())` 复用已有连接（注入的客户端不会被 `Close()` 关闭）
- `Requeue` 不支持，返回 `redis_stream.ErrRequeueUnsupported`
- `Retry=true` 时失败的消息不 XACK，留在 pending 列表，由后台 `XAUTOCLAIM` 在
  `claimMinIdle` 之后重新投递；该机制同时负责接管崩溃消费者遗留的消息
- **`claimMinIdle` 必须大于 handler 的最长处理耗时**，否则处理中的消息会被认领导致重复消费
- Redis Stream 没有 kafka 那样的自动 retention，务必配置 `maxLen` 限制 stream 长度
- 暂不支持死信队列：`Retry=true` 的消息会被无限重投，需要业务侧自行兜底

配置示例：

```yaml
broker:
  type: redis_stream
  redis_stream:
    addr: 127.0.0.1:6379    # 留空则使用 redis_stream.SetClient 注入的客户端
    password: ""
    db: 0
    maxLen: 10000           # stream 近似最大长度，<=0 不裁剪
    count: 16               # 单次 XREADGROUP 拉取条数
    block: 5s               # XREADGROUP 阻塞时长
    claimMinIdle: 30s       # pending 消息空闲多久后可被重投
    claimInterval: 10s      # XAUTOCLAIM 扫描间隔
```
