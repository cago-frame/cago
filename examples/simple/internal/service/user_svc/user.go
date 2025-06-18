package user_svc

import (
	"context"
	"strconv"

	"github.com/cago-frame/cago/examples/simple/internal/model"
	"github.com/cago-frame/cago/pkg/iam/audit"
	"github.com/cago-frame/cago/pkg/logger"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/trace"
	"go.uber.org/zap"

	api "github.com/cago-frame/cago/examples/simple/internal/api/user"
	"github.com/cago-frame/cago/examples/simple/internal/pkg/code"
	"github.com/cago-frame/cago/examples/simple/internal/repository/user_repo"
	"github.com/cago-frame/cago/pkg/i18n"
	"github.com/cago-frame/cago/pkg/iam/authn"
	"github.com/cago-frame/cago/pkg/iam/sessions"
	"github.com/gin-gonic/gin"
)

type UserSvc interface {
	// Register 注册
	Register(ctx context.Context, req *api.RegisterRequest) (*api.RegisterResponse, error)
	// User 登录
	Login(ctx *gin.Context, req *api.LoginRequest) error
	// Logout 登出
	Logout(ctx *gin.Context, req *api.LogoutRequest) (*api.LogoutResponse, error)
	// Ctx 从上下文获取用户信息
	Ctx(ctx context.Context) *model.AuthInfo
	// WithUser 设置用户信息到上下文
	WithUser(ctx context.Context, userId int64) (context.Context, error)
	// Middleware authn处理中间件
	Middleware(force bool) gin.HandlerFunc
	// AuditMiddleware 审计处理中间件
	AuditMiddleware(module string) gin.HandlerFunc
	// CurrentUser 当前登录用户
	CurrentUser(ctx context.Context, req *api.CurrentUserRequest) (*api.CurrentUserResponse, error)
	// RefreshToken 刷新token
	RefreshToken(ctx *gin.Context, req *api.RefreshTokenRequest) error
}

type userSvc struct {
}

var defaultUser = &userSvc{}

func User() UserSvc {
	return defaultUser
}

// Register 注册
func (l *userSvc) Register(ctx context.Context, req *api.RegisterRequest) (*api.RegisterResponse, error) {
	// 查找相同用户名
	user, err := user_repo.User().FindByUsername(ctx, req.Username)
	if err != nil {
		return nil, err
	}
	if user != nil {
		return nil, i18n.NewError(ctx, code.UsernameAlreadyExists)
	}
	// 创建用户
	_, err = user_repo.User().Register(ctx, &authn.RegisterRequest{
		Username: req.Password,
		Password: req.Password,
	})
	if err != nil {
		return nil, err
	}
	return &api.RegisterResponse{}, nil
}

// Login 登录
func (l *userSvc) Login(ctx *gin.Context, req *api.LoginRequest) error {
	_, err := authn.Default().LoginByPassword(ctx, req.Username, req.Password)
	return err
}

// Logout 登出
func (l *userSvc) Logout(ctx *gin.Context, req *api.LogoutRequest) (*api.LogoutResponse, error) {
	err := authn.Default().Logout(ctx)
	if err != nil {
		return nil, err
	}
	return &api.LogoutResponse{}, nil
}

func (l *userSvc) Ctx(ctx context.Context) *model.AuthInfo {
	user, _ := ctx.Value(model.AuthInfo{}).(*model.AuthInfo)
	return user
}

func (l *userSvc) WithUser(ctx context.Context, userId int64) (context.Context, error) {
	user, err := user_repo.User().Find(ctx, userId)
	if err != nil {
		return nil, err
	}
	if user == nil {
		return nil, i18n.NewError(ctx, code.UserNotFound)
	}
	if err := user.Check(ctx); err != nil {
		return nil, err
	}
	// 设置用户信息,链路追踪和日志也添加上用户信息
	authInfo := &model.AuthInfo{
		UserID:   user.ID,
		Username: user.Username,
	}
	trace.SpanFromContext(ctx).SetAttributes(
		attribute.Int64("user_id", user.ID),
	)
	return context.WithValue(
		logger.WithContextLogger(ctx, logger.Ctx(ctx).
			With(zap.Int64("user_id", user.ID))),
		model.AuthInfo{}, authInfo), nil
}

func (l *userSvc) Middleware(force bool) gin.HandlerFunc {
	return authn.Default().Middleware(force, func(ctx *gin.Context, userId string, session *sessions.Session) error {
		nUserId, err := strconv.ParseInt(userId, 10, 64)
		if err != nil {
			return err
		}
		gCtx, err := l.WithUser(ctx.Request.Context(), nUserId)
		if err != nil {
			return err
		}
		ctx.Request = ctx.Request.WithContext(gCtx)
		return nil
	})
}

func (l *userSvc) AuditMiddleware(module string) gin.HandlerFunc {
	return audit.Default().Middleware(module, func(ctx *gin.Context) []zap.Field {
		user := l.Ctx(ctx)
		fields := []zap.Field{
			zap.String("path", ctx.Request.URL.Path),
		}
		if user != nil {
			fields = append(fields,
				zap.Int64("user_id", user.UserID),
				zap.String("username", user.Username))
		}
		return fields
	})
}

// CurrentUser 当前登录用户
func (l *userSvc) CurrentUser(ctx context.Context, req *api.CurrentUserRequest) (*api.CurrentUserResponse, error) {
	user := l.Ctx(ctx)
	return &api.CurrentUserResponse{
		Username: user.Username,
	}, nil
}

// RefreshToken 刷新token
func (l *userSvc) RefreshToken(ctx *gin.Context, req *api.RefreshTokenRequest) error {
	return authn.Default().RefreshSession(ctx, req.RefreshToken)
}
