package httputils

import (
	"context"
	"net/http"

	"github.com/cago-frame/cago/pkg/errs"
	"github.com/cago-frame/cago/pkg/logger"
	pkgValidator "github.com/cago-frame/cago/pkg/utils/validator"
	"github.com/gin-gonic/gin"
	"github.com/go-playground/validator/v10"
	"go.opentelemetry.io/otel/trace"
	"go.uber.org/zap"
)

// Handle 处理请求
func Handle(ctx *gin.Context, f func() interface{}) {
	resp := f()
	if resp == nil {
		ctx.JSON(http.StatusOK, gin.H{
			"code": 0, "msg": "ok",
		})
		return
	}
	HandleResp(ctx, resp)
}

// HandleResp 处理响应
// 1. 如果是 httputils.Error 类型，会根据里面定义的状态码和数据返回
// 2. 如果是 validator.ValidationErrors 类型，会返回 400 错误码和错误信息
// 3. 如果是 error 类型，会返回 500 错误码，错误信息会记录到日志中
// 4. 其他情况会返回 200 状态码，并返回数据
func HandleResp(ctx *gin.Context, resp any) {
	if err, ok := resp.(error); ok {
		if err := HandleError(ctx, err); err != nil {
			return
		}
	}
	ctx.JSON(http.StatusOK, JSONResponse{
		Code: 0,
		Msg:  "success",
		Data: resp,
	})
}

// HandleError 处理错误
// 当err为nil时不做任何处理
// 否则根据err的类型进行处理，然后返回
func HandleError(ctx *gin.Context, err error) error {
	if err == nil {
		return nil
	}
	var field []zap.Field
	for {
		if errr, ok := err.(errs.Unwrap); ok {
			switch errr := errr.(type) {
			case *errs.Error:
				field = append(field, errr.Field()...)
			}
			err = errr.Unwrap()
		} else {
			break
		}
	}
	// 从trace中获取
	switch data := err.(type) {
	case *Error:
		data.RequestID = RequestID(ctx)
		ctx.AbortWithStatusJSON(data.Status, data)
	case validator.ValidationErrors:
		ctx.AbortWithStatusJSON(http.StatusBadRequest, gin.H{
			"code": -1, "msg": pkgValidator.TransError(data),
		})
	default:
		requestId := RequestID(ctx)
		field = append(field, zap.Error(data))
		logger.Ctx(ctx).Error(
			"internal server error",
			field...,
		)
		if requestId != "" {
			ctx.AbortWithStatusJSON(http.StatusInternalServerError, gin.H{
				"code": -1, "msg": "系统错误", "request_id": requestId,
			})
		} else {
			ctx.AbortWithStatusJSON(http.StatusInternalServerError, gin.H{
				"code": -1, "msg": "系统错误",
			})
		}
	}
	return err
}

// RequestID 获取请求ID
func RequestID(ctx context.Context) string {
	if span := trace.SpanFromContext(ctx); span != nil {
		if span.SpanContext().TraceID().IsValid() {
			return span.SpanContext().TraceID().String()
		}
	}
	return ""
}
