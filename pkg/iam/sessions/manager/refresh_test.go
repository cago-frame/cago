package manager

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/cago-frame/cago/database/cache/cache"
	"github.com/cago-frame/cago/database/cache/memory"
	"github.com/cago-frame/cago/pkg/iam/sessions"
	"github.com/cago-frame/cago/pkg/utils/httputils"
	"github.com/gin-gonic/gin"
	"github.com/smartystreets/goconvey/convey"
	"github.com/stretchr/testify/assert"
)

func TestNewRefreshHTTPSessionManager(t *testing.T) {
	m, _ := memory.NewMemoryCache()
	refreshSession := NewRefreshHTTPSessionManager(NewMemorySessionManager(), NewMemorySessionManager(), func(options *RefreshHTTPSessionManagerOptions) {
		options.AccessTokenMapping = NewCacheAccessTokenMapping(m)
	})

	convey.Convey("session", t, func() {
		w := httptest.NewRecorder()
		c, _ := gin.CreateTestContext(w)
		convey.Convey("session不存在", func() {
			c.Request = httptest.NewRequest(http.MethodGet, "/", nil)
			c.Request.Header.Set("Cookie", "access_token=xxxxx")
			// session不存在
			session, err := refreshSession.GetFromRequest(c)
			assert.Equal(t, sessions.ErrSessionNotFound, err)
			assert.Nil(t, session)
		})

		convey.Convey("创建一个新的session", func() {
			session, err := refreshSession.Start(c)
			assert.NoError(t, err)
			assert.NotNil(t, session)
			assert.Len(t, session.Values, 0)
			assert.Len(t, session.Metadata, 0)
			session.Values["uid"] = 1
			err = refreshSession.SaveToResponse(c, session)
			assert.NoError(t, err)
			// 自动添加了到期时间
			assert.Len(t, session.Values, 1)
			assert.Len(t, session.Metadata, 1)
			assert.Equal(t, 1, session.Values["uid"])
			refreshResp := &RefreshSessionResponse{}
			resp := &httputils.JSONResponse{
				Data: refreshResp,
			}
			err = json.Unmarshal(w.Body.Bytes(), resp)
			assert.NoError(t, err)
			assert.Equal(t, 0, resp.Code)
			convey.Convey("通过token获取信息", func() {
				c.Request = httptest.NewRequest(http.MethodGet, "/", nil)
				c.Request.Header.Set("Cookie", "access_token="+refreshResp.AccessToken)
				session, err = refreshSession.GetFromRequest(c)
				assert.NoError(t, err)
				assert.NotNil(t, session)
				assert.Equal(t, 1, session.Values["uid"])
			})
			w = httptest.NewRecorder()
			c, _ = gin.CreateTestContext(w)
			convey.Convey("刷新token(无效的access_token)", func() {
				c.Request = httptest.NewRequest(http.MethodPost, "/", strings.NewReader("refresh_token="+refreshResp.RefreshToken))
				c.Request.Header.Set("Content-Type", "application/x-www-form-urlencoded")
				c.Request.Header.Set("Cookie", "access_token=xxxx")
				session, err = refreshSession.GetFromRequest(c)
				assert.Equal(t, sessions.ErrSessionNotFound, err)
				assert.Nil(t, session)
			})
			convey.Convey("刷新token(带access_token)", func() {
				c.Request = httptest.NewRequest(http.MethodPost, "/", strings.NewReader("refresh_token="+refreshResp.RefreshToken))
				c.Request.Header.Set("Cookie", "access_token="+refreshResp.AccessToken)
				c.Request.Header.Set("Content-Type", "application/x-www-form-urlencoded")
				session, err = refreshSession.GetFromRequest(c)
				assert.NoError(t, err)
				assert.Equal(t, 1, session.Values["uid"])
				err = refreshSession.SaveToResponse(c, session)
				assert.NoError(t, err)
				refreshResp2 := &RefreshSessionResponse{}
				resp = &httputils.JSONResponse{
					Data: refreshResp2,
				}
				err = json.Unmarshal(w.Body.Bytes(), resp)
				assert.NoError(t, err)
				assert.NotEmpty(t, refreshResp2.AccessToken)
				assert.NotEmpty(t, refreshResp.AccessToken)
				assert.NotEqual(t, refreshResp.AccessToken, refreshResp2.AccessToken)
				assert.NotEqual(t, refreshResp.RefreshToken, refreshResp2.RefreshToken)
				convey.Convey("使用老的token访问", func() {
					c.Request = httptest.NewRequest(http.MethodPost, "/", nil)
					c.Request.Header.Set("Cookie", "access_token="+refreshResp.AccessToken)
					session, err = refreshSession.GetFromRequest(c)
					assert.Equal(t, sessions.ErrSessionNotFound, err)
					assert.Nil(t, session)
				})
			})
			convey.Convey("带过期的access_token刷新成功", func() {
				c.Request = httptest.NewRequest(http.MethodPost, "/", strings.NewReader("refresh_token="+refreshResp.RefreshToken))
				c.Request.Header.Set("Cookie", "access_token="+refreshResp.AccessToken)
				c.Request.Header.Set("Content-Type", "application/x-www-form-urlencoded")
				session.Metadata["expire"] = 0
				session, err = refreshSession.GetFromRequest(c)
				assert.NoError(t, err)
				assert.NotNil(t, session)
				assert.Equal(t, 1, session.Values["uid"])
			})
			convey.Convey("删除token", func() {
				c.Request = httptest.NewRequest(http.MethodPost, "/", nil)
				c.Request.Header.Set("Cookie", "access_token="+refreshResp.AccessToken)
				session, err = refreshSession.GetFromRequest(c)
				assert.NoError(t, err)
				assert.NotNil(t, session)
				err = refreshSession.Delete(c, session.ID)
				assert.NoError(t, err)
				convey.Convey("再刷新时失败", func() {
					c.Request = httptest.NewRequest(http.MethodPost, "/", strings.NewReader("refresh_token="+refreshResp.RefreshToken))
					c.Request.Header.Set("Cookie", "access_token="+refreshResp.AccessToken)
					c.Request.Header.Set("Content-Type", "application/x-www-form-urlencoded")
					err = refreshSession.SaveToResponse(c, session)
					assert.Equal(t, sessions.ErrSessionNotFound, err)
					// 缓存中也删除了
					err = m.Get(c, refreshResp.AccessToken).Err()
					assert.Equal(t, cache.ErrNil, err)
				})
			})
		})
	})
}
