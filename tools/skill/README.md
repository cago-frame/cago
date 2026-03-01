# Cago AI Skill

Provides Cago framework expertise to AI coding assistants, enabling them to generate code that follows framework conventions when developing Cago projects.

## Contents

```
tools/skill/
├── SKILL.md                      # Core knowledge: project layout, API definition, layer patterns, conventions
└── references/
    ├── examples.md               # Complete code examples: entry point, all layers, cron jobs, message queue, migrations
    └── components.md             # Components & configuration: component system, config format, database, Redis, cache, logger, trace, metrics, etc.
```

### SKILL.md

The main skill file containing core knowledge for AI:

- **Project Layout** — Responsibilities of `cmd/`, `internal/api/`, `controller/`, `service/`, `repository/`, etc.
- **API Definition Pattern** — Route declaration with `mux.Meta`, request/response struct tags (`form`, `uri`, `header`, `binding`)
- **Handler Signatures** — Four valid signature formats for Controller methods
- **Route Binding** — `Router()` function, `Group()`, `Bind()`, middleware usage
- **gRPC Service** — `grpc.GRPC()` component, interceptors, automatic OpenTelemetry integration
- **Service Pattern** — Interface + singleton accessor + private implementation
- **Repository Pattern** — Interface + Register/accessor + `db.Ctx(ctx)` queries
- **Database Access** — `db.Default()`, `db.Ctx(ctx)`, transaction pattern
- **Error Codes i18n** — Error code definition + multi-language registration
- **TDD Workflow** — Recommended Test-Driven Development process with step-by-step example
- **Conventions** — Comments, testing tools, lint rules, mock generation

### references/examples.md

Complete code examples for each layer, including:

- Application entry (`main.go`) — Component registration chain
- API Definition — Request/response structs + tag reference + custom Validate + pagination
- Router — Route registration, grouping, middleware
- Controller — Practical usage of handler signatures
- Service — Interface definition + business logic implementation
- Repository — CRUD operations, paginated queries, mock generation
- Entity — GORM entity definition + validation methods
- Error Codes — Definition + i18n registration + multiple error types
- Cron Jobs — Cron registration + handler + environment control
- Message Queue — Message definition, Publish/Subscribe, Handler, Producer/Consumer separation
- Database Migration — gormigrate usage
- Advanced Patterns — Transaction context propagation, Cache-Aside + KeyDepend, memory cache, RouterTree nested middleware, rate limiting, context enrichment, testing

### references/components.md

Detailed documentation of the component system and configuration:

- Component Interface — `Component`, `ComponentCancel`, `FuncComponent`
- Pre-built Components — Core, Database, Etcd, Redis, Cache, Broker, Mongo, Elasticsearch, Cron, HTTP, gRPC
- Configuration File — Complete YAML config examples (http, logger, db, redis, cache, broker, trace)
- Database — Multi-database support, transaction pattern, context-aware queries
- Redis — `redis.Ctx(ctx)` context-aware wrapper, Nil check, all operation types
- Cache — `cache.Ctx(ctx)`, GetOrSet cache-aside, KeyDepend invalidation, Memory Cache
- Etcd — `etcd.Default()`, cached client `NewCacheClient(WithCache())`, Watch auto-sync, config center
- Logger — zap logging, context logger, logger field enrichment
- OpenTelemetry — Auto-instrumented trace + manual spans, Prometheus Metrics
- Broker — Message publish/subscribe, Subscribe options, Event interface, NSQ vs EventBus
- Goroutines — `gogo.Go` safe goroutine spawning

## Installation

### Claude Code

```bash
ln -s $(pwd)/tools/skill ~/.claude/skills/cago
```

The skill auto-triggers when code imports `github.com/cago-frame/cago` or the project's `go.mod` contains a cago dependency. You can also explicitly invoke it with `/cago`:

```
/cago Create a CRUD API for user management
/cago Add a cron job that runs every hour
/cago Create a database migration for the order table
```

### Other AI Tools

Provide `SKILL.md` and the files under `references/` as context to the AI.
