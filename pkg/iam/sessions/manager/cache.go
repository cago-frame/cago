package manager

import (
	"context"
	"errors"
	"time"

	"github.com/cago-frame/cago/database/cache/cache"
	"github.com/cago-frame/cago/pkg/iam/sessions"
	"github.com/cago-frame/cago/pkg/utils"
)

type cacheSessionManager struct {
	cache cache.Cache
}

// NewCacheSessionManagerWithExpire 带上过期时间并且基于缓存组件的session管理器
func NewCacheSessionManagerWithExpire(cache cache.Cache, expireDuration int) sessions.SessionManager {
	return NewExpireSessionManager(expireDuration, &cacheSessionManager{
		cache: cache,
	})
}

// NewCacheSessionManager 基于缓存组件的session管理器
func NewCacheSessionManager(cache cache.Cache) sessions.SessionManager {
	return &cacheSessionManager{
		cache: cache,
	}
}

func (c *cacheSessionManager) key(id string) string {
	return "session:" + id
}

func (c *cacheSessionManager) Start(ctx context.Context) (*sessions.Session, error) {
	return &sessions.Session{
		Metadata: make(map[string]interface{}),
		Values:   make(map[string]interface{}),
	}, nil
}

func (c *cacheSessionManager) Get(ctx context.Context, id string) (*sessions.Session, error) {
	session := &sessions.Session{}
	data, err := c.cache.Get(ctx, c.key(id)).Bytes()
	if err != nil {
		if errors.Is(err, cache.ErrNil) {
			return nil, sessions.ErrSessionNotFound
		}
		return nil, err
	}
	err = deserialize(data, session)
	if err != nil {
		return nil, err
	}
	return session, nil
}

func (c *cacheSessionManager) Save(ctx context.Context, session *sessions.Session) error {
	if session.ID == "" {
		session.ID = utils.RandString(32, utils.Mix)
	}
	expire, ok := session.Metadata["expire"].(int64)
	if !ok || expire == 0 {
		expire = 86400
	} else {
		expire = expire - time.Now().Unix()
	}
	data, err := serialize(session)
	if err != nil {
		return err
	}
	return c.cache.Set(ctx, c.key(session.ID), data, cache.Expiration(time.Second*time.Duration(expire))).Err()
}

func (c *cacheSessionManager) Delete(ctx context.Context, id string) error {
	return c.cache.Del(ctx, c.key(id))
}

func (c *cacheSessionManager) Refresh(ctx context.Context, session *sessions.Session) error {
	// 删除老的
	if err := c.Delete(ctx, session.ID); err != nil {
		return err
	}
	// 重新生成
	session.ID = ""
	return c.Save(ctx, session)
}
