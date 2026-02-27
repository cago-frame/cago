package user_rpc

import (
	"context"

	"github.com/cago-frame/cago/examples/simple/internal/repository/user_repo"
	pb "github.com/cago-frame/cago/examples/simple/pkg/proto/user"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

type UserService struct {
	pb.UnimplementedUserServiceServer
}

func NewUserService() *UserService {
	return &UserService{}
}

// GetUser 根据用户ID获取用户信息
func (s *UserService) GetUser(ctx context.Context, req *pb.GetUserRequest) (*pb.GetUserResponse, error) {
	user, err := user_repo.User().Find(ctx, req.Id)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "查询用户失败: %v", err)
	}
	if user == nil {
		return nil, status.Error(codes.NotFound, "用户不存在")
	}
	return &pb.GetUserResponse{
		Id:       user.ID,
		Username: user.Username,
	}, nil
}
