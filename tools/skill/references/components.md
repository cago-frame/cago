# Cago Components & Configuration

## Table of Contents

- [Component System](#component-system)
- [Pre-built Components](#pre-built-components)
- [Configuration](#configuration)
- [Database](#database)
- [Redis](#redis)
- [Cache](#cache)
- [Logger](#logger)
- [OpenTelemetry (Trace & Metrics)](#opentelemetry)
- [gRPC Server](#grpc-server)
- [Etcd](#etcd)
- [Broker (Message Queue)](#broker)
- [Goroutines (gogo)](#goroutines)

## Component System

### Interface

```go
type Component interface {
    Start(ctx context.Context, cfg *configs.Config) error
    CloseHandle()
}

type ComponentCancel interface {
    Component
    StartCancel(ctx context.Context, cancel context.CancelFunc, cfg *configs.Config) error
}
```

### FuncComponent

For simple components without cleanup:

```go
cago.FuncComponent(func(ctx context.Context, cfg *configs.Config) error {
    // initialization logic
    return nil
})
```

### Registration Order

Components start in registration order. Use `Registry()` for normal components, `RegistryCancel()` for components that
can terminate the app (like HTTP server).

## Pre-built Components

| Constructor                 | Config Key                   | Description                  |
|-----------------------------|------------------------------|------------------------------|
| `component.Core()`          | `logger`, `trace`, `metrics` | Logger + OpenTelemetry       |
| `component.Database()`      | `db` or `dbs`                | GORM database                |
| `component.Redis()`         | `redis`                      | Redis client                 |
| `component.Cache()`         | `cache`                      | Cache (Redis or in-memory)   |
| `component.Broker()`        | `broker`                     | Message queue (NSQ/EventBus) |
| `component.Etcd()`          | `etcd`                       | Etcd client                  |
| `component.Mongo()`         | `mongo`                      | MongoDB                      |
| `component.Elasticsearch()` | `elasticsearch`              | Elasticsearch                |
| `cron.Cron()`               | -                            | Cron scheduler               |
| `mux.HTTP(callback)`        | `http`                       | Gin HTTP server              |
| `grpc.GRPC(callback, opts)` | `grpc`                       | gRPC server (auto otel)      |

## Configuration

YAML file loaded via `configs.NewConfig("appname")`. Default path: `./configs/config.yaml`.

```yaml
# configs/config.yaml
env: DEV           # DEV, TEST, PRE, PROD
debug: true

http:
  address:
    - "0.0.0.0:8080"

grpc:
  address: "0.0.0.0:9090"

logger:
  level: debug     # debug, info, warn, error
  disableConsole: false
  logFile:
    enable: true
    filename: ./runtime/logs/app.log
    errorFilename: ./runtime/logs/app.err.log
  # loki:
  #   url: "http://localhost:3100/loki/api/v1/push"

db:
  driver: "mysql"  # mysql, postgres, sqlite, clickhouse
  dsn: "user:password@tcp(127.0.0.1:3306)/dbname?charset=utf8mb4&parseTime=True&loc=Local&multiStatements=true"
  prefix: "t_"
  debug: false
  prepareStmt: true

# OR multi-database:
# dbs:
#   default:
#     driver: "mysql"
#     dsn: "..."
#   secondary:
#     driver: "postgres"
#     dsn: "..."

redis:
  addr: "127.0.0.1:6379"
  password: ""
  db: 0

cache:
  type: "redis"    # "redis" or "memory"
  addr: "127.0.0.1:6379"
  password: ""
  db: 1

broker:
  type: "eventbus" # "eventbus" (in-memory) or "nsq"
  # nsq:
  #   addr: "127.0.0.1:4150"
  #   nsqlookupaddr:
  #     - "127.0.0.1:4161"

trace:
  endpoint: "localhost:4317"
  sample: 1         # 0=never, 0-1=percentage, 1=always
  useSSL: false
  # type: "grpc"    # "grpc" (default), "http", or "noop"
```

### Access Config Values

```go
cfg.Scan(ctx, "db", &dbConfig)     // Scan into struct
cfg.Key("http.address").String()   // Dot notation access
cfg.Debug                          // bool
cfg.Env                            // "DEV", "TEST", etc.
```

## Database

```go
import "github.com/cago-frame/cago/database/db"

db.Default()                       // Default *gorm.DB
db.Use("secondary")               // Named database
db.Ctx(ctx)                        // From context (transaction-aware), fallback to default
db.CtxWith(ctx, "secondary")      // From context, fallback to named DB
db.WithContextDB(ctx, tx)          // Set *gorm.DB (e.g. transaction) in context
db.WithContext(ctx, "secondary")   // Set named DB in context
db.RecordNotFound(err)             // Check if gorm.ErrRecordNotFound
```

### Transaction Pattern

将事务 tx 放入 context 后，后续所有使用 `db.Ctx(ctx)` 的代码自动在事务中执行：

```go
err := db.Ctx(ctx).Transaction(func(tx *gorm.DB) error {
    // 关键：将 tx 放入 context，后续 db.Ctx(ctx) 自动使用事务
    ctx := db.WithContextDB(ctx, tx)

    // 所有 repository 方法透明使用事务
    if err := scriptRepo.Create(ctx, script); err != nil {
        return err
    }
    scriptCode.ScriptID = script.ID
    if err := scriptCodeRepo.Create(ctx, scriptCode); err != nil {
        return err
    }
    return nil
})
if err != nil {
    return nil, err
}

// 事务提交后再发布消息
if err := producer.PublishScriptCreate(ctx, script, scriptCode); err != nil {
    return nil, err
}
```

### Multi-database Usage

```go
// 通过 context 切换数据库
ctx = db.WithContext(ctx, "secondary")
db.Ctx(ctx)  // 使用 secondary 数据库

// 直接使用
db.Use("secondary").Where("id=?", 1).First(&entity)

// CtxWith: 从 context 获取，若无则使用指定的命名数据库
db.CtxWith(ctx, "secondary")
```

### Custom Driver Registration

```go
db.RegisterDriver(db.Driver("custom"), func(cfg *db.Config) gorm.Dialector {
    return customDialector(cfg.Dsn)
})
```

## Redis

```go
import "github.com/cago-frame/cago/database/redis"

redis.Default()           // *redis.Client (原始客户端)
redis.Ctx(ctx)            // *CtxRedis (自动传递 context 的包装)
redis.Nil(err)            // 检查 err 是否为 redis.Nil (key 不存在)
```

### Context-Aware Usage

`redis.Ctx(ctx)` 返回的 `CtxRedis` 自动为所有操作传递 context，无需每次手动传：

```go
// 使用 Ctx 包装 (推荐)
redis.Ctx(ctx).Set("key", "value", time.Hour)
redis.Ctx(ctx).Get("key")
redis.Ctx(ctx).Del("key")
redis.Ctx(ctx).SetNX("lock:key", 1, 10*time.Second)
redis.Ctx(ctx).Incr("counter")
redis.Ctx(ctx).HSet("hash", "field", "value")
redis.Ctx(ctx).HGet("hash", "field")
redis.Ctx(ctx).LPush("list", "item")
redis.Ctx(ctx).ZAdd("sorted_set", redis2.Z{Score: 1, Member: "item"})
redis.Ctx(ctx).Expire("key", time.Hour)
redis.Ctx(ctx).Exists("key")
redis.Ctx(ctx).TTL("key")

// 检查 key 不存在
val, err := redis.Ctx(ctx).Get("key").Result()
if redis.Nil(err) {
    // key 不存在
}

// 直接使用原始客户端
redis.Default().Set(ctx, "key", "value", 0).Err()
```

### CtxRedis 支持的操作

- **String**: Get, Set, SetNX, SetXX, SetEx, SetRange, GetRange, GetSet
- **Numeric**: Incr, IncrBy, IncrByFloat, Decr, DecrBy
- **Key**: Del, Exists, Expire, ExpireAt, TTL
- **List**: LPush, RPush, LPop, RPop, LLen, LRange, LTrim, LRem
- **Hash**: HGet, HSet, HDel, HGetAll, HIncrBy, HIncrByFloat, HExists, HKeys, HLen, HSetNX, HVals, HScan
- **HyperLogLog**: PFCount, PFAdd, PFMerge
- **Sorted Set**: ZAdd, ZAddNX, ZAddXX, ZAddArgs, ZAddArgsIncr, ZRemRangeByScore, ZRemRangeByLex, ZRemRangeByRank, ZRevRangeByScore, ZRangeWithScores, ZRangeByScore, ZRangeByLex 等

## Cache

```go
import (
    "github.com/cago-frame/cago/database/cache"
    cache2 "github.com/cago-frame/cago/database/cache/cache"
    "github.com/cago-frame/cago/database/cache/cache/memory"
)

cache.Default()              // Cache 接口
cache.Ctx(ctx)               // *CtxCache (自动传递 context)
cache.IsNil(err)             // 检查 err 是否为 ErrNil (key 不存在)
cache.Expiration(dur)        // 设置过期时间选项
cache.WithDepend(dep)        // 设置依赖失效选项
cache.NewPrefixCache(prefix, c)  // 带前缀的 cache 包装
```

### 基本操作

```go
// Set with expiration
cache.Ctx(ctx).Set("user:123", userData, cache.Expiration(time.Hour))

// Get with type conversion
val, err := cache.Ctx(ctx).Get("user:123").Scan(&user)
num, err := cache.Ctx(ctx).Get("counter").Int64()
str, err := cache.Ctx(ctx).Get("name").Result()
b, err := cache.Ctx(ctx).Get("flag").Bool()

// Check existence
exists, err := cache.Ctx(ctx).Has("key")

// Delete
err := cache.Ctx(ctx).Del("key")

// Check if key not found
if cache.IsNil(err) {
    // key 不存在
}
```

### GetOrSet 模式 (Cache-Aside)

自动处理缓存未命中，回源获取数据后存入缓存：

```go
user := &User{}
err := cache.Ctx(ctx).GetOrSet("user:123", func() (interface{}, error) {
    // 缓存未命中时执行，从数据库获取
    return userRepo.FindFromDB(ctx, 123)
}, cache.Expiration(time.Hour)).Scan(user)
```

### KeyDepend 依赖失效模式

通过依赖 key 实现缓存级联失效。当数据变更时，只需 InvalidKey 一次，所有依赖该 key 的缓存自动失效：

```go
// Repository 中定义 KeyDepend
func (u *scriptRepo) KeyDepend(id int64) *cache2.KeyDepend {
    return cache2.NewKeyDepend(cache.Default(), "script:"+strconv.FormatInt(id, 10)+":dep")
}

// 读取时带上依赖
func (u *scriptRepo) Find(ctx context.Context, id int64) (*entity.Script, error) {
    ret := &entity.Script{}
    err := cache.Ctx(ctx).GetOrSet("script:"+strconv.FormatInt(id, 10), func() (interface{}, error) {
        if err := db.Ctx(ctx).Where("id=?", id).First(ret).Error; err != nil {
            if db.RecordNotFound(err) {
                return nil, nil
            }
            return nil, err
        }
        return ret, nil
    }, cache.WithDepend(u.KeyDepend(id)), cache.Expiration(time.Hour)).Scan(ret)
    if err != nil {
        return nil, err
    }
    return ret, nil
}

// 写入时使依赖失效，所有使用该 KeyDepend 的缓存都会被刷新
func (u *scriptRepo) Update(ctx context.Context, script *entity.Script) error {
    if err := db.Ctx(ctx).Updates(script).Error; err != nil {
        return err
    }
    return u.KeyDepend(script.ID).InvalidKey(ctx)
}
```

### Memory Cache (进程内缓存)

适用于大数据量的本地缓存，避免频繁 Redis 请求：

```go
import "github.com/cago-frame/cago/database/cache/cache/memory"

// 创建内存缓存实例
memCache, _ := memory.NewMemoryCache()

// 用于 repository 中缓存大数据 (如代码内容)
type codeRepo struct {
    memoryCache cache2.Cache
}

func NewCodeRepo() *codeRepo {
    c, _ := memory.NewMemoryCache()
    return &codeRepo{memoryCache: c}
}

// 使用内存缓存 + KeyDepend
func (u *codeRepo) FindLatest(ctx context.Context, id int64) (*entity.Code, error) {
    ret := &entity.Code{}
    err := u.memoryCache.GetOrSet(ctx, "code:"+strconv.FormatInt(id, 10), func() (interface{}, error) {
        return u.findFromDB(ctx, id)
    }, cache2.Expiration(time.Hour), cache2.WithDepend(u.KeyDepend(id))).Scan(&ret)
    return ret, err
}
```

### PrefixCache (命名空间隔离)

```go
// 为不同模块创建独立的缓存命名空间
userCache := cache.NewPrefixCache("user:", cache.Default())
scriptCache := cache.NewPrefixCache("script:", cache.Default())
```

## Etcd

```go
import "github.com/cago-frame/cago/database/etcd"
```

### 基本用法

```go
etcd.Default()  // *clientv3.Client (原始客户端)
```

### 带缓存的客户端

`etcd.Client` 封装 `clientv3.Client`，方法签名完全一致。通过 `WithCache()` 启用可选的内存缓存：

```go
// 创建带缓存的客户端
cli := etcd.NewCacheClient(etcd.Default(), etcd.WithCache())

// Get — 启用缓存时首次从 etcd 读取并缓存，后续直接返回缓存
resp, err := cli.Get(ctx, "/config/key")

// Put — 在写锁内完成 etcd 写入和缓存更新，保证一致性
_, err := cli.Put(ctx, "/config/key", "value")

// Delete — 先移除缓存再删除 etcd，避免读到已删除的旧值
_, err := cli.Delete(ctx, "/config/key")

// Watch — 启用缓存时通过代理 channel 自动更新缓存
watchCh := cli.Watch(ctx, "/config/key")
for resp := range watchCh {
    // PUT 事件自动更新缓存，DELETE 事件自动移除缓存
    for _, ev := range resp.Events {
        // 处理事件...
    }
}
```

不启用缓存时所有方法直接透传到 `clientv3.Client`，无额外开销。

### 配置

```yaml
etcd:
  endpoints:
    - 127.0.0.1:2379
  username: ""
  password: ""
```

### 作为配置中心

etcd 可作为配置源，替代文件配置。配置中的 key 会自动缓存并通过 Watch 实时同步：

```yaml
source: etcd       # 切换为 etcd 配置源（默认 file）
etcd:
  endpoints:
    - 127.0.0.1:2379
  prefix: /config  # etcd 中的路径前缀
```

## Logger

```go
import "github.com/cago-frame/cago/pkg/logger"

logger.Default()                    // *zap.Logger
logger.Ctx(ctx)                     // Logger from context (with trace/user fields)
logger.WithContextLogger(ctx, l)    // Set logger in context
```

### Context Logger Enrichment

在 middleware 中丰富 logger 上下文，后续代码自动带上这些字段：

```go
// Middleware 中添加用户信息
ctx = logger.WithContextLogger(ctx, logger.Ctx(ctx).With(
    zap.Int64("user_id", user.ID),
))

// 后续代码自动带上 user_id
logger.Ctx(ctx).Info("操作成功")  // 日志中自动包含 user_id 字段
logger.Ctx(ctx).Error("操作失败", zap.Error(err))
```

## OpenTelemetry

### Trace (分布式追踪)

Auto-instrumented for DB, Redis, HTTP. 启用 trace 后自动：
- 为每个 HTTP 请求创建 span
- 将 trace_id 注入 HTTP 响应头 `X-Trace-Id`
- 为 DB 查询、Redis 操作、Broker 消息自动创建子 span
- 在 logger 中自动添加 trace_id 和 span_id 字段

#### Manual Spans

```go
import "go.opentelemetry.io/otel"

ctx, span := otel.Tracer("").Start(ctx, "operation-name")
defer span.End()
span.SetAttributes(attribute.String("key", "value"))
span.SetAttributes(attribute.Int64("user_id", userId))
```

#### Trace Context in Services

```go
import "github.com/cago-frame/cago/pkg/opentelemetry/trace"

// 从 context 获取当前 span 并设置属性
trace.SpanFromContext(ctx).SetAttributes(
    attribute.Int64("user_id", user.ID),
)
```

#### Trace Config

```yaml
trace:
  endpoint: "localhost:4317"    # OTLP collector 地址
  sample: 1                     # 采样率: 0=不采样, 0-1=百分比, 1=全采样
  useSSL: false                 # 是否启用 SSL/TLS
  # type: "grpc"                # "grpc" (默认), "http", "noop"
  # header:                     # 自定义 OTLP 头
  #   Authorization: "Bearer xxx"
```

### Metrics (Prometheus)

自动采集的 HTTP 指标：
- `http_request_total` — 请求总数
- `http_request_duration` — 请求耗时分布 (100ms, 300ms, 500ms, 1s, 2s, 5s, 10s)
- `http_request_body_size` — 请求体大小
- `http_response_body_size` — 响应体大小
- `http_status_code` — 状态码分布

指标暴露在 `GET /metrics` 端点 (Prometheus 格式)。

## gRPC Server

```go
import "github.com/cago-frame/cago/server/grpc"
```

### 基本用法

```go
grpc.GRPC(func(ctx context.Context, s *grpc.Server) error {
    pb.RegisterUserServiceServer(s, &userServiceImpl{})
    return nil
})
```

### 配置

```yaml
grpc:
  address: "0.0.0.0:9090"   # 默认 127.0.0.1:9090
```

### 自定义拦截器（中间件）

gRPC 的中间件通过 `grpc.ServerOption`（拦截器）实现，支持两种注册方式：

#### 1. 构造函数传入

```go
grpc.GRPC(registerServices,
    grpc.ChainUnaryInterceptor(authInterceptor, logInterceptor),
    grpc.ChainStreamInterceptor(streamAuthInterceptor),
)
```

#### 2. 全局注册（类似 mux.RegisterMiddleware）

```go
grpc.RegisterServerOption(func(cfg *configs.Config) ([]grpc.ServerOption, error) {
    return []grpc.ServerOption{
        grpc.ChainUnaryInterceptor(myInterceptor),
    }, nil
})
```

### OpenTelemetry 自动集成

当注册了 trace 或 metrics 组件时，gRPC 服务器自动接入 OpenTelemetry：
- **Tracing** — 每个 RPC 调用自动创建 span，通过 `otelgrpc.NewServerHandler()` 实现
- **Metrics** — 自动记录 RPC 请求数、延迟、消息大小等指标

无需手动配置，只要在 gRPC 之前注册 `component.Core()` 即可。

## Broker

```go
import (
    "github.com/cago-frame/cago/pkg/broker"
    broker2 "github.com/cago-frame/cago/pkg/broker/broker"
)

broker.Default()  // Broker 接口
```

### Publish

```go
err := broker.Default().Publish(ctx, "topic_name", &broker2.Message{
    Body: data,
})
```

### Subscribe

```go
subscriber, err := broker.Default().Subscribe(ctx, "topic_name",
    func(ctx context.Context, event broker2.Event) error {
        msg := event.Message()
        // 处理消息 msg.Body
        return nil
    },
    // Options:
    broker2.Retry(),              // 处理失败时重试
    broker2.NotAutoAck(),         // 禁用自动 ack，需手动 event.Ack()
    broker2.Group("my-group"),    // 消费者组 (默认使用 app name)
    broker2.WithConcurrent(3),    // 并发消费者数量 (默认 1)
)

// 取消订阅
defer subscriber.Unsubscribe()
```

### Event Interface

```go
type Event interface {
    Topic() string                        // Topic 名
    Message() *Message                    // 消息内容 (Header + Body)
    Ack() error                           // 确认消息已处理
    Requeue(delay time.Duration) error    // 重新入队 (延迟重试)
    Attempted() int                       // 重试次数
    Error() error                         // 处理错误
}
```

### 手动 Ack 模式

```go
broker.Default().Subscribe(ctx, "topic",
    func(ctx context.Context, event broker2.Event) error {
        msg := event.Message()
        if err := processMessage(ctx, msg); err != nil {
            // 失败时延迟 5 秒重试
            return event.Requeue(5 * time.Second)
        }
        return event.Ack()
    },
    broker2.NotAutoAck(),  // 必须手动 ack
)
```

### Trace Context Propagation

Broker 自动在 Message.Header 中传播 trace context。发布消息时注入 trace，订阅消息时提取 trace，实现跨服务链路追踪。

### NSQ vs EventBus

| 特性 | NSQ | EventBus |
|------|-----|----------|
| 持久化 | 是 | 否 (内存) |
| 重试 | 支持 | 不支持 |
| 多实例消费 | 支持 (group) | 不支持 |
| 适用场景 | 生产环境 | 开发/测试 |

```yaml
# NSQ
broker:
  type: nsq
  nsq:
    addr: "127.0.0.1:4150"
    nsqlookupaddr:
      - "127.0.0.1:4161"

# EventBus (开发/测试)
broker:
  type: eventbus
```

## Goroutines

Always use `gogo.Go` for spawning goroutines:

```go
import "github.com/cago-frame/cago/pkg/gogo"

gogo.Go(ctx, func(ctx context.Context) error {
    // async work, respond to ctx.Done() for graceful shutdown
    select {
    case <-ctx.Done():
        return nil
    default:
        // do work
    }
    return nil
})

// With options
gogo.Go(ctx, fn, gogo.WithIgnorePanic(true))
```

`gogo.Go` provides:

- Panic recovery with logging
- Graceful shutdown coordination via `gogo.Wait()` (framework waits up to 10s)
- Context propagation
