package authn

import (
	"context"
	"errors"

	"github.com/cago-frame/cago/pkg/iam/sessions"
	"github.com/cago-frame/cago/pkg/utils/httputils"
	"github.com/gin-gonic/gin"
)

type key int

const (
	authnSession key = iota
)

const (
	userIdKey       = "user_id"
	usernameKey     = "username"
	refreshTokenKey = "refresh_token"
)

// Authn 是一个认证模块 集成了用户登录、注册、认证等功能
// 你可以使用Middleware中间件来对你的请求进行认证
type Authn struct {
	options  *Options
	database Database
}

// New 创建一个Authn
func New(database Database, opts ...Option) *Authn {
	options := newOptions(opts...)
	return &Authn{
		options:  options,
		database: database,
	}
}

var defaultAuthn *Authn

// SetDefault 设置默认的Authn
func SetDefault(authn *Authn) {
	defaultAuthn = authn
}

// Default 获取默认的Authn 使用之前需要先调用SetDefault
func Default() *Authn {
	return defaultAuthn
}

// Middleware 中间件
// force 表示是否强制要求认证
func (a *Authn) Middleware(force bool, middleware Middleware) gin.HandlerFunc {
	return func(ctx *gin.Context) {
		session, err := a.options.sessionManager.GetFromRequest(ctx)
		if err != nil {
			if force {
				if errors.Is(err, sessions.ErrSessionNotFound) {
					httputils.HandleResp(ctx, ErrUnauthorized)
				} else if errors.Is(err, sessions.ErrSessionExpired) {
					httputils.HandleResp(ctx, ErrTokenInvalid)
				} else {
					httputils.HandleResp(ctx, err)
				}
				return
			}
			return
		}
		if session == nil {
			if force {
				httputils.HandleResp(ctx, ErrUnauthorized)
				return
			}
			return
		}
		// 取出用户
		userId, ok := session.Values[userIdKey].(string)
		if !ok {
			httputils.HandleResp(ctx, ErrUnauthorized)
			return
		}
		ctx.Request = ctx.Request.WithContext(WithSession(ctx.Request.Context(), session))
		if middleware != nil {
			err = middleware(ctx, userId, session)
			if err != nil {
				httputils.HandleResp(ctx, err)
				return
			}
		}
	}
}

// CtxSession 获取session
func CtxSession(ctx context.Context) *sessions.Session {
	session, _ := ctx.Value(authnSession).(*sessions.Session)
	return session
}

// WithSession 设置session
func WithSession(ctx context.Context, session *sessions.Session) context.Context {
	return context.WithValue(ctx, authnSession, session)
}
