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
cfg.Scan(ctx, "db", &dbConfig)         // Scan into struct
cfg.String(ctx, "http.address")        // Dot notation, returns string
cfg.Bool(ctx, "debug")                 // Returns bool
cfg.Has(ctx, "key")                    // Check if key exists
cfg.Watch(ctx, "key", callback)        // Watch config changes (callback receives source.Event)
cfg.Debug                              // bool (direct field access)
cfg.Env                                // Env type: "dev", "test", "pre", "prod"
```

### Etcd as Configuration Source

Default config source is file (`configs/config.yaml`). Set `source: etcd` to switch to etcd. The config file still needs basic etcd connection info — the framework reads it from the file first, then switches to etcd for all other config keys.

Key prefix rule: `{etcd.prefix}/{env}/{appName}` — e.g., if prefix is `/config`, env is `dev`, appName is `myapp`, the full key for database config is `/config/dev/myapp/db`.

```yaml
# configs/config.yaml — etcd config source
source: etcd
env: dev
etcd:
  endpoints:
    - 127.0.0.1:2379
  prefix: /config     # Final keys: /config/dev/appname/db, /config/dev/appname/redis, etc.
```

Each top-level config key (`db`, `redis`, `cache`, `logger`, `trace`, `broker`, etc.) is stored as a separate etcd key. If a key doesn't exist, it's auto-initialized with the default value and returns an error on first run — set the value in etcd and restart.

Config values in etcd use the same serialization format (YAML by default). Example etcd value for key `/config/dev/myapp/db`:

```yaml
driver: mysql
dsn: "user:password@tcp(127.0.0.1:3306)/dbname?charset=utf8mb4&parseTime=True&loc=Local"
prefix: "t_"
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

After putting the transaction tx into context, all subsequent code using `db.Ctx(ctx)` automatically executes within the transaction:

```go
err := db.Ctx(ctx).Transaction(func(tx *gorm.DB) error {
    // Key: put tx into context, subsequent db.Ctx(ctx) calls automatically use the transaction
    ctx := db.WithContextDB(ctx, tx)

    // All repository methods transparently use the transaction
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

// Publish messages after transaction commits
if err := producer.PublishScriptCreate(ctx, script, scriptCode); err != nil {
    return nil, err
}
```

### Multi-database Usage

```go
// Switch database via context
ctx = db.WithContext(ctx, "secondary")
db.Ctx(ctx)  // Uses secondary database

// Direct usage
db.Use("secondary").Where("id=?", 1).First(&entity)

// CtxWith: get from context, fallback to the specified named database
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

redis.Default()           // *redis.Client (raw client)
redis.Ctx(ctx)            // *CtxRedis (context-aware wrapper)
redis.Nil(err)            // Check if err is redis.Nil (key not found)
```

### Context-Aware Usage

`redis.Ctx(ctx)` returns a `CtxRedis` that automatically passes context for all operations, no need to pass it manually each time:

```go
// Using Ctx wrapper (recommended)
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

// Check if key does not exist
val, err := redis.Ctx(ctx).Get("key").Result()
if redis.Nil(err) {
    // key does not exist
}

// Direct usage with raw client
redis.Default().Set(ctx, "key", "value", 0).Err()
```

### Supported CtxRedis Operations

- **String**: Get, Set, SetNX, SetXX, SetEx, SetRange, GetRange, GetSet
- **Numeric**: Incr, IncrBy, IncrByFloat, Decr, DecrBy
- **Key**: Del, Exists, Expire, ExpireAt, TTL
- **List**: LPush, RPush, LPop, RPop, LLen, LRange, LTrim, LRem
- **Hash**: HGet, HSet, HDel, HGetAll, HIncrBy, HIncrByFloat, HExists, HKeys, HLen, HSetNX, HVals, HScan
- **HyperLogLog**: PFCount, PFAdd, PFMerge
- **Sorted Set**: ZAdd, ZAddNX, ZAddXX, ZAddArgs, ZAddArgsIncr, ZRemRangeByScore, ZRemRangeByLex, ZRemRangeByRank, ZRevRangeByScore, ZRangeWithScores, ZRangeByScore, ZRangeByLex, etc.

## Cache

```go
import (
    "github.com/cago-frame/cago/database/cache"
    cache2 "github.com/cago-frame/cago/database/cache/cache"
    "github.com/cago-frame/cago/database/cache/cache/memory"
)

cache.Default()              // Cache interface
cache.Ctx(ctx)               // *CtxCache (context-aware)
cache.IsNil(err)             // Check if err is ErrNil (key not found)
cache.Expiration(dur)        // Set expiration option
cache.WithDepend(dep)        // Set dependency invalidation option
cache.NewPrefixCache(prefix, c)  // Prefixed cache wrapper
```

