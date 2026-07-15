package user

import (
	"github.com/cago-frame/cago/pkg/iam/sessions/manager"
	"github.com/cago-frame/cago/server/mux"
)

// RegisterRequest 注册
type RegisterRequest struct {
	mux.Meta `path:"/user/register" method:"POST"`
	// 用户名
	Username string `form:"username" binding:"required"`
	// 密码
	Password string `form:"password" binding:"required"`
}

type RegisterResponse struct {
}

// LoginRequest 登录
type LoginRequest struct {
	mux.Meta `path:"/user/login" method:"POST"`
	// 用户名
	Username string `form:"username" binding:"required"`
	// 密码
	Password string `form:"password" binding:"required"`
}

type LoginResponse struct {
	Username                       string `json:"username"`
	manager.RefreshSessionResponse `json:"token"`
}

// CurrentUserRequest 当前登录用户
type CurrentUserRequest struct {
	mux.Meta `path:"/user/current" method:"GET"`
}

type CurrentUserResponse struct {
	Username string `json:"username"`
}

// LogoutRequest 登出
type LogoutRequest struct {
	mux.Meta `path:"/user/logout" method:"GET"`
}

type LogoutResponse struct {
}

// RefreshTokenRequest 刷新token
type RefreshTokenRequest struct {
	mux.Meta     `path:"/user/refresh" method:"POST"`
	RefreshToken string `form:"refresh_token" json:"refresh_token" binding:"required"`
}

type RefreshTokenResponse struct {
	Username                       string `json:"username"`
	manager.RefreshSessionResponse `json:"token"`
}
