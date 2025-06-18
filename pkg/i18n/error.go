package i18n

import (
	"context"
	"fmt"
	"net/http"

	"github.com/cago-frame/cago/pkg/utils/httputils"
)

type key int

const (
	languageKey key = iota
)

// T 根据ctx获取语言，根据key获取对应的文本，然后根据args进行格式化，返回结果
func T(ctx context.Context, key int, args ...interface{}) string {
	lang, ok := ctx.Value(languageKey).(string)
	if !ok {
		lang = DefaultLang
	}
	langMap, ok := langs[lang]
	if !ok {
		langMap = langs[DefaultLang]
	}
	return fmt.Sprintf(langMap[key], args...)
}

// WithLanguage 设置语言
func WithLanguage(ctx context.Context, lang string) context.Context {
	return context.WithValue(ctx, languageKey, lang)
}

// NewErrorWithStatus 自定义错误，可以设置状态码，code为错误码，会自动获取对应的错误文本，v为格式化参数
func NewErrorWithStatus(ctx context.Context, status int, code int, v ...interface{}) error {
	return &httputils.Error{
		Status: status,
		Code:   code,
		Msg:    T(ctx, code, v...),
	}
}

// NewError 参数校验错误
func NewError(ctx context.Context, code int, v ...interface{}) error {
	return NewErrorWithStatus(ctx, http.StatusBadRequest, code, v...)
}

// NewUnauthorizedError 401未授权
func NewUnauthorizedError(ctx context.Context, code int, v ...interface{}) error {
	return NewErrorWithStatus(ctx, http.StatusUnauthorized, code, v...)
}

// NewForbiddenError 403禁止访问
func NewForbiddenError(ctx context.Context, code int, v ...interface{}) error {
	return NewErrorWithStatus(ctx, http.StatusForbidden, code, v...)
}

// NewNotFoundError 404资源不存在
func NewNotFoundError(ctx context.Context, code int, v ...interface{}) error {
	return NewErrorWithStatus(ctx, http.StatusNotFound, code, v...)
}

// NewInternalError 500构造内部错误
func NewInternalError(ctx context.Context, code int, v ...interface{}) error {
	return NewErrorWithStatus(ctx, http.StatusInternalServerError, code, v...)
}
