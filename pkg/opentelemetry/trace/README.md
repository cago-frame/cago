# 链路追踪

```yaml
endpoint: "jaeger-collector.example.com"
useSsl: true
header:
  Authorization: "Basic ----"
sample: 0.001
# type 可选:
#   grpc(默认) 通过 OTLP/gRPC 上报
#   http       通过 OTLP/HTTP 上报
#   empty      不上报，但仍生成有效 trace_id/span_id，方便日志与响应头打出 trace_id
#   noop       完全空壳，不生成 trace_id（SpanContext 无效）
type: grpc
```
