package rpc

import (
	"context"

	"github.com/cago-frame/cago/examples/simple/internal/rpc/user_rpc"
	pb "github.com/cago-frame/cago/examples/simple/pkg/proto/user"
	"google.golang.org/grpc"
)

// Register gRPC服务注册
func Register(ctx context.Context, s *grpc.Server) error {
	pb.RegisterUserServiceServer(s, user_rpc.NewUserService())
	return nil
}
