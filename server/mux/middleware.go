package mux

import (
	"errors"

	"github.com/cago-frame/cago/configs"
	"github.com/cago-frame/cago/pkg/utils/httputils"
	"github.com/gin-gonic/gin"
)

type RegisterMiddlewareFunc func(cfg *configs.Config, router *gin.Engine) error

var registerMiddleware []RegisterMiddlewareFunc

func RegisterMiddleware(f RegisterMiddlewareFunc) {
	registerMiddleware = append(registerMiddleware, f)
}

func Recover() gin.HandlerFunc {
	return gin.CustomRecovery(func(c *gin.Context, err any) {
		switch err := err.(type) {
		case error:
			httputils.HandleResp(c, err)
		case string:
			httputils.HandleResp(c, errors.New(err))
		}
	})
}
