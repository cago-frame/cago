package manager

import (
	"bytes"
	"context"
	"encoding/gob"
	"errors"
	"time"

	"github.com/cago-frame/cago/pkg/iam/sessions"
	"github.com/cago-frame/cago/pkg/utils"
	"github.com/redis/go-redis/v9"
)

type redisSessionManager struct {
	prefix string
	redis  *redis.Client
}

func NewRedisSessionManager(prefix string, redis *redis.Client, expireDuration int) sessions.SessionManager {
	return NewExpireSessionManager(expireDuration, &redisSessionManager{
		prefix: prefix,
		redis:  redis,
	})
}

func (r *redisSessionManager) key(id string) string {
	return r.prefix + ":" + id
}

func (r *redisSessionManager) Start(ctx context.Context) (*sessions.Session, error) {
	return &sessions.Session{
		Metadata: make(map[string]interface{}),
		Values:   make(map[string]interface{}),
	}, nil
}

func serialize(session *sessions.Session) ([]byte, error) {
	buf := new(bytes.Buffer)
	enc := gob.NewEncoder(buf)
	err := enc.Encode(session)
	if err == nil {
		return buf.Bytes(), nil
	}
	return nil, err
}

func deserialize(d []byte, session *sessions.Session) error {
	dec := gob.NewDecoder(bytes.NewBuffer(d))
	return dec.Decode(&session)
}

func (r *redisSessionManager) Get(ctx context.Context, id string) (*sessions.Session, error) {
	session := &sessions.Session{}
	data, err := r.redis.Get(ctx, r.key(id)).Bytes()
	if err != nil {
		if errors.Is(err, redis.Nil) {
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

func (r *redisSessionManager) Save(ctx context.Context, session *sessions.Session) error {
	if session.ID == "" {
		session.ID = utils.RandString(32, utils.Mix)
	}
	expires, ok := session.Metadata["expire"].(int64)
	if !ok {
		expires = 86400 + time.Now().Unix()
	} else if expires == 0 {
		expires = 86400 + time.Now().Unix()
	}
	data, err := serialize(session)
	if err != nil {
		return err
	}
	return r.redis.Set(ctx, r.key(session.ID), data, time.Second*time.Duration(expires-time.Now().Unix())).Err()
}

func (r *redisSessionManager) Delete(ctx context.Context, id string) error {
	return r.redis.Del(ctx, r.key(id)).Err()
}

func (r *redisSessionManager) Refresh(ctx context.Context, session *sessions.Session) error {
	// 删除老的
	if err := r.Delete(ctx, session.ID); err != nil {
		return err
	}
	// 重新生成
	session.ID = ""
	return r.Save(ctx, session)
}
