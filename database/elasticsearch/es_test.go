package elasticsearch

import (
	"testing"

	"github.com/stretchr/testify/assert"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
)

func TestClientConfig_InstrumentationFollowsTraceProvider(t *testing.T) {
	cfg := &Config{Address: []string{"http://127.0.0.1:9200"}, Username: "u", Password: "p"}

	// 未配置链路追踪：不挂 instrumentation，行为与之前一致
	plain := clientConfig(cfg, nil, nil)
	assert.Nil(t, plain.Instrumentation)
	assert.Equal(t, cfg.Address, plain.Addresses)
	assert.Equal(t, "u", plain.Username)

	// 配置了链路追踪：每个 ES 请求产生 span（不记录查询体）
	traced := clientConfig(cfg, nil, sdktrace.NewTracerProvider())
	assert.NotNil(t, traced.Instrumentation)
}
