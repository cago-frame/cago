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
