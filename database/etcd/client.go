package etcd

import (
	"context"
	"sync"

	"go.etcd.io/etcd/api/v3/mvccpb"
	clientv3 "go.etcd.io/etcd/client/v3"
)

type Option func(*Client)

// WithCache 启用缓存，读取过的key会缓存到内存中，
// 配合Watch使用时会自动更新缓存。
func WithCache() Option {
	return func(c *Client) {
		c.enableCache = true
		c.cache = make(map[string][]byte)
	}
}

// Client 封装etcd客户端，可通过WithCache启用缓存。
// 方法签名与clientv3.Client保持一致。
type Client struct {
	*clientv3.Client
	enableCache bool
	mu          sync.RWMutex
	cache       map[string][]byte
}

// NewCacheClient 创建etcd客户端封装
func NewCacheClient(cli *clientv3.Client, opts ...Option) *Client {
	c := &Client{Client: cli}
	for _, o := range opts {
		o(c)
	}
	return c
}

// Get 获取key的值，启用缓存时优先从缓存读取
func (c *Client) Get(ctx context.Context, key string, opts ...clientv3.OpOption) (*clientv3.GetResponse, error) {
	if c.enableCache {
		c.mu.RLock()
		if v, ok := c.cache[key]; ok {
			c.mu.RUnlock()
			return &clientv3.GetResponse{
				Kvs: []*mvccpb.KeyValue{
					{Key: []byte(key), Value: v},
				},
			}, nil
		}
		c.mu.RUnlock()
	}

	resp, err := c.Client.Get(ctx, key, opts...)
	if err != nil {
		return nil, err
	}
	if c.enableCache && len(resp.Kvs) > 0 {
		c.mu.Lock()
		c.cache[key] = resp.Kvs[0].Value
		c.mu.Unlock()
	}
	return resp, nil
}

// Put 写入key的值，启用缓存时在锁内完成写入和缓存更新，保证一致性
func (c *Client) Put(ctx context.Context, key, val string, opts ...clientv3.OpOption) (*clientv3.PutResponse, error) {
	if c.enableCache {
		c.mu.Lock()
		defer c.mu.Unlock()
	}
	resp, err := c.Client.Put(ctx, key, val, opts...)
	if err != nil {
		return nil, err
	}
	if c.enableCache {
		c.cache[key] = []byte(val)
	}
	return resp, nil
}

// Delete 删除key，启用缓存时先移除缓存再删除etcd，避免中间窗口读到已删除的值
func (c *Client) Delete(ctx context.Context, key string, opts ...clientv3.OpOption) (*clientv3.DeleteResponse, error) {
	if c.enableCache {
		c.mu.Lock()
		delete(c.cache, key)
		c.mu.Unlock()
	}
	resp, err := c.Client.Delete(ctx, key, opts...)
	if err != nil {
		return nil, err
	}
	return resp, nil
}

// Watch 监听key变化，启用缓存时通过代理channel自动更新缓存
func (c *Client) Watch(ctx context.Context, key string, opts ...clientv3.OpOption) clientv3.WatchChan {
	origCh := c.Client.Watch(ctx, key, opts...)
	if !c.enableCache {
		return origCh
	}
	proxyCh := make(chan clientv3.WatchResponse)
	go func() {
		defer close(proxyCh)
		for resp := range origCh {
			for _, ev := range resp.Events {
				switch ev.Type {
				case clientv3.EventTypePut:
					c.mu.Lock()
					c.cache[key] = ev.Kv.Value
					c.mu.Unlock()
				case clientv3.EventTypeDelete:
					c.mu.Lock()
					delete(c.cache, key)
					c.mu.Unlock()
				}
			}
			proxyCh <- resp
		}
	}()
	return proxyCh
}
