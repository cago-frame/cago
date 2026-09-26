# 日志组件

封装zap作为日志组件

## 使用

```go
logger.Default().Info("info")
logger.With(zap.String("key", "value")).Info("info")

// 把日志实例放入context中
ctx:=logger.ContextWith(ctx, zap.String("key", "value"))
ctx:=logger.ContextWithLogger(ctx, logger.With(zap.String("key", "value")))

// 从context中取出日志实例使用
logger.Ctx(ctx).Info("info")
logger.CtxWith(ctx, zap.String("key", "value")).Info("info")
```

## 写文件与轮转

`logFile.enable` 打开后按 `filename` / `errorFilename` 写文件，由 lumberjack 轮转，两个文件共用下面的轮转参数。
每项零值取默认值，负数表示不限。

```yaml
logger:
  logFile:
    enable: true
    filename: ./runtime/logs/app.log
    errorFilename: ./runtime/logs/app.err.log
    maxSize: 2       # 单个文件大小上限（MB），默认 2
    maxBackups: 10   # 保留的已轮转文件个数，默认 10
    maxAge: 30       # 已轮转文件保留天数，默认 30
    compress: false  # 是否 gzip 压缩已轮转的文件，默认 false
```
