package api

import (
	"context"
	"net/http"
	"testing"

	"github.com/cago-frame/cago/examples/simple/internal/api/example"
	"github.com/cago-frame/cago/examples/simple/internal/api/user"
	"github.com/cago-frame/cago/examples/simple/internal/model/entity/user_entity"
	"github.com/cago-frame/cago/examples/simple/internal/repository/user_repo"
	mock_user_repo "github.com/cago-frame/cago/examples/simple/internal/repository/user_repo/mock"
	"github.com/cago-frame/cago/pkg/consts"
	"github.com/cago-frame/cago/pkg/iam/authn"
	"github.com/cago-frame/cago/pkg/utils/testutils"
	"github.com/cago-frame/cago/server/mux/muxclient"
	"github.com/cago-frame/cago/server/mux/muxtest"
	"github.com/smartystreets/goconvey/convey"
	"github.com/stretchr/testify/assert"
	"go.uber.org/mock/gomock"
	"golang.org/x/crypto/bcrypt"
)

func TestRouter(t *testing.T) {
	// 注册依赖和mock
	testutils.Cache(t)
	mockCtrl := gomock.NewController(t)
	defer mockCtrl.Finish()

	ctx := context.Background()
	mockUserRepo := mock_user_repo.NewMockUserRepo(mockCtrl)
	user_repo.RegisterUser(mockUserRepo)

	testutils.IAM(t, user_repo.User())

	// 注册路由
	testMux := muxtest.NewTestMux()
	err := Router(context.Background(), testMux.Router)
	assert.Nil(t, err)

	convey.Convey("登录用户", t, func() {
		hashedPassword, _ := bcrypt.GenerateFromPassword([]byte("qwe123"), bcrypt.DefaultCost)
		mockUserRepo.EXPECT().GetUserByUsername(gomock.Any(), "test", gomock.Any()).Return(&authn.User{
			ID:             "1",
			Username:       "test",
			HashedPassword: string(hashedPassword),
		}, nil)
		loginResp := &user.LoginResponse{}
		var httpResp *http.Response
		err := testMux.Do(ctx, &user.LoginRequest{
			Username: "test",
			Password: "qwe123",
		}, loginResp, muxclient.WithResponse(&httpResp))
		assert.NoError(t, err)
		assert.Equal(t, http.StatusOK, httpResp.StatusCode)
		assert.Equal(t, "test", loginResp.Username)
		assert.NotEmpty(t, loginResp.AccessToken)

		mockUserRepo.EXPECT().Find(gomock.Any(), int64(1)).Return(&user_entity.User{
			ID:       1,
			Username: "test",
			Status:   consts.ACTIVE,
		}, nil)

		convey.Convey("当前用户", func() {
			resp := &user.CurrentUserResponse{}
			err := testMux.Do(ctx, &user.CurrentUserRequest{}, resp, muxclient.WithHeader(http.Header{
				"Cookie": []string{"access_token=" + loginResp.AccessToken},
			}))
			assert.NoError(t, err)
			assert.Equal(t, "test", resp.Username)
		})
		convey.Convey("日志审计", func() {
			resp := &example.AuditResponse{}
			err := testMux.Do(ctx, &example.AuditRequest{}, resp, muxclient.WithHeader(http.Header{
				"Cookie": []string{"access_token=" + loginResp.AccessToken},
			}))
			assert.NoError(t, err)
		})
		convey.Convey("退出登录", func() {
			resp := &user.LogoutResponse{}
			err := testMux.Do(ctx, &user.LogoutRequest{}, resp, muxclient.WithHeader(http.Header{
				"Cookie": []string{"access_token=" + loginResp.AccessToken},
			}))
			assert.NoError(t, err)
			assert.NotNil(t, resp)
			convey.Convey("退出登录了再访问api出错", func() {
				err := testMux.Do(ctx, &user.CurrentUserRequest{}, resp, muxclient.WithHeader(http.Header{
					"Cookie": []string{"access_token=" + loginResp.AccessToken},
				}))
				assert.Equal(t, authn.ErrUnauthorized, err)
			})
		})
		convey.Convey("刷新token", func() {
			resp := &user.RefreshTokenResponse{}
			err := testMux.Do(ctx, &user.RefreshTokenRequest{
				RefreshToken: loginResp.RefreshToken,
			}, resp, muxclient.WithHeader(http.Header{
				"Cookie": []string{"access_token=" + loginResp.AccessToken},
			}))
			assert.NoError(t, err)
			assert.NotEmpty(t, resp.AccessToken)
			assert.NotEmpty(t, resp.RefreshToken)
			convey.Convey("使用老的token访问报错", func() {
				resp := &user.CurrentUserResponse{}
				err := testMux.Do(ctx, &user.CurrentUserRequest{}, resp, muxclient.WithHeader(http.Header{
					"Cookie": []string{"access_token=" + loginResp.AccessToken},
				}))
				assert.Equal(t, authn.ErrUnauthorized, err)
			})
		})
	})
}
