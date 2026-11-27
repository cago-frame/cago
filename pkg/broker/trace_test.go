package broker

import (
	"context"
	"errors"
	"strings"
	"testing"
	"time"

	broker2 "github.com/cago-frame/cago/pkg/broker/broker"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/propagation"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	"go.opentelemetry.io/otel/sdk/trace/tracetest"
	"go.opentelemetry.io/otel/trace"
)

// memBroker 把 Publish 的消息原样交给已订阅的 handler，用来观察 Header 是否携带了链路上下文
type memBroker struct {
	broker2.Broker
	published  *broker2.Message
	publishErr error
	handler    broker2.Handler
}

func (m *memBroker) Publish(_ context.Context, _ string, data *broker2.Message, _ ...broker2.PublishOption) error {
	m.published = data
	return m.publishErr
}

func (m *memBroker) Subscribe(_ context.Context, _ string, h broker2.Handler, _ ...broker2.SubscribeOption) (broker2.Subscriber, error) {
	m.handler = h
	return nil, nil
}

func (m *memBroker) String() string { return "mem" }

type memEvent struct {
	broker2.Event
	topic     string
	msg       *broker2.Message
	attempted int
}

func (e *memEvent) Topic() string               { return e.topic }
func (e *memEvent) Message() *broker2.Message   { return e.msg }
func (e *memEvent) Attempted() int              { return e.attempted }
func (e *memEvent) Requeue(time.Duration) error { return nil }

func newTracedBroker(t *testing.T) (*memBroker, broker2.Broker, *tracetest.SpanRecorder) {
	t.Helper()
	prev := otel.GetTextMapPropagator()
	otel.SetTextMapPropagator(propagation.TraceContext{})
	t.Cleanup(func() { otel.SetTextMapPropagator(prev) })

	rec := tracetest.NewSpanRecorder()
	tp := sdktrace.NewTracerProvider(sdktrace.WithSpanProcessor(rec))
	mem := &memBroker{}
	b, err := New(WithBroker(mem), WithTracer(tp.Tracer("test")))
	require.NoError(t, err)
	return mem, b, rec
}

func spanByKind(spans []sdktrace.ReadOnlySpan, kind trace.SpanKind) sdktrace.ReadOnlySpan {
	for _, s := range spans {
		if s.SpanKind() == kind {
			return s
		}
	}
	return nil
}

func attr(s sdktrace.ReadOnlySpan, key attribute.Key) (attribute.Value, bool) {
	for _, kv := range s.Attributes() {
		if kv.Key == key {
			return kv.Value, true
		}
	}
	return attribute.Value{}, false
}

func TestTrace_PublishInjectsContextAndConsumerContinuesTrace(t *testing.T) {
	mem, b, rec := newTracedBroker(t)

	var handlerCtx context.Context
	_, err := b.Subscribe(context.Background(), "script.create", func(ctx context.Context, _ broker2.Event) error {
		handlerCtx = ctx
		return nil
	})
	require.NoError(t, err)

	// 旧调用方传入 Header 为 nil 的消息
	require.NoError(t, b.Publish(context.Background(), "script.create", &broker2.Message{Body: []byte("x")}))
	require.NotNil(t, mem.published.Header)
	assert.NotEmpty(t, mem.published.Header["traceparent"], "Publish 应把链路上下文注入消息头")

	require.NoError(t, mem.handler(context.Background(), &memEvent{topic: "script.create", msg: mem.published, attempted: 2}))

	producer := spanByKind(rec.Ended(), trace.SpanKindProducer)
	consumer := spanByKind(rec.Ended(), trace.SpanKindConsumer)
	require.NotNil(t, producer)
	require.NotNil(t, consumer)
	assert.Equal(t, producer.SpanContext().TraceID(), consumer.SpanContext().TraceID(), "消费者应接上生产者的 trace")
	assert.Equal(t, producer.SpanContext().SpanID(), consumer.Parent().SpanID())
	assert.True(t, strings.Contains(consumer.Name(), "script.create"), "消费 span 名应带 topic，实际 %q", consumer.Name())
	assert.True(t, strings.Contains(producer.Name(), "script.create"), "生产 span 名应带 topic，实际 %q", producer.Name())
	v, ok := attr(consumer, "messaging.attempted")
	require.True(t, ok)
	assert.EqualValues(t, 2, v.AsInt64())
	assert.Equal(t, codes.Unset, consumer.Status().Code)
	assert.True(t, trace.SpanFromContext(handlerCtx).SpanContext().IsValid(), "handler 拿到的 ctx 应带消费 span")
}

func TestTrace_HandlerErrorMarksConsumerSpanError(t *testing.T) {
	mem, b, rec := newTracedBroker(t)
	_, err := b.Subscribe(context.Background(), "script.create", func(context.Context, broker2.Event) error {
		return errors.New("boom")
	})
	require.NoError(t, err)

	// 没有 Header 的旧消息也要能正常消费
	err = mem.handler(context.Background(), &memEvent{topic: "script.create", msg: &broker2.Message{Body: []byte("x")}, attempted: 1})
	assert.EqualError(t, err, "boom")

	consumer := spanByKind(rec.Ended(), trace.SpanKindConsumer)
	require.NotNil(t, consumer)
	assert.Equal(t, codes.Error, consumer.Status().Code)
	require.NotEmpty(t, consumer.Events())
	assert.Equal(t, "exception", consumer.Events()[0].Name)
}

func TestTrace_PublishErrorMarksProducerSpanError(t *testing.T) {
	mem, b, rec := newTracedBroker(t)
	mem.publishErr = errors.New("nsq down")

	err := b.Publish(context.Background(), "script.create", &broker2.Message{Body: []byte("x")})
	assert.EqualError(t, err, "nsq down")

	producer := spanByKind(rec.Ended(), trace.SpanKindProducer)
	require.NotNil(t, producer)
	assert.Equal(t, codes.Error, producer.Status().Code)
}

func TestTrace_NoTracerKeepsBehaviour(t *testing.T) {
	mem := &memBroker{}
	b, err := New(WithBroker(mem))
	require.NoError(t, err)
	var called bool
	_, err = b.Subscribe(context.Background(), "t", func(context.Context, broker2.Event) error {
		called = true
		return nil
	})
	require.NoError(t, err)
	require.NoError(t, b.Publish(context.Background(), "t", &broker2.Message{Body: []byte("x")}))
	require.NoError(t, mem.handler(context.Background(), &memEvent{topic: "t", msg: mem.published}))
	assert.True(t, called)
	assert.Nil(t, mem.published.Header, "未配置链路追踪时不应改动消息")
}
