package etcd

import (
	"context"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"go.etcd.io/etcd/api/v3/mvccpb"
	clientv3 "go.etcd.io/etcd/client/v3"
)

// mockKV 模拟etcd KV接口，记录调用次数
type mockKV struct {
	mu       sync.RWMutex
	data     map[string]string
	getCalls atomic.Int64
}

func newMockKV() *mockKV {
	return &mockKV{data: make(map[string]string)}
}

func (m *mockKV) Put(ctx context.Context, key, val string, opts ...clientv3.OpOption) (*clientv3.PutResponse, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.data[key] = val
	return &clientv3.PutResponse{}, nil
}

func (m *mockKV) Get(ctx context.Context, key string, opts ...clientv3.OpOption) (*clientv3.GetResponse, error) {
	m.getCalls.Add(1)
	m.mu.RLock()
	defer m.mu.RUnlock()
	if v, ok := m.data[key]; ok {
		return &clientv3.GetResponse{
			Kvs: []*mvccpb.KeyValue{
				{Key: []byte(key), Value: []byte(v)},
			},
		}, nil
	}
	return &clientv3.GetResponse{}, nil
}

func (m *mockKV) Delete(ctx context.Context, key string, opts ...clientv3.OpOption) (*clientv3.DeleteResponse, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	delete(m.data, key)
	return &clientv3.DeleteResponse{}, nil
}

func (m *mockKV) Compact(ctx context.Context, rev int64, opts ...clientv3.CompactOption) (*clientv3.CompactResponse, error) {
	return &clientv3.CompactResponse{}, nil
}

func (m *mockKV) Do(ctx context.Context, op clientv3.Op) (clientv3.OpResponse, error) {
	return clientv3.OpResponse{}, nil
}

func (m *mockKV) Txn(ctx context.Context) clientv3.Txn {
	return nil
}

// mockWatcher 模拟etcd Watcher接口
type mockWatcher struct {
	mu       sync.Mutex
	channels map[string]chan clientv3.WatchResponse
}

func newMockWatcher() *mockWatcher {
	return &mockWatcher{channels: make(map[string]chan clientv3.WatchResponse)}
}

func (m *mockWatcher) Watch(ctx context.Context, key string, opts ...clientv3.OpOption) clientv3.WatchChan {
	m.mu.Lock()
	defer m.mu.Unlock()
	ch := make(chan clientv3.WatchResponse, 10)
	m.channels[key] = ch
	return ch
}

func (m *mockWatcher) RequestProgress(ctx context.Context) error {
	return nil
}

func (m *mockWatcher) Close() error {
	m.mu.Lock()
	defer m.mu.Unlock()
	for _, ch := range m.channels {
		close(ch)
	}
	return nil
}

// waitChannel 等待指定key的watch channel注册完成
func (m *mockWatcher) waitChannel(key string, timeout time.Duration) chan clientv3.WatchResponse {
	deadline := time.Now().Add(timeout)
	for time.Now().Before(deadline) {
		m.mu.Lock()
		ch := m.channels[key]
		m.mu.Unlock()
		if ch != nil {
			return ch
		}
		time.Sleep(time.Millisecond)
	}
	return nil
}

// sendEvent 向指定key的watch channel发送事件，等待channel就绪
func (m *mockWatcher) sendEvent(t *testing.T, key string, resp clientv3.WatchResponse) {
	ch := m.waitChannel(key, time.Second)
	if ch == nil {
		t.Fatalf("watch channel for %s not registered", key)
	}
	ch <- resp
}

func newMockClientv3(kv *mockKV, w *mockWatcher) *clientv3.Client {
	cli := &clientv3.Client{}
	cli.KV = kv
	cli.Watcher = w
	return cli
}

// TestGetWithoutCache 不启用缓存，每次Get都访问etcd
func TestGetWithoutCache(t *testing.T) {
	kv := newMockKV()
	kv.data["/key1"] = "value1"
	cli := newMockClientv3(kv, newMockWatcher())
	c := NewCacheClient(cli)

	ctx := context.Background()
	resp, err := c.Get(ctx, "/key1")
	assert.NoError(t, err)
	assert.Equal(t, []byte("value1"), resp.Kvs[0].Value)
	assert.Equal(t, int64(1), kv.getCalls.Load())

	// 第二次调用，没有缓存，仍然访问etcd
	resp, err = c.Get(ctx, "/key1")
	assert.NoError(t, err)
	assert.Equal(t, []byte("value1"), resp.Kvs[0].Value)
	assert.Equal(t, int64(2), kv.getCalls.Load())
}

// TestGetWithCache 启用缓存，第二次Get从缓存读取
func TestGetWithCache(t *testing.T) {
	kv := newMockKV()
	kv.data["/key1"] = "value1"
	cli := newMockClientv3(kv, newMockWatcher())
	c := NewCacheClient(cli, WithCache())

	ctx := context.Background()
	resp, err := c.Get(ctx, "/key1")
	assert.NoError(t, err)
	assert.Equal(t, []byte("value1"), resp.Kvs[0].Value)
	assert.Equal(t, int64(1), kv.getCalls.Load())

	// 第二次调用，命中缓存，不再访问etcd
	resp, err = c.Get(ctx, "/key1")
	assert.NoError(t, err)
	assert.Equal(t, []byte("value1"), resp.Kvs[0].Value)
	assert.Equal(t, int64(1), kv.getCalls.Load())
}

