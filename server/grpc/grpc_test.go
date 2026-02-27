package grpc

import (
	"context"
	"testing"

	"github.com/cago-frame/cago/configs"
	"github.com/cago-frame/cago/configs/memory"
	"google.golang.org/grpc"
)

func TestGRPC(t *testing.T) {
	cfg, err := configs.NewConfig("grpc-test", configs.WithSource(memory.NewSource(map[string]interface{}{
		"grpc": map[string]interface{}{
			"address": "127.0.0.1:0",
		},
	})))
	if err != nil {
		t.Fatal("failed to create config: ", err)
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	_ = GRPC(func(ctx context.Context, s *grpc.Server) error {
		return nil
	}).Start(ctx, cfg)
}
