package httputils

import (
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	validator2 "github.com/cago-frame/cago/pkg/utils/validator"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
)

func TestHandleResp(t *testing.T) {
	// 成功返回
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	HandleResp(c, nil)
	assert.Equal(t, http.StatusOK, w.Code)
	assert.Equal(t, "{\"code\":0,\"msg\":\"success\",\"data\":null}", w.Body.String())
	// 内部错误
	w = httptest.NewRecorder()
	c, _ = gin.CreateTestContext(w)
	HandleResp(c, errors.New("error"))
	assert.Equal(t, http.StatusInternalServerError, w.Code)
	assert.Equal(t, "{\"code\":-1,\"msg\":\"系统错误\"}", w.Body.String())
	// 定义的错误
	w = httptest.NewRecorder()
	c, _ = gin.CreateTestContext(w)
	HandleResp(c, NewError(http.StatusBadRequest, 1000, "参数错误"))
	assert.Equal(t, http.StatusBadRequest, w.Code)
	assert.Equal(t, "{\"code\":1000,\"msg\":\"参数错误\"}", w.Body.String())
	// 参数校验错误
	w = httptest.NewRecorder()
	c, _ = gin.CreateTestContext(w)
	v1, err := validator2.NewValidator()
	v := v1.(*validator2.DefaultValidator)
	assert.Nil(t, err)
	err = v.ValidateStruct(struct {
		Mobile string `binding:"required,mobile" label:"手机"`
	}{})
	HandleResp(c, err)
	assert.Equal(t, http.StatusBadRequest, w.Code)
	assert.Equal(t, "{\"code\":-1,\"msg\":\"手机为必填字段\"}", w.Body.String())
	// 处理错误 不返回任何内容
	w = httptest.NewRecorder()
	c, _ = gin.CreateTestContext(w)
	err = HandleError(c, nil)
	assert.NoError(t, err)
	assert.Equal(t, http.StatusOK, w.Code)
	assert.Equal(t, "", w.Body.String())
}
