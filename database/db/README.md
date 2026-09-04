# 数据库

底层使用gorm库进行封装，支持多种数据库、支持单库与多库模式。

## 配置

```yaml
# 单库模式, key设置为db
db:
    driver: mysql
    dsn: root:password@tcp(127.0.0.1:3306)/db?charset=utf8mb4&collation=utf8mb4_unicode_ci&parseTime=True&loc=Local&multiStatements=true
    prefix: prefix_

# 多库模式, key设置为dbs，请注意需要注册对应的数据库驱动
dbs:
    default: # 默认链接, 必须设置
      driver: mysql
      dsn: root:password@tcp(127.0.0.1:3306)/db?charset=utf8mb4&collation=utf8mb4_unicode_ci&parseTime=True&loc=Local&multiStatements=true
      prefix: prefix_
    clickhouse: # clickhouse
      driver: clickhouse
      dsn: clickhouse://127.0.0.1:9009/default?read_timeout=10s
```

## 连接池

连接池参数写在库自己的配置下,多库模式每个库各配各的:

```yaml
db:
    driver: mysql
    dsn: root:password@tcp(127.0.0.1:3306)/db?charset=utf8mb4&parseTime=True&loc=Local
    maxOpenConns: 40      # 最大连接数, <=0 表示无上限
    maxIdleConns: 20      # 最大空闲连接数
    connMaxLifetime: 30m  # 连接最长存活时间
    connMaxIdleTime: 5m   # 连接最长空闲时间
```

**每项零值表示不设置**,保持 `database/sql` 自己的默认值。而那套默认值对长期跑的
服务是不合适的,上线前建议配一遍:

- `maxIdleConns` 默认 2 —— 并发一上来就不停地建/拆 TCP + 数据库握手;
- `maxOpenConns` 默认无上限 —— 一次流量尖峰就能顶穿数据库的最大连接数;
- `connMaxLifetime` 默认永不过期 —— 主从切换后会一直攥着指向旧主的死连接。

多副本部署时 `maxOpenConns` 是**每个副本**的上限,调之前先算
「副本数 × maxOpenConns ≤ 数据库的最大连接数」——MySQL 默认只有 151。

## 使用

```go
db.Default().Model(&User{}).Where("id = ?", 1).First(&user)
db.Ctx(ctx).Model(&User{}).Where("id = ?", 1).First(&user)

// 多库
db.With("clickhouse").Model(&User{}).Where("id = ?", 1).First(&user)
db.CtxWith(ctx, "clickhouse").Model(&User{}).Where("id = ?", 1).First(&user)
```

## 事务

推荐使用context去传递事务的数据库实例

```go
db.Default().Transaction(func(tx *gorm.DB) error {
	ctx:=db.ContextWithDB(context.Background(),tx)
	// 业务方法
    return SomeMethod(ctx)
})

func SomeMethod(ctx context.Context) error {
	db.Ctx(ctx).Model(&User{}).Where("id = ?", 1).First(&user)
    return nil
}
```

## 驱动

默认支持`mysql`，其它驱动需要使用`db.RegisterDriver`进行注册。可以参考[clickhouse](./clickhouse.go)的实现。

```go
import (
_ "github.com/cago-frame/cago/database/db/clickhouse"
)
```
