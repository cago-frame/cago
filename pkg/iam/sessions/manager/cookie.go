package manager

import (
	"github.com/cago-frame/cago/pkg/iam/sessions"
	"github.com/gin-gonic/gin"
)

type cookieSessionManager struct {
	sessions.SessionManager
}

// NewCookieSessionManager 从cookie中读取session id
func NewCookieSessionManager(session sessions.SessionManager) sessions.HTTPSessionManager {
	return &cookieSessionManager{
		SessionManager: session,
	}
}

func (c *cookieSessionManager) GetFromRequest(ctx *gin.Context) (*sessions.Session, error) {
	id, err := ctx.Cookie("SESSION")
	if err != nil {
		return nil, err
	}
	return c.Get(ctx, id)
}

func (c *cookieSessionManager) SaveToResponse(ctx *gin.Context, session *sessions.Session) error {
	if err := c.Save(ctx, session); err != nil {
		return err
	}
	ctx.SetCookie("SESSION", session.ID, 0, "/", "", false, true)
	return nil
}