### Basic Operations

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
    // key does not exist
}
```

### GetOrSet Pattern (Cache-Aside)

Automatically handles cache misses, fetches data from source and stores in cache:

```go
user := &User{}
err := cache.Ctx(ctx).GetOrSet("user:123", func() (interface{}, error) {
    // Executed on cache miss, fetches from database
    return userRepo.FindFromDB(ctx, 123)
}, cache.Expiration(time.Hour)).Scan(user)
```

### KeyDepend Dependency Invalidation Pattern

Cascading cache invalidation via dependency keys. When data changes, just call InvalidKey once and all caches depending on that key are automatically invalidated:

```go
// Define KeyDepend in Repository
func (u *scriptRepo) KeyDepend(id int64) *cache2.KeyDepend {
    return cache2.NewKeyDepend(cache.Default(), "script:"+strconv.FormatInt(id, 10)+":dep")
}

// Include dependency when reading
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

// Invalidate dependency on write, all caches using this KeyDepend will be refreshed
func (u *scriptRepo) Update(ctx context.Context, script *entity.Script) error {
    if err := db.Ctx(ctx).Updates(script).Error; err != nil {
        return err
    }
    return u.KeyDepend(script.ID).InvalidKey(ctx)
}
```

### Memory Cache (In-Process)

Suitable for local caching of large data to avoid frequent Redis requests:

```go
import "github.com/cago-frame/cago/database/cache/cache/memory"

// Create memory cache instance
memCache, _ := memory.NewMemoryCache()

// For caching large data in repository (e.g., code content)
type codeRepo struct {
    memoryCache cache2.Cache
}

func NewCodeRepo() *codeRepo {
    c, _ := memory.NewMemoryCache()
    return &codeRepo{memoryCache: c}
}

// Use memory cache + KeyDepend
func (u *codeRepo) FindLatest(ctx context.Context, id int64) (*entity.Code, error) {
    ret := &entity.Code{}
    err := u.memoryCache.GetOrSet(ctx, "code:"+strconv.FormatInt(id, 10), func() (interface{}, error) {
        return u.findFromDB(ctx, id)
    }, cache2.Expiration(time.Hour), cache2.WithDepend(u.KeyDepend(id))).Scan(&ret)
    return ret, err
}
```

### PrefixCache (Namespace Isolation)

```go
// Create isolated cache namespaces for different modules
userCache := cache.NewPrefixCache("user:", cache.Default())
scriptCache := cache.NewPrefixCache("script:", cache.Default())
```

## Etcd

```go
import "github.com/cago-frame/cago/database/etcd"
```

### Basic Usage

```go
etcd.Default()  // *clientv3.Client (raw client)
```

### Cached Client

`etcd.Client` wraps `clientv3.Client` with identical method signatures. Enable optional in-memory caching via `WithCache()`:

```go
// Create a cached client
cli := etcd.NewCacheClient(etcd.Default(), etcd.WithCache())

// Get — with cache enabled, first read from etcd and cache, subsequent reads return from cache
resp, err := cli.Get(ctx, "/config/key")

// Put — completes etcd write and cache update within a write lock, ensuring consistency
_, err := cli.Put(ctx, "/config/key", "value")

// Delete — removes cache first then deletes from etcd, avoiding stale reads
_, err := cli.Delete(ctx, "/config/key")

// Watch — with cache enabled, automatically updates cache via proxy channel
watchCh := cli.Watch(ctx, "/config/key")
for resp := range watchCh {
    // PUT events auto-update cache, DELETE events auto-remove cache
    for _, ev := range resp.Events {
        // Handle events...
    }
}
```

Without cache enabled, all methods pass through directly to `clientv3.Client` with no overhead.

### Configuration

```yaml
etcd:
  endpoints:
    - 127.0.0.1:2379
  username: ""
  password: ""
```

### As Configuration Center

Etcd can serve as a configuration source, replacing file-based config. Keys are automatically cached and synced in real-time via Watch:

```yaml
source: etcd       # Switch to etcd config source (default: file)
etcd:
  endpoints:
    - 127.0.0.1:2379
  prefix: /config  # Path prefix in etcd
```

## Logger

```go
import "github.com/cago-frame/cago/pkg/logger"

logger.Default()                    // *zap.Logger
logger.Ctx(ctx)                     // Logger from context (with trace/user fields)
logger.WithContextLogger(ctx, l)    // Set logger in context
```

### Context Logger Enrichment

Enrich logger context in middleware, subsequent code automatically includes these fields:

```go
// Add user info in middleware
ctx = logger.WithContextLogger(ctx, logger.Ctx(ctx).With(
    zap.Int64("user_id", user.ID),
))

