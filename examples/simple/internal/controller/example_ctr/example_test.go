package example_ctr

import (
	"context"
	"net/http"
	"testing"

	api "github.com/cago-frame/cago/examples/simple/internal/api/example"
	userapi "github.com/cago-frame/cago/examples/simple/internal/api/user"
	"github.com/cago-frame/cago/examples/simple/internal/controller/user_ctr"
	"github.com/cago-frame/cago/examples/simple/internal/model/entity/user_entity"
	"github.com/cago-frame/cago/examples/simple/internal/repository/user_repo"
	mock_user_repo "github.com/cago-frame/cago/examples/simple/internal/repository/user_repo/mock"
	"github.com/cago-frame/cago/examples/simple/internal/service/user_svc"
	"github.com/cago-frame/cago/pkg/broker"
	"github.com/cago-frame/cago/pkg/broker/event_bus"
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

func setupExampleTest(t *testing.T) (context.Context, *mock_user_repo.MockUserRepo, *muxtest.TestMux) {
	testutils.Cache(t)
	mockCtrl := gomock.NewController(t)
	t.Cleanup(func() { mockCtrl.Finish() })

	ctx := context.Background()
	mockUserRepo := mock_user_repo.NewMockUserRepo(mockCtrl)
	user_repo.RegisterUser(mockUserRepo)

	// 每次重新初始化IAM，避免onceDo缓存导致的mock引用问题
	iam.SetDefault(iam.New(user_repo.User()))
	broker.SetBroker(event_bus.NewEvBusBroker())

	// 注册路由
	testMux := muxtest.NewTestMux()
	r := testMux.Group("/api/v1")

	// 注册用户登录路由，供需要认证的测试使用
	userCtr := user_ctr.NewUser()
	r.Group("/").Bind(
		userCtr.Login,
	)

	exampleCtr := NewExample()
	r.Group("/").Bind(
		exampleCtr.Ping,
		exampleCtr.GinFun,
	)
	r.Group("/",
		user_svc.User().Middleware(true),
		user_svc.User().AuditMiddleware("example")).Bind(
		exampleCtr.Audit,
	)

	return ctx, mockUserRepo, testMux
}

// loginUser 辅助函数: 执行登录并返回登录响应
func loginUser(t *testing.T, ctx context.Context, testMux *muxtest.TestMux, mockUserRepo *mock_user_repo.MockUserRepo) *userapi.LoginResponse {
	t.Helper()
	hashedPassword, _ := bcrypt.GenerateFromPassword([]byte("qwe123"), bcrypt.DefaultCost)
	mockUserRepo.EXPECT().GetUserByUsername(gomock.Any(), "test", gomock.Any()).Return(&authn.User{
		ID:             "1",
		Username:       "test",
		HashedPassword: string(hashedPassword),
	}, nil)
	loginResp := &userapi.LoginResponse{}
	var httpResp *http.Response
	err := testMux.Do(ctx, &userapi.LoginRequest{
		Username: "test",
		Password: "qwe123",
	}, loginResp, muxclient.WithResponse(&httpResp))
	assert.NoError(t, err)
	assert.Equal(t, http.StatusOK, httpResp.StatusCode)
	assert.NotEmpty(t, loginResp.AccessToken)
	return loginResp
}

func TestExamplePing(t *testing.T) {
	ctx, _, testMux := setupExampleTest(t)

	convey.Convey("Ping", t, func() {
		resp := &api.PingResponse{}
		err := testMux.Do(ctx, &api.PingRequest{}, resp)
		assert.NoError(t, err)
		assert.NotEmpty(t, resp.Pong)
		assert.Len(t, resp.Pong, 6)
	})
}

func TestExampleAudit(t *testing.T) {
	ctx, mockUserRepo, testMux := setupExampleTest(t)

	convey.Convey("审计操作", t, func() {
		convey.Convey("未登录访问审计接口", func() {
			resp := &api.AuditResponse{}
			err := testMux.Do(ctx, &api.AuditRequest{}, resp)
			assert.Equal(t, authn.ErrUnauthorized, err)
		})
		convey.Convey("登录后访问审计接口", func() {
			loginResp := loginUser(t, ctx, testMux, mockUserRepo)
			mockUserRepo.EXPECT().Find(gomock.Any(), int64(1)).Return(&user_entity.User{
				ID:       1,
				Username: "test",
				Status:   consts.ACTIVE,
			}, nil)
			resp := &api.AuditResponse{}
			err := testMux.Do(ctx, &api.AuditRequest{}, resp, muxclient.WithHeader(http.Header{
				"Cookie": []string{"access_token=" + loginResp.AccessToken},
			}))
			assert.NoError(t, err)
		})
	})
}
