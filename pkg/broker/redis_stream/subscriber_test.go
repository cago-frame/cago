package redis_stream

import (
	"context"
	"errors"
	"sync/atomic"
	"testing"
	"time"

	"github.com/alicebob/miniredis/v2"
	broker2 "github.com/cago-frame/cago/pkg/broker/broker"
	"github.com/redis/go-redis/v9"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// newTestBroker 起一个 miniredis 并返回连到它的 broker
func newTestBroker(t *testing.T, cfg Config) (broker2.Broker, *redis.Client) {
	t.Helper()
	mr := miniredis.RunT(t)
	cfg.Addr = mr.Addr()
	if cfg.Block == 0 {
		cfg.Block = 50 * time.Millisecond
	}
	b, err := NewBroker(cfg)
	require.Nil(t, err)
	t.Cleanup(func() { _ = b.Close() })
	return b, redis.NewClient(&redis.Options{Addr: mr.Addr()})
}

func TestSubscribe_ReceivesPublishedMessage(t *testing.T) {
	ctx := context.Background()
	b, cli := newTestBroker(t, Config{})

	got := make(chan *broker2.Message, 1)
	sub, err := b.Subscribe(ctx, "orders", func(_ context.Context, e broker2.Event) error {
		got <- e.Message()
		return nil
	}, broker2.Group("g1"))
	require.Nil(t, err)
	defer func() { _ = sub.Unsubscribe() }()
	assert.Equal(t, "orders", sub.Topic())

	require.Nil(t, b.Publish(ctx, "orders", &broker2.Message{
		Header: map[string]string{"traceparent": "tp"},
		Body:   []byte("hello"),
	}))

	select {
	case msg := <-got:
		assert.Equal(t, []byte("hello"), msg.Body)
		assert.Equal(t, "tp", msg.Header["traceparent"])
	case <-time.After(3 * time.Second):
		t.Fatal("超时未收到消息")
	}

	// 成功处理后必须 XACK，pending 列表应清空
	assert.Eventually(t, func() bool {
		p, err := cli.XPending(ctx, "orders", "g1").Result()
		return err == nil && p.Count == 0
	}, 3*time.Second, 20*time.Millisecond)
}

func TestSubscribe_RequiresGroup(t *testing.T) {
	b, _ := newTestBroker(t, Config{})
	_, err := b.Subscribe(context.Background(), "orders", func(context.Context, broker2.Event) error {
		return nil
	})
	assert.NotNil(t, err)
	assert.Contains(t, err.Error(), "Group")
}

// 消费组已存在是正常情况，第二次 Subscribe 不应因 BUSYGROUP 失败
func TestSubscribe_BusyGroupIgnored(t *testing.T) {
	b, _ := newTestBroker(t, Config{})
	handler := func(context.Context, broker2.Event) error { return nil }

	first, err := b.Subscribe(context.Background(), "orders", handler, broker2.Group("g1"))
	require.Nil(t, err)
	defer func() { _ = first.Unsubscribe() }()

	second, err := b.Subscribe(context.Background(), "orders", handler, broker2.Group("g1"))
	require.Nil(t, err)
	require.Nil(t, second.Unsubscribe())
}

// Retry 语义：handler 失败不 Ack，消息留在 pending，由后台 claimer 重新投递
func TestSubscribe_RetryRedelivers(t *testing.T) {
	ctx := context.Background()
	b, _ := newTestBroker(t, Config{
		ClaimMinIdle:  10 * time.Millisecond,
		ClaimInterval: 20 * time.Millisecond,
	})

	var calls atomic.Int32
	done := make(chan struct{})
	sub, err := b.Subscribe(ctx, "orders", func(_ context.Context, _ broker2.Event) error {
		if calls.Add(1) == 1 {
			return errors.New("boom")
		}
		close(done)
		return nil
	}, broker2.Group("g1"), broker2.Retry())
	require.Nil(t, err)
	defer func() { _ = sub.Unsubscribe() }()

	require.Nil(t, b.Publish(ctx, "orders", &broker2.Message{Body: []byte("x")}))

	select {
	case <-done:
	case <-time.After(5 * time.Second):
		t.Fatal("超时未发生重投递")
	}
	assert.GreaterOrEqual(t, calls.Load(), int32(2))
}

// 失败且未开启 Retry 时应直接 Ack 丢弃，避免 pending 无限堆积
func TestSubscribe_NoRetryAcksOnFailure(t *testing.T) {
	ctx := context.Background()
	b, cli := newTestBroker(t, Config{})

	called := make(chan struct{}, 1)
	sub, err := b.Subscribe(ctx, "orders", func(_ context.Context, _ broker2.Event) error {
		called <- struct{}{}
		return errors.New("boom")
	}, broker2.Group("g1"))
	require.Nil(t, err)
	defer func() { _ = sub.Unsubscribe() }()

	require.Nil(t, b.Publish(ctx, "orders", &broker2.Message{Body: []byte("x")}))
	select {
	case <-called:
	case <-time.After(3 * time.Second):
		t.Fatal("超时未收到消息")
	}
	assert.Eventually(t, func() bool {
		p, err := cli.XPending(ctx, "orders", "g1").Result()
		return err == nil && p.Count == 0
	}, 3*time.Second, 20*time.Millisecond)
}

func TestPublish_TrimsToMaxLen(t *testing.T) {
	ctx := context.Background()
	b, cli := newTestBroker(t, Config{MaxLen: 2})
	for range 5 {
		require.Nil(t, b.Publish(ctx, "orders", &broker2.Message{Body: []byte("x")}))
	}
	n, err := cli.XLen(ctx, "orders").Result()
	require.Nil(t, err)
	assert.LessOrEqual(t, n, int64(2))
}

// Addr 为空时应使用 SetClient 注入的客户端，且 Close 不关闭它
func TestNewBroker_UsesInjectedClient(t *testing.T) {
	mr := miniredis.RunT(t)
	cli := redis.NewClient(&redis.Options{Addr: mr.Addr()})
	SetClient(cli)
	t.Cleanup(func() { SetClient(nil) })

	b, err := NewBroker(Config{})
	require.Nil(t, err)
	require.Nil(t, b.Publish(context.Background(), "orders", &broker2.Message{Body: []byte("x")}))
	require.Nil(t, b.Close())

	// broker.Close 不应关闭注入的客户端
	assert.Nil(t, cli.Ping(context.Background()).Err())
}

// 建组失败（且不是 BUSYGROUP）应直接让 Subscribe 失败
func TestSubscribe_CreateGroupError(t *testing.T) {
	mr := miniredis.RunT(t)
	b, err := NewBroker(Config{Addr: mr.Addr(), Block: 50 * time.Millisecond})
	require.Nil(t, err)
	t.Cleanup(func() { _ = b.Close() })

	mr.SetError("LOADING Redis is loading the dataset in memory")
	_, err = b.Subscribe(context.Background(), "orders", func(context.Context, broker2.Event) error {
		return nil
	}, broker2.Group("g1"))
	require.NotNil(t, err)
	assert.Contains(t, err.Error(), "create consumer group")
}

// XADD 失败时 Publish 应把错误透出给调用方
func TestPublish_Error(t *testing.T) {
	mr := miniredis.RunT(t)
	b, err := NewBroker(Config{Addr: mr.Addr(), Block: 50 * time.Millisecond})
	require.Nil(t, err)
	t.Cleanup(func() { _ = b.Close() })

	mr.SetError("LOADING Redis is loading the dataset in memory")
	err = b.Publish(context.Background(), "orders", &broker2.Message{Body: []byte("x")})
	require.NotNil(t, err)
	assert.Contains(t, err.Error(), "LOADING")
}