// Subsequent code automatically includes user_id
logger.Ctx(ctx).Info("operation succeeded")  // Logs automatically include user_id field
logger.Ctx(ctx).Error("operation failed", zap.Error(err))
```

## OpenTelemetry

### Trace (Distributed Tracing)

Auto-instrumented for DB, Redis, HTTP. When trace is enabled, it automatically:
- Creates a span for each HTTP request
- Injects trace_id into HTTP response header `X-Trace-Id`
- Creates child spans for DB queries, Redis operations, and Broker messages
- Adds trace_id and span_id fields to logger automatically

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

// Get current span from context and set attributes
trace.SpanFromContext(ctx).SetAttributes(
    attribute.Int64("user_id", user.ID),
)
```

#### Trace Config

```yaml
trace:
  endpoint: "localhost:4317"    # OTLP collector address
  sample: 1                     # Sample rate: 0=never, 0-1=percentage, 1=always
  useSSL: false                 # Enable SSL/TLS
  # type: "grpc"                # "grpc" (default), "http", "noop"
  # header:                     # Custom OTLP headers
  #   Authorization: "Bearer xxx"
```

### Metrics (Prometheus)

Auto-collected HTTP metrics:
- `http_request_total` — Total request count
- `http_request_duration` — Request duration distribution (100ms, 300ms, 500ms, 1s, 2s, 5s, 10s)
- `http_request_body_size` — Request body size
- `http_response_body_size` — Response body size
- `http_status_code` — Status code distribution

Metrics are exposed at `GET /metrics` endpoint (Prometheus format).

## gRPC Server

```go
import "github.com/cago-frame/cago/server/grpc"
```

### Basic Usage

```go
grpc.GRPC(func(ctx context.Context, s *grpc.Server) error {
    pb.RegisterUserServiceServer(s, &userServiceImpl{})
    return nil
})
```

### Configuration

```yaml
grpc:
  address: "0.0.0.0:9090"   # Default: 127.0.0.1:9090
```

### Custom Interceptors (Middleware)

gRPC middleware is implemented via `grpc.ServerOption` (interceptors), supporting two registration methods:

#### 1. Pass via Constructor

```go
grpc.GRPC(registerServices,
    grpc.ChainUnaryInterceptor(authInterceptor, logInterceptor),
    grpc.ChainStreamInterceptor(streamAuthInterceptor),
)
```

#### 2. Global Registration (similar to mux.RegisterMiddleware)

```go
grpc.RegisterServerOption(func(cfg *configs.Config) ([]grpc.ServerOption, error) {
    return []grpc.ServerOption{
        grpc.ChainUnaryInterceptor(myInterceptor),
    }, nil
})
```

### Automatic OpenTelemetry Integration

When trace or metrics components are registered, the gRPC server automatically integrates with OpenTelemetry:
- **Tracing** — Automatically creates a span for each RPC call via `otelgrpc.NewServerHandler()`
- **Metrics** — Automatically records RPC request count, latency, message size, and other metrics

No manual configuration needed — just register `component.Core()` before gRPC.

## Broker

```go
import (
    "github.com/cago-frame/cago/pkg/broker"
    broker2 "github.com/cago-frame/cago/pkg/broker/broker"
)

broker.Default()  // Broker interface
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
        // Process message msg.Body
        return nil
    },
    // Options:
    broker2.Retry(),              // Retry on processing failure
    broker2.NotAutoAck(),         // Disable auto ack, must manually call event.Ack()
    broker2.Group("my-group"),    // Consumer group (defaults to app name)
    broker2.WithConcurrent(3),    // Concurrent consumer count (default: 1)
)

// Unsubscribe
defer subscriber.Unsubscribe()
```

### Event Interface

```go
type Event interface {
    Topic() string                        // Topic name
    Message() *Message                    // Message content (Header + Body)
    Ack() error                           // Acknowledge message as processed
    Requeue(delay time.Duration) error    // Requeue (delayed retry)
    Attempted() int                       // Retry count
    Error() error                         // Processing error
}
```

### Manual Ack Mode

```go
broker.Default().Subscribe(ctx, "topic",
    func(ctx context.Context, event broker2.Event) error {
        msg := event.Message()
        if err := processMessage(ctx, msg); err != nil {
            // Retry with 5 second delay on failure
            return event.Requeue(5 * time.Second)
        }
        return event.Ack()
    },
    broker2.NotAutoAck(),  // Must manually ack
)
```

### Trace Context Propagation

Broker automatically propagates trace context in Message.Header. Trace is injected when publishing messages and extracted when subscribing, enabling cross-service distributed tracing.

### NSQ vs EventBus

| Feature | NSQ | EventBus |
|---------|-----|----------|
| Persistence | Yes | No (in-memory) |
| Retry | Supported | Not supported |
| Multi-instance consumption | Supported (group) | Not supported |
| Use case | Production | Development/Testing |

```yaml
# NSQ
broker:
  type: nsq
  nsq:
    addr: "127.0.0.1:4150"
    nsqlookupaddr:
      - "127.0.0.1:4161"

# EventBus (development/testing)
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
