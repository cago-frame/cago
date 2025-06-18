package manager

import (
	"context"
	"errors"
	"net/http"
	"time"

	"github.com/cago-frame/cago/database/cache/cache"
	"github.com/cago-frame/cago/pkg/iam/sessions"
	"github.com/cago-frame/cago/pkg/utils/httputils"
	"github.com/gin-gonic/gin"
)

var (
	ErrRefreshTokenNotFound = errors.New("refresh token not found")
)

type RefreshHTTPSessionManagerOptions struct {
	// token有效期
	TokenDuration int
	// 刷新有效期
	RefreshDuration int
	// AccessTokenMapping refresh token映射
	AccessTokenMapping AccessTokenMapping
	// ResponseFunc 返回函数
	ResponseFunc func(ctx *gin.Context, accessToken, refreshToken *sessions.Session) error
	// GetRefreshToken 获取refresh token
	GetRefreshToken func(ctx *gin.Context) (string, error)
}

type RefreshHTTPSessionManagerOption func(*RefreshHTTPSessionManagerOptions)

// refreshHTTPSessionManager 刷新session管理器
type refreshHTTPSessionManager struct {
	sessions.SessionManager
	originSessionManager  sessions.SessionManager
	refreshSessionManager sessions.SessionManager
	options               *RefreshHTTPSessionManagerOptions
}

// NewRefreshHTTPSessionManager 创建一个拥有两个token的session管理器
// 拥有一个access_token和一个refresh_token 用于token双刷
func NewRefreshHTTPSessionManager(sessionManager, refreshSessionManager sessions.SessionManager,
	opts ...RefreshHTTPSessionManagerOption) sessions.HTTPSessionManager {
	options := &RefreshHTTPSessionManagerOptions{
		TokenDuration:   7200,
		RefreshDuration: 86400 * 30,
		ResponseFunc: func(ctx *gin.Context, accessToken, refreshToken *sessions.Session) error {
			ctx.SetCookie("access_token", accessToken.ID,
				int(accessToken.Metadata["expire"].(int64)), "/", "", false, true)
			ctx.JSON(http.StatusOK, httputils.JSONResponse{
				Code: 0,
				Data: &RefreshSessionResponse{
					AccessToken:   accessToken.ID,
					RefreshToken:  refreshToken.ID,
					Expire:        accessToken.Metadata["expire"].(int64),
					RefreshExpire: refreshToken.Metadata["expire"].(int64),
				},
			})
			return nil
		},
		GetRefreshToken: func(ctx *gin.Context) (string, error) {
			return ctx.PostForm("refresh_token"), nil
		},
	}
	for _, opt := range opts {
		opt(options)
	}
	return &refreshHTTPSessionManager{
		SessionManager:        NewExpireSessionManager(options.TokenDuration, sessionManager),
		originSessionManager:  sessionManager,
		refreshSessionManager: NewExpireSessionManager(options.RefreshDuration, refreshSessionManager),
		options:               options,
	}
}

func copyMap(src map[string]interface{}) map[string]interface{} {
	dst := make(map[string]interface{})
	for k, v := range src {
		dst[k] = v
	}
	return dst
}

// Delete 删除session使其失效
// 删除access_token时同时删除refresh_token
// access_token与refresh_token的映射关系需要使用其它方式维护，需要设置AccessTokenMapping，如果不设置则不会删除refresh_token
func (h *refreshHTTPSessionManager) Delete(ctx context.Context, id string) error {
	_, err := h.SessionManager.Get(ctx, id)
	if err != nil {
		return err
	}
	if h.options.AccessTokenMapping != nil {
		refreshId, err := h.options.AccessTokenMapping.Get(ctx, id)
		if err != nil {
			return err
		}
		if err := h.refreshSessionManager.Delete(ctx, refreshId); err != nil {
			return err
		}
		if err := h.options.AccessTokenMapping.Delete(ctx, id); err != nil {
			return err
		}
	}
	return h.SessionManager.Delete(ctx, id)
}