// TestGetNotFound key不存在返回空Kvs
func TestGetNotFound(t *testing.T) {
	kv := newMockKV()
	cli := newMockClientv3(kv, newMockWatcher())
	c := NewCacheClient(cli, WithCache())

	resp, err := c.Get(context.Background(), "/nokey")
	assert.NoError(t, err)
	assert.Empty(t, resp.Kvs)
}

// TestPutUpdateCache Put后缓存被更新，后续Get命中缓存
func TestPutUpdateCache(t *testing.T) {
	kv := newMockKV()
	cli := newMockClientv3(kv, newMockWatcher())
	c := NewCacheClient(cli, WithCache())

	ctx := context.Background()
	_, err := c.Put(ctx, "/key1", "val1")
	assert.NoError(t, err)

	// Get应该命中缓存
	resp, err := c.Get(ctx, "/key1")
	assert.NoError(t, err)
	assert.Equal(t, []byte("val1"), resp.Kvs[0].Value)
	assert.Equal(t, int64(0), kv.getCalls.Load())
}

// TestDeleteRemoveCache Delete后缓存被移除，后续Get重新访问etcd
func TestDeleteRemoveCache(t *testing.T) {
	kv := newMockKV()
	kv.data["/key1"] = "value1"
	cli := newMockClientv3(kv, newMockWatcher())
	c := NewCacheClient(cli, WithCache())

	ctx := context.Background()
	// 先读取，缓存住
	_, err := c.Get(ctx, "/key1")
	assert.NoError(t, err)
	assert.Equal(t, int64(1), kv.getCalls.Load())

	// 删除
	_, err = c.Delete(ctx, "/key1")
	assert.NoError(t, err)

	// 再次Get，缓存已被清除，需要访问etcd（此时etcd里也没了）
	resp, err := c.Get(ctx, "/key1")
	assert.NoError(t, err)
	assert.Empty(t, resp.Kvs)
	assert.Equal(t, int64(2), kv.getCalls.Load())
}

// TestWatchUpdateCache Watch收到PUT事件后更新缓存
func TestWatchUpdateCache(t *testing.T) {
	kv := newMockKV()
	kv.data["/key1"] = "old"
	w := newMockWatcher()
	cli := newMockClientv3(kv, w)
	c := NewCacheClient(cli, WithCache())

	ctx := context.Background()
	// 先读取缓存住
	resp, err := c.Get(ctx, "/key1")
	assert.NoError(t, err)
	assert.Equal(t, []byte("old"), resp.Kvs[0].Value)

	// 启动Watch，从返回的channel消费事件
	watchCh := c.Watch(ctx, "/key1")

	// 模拟etcd推送PUT事件
	w.sendEvent(t, "/key1", clientv3.WatchResponse{
		Events: []*clientv3.Event{
			{
				Type: mvccpb.PUT,
				Kv:   &mvccpb.KeyValue{Key: []byte("/key1"), Value: []byte("new")},
			},
		},
	})

	// 等待代理channel转发事件
	select {
	case wresp := <-watchCh:
		assert.Equal(t, mvccpb.PUT, wresp.Events[0].Type)
	case <-time.After(time.Second):
		t.Fatal("等待Watch事件超时")
	}

	// 缓存应该已被更新，不再访问etcd
	resp, err = c.Get(ctx, "/key1")
	assert.NoError(t, err)
	assert.Equal(t, []byte("new"), resp.Kvs[0].Value)
	assert.Equal(t, int64(1), kv.getCalls.Load()) // 仍然只有最初那一次
}

// TestWatchDeleteCache Watch收到DELETE事件后移除缓存
func TestWatchDeleteCache(t *testing.T) {
	kv := newMockKV()
	kv.data["/key1"] = "value1"
	w := newMockWatcher()
	cli := newMockClientv3(kv, w)
	c := NewCacheClient(cli, WithCache())

	ctx := context.Background()
	_, err := c.Get(ctx, "/key1")
	assert.NoError(t, err)

	watchCh := c.Watch(ctx, "/key1")

	// 模拟etcd推送DELETE事件
	w.sendEvent(t, "/key1", clientv3.WatchResponse{
		Events: []*clientv3.Event{
			{
				Type: mvccpb.DELETE,
				Kv:   &mvccpb.KeyValue{Key: []byte("/key1")},
			},
		},
	})

	select {
	case wresp := <-watchCh:
		assert.Equal(t, mvccpb.DELETE, wresp.Events[0].Type)
	case <-time.After(time.Second):
		t.Fatal("等待Watch事件超时")
	}

	// 缓存已清除，Get会重新访问etcd
	resp, err := c.Get(ctx, "/key1")
	assert.NoError(t, err)
	assert.Equal(t, int64(2), kv.getCalls.Load())
	assert.Equal(t, []byte("value1"), resp.Kvs[0].Value)
}

// TestWatchWithoutCache 不启用缓存时Watch直接返回原始channel
func TestWatchWithoutCache(t *testing.T) {
	kv := newMockKV()
	w := newMockWatcher()
	cli := newMockClientv3(kv, w)
	c := NewCacheClient(cli)

	watchCh := c.Watch(context.Background(), "/key1")

	w.sendEvent(t, "/key1", clientv3.WatchResponse{
		Events: []*clientv3.Event{
			{
				Type: mvccpb.PUT,
				Kv:   &mvccpb.KeyValue{Key: []byte("/key1"), Value: []byte("v")},
			},
		},
	})

	select {
	case wresp := <-watchCh:
		assert.Equal(t, mvccpb.PUT, wresp.Events[0].Type)
	case <-time.After(time.Second):
		t.Fatal("等待Watch事件超时")
	}

	// 没有缓存，cache应为nil
	assert.Nil(t, c.cache)
}
