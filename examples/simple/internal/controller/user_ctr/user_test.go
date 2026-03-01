package user_ctr

import (
	"context"
	"net/http"
	"testing"

	api "github.com/cago-frame/cago/examples/simple/internal/api/user"
	"github.com/cago-frame/cago/examples/simple/internal/model/entity/user_entity"
	"github.com/cago-frame/cago/examples/simple/internal/repository/user_repo"
	mock_user_repo "github.com/cago-frame/cago/examples/simple/internal/repository/user_repo/mock"
	"github.com/cago-frame/cago/examples/simple/internal/service/user_svc"
	"github.com/cago-frame/cago/pkg/consts"
	"github.com/cago-frame/cago/pkg/iam"
	"github.com/cago-frame/cago/pkg/iam/authn"
	"github.com/cago-frame/cago/pkg/utils/testutils"
	"github.com/cago-frame/cago/server/mux/muxclient"
	"github.com/cago-frame/cago/server/mux/muxtest"
	"github.com/smartystreets/goconvey/convey"
	"github.com/stretchr/testify/assert"
	"go.uber.org/mock/gomock"
	"golang.org/x/crypto/bcrypt"
)

func setupUserTest(t *testing.T) (context.Context, *mock_user_repo.MockUserRepo, *muxtest.TestMux) {
	testutils.Cache(t)
	mockCtrl := gomock.NewController(t)
	t.Cleanup(func() { mockCtrl.Finish() })

	ctx := context.Background()
	mockUserRepo := mock_user_repo.NewMockUserRepo(mockCtrl)
	user_repo.RegisterUser(mockUserRepo)

	// 每次重新初始化IAM，避免onceDo缓存导致的mock引用问题
	iam.SetDefault(iam.New(user_repo.User()))

	// 注册路由
	testMux := muxtest.NewTestMux()
	r := testMux.Group("/api/v1")
	userCtr := NewUser()
	r.Group("/").Bind(
		userCtr.Register,
		userCtr.Login,
	)
	r.Group("/", user_svc.User().Middleware(true)).Bind(
		userCtr.CurrentUser,
		userCtr.Logout,
		userCtr.RefreshToken,
	)

	return ctx, mockUserRepo, testMux
}

// loginUser 辅助函数: 执行登录并返回登录响应
func loginUser(t *testing.T, ctx context.Context, testMux *muxtest.TestMux, mockUserRepo *mock_user_repo.MockUserRepo) *api.LoginResponse {
	t.Helper()
	hashedPassword, _ := bcrypt.GenerateFromPassword([]byte("qwe123"), bcrypt.DefaultCost)
	mockUserRepo.EXPECT().GetUserByUsername(gomock.Any(), "test", gomock.Any()).Return(&authn.User{
		ID:             "1",
		Username:       "test",
		HashedPassword: string(hashedPassword),
	}, nil)
	loginResp := &api.LoginResponse{}
	var httpResp *http.Response
	err := testMux.Do(ctx, &api.LoginRequest{
		Username: "test",
		Password: "qwe123",
	}, loginResp, muxclient.WithResponse(&httpResp))
	assert.NoError(t, err)
	assert.Equal(t, http.StatusOK, httpResp.StatusCode)
	assert.Equal(t, "test", loginResp.Username)
	assert.NotEmpty(t, loginResp.AccessToken)
	return loginResp
}

func TestUserLogin(t *testing.T) {
	ctx, mockUserRepo, testMux := setupUserTest(t)

	convey.Convey("登录", t, func() {
		convey.Convey("登录成功", func() {
			loginResp := loginUser(t, ctx, testMux, mockUserRepo)
			assert.NotEmpty(t, loginResp.RefreshToken)
		})
		convey.Convey("用户名不存在", func() {
			mockUserRepo.EXPECT().GetUserByUsername(gomock.Any(), "notexist", gomock.Any()).Return(nil, nil)
			loginResp := &api.LoginResponse{}
			err := testMux.Do(ctx, &api.LoginRequest{
				Username: "notexist",
				Password: "qwe123",
			}, loginResp)
			assert.Equal(t, authn.UsernameNotFound, err)
		})
		convey.Convey("密码错误", func() {
			hashedPassword, _ := bcrypt.GenerateFromPassword([]byte("qwe123"), bcrypt.DefaultCost)
			mockUserRepo.EXPECT().GetUserByUsername(gomock.Any(), "test", gomock.Any()).Return(&authn.User{
				ID:             "1",
				Username:       "test",
				HashedPassword: string(hashedPassword),
			}, nil)
			loginResp := &api.LoginResponse{}
			err := testMux.Do(ctx, &api.LoginRequest{
				Username: "test",
				Password: "wrongpassword",
			}, loginResp)
			assert.Equal(t, authn.PasswordWrong, err)
		})
	})
}