func (h *refreshHTTPSessionManager) GetFromRequest(ctx *gin.Context) (*sessions.Session, error) {
	id, err := ctx.Cookie("access_token")
	if err != nil {
		return nil, sessions.ErrSessionNotFound
	}
	session, err := h.SessionManager.Get(ctx, id)
	if err != nil {
		// 如果access_token不存在，判断是否有refresh_token 并校验是否有效
		refreshID := ctx.PostForm("refresh_token")
		if refreshID != "" {
			// 有refresh_token，获取refresh session
			refreshSession, err := h.refreshSessionManager.Get(ctx, refreshID)
			if err != nil {
				return nil, err
			}
			if refreshSession.Values["access_token"].(string) != id {
				return nil, sessions.ErrSessionNotFound
			}
			session, err = h.SessionManager.Start(ctx)
			if err != nil {
				return nil, err
			}
			session.ID = id
			session.Values = copyMap(refreshSession.Values)
			delete(session.Values, "access_token")
			return session, err
		}
		return nil, err
	}
	return session, nil
}

type RefreshSessionResponse struct {
	AccessToken   string `json:"access_token"`
	RefreshToken  string `json:"refresh_token"`
	Expire        int64  `json:"expire"`
	RefreshExpire int64  `json:"refresh_expire"`
}

func (h *refreshHTTPSessionManager) SaveToResponse(ctx *gin.Context, session *sessions.Session) error {
	if session.ID == "" {
		// 如果是新的session，创建一个refresh session与之绑定
		refreshSession, err := h.refreshSessionManager.Start(ctx)
		if err != nil {
			return err
		}
		if err := h.SessionManager.Save(ctx, session); err != nil {
			return err
		}
		refreshSession.Values = copyMap(session.Values)
		refreshSession.Values["access_token"] = session.ID
		if err := h.refreshSessionManager.Save(ctx, refreshSession); err != nil {
			return err
		}
		// 设置映射关系
		if h.options.AccessTokenMapping != nil {
			if err := h.options.AccessTokenMapping.Set(ctx, session.ID,
				refreshSession.ID, time.Duration(h.options.RefreshDuration)*time.Second); err != nil {
				return err
			}
		}
		// 返回
		return h.options.ResponseFunc(ctx, session, refreshSession)
	}
	// 判断有没有刷新token，如果有则做一次刷新
	refreshID, err := h.options.GetRefreshToken(ctx)
	if err != nil {
		return err
	}
	if refreshID != "" {
		refreshSession, err := h.refreshSessionManager.Get(ctx, refreshID)
		if err != nil {
			return err
		}
		oldSessionId := session.ID
		// 刷新access token
		if err := h.SessionManager.Refresh(ctx, session); err != nil {
			return err
		}
		// 刷新refresh token
		refreshSession.Values = copyMap(session.Values)
		refreshSession.Values["access_token"] = session.ID
		if err := h.refreshSessionManager.Refresh(ctx, refreshSession); err != nil {
			return err
		}
		// 删除老的映射关系
		if h.options.AccessTokenMapping != nil {
			if err := h.options.AccessTokenMapping.Delete(ctx, oldSessionId); err != nil {
				return err
			}
			// 设置新的映射关系
			if err := h.options.AccessTokenMapping.Set(ctx, session.ID,
				refreshSession.ID, time.Duration(h.options.RefreshDuration)*time.Second); err != nil {
				return err
			}
		}
		// 返回
		return h.options.ResponseFunc(ctx, session, refreshSession)
	}
	return ErrRefreshTokenNotFound
}

// AccessTokenMapping access_token与refresh_token的映射关系
type AccessTokenMapping interface {
	// Set 设置refresh_token
	Set(ctx context.Context, accessToken, refreshToken string, duration time.Duration) error
	// Get 获取refresh_token
	Get(ctx context.Context, accessToken string) (string, error)
	// Delete 删除refresh_token
	Delete(ctx context.Context, accessToken string) error
}

type cacheAccessTokenMapping struct {
	cache cache.Cache
}

func NewCacheAccessTokenMapping(cache cache.Cache) AccessTokenMapping {
	return &cacheAccessTokenMapping{
		cache: cache,
	}
}

func (m *cacheAccessTokenMapping) Set(ctx context.Context, accessToken, refreshToken string, duration time.Duration) error {
	return m.cache.Set(ctx, "access_token:"+accessToken, refreshToken, cache.Expiration(duration)).Err()
}

func (m *cacheAccessTokenMapping) Get(ctx context.Context, accessToken string) (string, error) {
	return m.cache.Get(ctx, "access_token:"+accessToken).Result()
}

func (m *cacheAccessTokenMapping) Delete(ctx context.Context, accessToken string) error {
	return m.cache.Del(ctx, "access_token:"+accessToken)
}
