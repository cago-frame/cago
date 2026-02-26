# Cago Components & Configuration

## Table of Contents

- [Component System](#component-system)
- [Pre-built Components](#pre-built-components)
- [Configuration](#configuration)
- [Database](#database)
- [Redis & Cache](#redis--cache)
- [Logger](#logger)
- [OpenTelemetry](#opentelemetry)
- [Broker (Message Queue)](#broker)
- [IAM (Authentication)](#iam)
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
| `component.Mongo()`         | `mongo`                      | MongoDB                      |
| `component.Elasticsearch()` | `elasticsearch`              | Elasticsearch                |
| `cron.Cron()`               | -                            | Cron scheduler               |
| `mux.HTTP(callback)`        | `http`                       | Gin HTTP server              |

## Configuration

YAML file loaded via `configs.NewConfig("appname")`. Default path: `./configs/config.yaml`.

```yaml
# configs/config.yaml
env: DEV           # DEV, TEST, PRE, PROD
debug: true

http:
  address:
    - "0.0.0.0:8080"

logger:
  level: debug     # debug, info, warn, error
  filename: ""     # Empty = stdout only
  # loki:
  #   url: "http://localhost:3100/loki/api/v1/push"

db:
  driver: "mysql"  # mysql, postgres, sqlite, clickhouse
  dsn: "user:password@tcp(127.0.0.1:3306)/dbname?charset=utf8mb4&parseTime=True&loc=Local"
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

broker:
  type: "eventbus" # "eventbus" (in-memory) or "nsq"
  # nsq:
  #   addr: ["127.0.0.1:4150"]
  #   lookupAddr: ["127.0.0.1:4161"]

trace:
  type: "jaeger"
  endpoint: "http://localhost:14268/api/traces"
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
db.Ctx(ctx)                        // From context (transaction-aware)
db.WithContextDB(ctx, tx)          // Set transaction in context
db.WithContext(ctx, "secondary")   // Use named DB from context
db.RecordNotFound(err)             // Check if record not found
```

### Transaction Pattern

```go
err := db.Default().Transaction(func(tx *gorm.DB) error {
    ctx := db.WithContextDB(ctx, tx)
    // All db.Ctx(ctx) calls now use the transaction
    return repo.Create(ctx, entity)
})
```

## Redis & Cache

```go
import "github.com/cago-frame/cago/database/redis"
import "github.com/cago-frame/cago/database/cache"

redis.Default()     // *redis.Client
cache.Default()     // Cache interface (Get, Set, Delete)
```

## Logger

```go
import "github.com/cago-frame/cago/pkg/logger"

logger.Default()          // *zap.Logger
logger.Ctx(ctx)           // Logger from context (with trace/user fields)
logger.WithContextLogger(ctx, l)  // Set logger in context
```

## OpenTelemetry

Auto-instrumented for DB, Redis, HTTP. Manual spans:

```go
import "go.opentelemetry.io/otel"

ctx, span := otel.Tracer("").Start(ctx, "operation-name")
defer span.End()
span.SetAttributes(attribute.String("key", "value"))
```

## Broker

```go
import (
    "github.com/cago-frame/cago/pkg/broker"
    broker2 "github.com/cago-frame/cago/pkg/broker/broker"
)

// Publish
broker.Default().Publish(ctx, "topic", &broker2.Message{Body: data})

// Subscribe
broker.Default().Subscribe(ctx, "topic", func(ctx context.Context, event broker2.Event) error {
    msg := event.Message()
    // process msg.Body
    return nil
}, broker2.Retry())
```

## IAM

Authentication and session management:

```go
import (
    "github.com/cago-frame/cago/pkg/iam"
    "github.com/cago-frame/cago/pkg/iam/authn"
    "github.com/cago-frame/cago/pkg/iam/audit"
)

// Initialize in main.go
iam.IAM(userRepo, iam.WithAuthnOptions(), iam.WithAuditOptions(...))

// Login/Logout
authn.Default().LoginByPassword(ctx, username, password)
authn.Default().Logout(ctx)
authn.Default().RefreshSession(ctx, refreshToken)

// Middleware
authn.Default().Middleware(force, userLoadCallback)

// Audit
audit.Default().Middleware(module, fieldsCallback)
audit.Ctx(ctx).Record("action", zap.String("key", "value"))
```

## Goroutines

Always use `gogo.Go` for spawning goroutines:

```go
import "github.com/cago-frame/cago/pkg/gogo"

gogo.Go(ctx, func(ctx context.Context) error {
    // async work
    return nil
})

// With options
gogo.Go(ctx, fn, gogo.WithIgnorePanic())
```

`gogo.Go` provides:

- Panic recovery with logging
- Graceful shutdown coordination via `gogo.Wait()`
- Context propagation