func TestUserRegister(t *testing.T) {
	ctx, mockUserRepo, testMux := setupUserTest(t)

	convey.Convey("注册", t, func() {
		convey.Convey("注册成功", func() {
			mockUserRepo.EXPECT().FindByUsername(gomock.Any(), "newuser").Return(nil, nil)
			mockUserRepo.EXPECT().Register(gomock.Any(), gomock.Any()).Return(&authn.RegisterResponse{
				UserID: "2",
			}, nil)
			resp := &api.RegisterResponse{}
			err := testMux.Do(ctx, &api.RegisterRequest{
				Username: "newuser",
				Password: "password123",
			}, resp)
			assert.NoError(t, err)
		})
		convey.Convey("用户名已存在", func() {
			mockUserRepo.EXPECT().FindByUsername(gomock.Any(), "existuser").Return(&user_entity.User{
				ID:       1,
				Username: "existuser",
				Status:   consts.ACTIVE,
			}, nil)
			resp := &api.RegisterResponse{}
			err := testMux.Do(ctx, &api.RegisterRequest{
				Username: "existuser",
				Password: "password123",
			}, resp)
			assert.Error(t, err)
		})
	})
}

func TestUserCurrentUser(t *testing.T) {
	ctx, mockUserRepo, testMux := setupUserTest(t)

	convey.Convey("当前用户", t, func() {
		convey.Convey("未登录访问", func() {
			resp := &api.CurrentUserResponse{}
			err := testMux.Do(ctx, &api.CurrentUserRequest{}, resp)
			assert.Equal(t, authn.ErrUnauthorized, err)
		})
		convey.Convey("登录后获取当前用户", func() {
			loginResp := loginUser(t, ctx, testMux, mockUserRepo)
			mockUserRepo.EXPECT().Find(gomock.Any(), int64(1)).Return(&user_entity.User{
				ID:       1,
				Username: "test",
				Status:   consts.ACTIVE,
			}, nil)
			resp := &api.CurrentUserResponse{}
			err := testMux.Do(ctx, &api.CurrentUserRequest{}, resp, muxclient.WithHeader(http.Header{
				"Cookie": []string{"access_token=" + loginResp.AccessToken},
			}))
			assert.NoError(t, err)
			assert.Equal(t, "test", resp.Username)
		})
		convey.Convey("登录后用户被封禁", func() {
			loginResp := loginUser(t, ctx, testMux, mockUserRepo)
			mockUserRepo.EXPECT().Find(gomock.Any(), int64(1)).Return(&user_entity.User{
				ID:       1,
				Username: "test",
				Status:   consts.DELETE,
			}, nil)
			resp := &api.CurrentUserResponse{}
			err := testMux.Do(ctx, &api.CurrentUserRequest{}, resp, muxclient.WithHeader(http.Header{
				"Cookie": []string{"access_token=" + loginResp.AccessToken},
			}))
			assert.Error(t, err)
		})
	})
}

func TestUserLogout(t *testing.T) {
	ctx, mockUserRepo, testMux := setupUserTest(t)

	convey.Convey("退出登录", t, func() {
		convey.Convey("未登录退出", func() {
			resp := &api.LogoutResponse{}
			err := testMux.Do(ctx, &api.LogoutRequest{}, resp)
			assert.Equal(t, authn.ErrUnauthorized, err)
		})
		convey.Convey("登录后退出", func() {
			loginResp := loginUser(t, ctx, testMux, mockUserRepo)
			mockUserRepo.EXPECT().Find(gomock.Any(), int64(1)).Return(&user_entity.User{
				ID:       1,
				Username: "test",
				Status:   consts.ACTIVE,
			}, nil)
			resp := &api.LogoutResponse{}
			err := testMux.Do(ctx, &api.LogoutRequest{}, resp, muxclient.WithHeader(http.Header{
				"Cookie": []string{"access_token=" + loginResp.AccessToken},
			}))
			assert.NoError(t, err)
			assert.NotNil(t, resp)

			convey.Convey("退出后再访问接口失败", func() {
				resp := &api.CurrentUserResponse{}
				err := testMux.Do(ctx, &api.CurrentUserRequest{}, resp, muxclient.WithHeader(http.Header{
					"Cookie": []string{"access_token=" + loginResp.AccessToken},
				}))
				assert.Equal(t, authn.ErrUnauthorized, err)
			})
		})
	})
}

func TestUserRefreshToken(t *testing.T) {
	ctx, mockUserRepo, testMux := setupUserTest(t)

	convey.Convey("刷新token", t, func() {
		convey.Convey("刷新成功", func() {
			loginResp := loginUser(t, ctx, testMux, mockUserRepo)
			mockUserRepo.EXPECT().Find(gomock.Any(), int64(1)).Return(&user_entity.User{
				ID:       1,
				Username: "test",
				Status:   consts.ACTIVE,
			}, nil)
			resp := &api.RefreshTokenResponse{}
			err := testMux.Do(ctx, &api.RefreshTokenRequest{
				RefreshToken: loginResp.RefreshToken,
			}, resp, muxclient.WithHeader(http.Header{
				"Cookie": []string{"access_token=" + loginResp.AccessToken},
			}))
			assert.NoError(t, err)
			assert.NotEmpty(t, resp.AccessToken)
			assert.NotEmpty(t, resp.RefreshToken)

			convey.Convey("使用老的token访问失败", func() {
				resp := &api.CurrentUserResponse{}
				err := testMux.Do(ctx, &api.CurrentUserRequest{}, resp, muxclient.WithHeader(http.Header{
					"Cookie": []string{"access_token=" + loginResp.AccessToken},
				}))
				assert.Equal(t, authn.ErrUnauthorized, err)
			})
		})
	})
}
