package manager

import (
	"context"
	"testing"
	"time"

	"github.com/cago-frame/cago/pkg/iam/sessions"
	"github.com/redis/go-redis/v9"
	"github.com/stretchr/testify/assert"

	miniredis "github.com/alicebob/miniredis/v2"
)

func TestNewRedisSessionManager(t *testing.T) {
	m := miniredis.RunT(t)
	db := redis.NewClient(&redis.Options{
		Addr: m.Addr(),
	})
	testExpireSession(t, NewRedisSessionManager("aa", db, 60))
}

func testExpireSession(t *testing.T, sm sessions.SessionManager) {
	ctx := context.Background()
	session, err := sm.Start(ctx)
	assert.NoError(t, err)
	assert.NotNil(t, session)
	session.Values["int64"] = int64(1)
	session.Values["string"] = "string"
	session.Values["float64"] = 1.1
	session.Values["bool"] = true
	session.Values["nil"] = nil
	err = sm.Save(ctx, session)
	assert.NoError(t, err)
	// 读取
	session2, err := sm.Get(ctx, session.ID)
	assert.NoError(t, err)
	assert.NotNil(t, session2)
	assert.Equal(t, int64(1), session2.Values["int64"])
	assert.Equal(t, "string", session2.Values["string"])
	assert.Equal(t, 1.1, session2.Values["float64"])
	assert.Equal(t, true, session2.Values["bool"])
	v, ok := session2.Values["nil"]
	assert.True(t, ok)
	assert.Nil(t, v)
	// 删除
	err = sm.Delete(ctx, session.ID)
	assert.NoError(t, err)
	// 读取
	session3, err := sm.Get(ctx, session.ID)
	assert.Equal(t, sessions.ErrSessionNotFound, err)
	assert.Nil(t, session3)

	// 过期测试
	session, err = sm.Start(ctx)
	assert.NoError(t, err)
	session.Metadata["expire"] = time.Now().Unix() - 10
	err = sm.Save(ctx, session)
	assert.NoError(t, err)
	session2, err = sm.Get(ctx, session.ID)
	assert.Equal(t, sessions.ErrSessionExpired, err)
	assert.Nil(t, session2)
}
