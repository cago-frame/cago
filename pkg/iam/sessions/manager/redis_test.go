package manager

import (
	"context"
	"testing"
	"time"

	"github.com/cago-frame/cago/pkg/iam/sessions"
	"github.com/redis/go-redis/v9"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

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

// Refresh 会换一个新 ID：旧 key 删掉，新 key 写入，过期时间重置
func TestRedisSessionManager_Refresh(t *testing.T) {
	m := miniredis.RunT(t)
	db := redis.NewClient(&redis.Options{Addr: m.Addr()})
	sm := NewRedisSessionManager("aa", db, 60)
	ctx := context.Background()

	session, err := sm.Start(ctx)
	require.NoError(t, err)
	session.Values["user"] = "tom"
	require.NoError(t, sm.Save(ctx, session))
	oldID := session.ID

	require.NoError(t, sm.Refresh(ctx, session))
	assert.NotEqual(t, oldID, session.ID)
	assert.False(t, m.Exists("aa:"+oldID))
	assert.True(t, m.Exists("aa:"+session.ID))

	refreshed, err := sm.Get(ctx, session.ID)
	require.NoError(t, err)
	assert.Equal(t, "tom", refreshed.Values["user"])
}

// 存储中的数据损坏时应报错，而不是返回一个半初始化的会话
func TestRedisSessionManager_GetBadPayload(t *testing.T) {
	m := miniredis.RunT(t)
	db := redis.NewClient(&redis.Options{Addr: m.Addr()})
	sm := NewRedisSessionManager("aa", db, 60)

	require.NoError(t, m.Set("aa:sid", "not a gob payload"))

	session, err := sm.Get(context.Background(), "sid")
	assert.Error(t, err)
	assert.Nil(t, session)
}

// redis 报错时应原样透出，不能被误判成“会话不存在”
func TestRedisSessionManager_GetError(t *testing.T) {
	m := miniredis.RunT(t)
	db := redis.NewClient(&redis.Options{Addr: m.Addr()})
	sm := NewRedisSessionManager("aa", db, 60)

	m.SetError("LOADING Redis is loading the dataset in memory")
	session, err := sm.Get(context.Background(), "sid")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "LOADING")
	assert.Nil(t, session)
}
