package manager

import (
	"context"
	"strconv"
	"time"

	"github.com/cago-frame/cago/pkg/iam/sessions"
)

type expireSessionManager struct {
	sessions.SessionManager
	// 过期时间
	expireDuration int
}

// NewExpireSessionManager 创建过期session管理器
// expireDuration 过期时间 单位为秒
// 当session过期时，get会返回原session并且报错
// 刷新和保存时会刷新过期时间
func NewExpireSessionManager(expireDuration int, sessionManager sessions.SessionManager) sessions.SessionManager {
	return &expireSessionManager{
		expireDuration: expireDuration,
		SessionManager: sessionManager,
	}
}

func (e *expireSessionManager) Save(ctx context.Context, session *sessions.Session) error {
	if _, ok := session.Metadata["expire"]; !ok {
		session.Metadata["expire"] = time.Now().
			Add(time.Duration(e.expireDuration) * time.Second).Unix()
	}
	return e.SessionManager.Save(ctx, session)
}

func (e *expireSessionManager) Refresh(ctx context.Context, session *sessions.Session) error {
	session.Metadata["expire"] = time.Now().
		Add(time.Duration(e.expireDuration) * time.Second).Unix()
	return e.SessionManager.Refresh(ctx, session)
}

func (e *expireSessionManager) Get(ctx context.Context, id string) (*sessions.Session, error) {
	session, err := e.SessionManager.Get(ctx, id)
	if err != nil {
		return nil, err
	}
	var t int64
	switch expire := session.Metadata["expire"].(type) {
	case string:
		t, err = strconv.ParseInt(expire, 10, 64)
		session.Metadata["expire"] = t
	case int:
		t = int64(expire)
		session.Metadata["expire"] = t
	case int64:
		t = expire
	case float64:
		t = int64(expire)
		session.Metadata["expire"] = t
	default:
		return nil, sessions.ErrSessionExpired
	}
	if err != nil {
		return nil, err
	}
	if time.Now().Unix() > t {
		return nil, sessions.ErrSessionExpired
	}
	return session, nil
}
