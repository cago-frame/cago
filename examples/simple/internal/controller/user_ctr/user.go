package user_ctr

import (
	"context"

	"github.com/gin-gonic/gin"

	api "github.com/cago-frame/cago/examples/simple/internal/api/user"
	"github.com/cago-frame/cago/examples/simple/internal/service/user_svc"
)

type User struct {
}

func NewUser() *User {
	return &User{}
}

// Register 注册
func (l *User) Register(ctx context.Context, req *api.RegisterRequest) (*api.RegisterResponse, error) {
	return user_svc.User().Register(ctx, req)
}

// Login 登录
func (l *User) Login(ctx *gin.Context, req *api.LoginRequest) error {
	return user_svc.User().Login(ctx, req)
}

// Logout 登出
func (l *User) Logout(ctx *gin.Context, req *api.LogoutRequest) (*api.LogoutResponse, error) {
	return user_svc.User().Logout(ctx, req)
}

// CurrentUser 当前登录用户
func (l *User) CurrentUser(ctx context.Context, req *api.CurrentUserRequest) (*api.CurrentUserResponse, error) {
	return user_svc.User().CurrentUser(ctx, req)
}

// RefreshToken 刷新token
func (l *User) RefreshToken(ctx *gin.Context, req *api.RefreshTokenRequest) error {
	return user_svc.User().RefreshToken(ctx, req)
}
