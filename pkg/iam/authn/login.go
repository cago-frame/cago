package authn

import (
	"github.com/cago-frame/cago/pkg/iam/sessions"
	"github.com/cago-frame/cago/pkg/utils/httputils"
	"github.com/gin-gonic/gin"
)

// LoginHandler 登录操作 登录成功后返回用户信息
type LoginHandler func(ctx *gin.Context) (*User, error)

// SaveSession 保存session
type SaveSession func(ctx *gin.Context, user *User, session *sessions.Session) error

var (
	UsernameNotFound = httputils.NewBadRequestError(-1, "用户名不存在")
	PasswordWrong    = httputils.NewBadRequestError(-1, "密码错误")
)

func (a *Authn) LoginByPassword(ctx *gin.Context, username, password string) (*User, error) {
	return a.LoginBy(ctx, func(ctx *gin.Context) (*User, error) {
		user, err := a.database.GetUserByUsername(ctx, username, WithPassword())
		if err != nil {
			return nil, err
		}
		if user == nil {
			return nil, UsernameNotFound
		}
		if err := user.CheckPassword(password); err != nil {
			return nil, PasswordWrong
		}
		return user, nil
	})
}

func (a *Authn) LoginBy(ctx *gin.Context, login LoginHandler, opts ...LoginOption) (*User, error) {
	options := newLoginOptions(opts...)
	user, err := login(ctx)
	if err != nil {
		return nil, err
	}
	// 登录成功，设置session
	session, err := a.options.sessionManager.Start(ctx)
	if err != nil {
		return nil, err
	}
	session.Values[userIdKey] = user.ID
	session.Values[usernameKey] = user.Username
	if options.saveSession != nil {
		if err := options.saveSession(ctx, user, session); err != nil {
			return nil, err
		}
	}
	return user, a.options.sessionManager.SaveToResponse(ctx, session)
}

func (a *Authn) Logout(ctx *gin.Context) error {
	session, err := a.options.sessionManager.GetFromRequest(ctx)
	if err != nil {
		return err
	}
	if session == nil {
		return nil
	}
	if err := a.options.sessionManager.Delete(ctx, session.ID); err != nil {
		return err
	}
	return nil
}

// RefreshSession 刷新session
func (a *Authn) RefreshSession(ctx *gin.Context, refreshToken string) error {
	ctx.Set("refresh_token", refreshToken)
	session, err := a.options.sessionManager.GetFromRequest(ctx)
	if err != nil {
		return err
	}
	if session == nil {
		return nil
	}
	if err := a.options.sessionManager.Refresh(ctx, session); err != nil {
		return err
	}
	return a.options.sessionManager.SaveToResponse(ctx, session)
}
