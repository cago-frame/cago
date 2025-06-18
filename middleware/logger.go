package middleware

import (
	"time"

	"github.com/cago-frame/cago/pkg/logger"
	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
)

func Logger(log *zap.Logger) gin.HandlerFunc {
	return func(ctx *gin.Context) {
		tm := time.Now()
		log := log.With(
			zap.String("client_ip", ctx.ClientIP()),
			zap.String("method", ctx.Request.Method),
			zap.String("path", ctx.Request.URL.Path),
			zap.String("user_agent", ctx.Request.UserAgent()),
			// 请求开始时间
			zap.Time("start_time", tm),
		)
		ctx.Request = ctx.Request.WithContext(logger.WithContextLogger(ctx.Request.Context(), log))
		// 处理错误日志
		ctx.Next()
	}
}
