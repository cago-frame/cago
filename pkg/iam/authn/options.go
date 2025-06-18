package authn

import (
	"net/http"

	cache2 "github.com/cago-frame/cago/database/cache"
	"github.com/cago-frame/cago/database/cache/cache"
	"github.com/cago-frame/cago/pkg/iam/sessions/manager"
	"github.com/gin-gonic/gin"

	"github.com/cago-frame/cago/pkg/iam/sessions"
	"github.com/cago-frame/cago/pkg/utils/httputils"
)

var (
	ErrUnauthorized         = httputils.NewUnauthorizedError(-1, "未登录")
	ErrTokenInvalid         = httputils.NewUnauthorizedError(-1, "token失效")
	ErrRefreshTokenNotFound = httputils.NewUnauthorizedError(-1, "必须携带refresh_token")
)

type Middleware func(ctx *gin.Context, userId string, session *sessions.Session) error

type Options struct {
	sessionManager sessions.HTTPSessionManager
}

type Option func(*Options)

func newOptions(opts ...Option) *Options {
	opt := &Options{
		sessionManager: manager.NewRefreshHTTPSessionManager(
			manager.NewCacheSessionManager(
				cache.NewPrefixCache("iam:access:", cache2.Default()),
			),
			manager.NewCacheSessionManager(
				cache.NewPrefixCache("iam:refresh:", cache2.Default()),
			),
			func(options *manager.RefreshHTTPSessionManagerOptions) {
				options.AccessTokenMapping = manager.NewCacheAccessTokenMapping(cache2.Default())
				options.ResponseFunc = func(ctx *gin.Context, accessToken, refreshToken *sessions.Session) error {
					ctx.SetCookie("access_token", accessToken.ID,
						int(accessToken.Metadata["expire"].(int64)), "/", "", false, true)
					ctx.JSON(http.StatusOK, httputils.JSONResponse{
						Code: 0,
						Data: gin.H{
							"username": accessToken.Values[usernameKey],
							"token": &manager.RefreshSessionResponse{
								AccessToken:   accessToken.ID,
								RefreshToken:  refreshToken.ID,
								Expire:        accessToken.Metadata["expire"].(int64),
								RefreshExpire: refreshToken.Metadata["expire"].(int64),
							},
						},
					})
					return nil
				}
				options.GetRefreshToken = func(ctx *gin.Context) (string, error) {
					val, ok := ctx.Get(refreshTokenKey)
					if !ok {
						return "", ErrRefreshTokenNotFound
					}
					return val.(string), nil
				}
			},
		),
	}
	for _, o := range opts {
		o(opt)
	}
	return opt
}

func WithSessionManager(session sessions.HTTPSessionManager) Option {
	return func(o *Options) {
		o.sessionManager = session
	}
}

type LoginOptions struct {
	saveSession SaveSession
}

type LoginOption func(*LoginOptions)

func newLoginOptions(opts ...LoginOption) *LoginOptions {
	opt := &LoginOptions{}
	for _, o := range opts {
		o(opt)
	}
	return opt
}

func WithSaveSession(saveSession SaveSession) LoginOption {
	return func(o *LoginOptions) {
		o.saveSession = saveSession
	}
}
